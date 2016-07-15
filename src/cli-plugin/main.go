package main

import (
	"cli-plugin/cli_plugin"
	"cli-plugin/styles"
	"encoding/json"
	"lib/marshal"

	"github.com/cloudfoundry/cli/plugin"
)

func main() {
	plugin.Start(&cli_plugin.Plugin{
		Marshaler:   marshal.MarshalFunc(json.Marshal),
		Unmarshaler: marshal.UnmarshalFunc(json.Unmarshal),
		Styler:      styles.NewGroup(),
	})
}
