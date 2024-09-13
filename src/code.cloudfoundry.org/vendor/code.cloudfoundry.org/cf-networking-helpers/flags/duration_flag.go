package flags

import (
	"encoding/json"
	"strconv"
	"time"
)

type DurationFlag time.Duration

func (f DurationFlag) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(f).String())
}

func (f *DurationFlag) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	parsedDuration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	*f = DurationFlag(parsedDuration)

	return nil
}
