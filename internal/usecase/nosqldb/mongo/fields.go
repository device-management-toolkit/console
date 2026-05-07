package mongo

// BSON field-name constants for keys used as filter, sort, or index keys —
// a typo would silently produce a query that never matches.
// Write-only $set keys live next to their entity Update method as raw strings.
const (
	fieldTenantID             = "tenantid"
	fieldGUID                 = "guid"
	fieldProfileName          = "profilename"
	fieldConfigName           = "configname"
	fieldDomainSuffix         = "domainsuffix"
	fieldTags                 = "tags"
	fieldIEEE8021xProfileName = "ieee8021xprofilename"
	fieldWirelessProfileName  = "wirelessprofilename"
	fieldPriority             = "priority"
	fieldWiredInterface       = "wiredinterface"
)

const (
	opSet   = "$set"
	opRegex = "$regex"
)
