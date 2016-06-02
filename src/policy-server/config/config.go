package config

import "io"

type Config struct {
	ListenHost string
	ListenPort int
}

func (c Config) Marshal(output io.Writer) error {
	return nil
}
