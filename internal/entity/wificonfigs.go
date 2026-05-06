package entity

type WirelessConfig struct {
	ProfileName          string  `bson:"profilename"`
	AuthenticationMethod int     `bson:"authenticationmethod"`
	EncryptionMethod     int     `bson:"encryptionmethod"`
	SSID                 string  `bson:"ssid"`
	PSKValue             int     `bson:"pskvalue"`
	PSKPassphrase        string  `bson:"pskpassphrase"`
	LinkPolicy           *string `bson:"linkpolicy"`
	TenantID             string  `bson:"tenantid"`
	IEEE8021xProfileName *string `bson:"ieee8021xprofilename"`
	Version              string  `bson:"version"`
	//	columns to populate from join query IEEE8021xProfileName
	AuthenticationProtocol *int  `bson:"authenticationprotocol"`
	PXETimeout             *int  `bson:"pxetimeout"`
	WiredInterface         *bool `bson:"wiredinterface"`
}
