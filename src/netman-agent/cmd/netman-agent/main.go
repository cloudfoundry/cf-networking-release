package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"lib/flannel"
	"log"
	"netman-agent/config"
	"netman-agent/planner"
	"netman-agent/rules"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/coreos/go-iptables/iptables"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

func main() {
	conf := &config.Config{}

	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	logger := lager.NewLogger("netman-agent")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	configBytes, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		log.Fatal("error reading config")
	}

	err = json.Unmarshal(configBytes, conf)
	if err != nil {
		log.Fatal("error unmarshalling config")
	}

	flannelInfoReader := &flannel.NetworkInfo{
		FlannelSubnetFilePath: conf.FlannelSubnetFile,
	}
	localSubnetCIDR, overlayNetwork, err := flannelInfoReader.DiscoverNetworkInfo()
	if err != nil {
		log.Fatalf("discovering network info: %s", err)
	}

	ipt, err := iptables.New()
	if err != nil {
		log.Fatal(err)
	}

	timestamper := &rules.Timestamper{}

	ruleEnforcer := rules.NewEnforcer(
		logger.Session("rules-enforcer"),
		timestamper,
		ipt,
	)

	defaultPlanner := planner.Planner{
		LocalSubnet:    localSubnetCIDR,
		OverlayNetwork: overlayNetwork,
		RuleEnforcer:   ruleEnforcer,
	}

	err = defaultPlanner.DefaultEgressRules()
	if err != nil {
		log.Fatal(err)
	}

	members := grouper.Members{
		{"noop", ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
			close(ready)
			<-signals
			return nil
		})},
	}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))
	err = <-monitor.Wait()
	if err != nil {
		log.Fatalf("daemon terminated: %s", err)
	}
}
