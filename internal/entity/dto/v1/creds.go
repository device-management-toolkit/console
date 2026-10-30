package dto

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type PasswordChangeRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}

type PasswordChangeResponse struct {
	Message string `json:"message" example:"password changed"`
}
