package entity

type Domain struct {
	ProfileName                   string `bson:"profilename"`
	DomainSuffix                  string `bson:"domainsuffix"`
	ProvisioningCert              string `bson:"provisioningcert"`
	ProvisioningCertStorageFormat string `bson:"provisioningcertstorageformat"`
	ProvisioningCertPassword      string `bson:"provisioningcertpassword"`
	ExpirationDate                string `bson:"expirationdate"`
	TenantID                      string `bson:"tenantid"`
	Version                       string `bson:"version"`
}
