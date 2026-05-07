package usecase

import (
	"io"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/usecase/amtexplorer"
	"github.com/device-management-toolkit/console/internal/usecase/ciraconfigs"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/internal/usecase/domains"
	"github.com/device-management-toolkit/console/internal/usecase/export"
	"github.com/device-management-toolkit/console/internal/usecase/ieee8021xconfigs"
	"github.com/device-management-toolkit/console/internal/usecase/profiles"
	"github.com/device-management-toolkit/console/internal/usecase/profilewificonfigs"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/internal/usecase/wificonfigs"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// Repos is the bundle of repository interfaces an app instance needs.
// Concrete backend constructors live alongside their dialer (e.g. in
// internal/app) so this package never imports a storage driver.
type Repos struct {
	Devices            devices.Repository
	Domains            domains.Repository
	Profiles           profiles.Repository
	ProfileWiFiConfigs profilewificonfigs.Repository
	IEEE8021xConfigs   ieee8021xconfigs.Repository
	CIRAConfigs        ciraconfigs.Repository
	WirelessConfigs    wificonfigs.Repository

	// Closer releases the underlying driver.
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
		Closer: CloserFunc(func() error {
			database.Close()

			return nil
		}),
	}
}

// CloserFunc adapts a plain function to io.Closer.
type CloserFunc func() error

func (f CloserFunc) Close() error { return f() }

// Usecases bundles every feature the HTTP handlers depend on.
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
