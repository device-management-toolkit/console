package mongo

import (
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

// Reuse semantic errors from sqldb so use cases can keep their existing
// `errors.As(&sqldb.NotUniqueError{})` checks unchanged.
var (
	errDeviceDatabase              = sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("MongoDeviceRepo")}
	errDeviceNotUnique             = sqldb.NotUniqueError{Console: consoleerrors.CreateConsoleError("MongoDeviceRepo")}
	errProfileDatabase             = sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("MongoProfileRepo")}
	errProfileNotUnique            = sqldb.NotUniqueError{Console: consoleerrors.CreateConsoleError("MongoProfileRepo")}
	errDomainDatabase              = sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("MongoDomainRepo")}
	errDomainNotUnique             = sqldb.NotUniqueError{Console: consoleerrors.CreateConsoleError("MongoDomainRepo")}
	errCIRADatabase                = sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("MongoCIRARepo")}
	errCIRANotUnique               = sqldb.NotUniqueError{Console: consoleerrors.CreateConsoleError("MongoCIRARepo")}
	errIEEEDatabase                = sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("MongoIEEE8021xRepo")}
	errIEEENotUnique               = sqldb.NotUniqueError{Console: consoleerrors.CreateConsoleError("MongoIEEE8021xRepo")}
	errWiFiDatabase                = sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("MongoWirelessRepo")}
	errWiFiNotUnique               = sqldb.NotUniqueError{Console: consoleerrors.CreateConsoleError("MongoWirelessRepo")}
	errProfileWiFiConfigsDatabase  = sqldb.DatabaseError{Console: consoleerrors.CreateConsoleError("MongoProfileWiFiConfigsRepo")}
	errProfileWiFiConfigsNotUnique = sqldb.NotUniqueError{Console: consoleerrors.CreateConsoleError("MongoProfileWiFiConfigsRepo")}
)

// isDuplicateKey matches Mongo duplicate-key write errors (code 11000).
// Used to produce the same NotUniqueError the SQL path would.
func isDuplicateKey(err error) bool {
	return mongo.IsDuplicateKeyError(err)
}
