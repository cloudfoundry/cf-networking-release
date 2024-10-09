package testsupport

import (
	"fmt"
	"os"
	"sync"
)

var uniquePort int
var portMutex sync.Mutex

func init() {
	uniquePort = 1
}

func BuildASG(n int) string {
	portMutex.Lock()
	defer portMutex.Unlock()
	asg := "["
	for i := 1; i < n; i++ {
		t := `{"protocol": "tcp", "destination": "` + fmt.Sprintf("169.254.%d.%d", i/254, i%254) + `", "ports": "` + fmt.Sprintf("%d", uniquePort) + `" },`
		asg = asg + t
		uniquePort++
	}

	t := `{"protocol": "tcp", "destination": "` + fmt.Sprintf("169.254.%d.%d", n/254, n%254) + `", "ports": "` + fmt.Sprintf("%d", uniquePort) + `" }`
	return asg + t + "]"
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
