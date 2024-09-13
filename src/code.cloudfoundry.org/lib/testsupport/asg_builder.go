package testsupport

import (
	"fmt"
	"os"
)

func BuildASG(n int) string {
	asg := "["
	for i := 1; i < n; i++ {
		t := `{"protocol": "tcp", "destination": "` + fmt.Sprintf("169.254.%d.%d", i/254, i%254) + `", "ports": "80" },`
		asg = asg + t
	}

	t := `{"protocol": "tcp", "destination": "` + fmt.Sprintf("169.254.%d.%d", n/254, n%254) + `", "ports": "80" }`
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
