package cli_plugin

import (
	"bytes"
	"cli-plugin/styles"
	"flag"
	"fmt"
	"io/ioutil"
	"lib/marshal"
	"lib/models"
	"lib/policy_client"
	"log"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/cloudfoundry/cli/plugin"
)

type Plugin struct {
	Marshaler    marshal.Marshaler
	Unmarshaler  marshal.Unmarshaler
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

const AllowCommand = "access-allow"
const ListCommand = "access-list"
const DenyCommand = "access-deny"

var ListUsageRegex = fmt.Sprintf(`\A%s\s*(--app(\s+|=)\S+\z|\z)`, ListCommand)
var AllowUsageRegex = fmt.Sprintf(`\A%s\s+\S+\s+\S+\s+(--|-)\w+(\s+|=)\w+\s+(--|-)\w+(\s+|=)\w+\z`, AllowCommand)
var DenyUsageRegex = fmt.Sprintf(`\A%s\s+\S+\s+\S+\s+(--|-)\w+(\s+|=)\w+\s+(--|-)\w+(\s+|=)\w+\z`, DenyCommand)

func (p *Plugin) GetMetadata() plugin.PluginMetadata {
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
					Usage: fmt.Sprintf("cf %s SOURCE_APP DESTINATION_APP --protocol <tcp|udp> --port [1-65535]", AllowCommand),
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
				Name:     DenyCommand,
				HelpText: "Remove direct network traffic from one app to another",
				UsageDetails: plugin.Usage{
					Usage: fmt.Sprintf("cf %s SOURCE_APP DESTINATION_APP --protocol <tcp|udp> --port [1-65535]", DenyCommand),
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
	switch args[0] {
	case AllowCommand:
		return p.AllowCommand(cliConnection, args)
	case ListCommand:
		return p.ListCommand(cliConnection, args)
	case DenyCommand:
		return p.DenyCommand(cliConnection, args)
	}

	return "", nil
}

func (p *Plugin) ListCommand(cliConnection plugin.CliConnection, args []string) (string, error) {
	err := validateUsage(cliConnection, ListUsageRegex, args)
	if err != nil {
		return "", err
	}

	username, err := cliConnection.Username()
	if err != nil {
		return "", fmt.Errorf("could not resolve username: %s", err)
	}
	p.Logger.Printf(p.Styler.ApplyStyles("Listing policies as " + p.Styler.AddStyle(username, "cyan") + "..."))

	accessToken, err := cliConnection.AccessToken()
	if err != nil {
		return "", fmt.Errorf("getting access token: %s", err)
	}

	flags := flag.NewFlagSet("cf list-policies", flag.ContinueOnError)
	appName := flags.String("app", "", "app name to filter results")
	flags.Parse(args[1:])

	var appGuid string
	if *appName != "" {
		app, err := cliConnection.GetApp(*appName)
		if err != nil {
			return "", fmt.Errorf("getting app: %s", err)
		}
		appGuid = app.Guid
	}

	var policies []models.Policy
	if appGuid != "" {
		var err error
		policies, err = p.PolicyClient.GetPoliciesByID(accessToken, appGuid)
		if err != nil {
			return "", fmt.Errorf("getting policies by id: %s", err)
		}
	} else {
		var err error
		policies, err = p.PolicyClient.GetPolicies(accessToken)
		if err != nil {
			return "", fmt.Errorf("getting policies: %s", err)
		}
	}

	apps, err := cliConnection.GetApps()
	if err != nil {
		return "", fmt.Errorf("getting apps: %s", err)
	}

	buffer := &bytes.Buffer{}
	tabWriter := tabwriter.NewWriter(buffer, 0, 8, 2, '\t', tabwriter.FilterHTML)
	fmt.Fprintf(tabWriter, p.Styler.AddStyle("Source\tDestination\tProtocol\tPort\n", "bold"))

	for _, policy := range policies {
		srcName := ""
		dstName := ""
		for _, app := range apps {
			if policy.Source.ID == app.Guid {
				srcName = app.Name
			}
			if policy.Destination.ID == app.Guid {
				dstName = app.Name
			}
		}
		if srcName != "" && dstName != "" {
			fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%d\n",
				p.Styler.AddStyle(srcName, "cyan"),
				p.Styler.AddStyle(dstName, "cyan"),
				policy.Destination.Protocol,
				policy.Destination.Port,
			)
		}
	}

	tabWriter.Flush()
	outBytes, err := ioutil.ReadAll(buffer)
	if err != nil {
		//untested
		return "", fmt.Errorf("formatting output: %s", err)
	}

	return string(outBytes), nil
}

func (p *Plugin) AllowCommand(cliConnection plugin.CliConnection, args []string) (string, error) {
	err := validateUsage(cliConnection, AllowUsageRegex, args)
	if err != nil {
		return "", err
	}

	validArgs, err := ValidateArgs(cliConnection, args)
	if err != nil {
		return "", err
	}

	username, err := cliConnection.Username()
	if err != nil {
		return "", fmt.Errorf("could not resolve username: %s", err)
	}
	p.Logger.Printf(p.Styler.ApplyStyles(
		"Allowing traffic from " + p.Styler.AddStyle(validArgs.SourceAppName, "cyan") +
			" to " + p.Styler.AddStyle(validArgs.DestAppName, "cyan") +
			" as " + p.Styler.AddStyle(username, "cyan") + "..."))

	srcAppModel, err := cliConnection.GetApp(validArgs.SourceAppName)
	if err != nil {
		return "", fmt.Errorf("resolving source app: %s", err)
	}
	if srcAppModel.Guid == "" {
		return "", fmt.Errorf("resolving source app: %s not found", validArgs.SourceAppName)
	}
	dstAppModel, err := cliConnection.GetApp(validArgs.DestAppName)
	if err != nil {
		return "", fmt.Errorf("resolving destination app: %s", err)
	}
	if dstAppModel.Guid == "" {
		return "", fmt.Errorf("resolving destination app: %s not found", validArgs.DestAppName)
	}

	policy := models.Policy{
		Source: models.Source{
			ID: srcAppModel.Guid,
		},
		Destination: models.Destination{
			ID:       dstAppModel.Guid,
			Protocol: validArgs.Protocol,
			Port:     validArgs.Port,
		},
	}

	token, err := cliConnection.AccessToken()
	if err != nil {
		return "", fmt.Errorf("getting access token: %s", err)
	}

	err = p.PolicyClient.AddPolicies(token, []models.Policy{policy})
	if err != nil {
		return "", fmt.Errorf("adding policies: %s", err)
	}

	return "", nil
}

func (p *Plugin) DenyCommand(cliConnection plugin.CliConnection, args []string) (string, error) {
	err := validateUsage(cliConnection, DenyUsageRegex, args)
	if err != nil {
		return "", err
	}

	validArgs, err := ValidateArgs(cliConnection, args)
	if err != nil {
		return "", err
	}

	username, err := cliConnection.Username()
	if err != nil {
		return "", fmt.Errorf("could not resolve username: %s", err)
	}
	p.Logger.Printf(p.Styler.ApplyStyles(
		"Denying traffic from " + p.Styler.AddStyle(validArgs.SourceAppName, "cyan") +
			" to " + p.Styler.AddStyle(validArgs.DestAppName, "cyan") +
			" as " + p.Styler.AddStyle(username, "cyan") + "..."))

	srcAppModel, err := cliConnection.GetApp(validArgs.SourceAppName)
	if err != nil {
		return "", fmt.Errorf("resolving source app: %s", err)
	}
	if srcAppModel.Guid == "" {
		return "", fmt.Errorf("resolving source app: %s not found", validArgs.SourceAppName)
	}
	dstAppModel, err := cliConnection.GetApp(validArgs.DestAppName)
	if err != nil {
		return "", fmt.Errorf("resolving destination app: %s", err)
	}
	if dstAppModel.Guid == "" {
		return "", fmt.Errorf("resolving destination app: %s not found", validArgs.DestAppName)
	}

	policy := models.Policy{
		Source: models.Source{
			ID: srcAppModel.Guid,
		},
		Destination: models.Destination{
			ID:       dstAppModel.Guid,
			Protocol: validArgs.Protocol,
			Port:     validArgs.Port,
		},
	}

	accessToken, err := cliConnection.AccessToken()
	if err != nil {
		return "", fmt.Errorf("getting access token: %s", err)
	}

	err = p.PolicyClient.DeletePolicies(accessToken, []models.Policy{policy})
	if err != nil {
		return "", fmt.Errorf("deleting policies: %s", err)
	}

	return "", nil
}

func validateUsage(cliConnection plugin.CliConnection, regex string, args []string) error {
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
