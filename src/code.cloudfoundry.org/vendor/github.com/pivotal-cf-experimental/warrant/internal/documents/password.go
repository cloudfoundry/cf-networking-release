package documents

// SetPasswordRequest represents the JSON transport data structure
// for a request to set a user password.
type SetPasswordRequest struct {
	// Password is the new password to set.
	Password string `json:"password"`
}

// ChangePasswordRequest represents the JSON transport data structure
// for a request to change a user password.
type ChangePasswordRequest struct {
	// Password is the new password to set.
	Password string `json:"password"`

	// OldPassword is the existing password to check before setting
	// a new one.
	OldPassword string `json:"oldPassword"`
}
