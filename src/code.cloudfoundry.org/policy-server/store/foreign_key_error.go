package store

type ForeignKeyError struct {
	innerError error
}

func NewForeignKeyError(innerError error) ForeignKeyError {
	return ForeignKeyError{
		innerError: innerError,
	}
}

func (f ForeignKeyError) Error() string {
	return f.innerError.Error()
}
