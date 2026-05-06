package mongo

import (
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/device-management-toolkit/console/internal/repoerrors"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

// Reused from repoerrors so use cases' errors.As checks work for both backends.
var (
	errDeviceDatabase              = repoerrors.DatabaseError{Console: consoleerrors.CreateConsoleError("MongoDeviceRepo")}
	errDeviceNotUnique             = repoerrors.NotUniqueError{Console: consoleerrors.CreateConsoleError("MongoDeviceRepo")}
	errProfileDatabase             = repoerrors.DatabaseError{Console: consoleerrors.CreateConsoleError("MongoProfileRepo")}
	errProfileNotUnique            = repoerrors.NotUniqueError{Console: consoleerrors.CreateConsoleError("MongoProfileRepo")}
	errDomainDatabase              = repoerrors.DatabaseError{Console: consoleerrors.CreateConsoleError("MongoDomainRepo")}
	errDomainNotUnique             = repoerrors.NotUniqueError{Console: consoleerrors.CreateConsoleError("MongoDomainRepo")}
	errCIRADatabase                = repoerrors.DatabaseError{Console: consoleerrors.CreateConsoleError("MongoCIRARepo")}
	errCIRANotUnique               = repoerrors.NotUniqueError{Console: consoleerrors.CreateConsoleError("MongoCIRARepo")}
	errIEEEDatabase                = repoerrors.DatabaseError{Console: consoleerrors.CreateConsoleError("MongoIEEE8021xRepo")}
	errIEEENotUnique               = repoerrors.NotUniqueError{Console: consoleerrors.CreateConsoleError("MongoIEEE8021xRepo")}
	errWiFiDatabase                = repoerrors.DatabaseError{Console: consoleerrors.CreateConsoleError("MongoWirelessRepo")}
	errWiFiNotUnique               = repoerrors.NotUniqueError{Console: consoleerrors.CreateConsoleError("MongoWirelessRepo")}
	errProfileWiFiConfigsDatabase  = repoerrors.DatabaseError{Console: consoleerrors.CreateConsoleError("MongoProfileWiFiConfigsRepo")}
	errProfileWiFiConfigsNotUnique = repoerrors.NotUniqueError{Console: consoleerrors.CreateConsoleError("MongoProfileWiFiConfigsRepo")}
)

// isDuplicateKey matches Mongo E11000 errors (mapped to NotUniqueError, mirroring SQL).
func isDuplicateKey(err error) bool {
	return mongo.IsDuplicateKeyError(err)
}
