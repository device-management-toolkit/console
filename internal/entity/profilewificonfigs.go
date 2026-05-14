package entity

type ProfileWiFiConfigs struct {
	Priority            int    `bson:"priority"`
	ProfileName         string `bson:"profilename"`
	WirelessProfileName string `bson:"wirelessprofilename"`
	TenantID            string `bson:"tenantid"`
}
