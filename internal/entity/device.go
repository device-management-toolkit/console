package entity

import "time"

// BSON tags are lowercase to match the mongo-driver codec default — do not
// change to camelCase or existing data won't decode.
type Device struct {
	ConnectionStatus bool       `bson:"connectionstatus"`
	MPSInstance      string     `bson:"mpsinstance"`
	Hostname         string     `bson:"hostname"`
	GUID             string     `bson:"guid"`
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
	// Queryable mirrors of the matching deviceinfo JSON fields, synced on every insert/update.
	CurrentMode string `bson:"currentmode"`
	Discovered  *bool  `bson:"discovered"`
}

// DeviceState narrows a device query to one tab of the device list.
type DeviceState string

const (
	// DeviceStateAny applies no state filter.
	DeviceStateAny DeviceState = ""
	// DeviceStateActivated matches managed devices: anything not left in Discovered.
	DeviceStateActivated DeviceState = "activated"
	// DeviceStateDiscovered matches devices found by discovery and not yet moved to Managed.
	DeviceStateDiscovered DeviceState = "discovered"
)

type Explorer struct {
	XMLInput  string
	XMLOutput string
}
