package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// Lint constants
const (
	maxSystemsList        = 100
	powerStateUnknown     = "Unknown"
	powerStateOn          = "On"
	powerStateOff         = "Off"
	resetTypeOn           = "On"
	resetTypeForceOff     = "ForceOff"
	resetTypeForceRestart = "ForceRestart"
	resetTypePowerCycle   = "PowerCycle"
	actionPowerUp         = 2
	actionPowerCycle      = 5
	actionPowerDown       = 8
	actionReset           = 10
	// CIM PowerState enum values (Device.PowerState)
	cimPowerOn      = 2
	cimPowerSleep   = 3
	cimPowerStandby = 4
	cimPowerSoftOff = 7
	cimPowerHardOff = 8
)

// NewSystemsRoutes registers minimal Redfish ComputerSystem routes.
// It exposes:
// - GET /redfish/v1/Systems
// - GET /redfish/v1/Systems/:id
// - POST /redfish/v1/Systems/:id/Actions/ComputerSystem.Reset
// The :id is expected to be the device GUID and will be mapped directly to SendPowerAction.
func NewSystemsRoutes(r *gin.RouterGroup, d devices.Feature, l logger.Interface) {
	systems := r.Group("/Systems")
	systems.GET("", getSystemsCollectionHandler(d, l))
	systems.GET(":id", getSystemInstanceHandler(d, l))
	systems.POST(":id/Actions/ComputerSystem.Reset", postSystemResetHandler(d, l))
	l.Info("Registered Redfish Systems routes under %s", r.BasePath()+"/Systems")
}

func getSystemsCollectionHandler(d devices.Feature, l logger.Interface) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := d.Get(c.Request.Context(), maxSystemsList, 0, "")
		if err != nil {
			l.Error(err, "http - redfish - Systems collection")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

			return
		}

		members := make([]any, 0, len(items))
		for i := range items { // avoid value copy
			it := &items[i]
			if it.GUID == "" {
				continue
			}

			members = append(members, map[string]any{
				"@odata.id": "/redfish/v1/Systems/" + it.GUID,
			})
		}

		payload := map[string]any{
			"@odata.type":         "#ComputerSystemCollection.ComputerSystemCollection",
			"@odata.id":           "/redfish/v1/Systems",
			"Name":                "Computer System Collection",
			"Members@odata.count": len(members),
			"Members":             members,
		}
		c.JSON(http.StatusOK, payload)
	}
}

func getSystemInstanceHandler(d devices.Feature, l logger.Interface) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		powerState := powerStateUnknown

		if ps, err := d.GetPowerState(c.Request.Context(), id); err != nil {
			l.Warn("redfish - Systems instance: failed to get power state for %s: %v", id, err)
		} else {
			switch ps.PowerState { // CIM PowerState values
			case actionPowerUp: // 2 (On)
				powerState = powerStateOn
			case cimPowerSleep, cimPowerStandby: // Sleep/Standby -> treat as On
				powerState = powerStateOn
			case cimPowerSoftOff, cimPowerHardOff: // Soft Off / Hard Off
				powerState = powerStateOff
			default:
				powerState = powerStateUnknown
			}
		}

		payload := map[string]any{
			"@odata.type": "#ComputerSystem.v1_0_0.ComputerSystem",
			"@odata.id":   "/redfish/v1/Systems/" + id,
			"Id":          id,
			"Name":        "Computer System " + id,
			"PowerState":  powerState,
			"Actions": map[string]any{
				"#ComputerSystem.Reset": map[string]any{
					"target":                            "/redfish/v1/Systems/" + id + "/Actions/ComputerSystem.Reset",
					"ResetType@Redfish.AllowableValues": []string{resetTypeOn, resetTypeForceOff, resetTypeForceRestart, resetTypePowerCycle},
				},
			},
		}
		c.JSON(http.StatusOK, payload)
	}
}

func postSystemResetHandler(d devices.Feature, l logger.Interface) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var body struct {
			ResetType string `json:"ResetType"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

			return
		}

		var action int

		switch body.ResetType {
		case resetTypeOn:
			action = actionPowerUp
		case resetTypeForceOff:
			action = actionPowerDown
		case resetTypeForceRestart:
			action = actionReset
		case resetTypePowerCycle:
			action = actionPowerCycle
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported ResetType"})

			return
		}

		res, err := d.SendPowerAction(c.Request.Context(), id, action)
		if err != nil {
			l.Error(err, "http - redfish - ComputerSystem.Reset")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

			return
		}

		c.JSON(http.StatusOK, res)
	}
}
