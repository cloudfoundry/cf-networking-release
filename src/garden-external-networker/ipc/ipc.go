package ipc

import (
	"encoding/json"
	"fmt"
	"garden-external-networker/manager"
	"io"
)

type Mux struct {
	Up         func(handle string, inputs manager.UpInputs) (*manager.UpOutputs, error)
	Down       func(handle string) error
	NetOut     func(handle string, inputs manager.NetOutInputs) error
	NetIn      func(handle string, inputs manager.NetInInputs) (*manager.NetInOutputs, error)
	BulkNetOut func(handle string, inputs manager.BulkNetOutInputs) error
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
	case "net-out":
		var inputs manager.NetOutInputs
		if err := json.NewDecoder(stdin).Decode(&inputs); err != nil {
			return err
		}
		err := m.NetOut(handle, inputs)
		if err != nil {
			return err
		}
	case "net-in":
		var inputs manager.NetInInputs
		if err := json.NewDecoder(stdin).Decode(&inputs); err != nil {
			return err
		}
		outputs, err := m.NetIn(handle, inputs)
		if err != nil {
			return err
		}
		if err := json.NewEncoder(stdout).Encode(outputs); err != nil {
			return err
		}
	case "bulk-net-out":
		var inputs manager.BulkNetOutInputs
		if err := json.NewDecoder(stdin).Decode(&inputs); err != nil {
			return err
		}
		err := m.BulkNetOut(handle, inputs)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unrecognized action: %s", action)
	}
	return nil
}
