package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"lib/flannel"
	"net/http"
	"os"
	"regexp"
	"time"

	"code.cloudfoundry.org/lager"

	"flannel-watchdog/config"
	"flannel-watchdog/validator"

	"github.com/cloudfoundry/dropsonde"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

const dropsondeOrigin = "flannel-watchdog"

var ipAddrParseRegex = regexp.MustCompile(`((?:[0-9]{1,3}\.){3}[0-9]{1,3}/[0-9]{1,2})`)

type ipValidator interface {
	Validate(string) error
}

type Runner struct {
	SubnetFile string
	Logger     lager.Logger
	Validator  ipValidator
}

func (r *Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)

	errCh := make(chan error)
	go func() {
		for {
			time.Sleep(1 * time.Second)

			localSubnetter := flannel.NetworkInfo{
				FlannelSubnetFilePath: r.SubnetFile,
			}
			flannelIP, _, err := localSubnetter.DiscoverNetworkInfo()
			if err != nil {
				errCh <- fmt.Errorf("discovering flannel subnet: %s", err)
				return
			}

			err = r.Validator.Validate(flannelIP)
			if err != nil {
				fmt.Println(metrics.SendValue("flannelDown", 1.0, "bool"))
				errCh <- err
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

	var ipValidator ipValidator
	if conf.NoBridge {
		ipValidator = &validator.NoBridge{
			Logger:           logger,
			MetadataFileName: conf.MetadataFilename,
		}
	} else {
		ipValidator = &validator.Bridge{
			Logger:         logger,
			BridgeName:     conf.BridgeName,
			NetlinkAdapter: &validator.NetlinkAdapter{},
		}
	}

	runner := &Runner{
		SubnetFile: conf.FlannelSubnetFile,
		Logger:     logger,
		Validator:  ipValidator,
	}

	server := http_server.New(fmt.Sprintf("127.0.0.1:%d", conf.HealthCheckPort), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("The cell is healthy! The cell is configured with the correct subnet."))
	}))

	members := grouper.Members{
		{"runner", runner},
		{"server", server},
	}
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
