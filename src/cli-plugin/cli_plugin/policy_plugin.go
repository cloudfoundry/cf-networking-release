package cli_plugin

import (
	"cli-plugin/styles"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"lib/policy_client"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/lager"
)

type Plugin struct {
	Styler       *styles.StyleGroup
	Logger       *log.Logger
	PolicyClient policy_client.ExternalPolicyClient
}

type ValidArgs struct {
	SourceAppName string
	DestAppName   string
	Protocol      string
	Port          int
}

const AllowCommand = "allow-access"
const ListCommand = "list-access"
const RemoveCommand = "remove-access"
const DenyCommand = "deny-access" //deprecated

var ListUsageRegex = fmt.Sprintf(`\A%s\s*(--app(\s+|=)\S+\z|\z)`, ListCommand)
var AllowUsageRegex = fmt.Sprintf(`\A%s\s+\S+\s+\S+\s+(--|-)\w+(\s+|=)\w+\s+(--|-)\w+(\s+|=)\w+\z`, AllowCommand)
var RemoveUsageRegex = fmt.Sprintf(`\A%s\s+\S+\s+\S+\s+(--|-)\w+(\s+|=)\w+\s+(--|-)\w+(\s+|=)\w+\z`, RemoveCommand)
var DenyUsageRegex = fmt.Sprintf(`\A%s\s+\S+\s+\S+\s+(--|-)\w+(\s+|=)\w+\s+(--|-)\w+(\s+|=)\w+\z`, DenyCommand)

const MinPort = 1
const MaxPort = 65535

func (p *Plugin) GetMetadata() plugin.PluginMetadata {
	const usageTemplate = "cf %s SOURCE_APP DESTINATION_APP --protocol <tcp|udp> --port <%d-%d>"

	return plugin.PluginMetadata{
		Name: "network-policy",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 0,
		},
		MinCliVersion: plugin.VersionType{
			Major: 6,
			Minor: 15,
		},
		Commands: []plugin.Command{
			plugin.Command{
				Name:     AllowCommand,
				HelpText: "Allow direct network traffic from one app to another",
				UsageDetails: plugin.Usage{
					Usage: fmt.Sprintf(usageTemplate, AllowCommand, MinPort, MaxPort),
					Options: map[string]string{
						"-protocol": "Protocol to connect apps with. (required)",
						"-port":     "Port to connect to destination app with. (required)",
					},
				},
			},
			plugin.Command{
				Name:     ListCommand,
				HelpText: "List policy for direct network traffic from one app to another",
				UsageDetails: plugin.Usage{
					Usage:   fmt.Sprintf("cf %s [--app appName]", ListCommand),
					Options: map[string]string{"-app": "Application to filter results by. (optional)"},
				},
			},
			plugin.Command{
				Name:     RemoveCommand,
				HelpText: "Remove policy and deny direct network traffic from one app to another",
				UsageDetails: plugin.Usage{
					Usage: fmt.Sprintf(usageTemplate, RemoveCommand, MinPort, MaxPort),
					Options: map[string]string{
						"-protocol": "Protocol to connect apps with. (required)",
						"-port":     "Port to connect to destination app with. (required)",
					},
				},
			},
			plugin.Command{
				Name:     DenyCommand,
				HelpText: "Deprecated! Use remove-access",
				UsageDetails: plugin.Usage{
					Usage: fmt.Sprintf(usageTemplate, RemoveCommand, MinPort, MaxPort),
					Options: map[string]string{
						"-protocol": "Protocol to connect apps with. (required)",
						"-port":     "Port to connect to destination app with. (required)",
					},
				},
			},
		},
	}
}

func (p *Plugin) Run(cliConnection plugin.CliConnection, args []string) {
	output, err := p.RunWithErrors(cliConnection, args)
	if err != nil {
		p.Logger.Printf(p.Styler.ApplyStyles(p.Styler.AddStyle("FAILED", "red")))
		p.Logger.Fatalf("%s", err)
	}

	p.Logger.Printf(p.Styler.ApplyStyles(p.Styler.AddStyle("OK\n", "green")))
	p.Logger.Print(p.Styler.ApplyStyles(output))
}

func (p *Plugin) RunWithErrors(cliConnection plugin.CliConnection, args []string) (string, error) {
	apiEndpoint, err := cliConnection.ApiEndpoint()
	if err != nil {
		return "", fmt.Errorf("getting api endpoint: %s", err)
	}

	skipSSL, err := cliConnection.IsSSLDisabled()
	if err != nil {
		return "", fmt.Errorf("checking if ssl disabled: %s", err)
	}

	tracingEnabled := (os.Getenv("CF_TRACE") == "true")

	httpTransport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 3 * time.Second,
		}).Dial,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipSSL,
		},
	}
	runner := &CommandRunner{
		Styler: p.Styler,
		Logger: p.Logger,
		PolicyClient: policy_client.NewExternal(
			lager.NewLogger("command"),
			&http.Client{Transport: wrapWithHTTPTracing(httpTransport, tracingEnabled)},
			apiEndpoint),
		CliConnection: cliConnection,
		Args:          args,
	}

	switch args[0] {
	case AllowCommand:
		return runner.Allow()
	case ListCommand:
		return runner.List()
	case RemoveCommand:
		return runner.Remove()
	case DenyCommand:
		return runner.Remove()
	}

	return "", nil
}

func validateUsage(cliConnection plugin.CliConnection, args []string) error {
	var regex string
	switch args[0] {
	case ListCommand:
		regex = ListUsageRegex
	case AllowCommand:
		regex = AllowUsageRegex
	case RemoveCommand:
		regex = RemoveUsageRegex
	case DenyCommand:
		regex = DenyUsageRegex
	default:
		return errors.New("Invalid command")
	}
	rx := regexp.MustCompile(regex)
	if !rx.MatchString(strings.Join(args, " ")) {
		return errorWithUsage("", args[0], cliConnection)
	}
	return nil
}

func ValidateArgs(cliConnection plugin.CliConnection, args []string) (ValidArgs, error) {
	validArgs := ValidArgs{}

	srcAppName := args[1]
	dstAppName := args[2]

	flags := flag.NewFlagSet("cf "+args[0]+" <src> <dest>", flag.ContinueOnError)
	protocol := flags.String("protocol", "", "the protocol allowed")
	portString := flags.String("port", "", "the destination port")
	err := flags.Parse(args[3:])
	if err != nil {
		return ValidArgs{}, errorWithUsage(err.Error(), args[0], cliConnection)
	}

	port, err := strconv.Atoi(*portString)
	if err != nil {
		return ValidArgs{}, errorWithUsage(fmt.Sprintf("Port is not valid: %s", *portString), args[0], cliConnection)
	}
	if port < MinPort || port > MaxPort {
		return ValidArgs{}, errorWithUsage(fmt.Sprintf("Port is not valid. Must be in range <%d-%d>.", MinPort, MaxPort), args[0], cliConnection)
	}

	if *protocol != "tcp" && *protocol != "udp" {
		return ValidArgs{}, errorWithUsage("Protocol is not valid. Must be tcp or udp.", args[0], cliConnection)
	}

	validArgs.SourceAppName = srcAppName
	validArgs.DestAppName = dstAppName
	validArgs.Protocol = *protocol
	validArgs.Port = port

	return validArgs, nil
}

func errorWithUsage(errorString, cmd string, cliConnection plugin.CliConnection) error {
	output, err := cliConnection.CliCommandWithoutTerminalOutput("help", cmd)
	if err != nil {
		return fmt.Errorf("cf cli error: %s", err)
	}
	return fmt.Errorf("Incorrect usage. %s\n\n%s", errorString, strings.Join(output, "\n"))
}
