package ipc

import (
	"encoding/json"
	"fmt"
	"io"

	"code.cloudfoundry.org/garden-external-networker/manager"
)

type Mux struct {
	Up   func(handle string, inputs manager.UpInputs) (*manager.UpOutputs, error)
	Down func(handle string) error
}

func (m *Mux) Handle(action string, handle string, stdin io.Reader, stdout io.Writer) error {
	if handle == "" {
		return fmt.Errorf("missing handle")
	}

	switch action {
	case "up":
		var inputs manager.UpInputs
		if err := json.NewDecoder(stdin).Decode(&inputs); err != nil {
			return err
		}
		outputs, err := m.Up(handle, inputs)
		if err != nil {
			return err
		}
		if err := json.NewEncoder(stdout).Encode(outputs); err != nil {
			return err
		}
	case "down":
		err := m.Down(handle)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unrecognized action: %s", action)
	}
	return nil
}
