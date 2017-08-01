package cli_plugin

import (
	"bytes"
	"cli-plugin/styles"
	"flag"
	"fmt"
	"io/ioutil"
	"lib/policy_client"
	"log"
	"policy-server/api/api_v0"
	"text/tabwriter"

	"code.cloudfoundry.org/cli/plugin"
	"policy-server/api"
)

type CommandRunner struct {
	Styler        *styles.StyleGroup
	Logger        *log.Logger
	PolicyClient  policy_client.ExternalPolicyClient
	CliConnection plugin.CliConnection
	Args          []string
}

func (r *CommandRunner) List() (string, error) {
	err := validateUsage(r.CliConnection, r.Args)
	if err != nil {
		return "", err
	}

	username, err := r.CliConnection.Username()
	if err != nil {
		return "", fmt.Errorf("could not resolve username: %s", err)
	}

	r.Logger.Printf(r.Styler.ApplyStyles(
		"Listing policies as " + r.Styler.AddStyle(username, "cyan") + "..."))

	accessToken, err := r.CliConnection.AccessToken()
	if err != nil {
		return "", fmt.Errorf("getting access token: %s", err)
	}

	flags := flag.NewFlagSet("cf list-access", flag.ContinueOnError)
	appName := flags.String("app", "", "app name to filter results")
	flags.Parse(r.Args[1:])

	var appGuid string
	if *appName != "" {
		app, err := r.CliConnection.GetApp(*appName)
		if err != nil {
			return "", fmt.Errorf("getting app: %s", err)
		}
		appGuid = app.Guid
	}

	var policies []api_v0.Policy
	if appGuid != "" {
		var err error
		policies, err = r.PolicyClient.GetPoliciesByID(accessToken, appGuid)
		if err != nil {
			return "", fmt.Errorf("getting policies by id: %s", err)
		}
	} else {
		var err error
		policies, err = r.PolicyClient.GetPolicies(accessToken)
		if err != nil {
			return "", fmt.Errorf("getting policies: %s", err)
		}
	}

	apps, err := r.CliConnection.GetApps()
	if err != nil {
		return "", fmt.Errorf("getting apps: %s", err)
	}

	buffer := &bytes.Buffer{}
	tabWriter := tabwriter.NewWriter(buffer, 0, 8, 2, '\t', tabwriter.FilterHTML)
	fmt.Fprintf(tabWriter, r.Styler.AddStyle("Source\tDestination\tProtocol\tPort\n", "bold"))

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
				r.Styler.AddStyle(srcName, "cyan"),
				r.Styler.AddStyle(dstName, "cyan"),
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

func (r *CommandRunner) Allow() (string, error) {
	err := validateUsage(r.CliConnection, r.Args)
	if err != nil {
		return "", err
	}

	validArgs, err := ValidateArgs(r.CliConnection, r.Args)
	if err != nil {
		return "", err
	}

	username, err := r.CliConnection.Username()
	if err != nil {
		return "", fmt.Errorf("could not resolve username: %s", err)
	}

	r.Logger.Printf(r.Styler.ApplyStyles(
		"Allowing traffic from " + r.Styler.AddStyle(validArgs.SourceAppName, "cyan") +
			" to " + r.Styler.AddStyle(validArgs.DestAppName, "cyan") +
			" as " + r.Styler.AddStyle(username, "cyan") + "..."))

	token, err := r.CliConnection.AccessToken()
	if err != nil {
		return "", fmt.Errorf("getting access token: %s", err)
	}

	if validArgs.StartPort > 0 {
		policy, err := r.constructPolicy(validArgs)
		if err != nil {
			return "", err
		}
		err = r.PolicyClient.AddPolicies(token, []api.Policy{policy})
		if err != nil {
			return "", fmt.Errorf("adding policies: %s", err)
		}
	} else {
		policy, err := r.constructPolicyV0(validArgs)
		if err != nil {
			return "", err
		}

		err = r.PolicyClient.AddPoliciesV0(token, []api_v0.Policy{policy})
		if err != nil {
			return "", fmt.Errorf("adding policies: %s", err)
		}
	}

	return "", nil
}

func (r *CommandRunner) Remove() (string, error) {
	err := validateUsage(r.CliConnection, r.Args)
	if err != nil {
		return "", err
	}

	validArgs, err := ValidateArgs(r.CliConnection, r.Args)
	if err != nil {
		return "", err
	}

	username, err := r.CliConnection.Username()
	if err != nil {
		return "", fmt.Errorf("could not resolve username: %s", err)
	}

	r.Logger.Printf(r.Styler.ApplyStyles(
		"Denying traffic from " + r.Styler.AddStyle(validArgs.SourceAppName, "cyan") +
			" to " + r.Styler.AddStyle(validArgs.DestAppName, "cyan") +
			" as " + r.Styler.AddStyle(username, "cyan") + "..."))

	policy, err := r.constructPolicyV0(validArgs)
	if err != nil {
		return "", err
	}

	accessToken, err := r.CliConnection.AccessToken()
	if err != nil {
		return "", fmt.Errorf("getting access token: %s", err)
	}

	err = r.PolicyClient.DeletePolicies(accessToken, []api_v0.Policy{policy})
	if err != nil {
		return "", fmt.Errorf("deleting policies: %s", err)
	}

	return "", nil
}

func (r *CommandRunner) constructPolicyV0(validArgs ValidArgs) (api_v0.Policy, error) {
	srcAppModel, err := r.CliConnection.GetApp(validArgs.SourceAppName)
	if err != nil {
		return api_v0.Policy{}, fmt.Errorf("resolving source app: %s", err)
	}
	if srcAppModel.Guid == "" {
		return api_v0.Policy{}, fmt.Errorf("resolving source app: %s not found", validArgs.SourceAppName)
	}
	dstAppModel, err := r.CliConnection.GetApp(validArgs.DestAppName)
	if err != nil {
		return api_v0.Policy{}, fmt.Errorf("resolving destination app: %s", err)
	}
	if dstAppModel.Guid == "" {
		return api_v0.Policy{}, fmt.Errorf("resolving destination app: %s not found", validArgs.DestAppName)
	}

	return api_v0.Policy{
		Source: api_v0.Source{
			ID: srcAppModel.Guid,
		},
		Destination: api_v0.Destination{
			ID:       dstAppModel.Guid,
			Protocol: validArgs.Protocol,
			Port:     validArgs.Port,
		},
	}, nil
}
func (r *CommandRunner) constructPolicy(validArgs ValidArgs) (api.Policy, error) {
	srcAppModel, err := r.CliConnection.GetApp(validArgs.SourceAppName)
	if err != nil {
		return api.Policy{}, fmt.Errorf("resolving source app: %s", err)
	}
	if srcAppModel.Guid == "" {
		return api.Policy{}, fmt.Errorf("resolving source app: %s not found", validArgs.SourceAppName)
	}
	dstAppModel, err := r.CliConnection.GetApp(validArgs.DestAppName)
	if err != nil {
		return api.Policy{}, fmt.Errorf("resolving destination app: %s", err)
	}
	if dstAppModel.Guid == "" {
		return api.Policy{}, fmt.Errorf("resolving destination app: %s not found", validArgs.DestAppName)
	}

	return api.Policy{
		Source: api.Source{
			ID: srcAppModel.Guid,
		},
		Destination: api.Destination{
			ID:       dstAppModel.Guid,
			Protocol: validArgs.Protocol,
			Ports:    api.Ports{Start: validArgs.StartPort, End: validArgs.FinishPort },
		},
	}, nil
}
