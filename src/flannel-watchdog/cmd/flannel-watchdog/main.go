package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"lib/flannel"
	"os"
	"os/exec"
	"regexp"
	"time"

	"code.cloudfoundry.org/lager"

	"flannel-watchdog/config"

	"github.com/cloudfoundry/dropsonde"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

const dropsondeOrigin = "flannel-watchdog"

var ipAddrParseRegex = regexp.MustCompile(`((?:[0-9]{1,3}\.){3}[0-9]{1,3}/24)`)

type Runner struct {
	SubnetFile string
	BridgeName string
	Logger     lager.Logger
}

func (r *Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)

	errCh := make(chan error)
	go func() {
		found := false
		for {
			time.Sleep(1 * time.Second)

			localSubnetter := flannel.NetworkInfo{
				FlannelSubnetFilePath: r.SubnetFile,
			}
			flannelIP, _, err := localSubnetter.DiscoverNetworkInfo()
			if err != nil {
				errCh <- fmt.Errorf("discovering flannel subnet: %s", err)
			}

			output, err := exec.Command("ip", "addr", "show", "dev", r.BridgeName).CombinedOutput()
			if err != nil {
				r.Logger.Info("no bridge device found")
				found = false
				continue
			}

			matches := ipAddrParseRegex.FindStringSubmatch(string(output))
			if len(matches) < 2 {
				errCh <- fmt.Errorf(`device '%s' has no ip`, r.BridgeName)
				return
			}

			deviceIP := matches[1]
			if !found {
				found = true
				r.Logger.Info("Found bridge", lager.Data{"name": r.BridgeName})
			}

			if flannelIP != deviceIP {
				metrics.SendValue("flannelDown", 1.0, "bool")
				errCh <- fmt.Errorf(`This cell must be recreated.  Flannel is out of sync with the local bridge. `+
					`flannel (%s): %s bridge (%s): %s`, r.SubnetFile, flannelIP, r.BridgeName, deviceIP)
				return
			}

			metrics.SendValue("flannelDown", 0.0, "bool")
		}
	}()

	select {
	case <-signals:
		return nil
	case err := <-errCh:
		return err
	}
}

func mainWithErr(logger lager.Logger) error {
	conf := &config.Config{}
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	config, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		return fmt.Errorf("reading config: %s", err)
	}

	err = json.Unmarshal(config, conf)
	if err != nil {
		return fmt.Errorf("unmarshaling config: %s", err)
	}

	err = dropsonde.Initialize(conf.MetronAddress, dropsondeOrigin)
	if err != nil {
		return fmt.Errorf("initializing dropsonde: %s", err)
	}

	runner := &Runner{
		SubnetFile: conf.FlannelSubnetFile,
		BridgeName: conf.BridgeName,
		Logger:     logger,
	}
	members := grouper.Members{{"runner", runner}}
	group := grouper.NewOrdered(os.Interrupt, members)
	monitor := ifrit.Invoke(sigmon.New(group))

	err = <-monitor.Wait()
	return err
}

func main() {
	logger := lager.NewLogger("container-networking.flannel-watchdog")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))
	logger.Info("starting")

	if err := mainWithErr(logger); err != nil {
		logger.Error("fatal", err)
		os.Exit(1)
	}
}
