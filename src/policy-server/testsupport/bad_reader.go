package testsupport

import "errors"

type BadReader struct {
	Error error
}

func (r *BadReader) Read(buffer []byte) (int, error) {
	if r.Error != nil {
		return 0, r.Error
	}
	return 0, errors.New("banana")
}

func (r *BadReader) Close() error {
	return nil
}
