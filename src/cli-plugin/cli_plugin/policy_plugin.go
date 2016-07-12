package cli_plugin

import (
	"errors"
	"flag"
	"fmt"
	"lib/marshal"
	"log"
	"netman-agent/models"
	"os"
	"strconv"

	"github.com/cloudfoundry/cli/plugin"
)

type Plugin struct {
	Marshaler marshal.Marshaler
}

const AllowCommand = "allow-access"

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
					Usage: "cf allow-access SOURCE_APP DESTINATION_APP --protocol <tcp|udp> --port [1-65535]",
				},
			},
		},
	}
}

func (p *Plugin) RunWithErrors(cliConnection plugin.CliConnection, args []string) error {
	if len(args) < 2 {
		return errors.New("not enough arguments")
	}
	srcAppName := args[1]
	dstAppName := args[2]

	srcAppModel, err := cliConnection.GetApp(srcAppName)
	if err != nil {
		return fmt.Errorf("resolving source app: %s", err)
	}
	if srcAppModel.Guid == "" {
		return fmt.Errorf("resolving source app: %s not found", srcAppName)
	}

	dstAppModel, err := cliConnection.GetApp(dstAppName)
	if err != nil {
		return fmt.Errorf("resolving destination app: %s", err)
	}
	if dstAppModel.Guid == "" {
		return fmt.Errorf("resolving destination app: %s not found", dstAppName)
	}

	flags := flag.NewFlagSet("cf allow-policy <src> <dest>", flag.ContinueOnError)
	protocol := flags.String("protocol", "", "the protocol allowed")
	portString := flags.String("port", "", "the destination port")
	flags.Parse(args[3:])

	if *protocol == "" {
		return fmt.Errorf("Requires --protocol PROTOCOL as argument.")
	}

	if *portString == "" {
		return fmt.Errorf("Requires --port PORT as argument.")
	}

	port, err := strconv.Atoi(*portString)
	if err != nil {
		return fmt.Errorf("port is not valid: %s", *portString)
	}

	policy := models.Policy{
		Source: models.Source{
			ID: srcAppModel.Guid,
		},
		Destination: models.Destination{
			ID:       dstAppModel.Guid,
			Protocol: *protocol,
			Port:     port,
		},
	}

	var policies = struct {
		Policies []models.Policy `json:"policies"`
	}{
		[]models.Policy{policy},
	}

	payload, err := p.Marshaler.Marshal(policies)
	if err != nil {
		return fmt.Errorf("payload cannot be marshaled: %s", err)
	}

	_, err = cliConnection.CliCommand("curl", "-X", "POST", "/networking/v0/external/policies", "-d", "'"+string(payload)+"'")
	if err != nil {
		return fmt.Errorf("policy creation failed: %s", err)
	}

	return nil
}

func (p *Plugin) Run(cliConnection plugin.CliConnection, args []string) {
	logger := log.New(os.Stdout, "", 0)

	err := p.RunWithErrors(cliConnection, args)
	if err != nil {
		logger.Fatalf("%s", err)
	}
}
