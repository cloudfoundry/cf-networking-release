package viron

import "fmt"

const (
	RequiredFieldErrorFormat = "%s is a required environment variable"
)

type ParseError struct {
	Name  string
	Value string
	Kind  string
}

func NewParseError(name, value, kind string) ParseError {
	return ParseError{
		Name:  name,
		Value: value,
		Kind:  kind,
	}
}

func (err ParseError) Error() string {
	return fmt.Sprintf("%s value \"%s\" could not be parsed into %s value", err.Name, err.Value, err.Kind)
}

type InvalidArgumentError struct {
	Value interface{}
}

func NewInvalidArgumentError(value interface{}) InvalidArgumentError {
	return InvalidArgumentError{
		Value: value,
	}
}

func (err InvalidArgumentError) Error() string {
	return fmt.Sprintf("%v is not a non-zero struct pointer", err.Value)
}

type RequiredFieldError struct {
	Name string
}

func NewRequiredFieldError(name string) RequiredFieldError {
	return RequiredFieldError{
		Name: name,
	}
}

func (err RequiredFieldError) Error() string {
	return fmt.Sprintf("%s is a required environment variable", err.Name)
}
