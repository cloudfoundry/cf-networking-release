package marshal

//go:generate counterfeiter -o ../fakes/unmarshaler.go --fake-name Unmarshaler . Unmarshaler
type Unmarshaler interface {
	Unmarshal(input []byte, output interface{}) error
}

//go:generate counterfeiter -o ../fakes/marshaler.go --fake-name Marshaler . Marshaler
type Marshaler interface {
	Marshal(input interface{}) ([]byte, error)
}

type UnmarshalFunc func([]byte, interface{}) error

func (u UnmarshalFunc) Unmarshal(input []byte, output interface{}) error {
	return u(input, output)
}

type MarshalFunc func(interface{}) ([]byte, error)

func (u MarshalFunc) Marshal(input interface{}) ([]byte, error) {
	return u(input)
}
