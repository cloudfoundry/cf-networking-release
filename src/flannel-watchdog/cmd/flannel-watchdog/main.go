package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
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

	checkError := make(chan error)
	go func() {
		for {
			file, err := os.Open(r.SubnetFile)
			if err != nil {
				checkError <- err
				return
			}

			var flannelIP string
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				trimmedLine := strings.TrimPrefix(line, "FLANNEL_SUBNET=")
				if line != trimmedLine {
					flannelIP = trimmedLine
				}
			}
			file.Close()

			output, err := exec.Command("ip", "addr", "show", "dev", r.BridgeName).CombinedOutput()
			if err != nil {
				checkError <- fmt.Errorf("%s: %s", err, string(output))
				return
			}

			matches := regexp.MustCompile(ipAddrParseRegex).FindStringSubmatch(string(output))
			if len(matches) < 2 {
				checkError <- fmt.Errorf(`device "%s" has no ip`, r.BridgeName)
				return
			}

			if flannelIP != matches[1] {
				checkError <- errors.New("out of sync")
				return
			}
			time.Sleep(1 * time.Second)
		}
	}()

	select {
	case <-signals:
		return nil
	case err := <-checkError:
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
