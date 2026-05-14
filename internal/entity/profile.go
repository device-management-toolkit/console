package entity

type Profile struct {
	ProfileName                string  `bson:"profilename"`
	AMTPassword                string  `bson:"amtpassword"`
	CreationDate               string  `bson:"creationdate,omitempty"`
	CreatedBy                  string  `bson:"createdby,omitempty"`
	GenerateRandomPassword     bool    `bson:"generaterandompassword"`
	CIRAConfigName             *string `bson:"ciraconfigname"`
	Activation                 string  `bson:"activation"`
	MEBXPassword               string  `bson:"mebxpassword"`
	GenerateRandomMEBxPassword bool    `bson:"generaterandommebxpassword"`
	Tags                       string  `bson:"tags"`
	DHCPEnabled                bool    `bson:"dhcpenabled"`
	IPSyncEnabled              bool    `bson:"ipsyncenabled"`
	LocalWiFiSyncEnabled       bool    `bson:"localwifisyncenabled"`
	TenantID                   string  `bson:"tenantid"`
	TLSMode                    int     `bson:"tlsmode"`
	TLSSigningAuthority        string  `bson:"tlssigningauthority"`
	UserConsent                string  `bson:"userconsent"`
	IDEREnabled                bool    `bson:"iderenabled"`
	KVMEnabled                 bool    `bson:"kvmenabled"`
	SOLEnabled                 bool    `bson:"solenabled"`
	IEEE8021xProfileName       *string `bson:"ieee8021xprofilename"`
	UEFIWiFiSyncEnabled        bool    `bson:"uefiwifisyncenabled"`

	// columns to populate from join query — never persisted (bson:"-").
	Version                string `bson:"-"`
	AuthenticationProtocol *int   `bson:"-"`
	ServerName             string `bson:"-"`
	Domain                 string `bson:"-"`
	Username               string `bson:"-"`
	Password               string `bson:"-"`
	RoamingIdentity        string `bson:"-"`
	ActiveInS0             bool   `bson:"-"`
	PXETimeout             *int   `bson:"-"`
	WiredInterface         *bool  `bson:"-"`
}

const (
	TLSModeNone int = iota
	TLSModeServerOnly
	TLSModeServerAllowNonTLS
	TLSModeMutualOnly
	TLSModeMutualAllowNonTLS
)

const (
	TLSSigningAuthoritySelfSigned  string = "SelfSigned"
	TLSSigningAuthorityMicrosoftCA string = "MicrosoftCA"
)

const (
	UserConsentNone    string = "None"
	UserConsentAll     string = "All"
	UserConsentKVMOnly string = "KVM"
)
