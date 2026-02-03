package api

// ChangePasswordRequest is the request body for changing the password
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}
