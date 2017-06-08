package integration_test

import "golang.org/x/sys/windows"

func link(src, dest string) error {
	return windows.CreateHardLink(windows.StringToUTF16Ptr(dest+".exe"), windows.StringToUTF16Ptr(src), 0)
}
