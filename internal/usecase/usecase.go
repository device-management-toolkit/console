package usecase

import (
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

// New -.
func NewUseCases(database *db.SQL, log logger.Interface) *Usecases {
	pwc := profilewificonfigs.New(sqldb.NewProfileWiFiConfigsRepo(database, log), log)
	ieee := ieee8021xconfigs.New(sqldb.NewIEEE8021xRepo(database, log), log)
	wifiConfigRepo := sqldb.NewWirelessRepo(database, log)
	key := config.ConsoleConfig.EncryptionKey
	safeRequirements := security.Crypto{
		EncryptionKey: key,
	}
	wsman1 := wsman.NewGoWSMANMessages(log, safeRequirements)
	wsman2 := amtexplorer.NewGoWSMANMessages(log, safeRequirements)
	domainRepo := sqldb.NewDomainRepo(database, log)
	deviceRepo := sqldb.NewDeviceRepo(database, log)
	ciraRepo := sqldb.NewCIRARepo(database, log)
	profileRepo := sqldb.NewProfileRepo(database, log)

	domains1 := domains.New(domainRepo, log, safeRequirements)
	wificonfig := wificonfigs.New(wifiConfigRepo, ieee, log, safeRequirements)

	return &Usecases{
		Domains:            domains1,
		Devices:            devices.New(deviceRepo, wsman1, devices.NewRedirector(safeRequirements), log, safeRequirements),
		AMTExplorer:        amtexplorer.New(deviceRepo, wsman2, log, safeRequirements),
		Profiles:           profiles.New(profileRepo, wifiConfigRepo, pwc, ieee, log, domainRepo, ciraRepo, safeRequirements),
		IEEE8021xProfiles:  ieee,
		CIRAConfigs:        ciraconfigs.New(ciraRepo, log, safeRequirements),
		WirelessProfiles:   wificonfig,
		ProfileWiFiConfigs: pwc,
		Exporter:           export.NewFileExporter(),
	}
}
