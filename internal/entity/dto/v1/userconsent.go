package dto

type (
	UserConsentMessage struct {
		Header UserConsentHeader `json:"Header,omitempty"`
		Body   UserConsentBody   `json:"Body,omitempty"`
	}

	UserConsentHeader struct {
		Action      string `json:"Action,omitempty" example:"http://intel.com/wbem/wscim/1/ips-schema/1/IPS_OptInService/StartOptInResponse"`
		MessageID   string `json:"MessageID,omitempty" example:"uuid:00000000-8086-8086-8086-000000001ACD"`
		Method      string `json:"Method,omitempty" example:"StartOptIn"`
		RelatesTo   string `json:"RelatesTo,omitempty" example:"1"`
		ResourceURI string `json:"ResourceURI,omitempty" example:"http://intel.com/wbem/wscim/1/ips-schema/1/IPS_OptInService"`
		To          string `json:"To,omitempty" example:"http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous"`
	}

	UserConsentBody struct {
		ReturnValue int `json:"ReturnValue" example:"0"`
	}

	UserConsentCode struct {
		ConsentCode string `json:"consentCode" binding:"required" example:"123456"`
	}
)
