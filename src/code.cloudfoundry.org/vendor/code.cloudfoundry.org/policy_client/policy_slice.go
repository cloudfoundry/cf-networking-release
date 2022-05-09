package policy_client

import (
	"encoding/json"
	"strings"
)

type PolicySlice []Policy

func (s PolicySlice) Len() int {
	return len(s)
}

func (s PolicySlice) Less(i, j int) bool {
	a, err := json.Marshal(s[i])
	if err != nil {
		panic(err)
	}

	b, err := json.Marshal(s[j])
	if err != nil {
		panic(err)
	}

	return strings.Compare(string(a), string(b)) < 0
}

func (s PolicySlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
