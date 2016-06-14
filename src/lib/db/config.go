package db

import (
	"errors"
	"fmt"
)

type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
	SSLMode  string `json:"ssl_mode"`
}

func (c Config) PostgresURL() (string, error) {
	if c.Host == "" {
		return "", errors.New(`"host" is required`)
	}

	if c.Port == 0 {
		return "", errors.New(`"port" is required`)
	}

	if c.Username == "" {
		return "", errors.New(`"username" is required`)
	}

	if c.Name == "" {
		return "", errors.New(`"name" is required`)
	}

	if c.SSLMode == "" {
		return "", errors.New(`"ssl_mode" is required`)
	}

	return fmt.Sprintf(
		"%s://%s:%s@%s:%d/%s?sslmode=%s",
		"postgres",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Name,
		c.SSLMode,
	), nil
}
