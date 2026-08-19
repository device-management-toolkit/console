package dto

type WirelessProfileSyncRequest struct {
	LocalProfileSync *bool `json:"localProfileSync"`
	UEFIProfileSync  *bool `json:"uefiProfileSync"`
}

type WirelessProfileSyncResponse struct {
	LocalProfileSync         bool `json:"localProfileSync"`
	UEFIProfileSync          bool `json:"uefiProfileSync"`
	UEFIProfileSyncSupported bool `json:"uefiProfileSyncSupported"`
}
