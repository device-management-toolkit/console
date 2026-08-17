package entity

import "time"

// BSON tags are lowercase to match the mongo-driver codec default — do not
// change to camelCase or existing data won't decode.
type Device struct {
	ConnectionStatus bool       `bson:"connectionstatus"`
	MPSInstance      string     `bson:"mpsinstance"`
	Hostname         string     `bson:"hostname"`
	GUID             string     `bson:"guid"`
	ID               string     `bson:"id"`             // app-generated surrogate key; stable, immutable
	CreatedDate      string     `bson:"createddate"`    // server-set insert timestamp (RFC3339Nano UTC); immutable
	LastUpdate       string     `bson:"lastupdate"`     // server-set on insert + refreshed on record edits (RFC3339Nano UTC); NOT touched by heartbeats
	IsDeleted        bool       `bson:"isdeleted"`      // logical-deletion flag
	DeletedDate      string     `bson:"deleteddate"`    // server-set on soft-delete (RFC3339Nano UTC); read-only
	ProductType      string     `bson:"producttype"`    // manageability SKU (vPro/ISM)
	ConnectionType   string     `bson:"connectiontype"` // device connection type (CIRA/Direct)
	MPSUsername      string     `bson:"mpsusername"`
	Tags             string     `bson:"tags"`
	TenantID         string     `bson:"tenantid"`
	FriendlyName     string     `bson:"friendlyname"`
	DNSSuffix        string     `bson:"dnssuffix"`
	LastConnected    *time.Time `bson:"lastconnected"`
	LastSeen         *time.Time `bson:"lastseen"`
	LastDisconnected *time.Time `bson:"lastdisconnected"`
	DeviceInfo       string     `bson:"deviceinfo"`
	Username         string     `bson:"username"`
	Password         string     `bson:"password"`
	MPSPassword      *string    `bson:"mpspassword"`
	MEBXPassword     *string    `bson:"mebxpassword"`
	UseTLS           bool       `bson:"usetls"`
	AllowSelfSigned  bool       `bson:"allowselfsigned"`
	CertHash         *string    `bson:"certhash"`
}

type Explorer struct {
	XMLInput  string
	XMLOutput string
}
