package v2

type RemoteEraseState string

const (
	RemoteEraseStatePending   RemoteEraseState = "pending"
	RemoteEraseStateSucceeded RemoteEraseState = "succeeded"
	RemoteEraseStateFailed    RemoteEraseState = "failed"
	RemoteEraseStateUnknown   RemoteEraseState = "unknown"
)

type RemoteEraseStatus struct {
	Status     RemoteEraseState `json:"status"`
	Message    string           `json:"message"`
	BIOSStatus *uint16          `json:"biosStatus,omitempty"`
	BIOSDetail *uint16          `json:"biosDetail,omitempty"`
}
