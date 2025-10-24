package dto

import "encoding/xml"


// FromXMLName converts an encoding/xml Name to the JSON-friendly XMLName.
func FromXMLName(n xml.Name) XMLName {
	return XMLName{Space: n.Space, Local: n.Local}
}

// ToXMLName converts the JSON-friendly XMLName back to encoding/xml Name.
func (x XMLName) ToXMLName() xml.Name {
	return xml.Name{Space: x.Space, Local: x.Local}
}

type (
	GetUserConsentMessage struct {
		Body UserConsentMessage `json:"Body" binding:"required"`
	}

	// XMLName is a JSON-friendly wrapper for xml.Name. We keep XML semantics
	// via explicit conversion functions so JSON/OpenAPI schemas show
	// the exported Space and Local properties while XML marshalling can use
	// the underlying xml.Name where needed.
	XMLName struct {
		Space string `json:"Space,omitempty" example:"http://schemas.xmlsoap.org/ws/2004/08/addressing"`
		Local string `json:"Local" example:"StartOptIn_OUTPUT"`
	}

	UserConsentMessage struct {
		Name        XMLName `json:"XMLName" binding:"required"`
		ReturnValue int     `json:"ReturnValue" binding:"required"`
	}

	UserConsentCode struct {
		ConsentCode string `json:"consentCode" binding:"required" example:"123456"`
	}
)

