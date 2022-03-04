package cc_client

type UnstableSecurityGroupListError struct {
	innerError error
}

func NewUnstableSecurityGroupListError(innerError error) UnstableSecurityGroupListError {
	return UnstableSecurityGroupListError{
		innerError: innerError,
	}
}

func (f UnstableSecurityGroupListError) Error() string {
	return f.innerError.Error()
}
