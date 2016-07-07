package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"lib/flannel"
	"log"
	"os"
	"os/exec"
	"regexp"
	"time"

	"flannel-watchdog/config"

	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
)

const (
	ipAddrParseRegex = `((?:[0-9]{1,3}\.){3}[0-9]{1,3}/24)`
)

type Runner struct {
	SubnetFile string
	BridgeName string
}

func (r *Runner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)

	errCh := make(chan error)
	go func() {
		for {
			time.Sleep(1 * time.Second)

			localSubnetter := flannel.LocalSubnet{
				FlannelSubnetFilePath: r.SubnetFile,
			}
			flannelIP, err := localSubnetter.DiscoverLocalSubnet()
			if err != nil {
				errCh <- fmt.Errorf("discovering flannel subnet: %s", err)
			}

			output, err := exec.Command("ip", "addr", "show", "dev", r.BridgeName).CombinedOutput()
			if err != nil {
				fmt.Println("no bridge device found")
				continue
			}

			matches := regexp.MustCompile(ipAddrParseRegex).FindStringSubmatch(string(output))
			if len(matches) < 2 {
				errCh <- fmt.Errorf(`device "%s" has no ip`, r.BridgeName)
				return
			}

			deviceIP := matches[1]
			if flannelIP != deviceIP {
				errCh <- fmt.Errorf("out of sync: flannel subnet.net has %s but bridge device has %s", flannelIP, deviceIP)
				return
			}
		}
	}()

	select {
	case <-signals:
		return nil
	case err := <-errCh:
		return err
	}
}

func main() {
	fmt.Println("hello")

	conf := &config.Config{}

	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	configData, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		log.Fatal("error reading config")
	}

	err = json.Unmarshal(configData, conf)
	if err != nil {
		log.Fatal("error unmarshalling config")
	}

	runner := &Runner{
		SubnetFile: conf.FlannelSubnetFile,
		BridgeName: conf.BridgeName,
	}
	members := grouper.Members{
		{"runner", runner},
	}
	group := grouper.NewOrdered(os.Interrupt, members)
	monitor := ifrit.Invoke(group)

	err = <-monitor.Wait()
	if err != nil {
		log.Fatalf("%s\n", err)
	}
}
