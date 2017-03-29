package validator

import (
	"errors"
	"fmt"
	"lib/datastore"
	"net"
)

type Validator struct {
	Store datastore.Datastore
}

func (v *Validator) Validate(subnet string) error {
	metadata, err := v.Store.ReadAll()
	if err != nil {
		return fmt.Errorf("reading metadata: %s", err)
	}

	_, ipRange, err := net.ParseCIDR(subnet)
	if err != nil {
		return fmt.Errorf("parsing subnet: %s", err)
	}

	for _, container := range metadata {
		var containerIP net.IP
		containerIP = net.ParseIP(container.IP)

		if !ipRange.Contains(containerIP) {
			return errors.New(`This cell must be restarted (run "bosh restart <job>").  Flannel is out of sync with current containers.`)
		}

	}

	return nil
}
