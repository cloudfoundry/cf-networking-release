package main

import (
	"cli-plugin/cli_plugin"
	"cli-plugin/styles"
	"log"
	"os"

	"code.cloudfoundry.org/cli/plugin"
)

func main() {
	plugin.Start(&cli_plugin.Plugin{
		Styler: styles.NewGroup(),
		Logger: log.New(os.Stdout, "", 0),
	})
}
