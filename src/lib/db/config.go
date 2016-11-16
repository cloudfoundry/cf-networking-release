package db

type Config struct {
	Type             string `json:"type" validate:"nonzero"`
	ConnectionString string `json:"connection_string" validate:"nonzero"`
}
