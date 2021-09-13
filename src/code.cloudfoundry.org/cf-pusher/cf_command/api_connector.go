package cf_command

import "fmt"

//go:generate counterfeiter -o fakes/api_cli_adapter.go --fake-name ApiCliAdapter . apiCliAdapter
type apiCliAdapter interface {
	SetApiWithSsl(api string) error
	SetApiWithoutSsl(api string) error
	Auth(user, password string) error
}

type ApiConnector struct {
	Api               string
	AdminUser         string
	AdminPassword     string
	SkipSSLValidation bool
	Adapter           apiCliAdapter
}

func (c *ApiConnector) Connect() error {
	if c.SkipSSLValidation {
		err := c.Adapter.SetApiWithoutSsl(c.Api)
		if err != nil {
			return fmt.Errorf("setting api without ssl: %s", err)
		}
	} else {
		err := c.Adapter.SetApiWithSsl(c.Api)
		if err != nil {
			return fmt.Errorf("setting api with ssl: %s", err)
		}
	}

	err := c.Adapter.Auth(c.AdminUser, c.AdminPassword)
	if err != nil {
		return fmt.Errorf("authenticating: %s", err)
	}
	return nil
}
