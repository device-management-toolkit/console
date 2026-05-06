package entity

type IEEE8021xConfig struct {
	ProfileName            string `bson:"profilename"`
	AuthenticationProtocol int    `bson:"authenticationprotocol"`
	PXETimeout             *int   `bson:"pxetimeout"`
	WiredInterface         bool   `bson:"wiredinterface"`
	TenantID               string `bson:"tenantid"`
	Version                string `bson:"version"`
}
