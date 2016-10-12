package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"lib/flannel"
	"lib/metrics"
	"lib/mutualtls"
	"lib/policy_client"
	"lib/poller"
	"lib/rules"
	"log"
	"net/http"
	"os"
	"time"
	"vxlan-policy-agent/config"
	"vxlan-policy-agent/planner"

	"code.cloudfoundry.org/garden/client"
	"code.cloudfoundry.org/garden/client/connection"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/dropsonde"
	"github.com/coreos/go-iptables/iptables"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

const (
	dropsondeOrigin = "vxlan-policy-agent"
	emitInterval    = 30 * time.Second
)

func die(logger lager.Logger, action string, err error) {
	logger.Error(action, err)
	os.Exit(1)
}

func main() {
	conf := &config.VxlanPolicyAgent{}

	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	logger := lager.NewLogger("vxlan-policy-agent")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	configBytes, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		die(logger, "error reading config", err)
	}

	err = json.Unmarshal(configBytes, conf)
	if err != nil {
		die(logger, "error unmarshalling config", err)
	}
	logger.Info("parsed-config", lager.Data{"config": conf})

	pollInterval := time.Duration(conf.PollInterval) * time.Second
	if pollInterval == 0 {
		pollInterval = time.Second
	}

	flannelInfoReader := &flannel.NetworkInfo{
		FlannelSubnetFilePath: conf.FlannelSubnetFile,
	}
	localSubnetCIDR, overlayNetwork, err := flannelInfoReader.DiscoverNetworkInfo()
	if err != nil {
		die(logger, "discovering network info", err)
	}

	clientTLSConfig, err := mutualtls.NewClientTLSConfig(conf.ClientCertFile, conf.ClientKeyFile, conf.ServerCACertFile)
	if err != nil {
		die(logger, "mutal tls config", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: clientTLSConfig,
		},
	}

	policyClient := policy_client.NewInternal(
		logger.Session("policy-client"),
		httpClient,
		conf.PolicyServerURL,
	)

	gardenClient := client.New(connection.New(conf.GardenProtocol, conf.GardenAddress))

	ipt, err := iptables.New()
	if err != nil {
		die(logger, "iptables-new", err)
	}

	dynamicPlanner := &planner.VxlanPolicyPlanner{
		GardenClient: gardenClient,
		PolicyClient: policyClient,
		Logger:       logger.Session("rules-updater"),
		VNI:          conf.VNI,
	}

	timestamper := &rules.Timestamper{}
	ruleEnforcer := rules.NewEnforcer(
		logger.Session("rules-enforcer"),
		timestamper,
		ipt,
	)

	dynamicChain := rules.Chain{
		Table:       "filter",
		ParentChain: "FORWARD",
		Prefix:      "vpa--",
	}

	vxlanDefaultLocalPlanner := planner.VxlanDefaultLocalPlanner{
		Logger:      logger,
		LocalSubnet: localSubnetCIDR,
		Chain: rules.Chain{
			Table:       "filter",
			ParentChain: "FORWARD",
			Prefix:      "vpa--local-",
		},
	}

	vxlanDefaultRemotePlanner := planner.VxlanDefaultRemotePlanner{
		Logger: logger,
		VNI:    conf.VNI,
		Chain: rules.Chain{
			Table:       "filter",
			ParentChain: "FORWARD",
			Prefix:      "vpa--remote-",
		},
	}

	vxlanDefaultMasqueradePlanner := planner.VxlanDefaultMasqueradePlanner{
		Logger:         logger,
		LocalSubnet:    localSubnetCIDR,
		OverlayNetwork: overlayNetwork,
		Chain: rules.Chain{
			Table:       "nat",
			ParentChain: "POSTROUTING",
			Prefix:      "vpa--masq-",
		},
	}

	defaultLocalStuff, err := vxlanDefaultLocalPlanner.GetRulesAndChain()
	if err != nil {
		die(logger, "default-local-rules.GetRules", err)
	}

	err = ruleEnforcer.EnforceRulesAndChain(defaultLocalStuff)
	if err != nil {
		die(logger, "enforce-default-local", err)
	}

	defaultRemoteStuff, err := vxlanDefaultRemotePlanner.GetRulesAndChain()
	if err != nil {
		die(logger, "default-local-rules.GetRules", err)
	}
	err = ruleEnforcer.EnforceRulesAndChain(defaultRemoteStuff)
	if err != nil {
		die(logger, "enforce-default-remote", err)
	}

	defaultMasqueradeStuff, err := vxlanDefaultMasqueradePlanner.GetRulesAndChain()
	if err != nil {
		die(logger, "default-masquerade-rules.GetRules", err)
	}

	err = ruleEnforcer.EnforceRulesAndChain(defaultMasqueradeStuff)
	if err != nil {
		die(logger, "enforce-default-masquerade", err)
	}

	err = dropsonde.Initialize(conf.MetronAddress, dropsondeOrigin)
	if err != nil {
		log.Fatalf("initializing dropsonde: %s", err)
	}

	uptimeSource := metrics.NewUptimeSource()
	metricsEmitter := metrics.NewMetricsEmitter(logger, emitInterval, uptimeSource)

	policyPoller := &poller.Poller{
		Logger:       logger,
		PollInterval: pollInterval,
		Planner:      dynamicPlanner,

		Chain:    dynamicChain,
		Enforcer: ruleEnforcer,
	}

	members := grouper.Members{
		{"metrics_emitter", metricsEmitter},
		{"policy_poller", policyPoller},
	}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))
	logger.Info("starting")
	err = <-monitor.Wait()
	if err != nil {
		die(logger, "ifrit monitor", err)
	}
}
