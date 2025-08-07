package testsupport

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var uniquePort int
var portMutex sync.Mutex

func init() {
	uniquePort = 1
}

func BuildASG(size int) string {
	portMutex.Lock()
	defer portMutex.Unlock()
	asg := "["
	for i := 1; len(asg) < size; i++ {
		t := `{"protocol": "tcp", "destination": "` + fmt.Sprintf("169.254.%d.%d", i/254, i%254) + `", "ports": "` + fmt.Sprintf("%d", uniquePort) + `" },`
		asg = asg + t
		uniquePort++
		if uniquePort > 65535 {
			uniquePort = 1
		}
	}

	return strings.TrimSuffix(asg, ",") + "]"
}

func CreateTempFile(content string) (string, error) {
	tmpFile, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}

	path := tmpFile.Name()
	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", err
	}

	return path, nil
}
