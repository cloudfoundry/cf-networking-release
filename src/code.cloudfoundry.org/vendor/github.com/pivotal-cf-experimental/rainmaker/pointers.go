package rainmaker

func IntPtr(integer int) *int {
	return &integer
}

func BoolPtr(boolean bool) *bool {
	return &boolean
}

func StringPtr(str string) *string {
	return &str
}
