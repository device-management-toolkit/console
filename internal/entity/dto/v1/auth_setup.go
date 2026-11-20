package dto

type AuthSetupRequest struct {
	Token           string `json:"token"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

type AuthSetupStatus struct {
	SetupRequired bool `json:"setupRequired"`
}

type AuthSetupResponse struct {
	Status string `json:"status"`
}
