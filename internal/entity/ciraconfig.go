package entity

type CIRAConfig struct {
	ConfigName             string `bson:"configname"`
	MPSAddress             string `bson:"mpsaddress"`
	MPSPort                int    `bson:"mpsport"`
	Username               string `bson:"username"`
	Password               string `bson:"password"`
	CommonName             string `bson:"commonname"`
	ServerAddressFormat    int    `bson:"serveraddressformat"`
	AuthMethod             int    `bson:"authmethod"`
	MPSRootCertificate     string `bson:"mpsrootcertificate"`
	ProxyDetails           string `bson:"proxydetails"`
	TenantID               string `bson:"tenantid"`
	GenerateRandomPassword bool   `bson:"generaterandompassword"`
	Version                string `bson:"version"`
}
