package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"lib/datastore"
	"net"
	"os"

	"code.cloudfoundry.org/lager"
)

type NoBridge struct {
	Logger           lager.Logger
	MetadataFileName string
}

func (n *NoBridge) Validate(subnet string) error {
	metadata, err := ioutil.ReadFile(n.MetadataFileName)
	if err != nil {
		if os.IsNotExist(err) {
			n.Logger.Info("metadata file does not exist", lager.Data{"filename": n.MetadataFileName})
			return nil
		} else {
			return fmt.Errorf("reading file: %s", err) // untested
		}
	}

	var metadataStruct map[string]datastore.Container
	err = json.Unmarshal(metadata, &metadataStruct)
	if err != nil {
		return fmt.Errorf("unmarshalling metadata: %s", err)
	}

	_, ipRange, err := net.ParseCIDR(subnet)
	if err != nil {
		return fmt.Errorf("parsing subnet: %s", err)
	}

	for _, container := range metadataStruct {
		var containerIP net.IP
		containerIP = net.ParseIP(container.IP)

		if !ipRange.Contains(containerIP) {
			return errors.New(`This cell must be restarted (run "bosh restart <job>").  Flannel is out of sync with current containers.`)
		}

	}

	return nil
}
