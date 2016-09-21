package poller

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/runtimeschema/metric"
)

const netInterfaceCount = metric.Metric("NetInterfaceCount")
const iptablesRuleCount = metric.Metric("IPTablesRuleCount")
const overlayTxBytes = metric.Metric("OverlayTxBytes")
const overlayRxBytes = metric.Metric("OverlayRxBytes")

type SystemMetrics struct {
	Logger        lager.Logger
	PollInterval  time.Duration
	InterfaceName string
}

func (m *SystemMetrics) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)
	for {
		select {
		case <-signals:
			return nil
		case <-time.After(m.PollInterval):
			m.measure(m.Logger.Session("measure"))
		}
	}
}

func countNetworkInterfaces() (int, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return 0, err
	}
	return len(ifaces), nil
}

func lineCount(data []byte) int {
	lines := strings.Split(string(data), "\n")
	counter := 0
	for _, line := range lines {
		if len(strings.TrimSpace(line)) > 0 {
			counter++
		}
	}
	return counter
}

func countIPTablesRules(logger lager.Logger) (int, error) {
	cmd := exec.Command("iptables", "-w", "-S")
	filterRules, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("failed-getting-filter-rules", err)
		return 0, err
	}

	cmd = exec.Command("iptables", "-w", "-t", "nat", "-S")
	natRules, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("failed-getting-nat-rules", err)
		return 0, err
	}

	return lineCount(filterRules) + lineCount(natRules), nil
}

func readStatsFile(ifName, stat string) (int, error) {
	txBytesData, err := ioutil.ReadFile(filepath.Join("/sys/class/net/", ifName, "/statistics/", stat))
	if err != nil {
		return 0, fmt.Errorf("failed reading txbytes file: %s", err)
	}

	trimmedString := strings.TrimSpace(string(txBytesData))
	nBytes, err := strconv.Atoi(trimmedString)
	if err != nil {
		return 0, fmt.Errorf("txbytes could not be converted to int: %s", err)
	}

	return nBytes, nil
}

func (m *SystemMetrics) measure(logger lager.Logger) {
	logger.Debug("measure-start")
	defer logger.Debug("measure-complete")

	nInterfaces, err := countNetworkInterfaces()
	if err != nil {
		logger.Error("count-network-interfaces", err)
		return
	}

	if err := netInterfaceCount.Send(nInterfaces); err != nil {
		logger.Error("failed-to-send-metric", err, lager.Data{
			"metric": netInterfaceCount})
		return
	}
	logger.Debug("metric-sent", lager.Data{"NetInterfaceCount": nInterfaces})

	nIpTablesRule, err := countIPTablesRules(logger)
	if err != nil {
		logger.Error("count-iptables-rules", err)
		return
	}

	if err := iptablesRuleCount.Send(nIpTablesRule); err != nil {
		logger.Error("failed-to-send-metric", err, lager.Data{
			"metric": iptablesRuleCount})
		return
	}
	logger.Debug("metric-sent", lager.Data{"IPTablesRuleCount": nIpTablesRule})

	nTxBytes, err := readStatsFile(m.InterfaceName, "tx_bytes")
	if err != nil {
		logger.Error("read-tx-bytes", err)
		return
	}

	if err := overlayTxBytes.Send(nTxBytes); err != nil {
		logger.Error("failed-to-send-metric", err, lager.Data{
			"metric": overlayTxBytes})
		return
	}
	logger.Debug("metric-sent", lager.Data{"OverlayTxBytes": nTxBytes})

	nRxBytes, err := readStatsFile(m.InterfaceName, "rx_bytes")
	if err != nil {
		logger.Error("read-rx-bytes", err)
		return
	}

	if err := overlayRxBytes.Send(nRxBytes); err != nil {
		logger.Error("failed-to-send-metric", err, lager.Data{
			"metric": overlayRxBytes})
		return
	}
	logger.Debug("metric-sent", lager.Data{"OverlayRxBytes": nRxBytes})
}
