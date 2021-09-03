package serial

import (
	"encoding/json"
	"io"
)

//go:generate counterfeiter -o ../fakes/overwriteable_file.go --fake-name OverwriteableFile . OverwriteableFile
type OverwriteableFile interface {
	io.Reader
	io.Writer
	io.Seeker
	Truncate(size int64) error
}

//go:generate counterfeiter -o ../fakes/serializer.go --fake-name Serializer . Serializer
type Serializer interface {
	DecodeAll(file io.ReadSeeker, outData interface{}) error
	EncodeAndOverwrite(file OverwriteableFile, outData interface{}) error
}

type Serial struct{}

func (s *Serial) DecodeAll(file io.ReadSeeker, outData interface{}) error {
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	err = json.NewDecoder(file).Decode(outData)
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

func (s *Serial) EncodeAndOverwrite(file OverwriteableFile, outData interface{}) error {
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	err = file.Truncate(0)
	if err != nil {
		return err
	}

	return json.NewEncoder(file).Encode(outData)
}
