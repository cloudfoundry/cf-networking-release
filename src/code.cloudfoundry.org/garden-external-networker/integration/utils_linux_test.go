package integration_test

import "os"

func link(src, dest string) error {
	return os.Link(src, dest)
}
