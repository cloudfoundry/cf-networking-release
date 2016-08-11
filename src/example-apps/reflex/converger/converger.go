package converger

import "code.cloudfoundry.org/lager"

//go:generate counterfeiter -o ../fakes/reflex_client.go --fake-name ReflexClient . reflexClient
type reflexClient interface {
	GetAddressesViaRouter() ([]string, error)
	CheckInstance(address string) bool
}

//go:generate counterfeiter -o ../fakes/store_writer.go --fake-name StoreWriter . storeWriter
type storeWriter interface {
	Add(addresses []string)
}

type Converger struct {
	Logger lager.Logger
	Client reflexClient
	Store  storeWriter
}

func (c *Converger) Converge() error {
	addresses, err := c.Client.GetAddressesViaRouter()
	if err != nil {
		c.Logger.Error("get-addresses-via-router", err)
		return err
	}

	var goodAddresses, badAddresses []string
	for _, addr := range addresses {
		ok := c.Client.CheckInstance(addr)
		if ok {
			goodAddresses = append(goodAddresses, addr)
		} else {
			badAddresses = append(badAddresses, addr)
		}
	}
	c.Logger.Info("check-instance", lager.Data{
		"good-addresses": goodAddresses,
		"bad-addresses":  badAddresses,
	})
	c.Store.Add(goodAddresses)

	return nil
}
