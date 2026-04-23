package usecase

import (
	"context"
	"io"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/usecase/amtexplorer"
	"github.com/device-management-toolkit/console/internal/usecase/ciraconfigs"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/internal/usecase/domains"
	"github.com/device-management-toolkit/console/internal/usecase/export"
	"github.com/device-management-toolkit/console/internal/usecase/ieee8021xconfigs"
	mongodb "github.com/device-management-toolkit/console/internal/usecase/nosqldb/mongo"
	"github.com/device-management-toolkit/console/internal/usecase/profiles"
	"github.com/device-management-toolkit/console/internal/usecase/profilewificonfigs"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/internal/usecase/wificonfigs"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// Repos is the full set of repositories an app instance needs. A single
// backend populates all fields; the app is single-backend at a time.
type Repos struct {
	Devices            devices.Repository
	Domains            domains.Repository
	Profiles           profiles.Repository
	ProfileWiFiConfigs profilewificonfigs.Repository
	IEEE8021xConfigs   ieee8021xconfigs.Repository
	CIRAConfigs        ciraconfigs.Repository
	WirelessConfigs    wificonfigs.Repository

	// Closer releases the underlying driver (DB pool / Mongo client).
	Closer io.Closer
}

// NewSQLRepos builds the repo bundle from an open *db.SQL pool (Postgres or
// embedded SQLite — same code path, picked by URL).
func NewSQLRepos(database *db.SQL, log logger.Interface) *Repos {
	return &Repos{
		Devices:            sqldb.NewDeviceRepo(database, log),
		Domains:            sqldb.NewDomainRepo(database, log),
		Profiles:           sqldb.NewProfileRepo(database, log),
		ProfileWiFiConfigs: sqldb.NewProfileWiFiConfigsRepo(database, log),
		IEEE8021xConfigs:   sqldb.NewIEEE8021xRepo(database, log),
		CIRAConfigs:        sqldb.NewCIRARepo(database, log),
		WirelessConfigs:    sqldb.NewWirelessRepo(database, log),
		Closer: closerFunc(func() error {
			database.Close()

			return nil
		}),
	}
}

// NewMongoRepos builds the repo bundle from an open Mongo client + database.
// Closing the Repos disconnects the client.
func NewMongoRepos(client *mongo.Client, database *mongo.Database, log logger.Interface) *Repos {
	return &Repos{
		Devices:            mongodb.NewDeviceRepo(database, log),
		Domains:            mongodb.NewDomainRepo(database, log),
		Profiles:           mongodb.NewProfileRepo(database, log),
		ProfileWiFiConfigs: mongodb.NewProfileWiFiConfigsRepo(database, log),
		IEEE8021xConfigs:   mongodb.NewIEEE8021xRepo(database, log),
		CIRAConfigs:        mongodb.NewCIRARepo(database, log),
		WirelessConfigs:    mongodb.NewWirelessRepo(database, log),
		Closer:             closerFunc(func() error { return client.Disconnect(context.Background()) }),
	}
}

// closerFunc adapts a plain function to io.Closer.
type closerFunc func() error

func (f closerFunc) Close() error { return f() }

// Usecases -.
type Usecases struct {
	Devices            devices.Feature
	Domains            domains.Feature
	AMTExplorer        amtexplorer.Feature
	Profiles           profiles.Feature
	ProfileWiFiConfigs profilewificonfigs.Feature
	IEEE8021xProfiles  ieee8021xconfigs.Feature
	CIRAConfigs        ciraconfigs.Feature
	WirelessProfiles   wificonfigs.Feature
	Exporter           export.Exporter
}

// NewUseCases wires every use case from a repo bundle. The caller picks the
// backend via one of the Repos constructors; this function doesn't care which.
func NewUseCases(repos *Repos, log logger.Interface, certStore security.Storager) *Usecases {
	key := config.ConsoleConfig.EncryptionKey
	safeRequirements := security.Crypto{
		EncryptionKey: key,
	}

	wsman1 := wsman.NewGoWSMANMessages(log, safeRequirements)
	wsman2 := amtexplorer.NewGoWSMANMessages(log, safeRequirements)

	pwc := profilewificonfigs.New(repos.ProfileWiFiConfigs, log)
	ieee := ieee8021xconfigs.New(repos.IEEE8021xConfigs, log)
	domains1 := domains.New(repos.Domains, log, safeRequirements, certStore)
	wificonfig := wificonfigs.New(repos.WirelessConfigs, ieee, log, safeRequirements)

	return &Usecases{
		Domains:            domains1,
		Devices:            devices.New(repos.Devices, wsman1, devices.NewRedirector(safeRequirements), log, safeRequirements),
		AMTExplorer:        amtexplorer.New(repos.Devices, wsman2, log, safeRequirements),
		Profiles:           profiles.New(repos.Profiles, repos.WirelessConfigs, pwc, ieee, log, domains1, repos.CIRAConfigs, safeRequirements),
		IEEE8021xProfiles:  ieee,
		CIRAConfigs:        ciraconfigs.New(repos.CIRAConfigs, log, safeRequirements),
		WirelessProfiles:   wificonfig,
		ProfileWiFiConfigs: pwc,
		Exporter:           export.NewFileExporter(),
	}
}
