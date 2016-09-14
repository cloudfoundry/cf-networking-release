package poller

import (
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

func readTxBytes(logger lager.Logger, ifName string) (int, error) {
	cmd := exec.Command("cat", filepath.Join("/sys/class/net/", ifName, "/statistics/tx_bytes"))
	txBytesData, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("failed-reading-txbytes-file", err)
		return 0, err
	}

	trimmedString := strings.TrimSpace(string(txBytesData))
	nBytes, err := strconv.Atoi(trimmedString)
	if err != nil {
		logger.Error("txbytes-could-not-be-converted-to-int", err)
		return 0, err
	}

	return nBytes, nil
}

func (m *SystemMetrics) measure(logger lager.Logger) {
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

	nTxBytes, err := readTxBytes(logger, m.InterfaceName)
	if err != nil {
		logger.Error("read-tx-bytes", err)
		return
	}

	if err := overlayTxBytes.Send(nTxBytes); err != nil {
		logger.Error("failed-to-send-metric", err, lager.Data{
			"metric": overlayTxBytes})
		return
	}
}
