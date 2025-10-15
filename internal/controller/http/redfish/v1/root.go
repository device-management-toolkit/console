package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/pkg/logger"
)

// NewRoutes registers the Redfish Service Root handler.
// It responds to GET on the group base path (e.g., /redfish/v1).
func NewRoutes(r *gin.RouterGroup, l logger.Interface) {
	// Service Root per DMTF Redfish spec. Keep minimal and compliant.
	handler := func(c *gin.Context) {
		// Minimal ServiceRoot payload. Additional links can be added later.
		payload := map[string]any{
			"@odata.type":    "#ServiceRoot.v1_0_0.ServiceRoot",
			"@odata.id":      "/redfish/v1/",
			"Id":             "RootService",
			"Name":           "Redfish Service Root",
			"RedfishVersion": "1.0.0",
			"SessionService": map[string]any{
				"@odata.id": "/redfish/v1/SessionService",
			},
			"Systems": map[string]any{
				"@odata.id": "/redfish/v1/Systems",
			},
			"Links": map[string]any{
				"Sessions": map[string]any{
					"@odata.id": "/redfish/v1/SessionService/Sessions",
				},
			},
		}

		c.JSON(http.StatusOK, payload)
	}

	// Register for both the group root and explicit trailing slash.
	r.GET("", handler)
	r.GET("/", handler)

	// $metadata endpoint (minimal OData metadata document)
	r.GET("/$metadata", func(c *gin.Context) {
		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.String(http.StatusOK, `<?xml version="1.0" encoding="UTF-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
	<edmx:DataServices>
		<Schema Namespace="Redfish" xmlns="http://docs.oasis-open.org/odata/ns/edm">
			<EntityType Name="ServiceRoot">
				<Key><PropertyRef Name="Id"/></Key>
				<Property Name="Id" Type="Edm.String" Nullable="false"/>
				<Property Name="Name" Type="Edm.String"/>
				<Property Name="RedfishVersion" Type="Edm.String"/>
				<NavigationProperty Name="SessionService" Type="Redfish.SessionService"/>
				<NavigationProperty Name="Systems" Type="Collection(Redfish.ComputerSystem)"/>
			</EntityType>
			<EntityType Name="SessionService">
				<Key><PropertyRef Name="Id"/></Key>
				<Property Name="Id" Type="Edm.String" Nullable="false"/>
				<Property Name="Name" Type="Edm.String"/>
				<Property Name="ServiceEnabled" Type="Edm.Boolean"/>
				<Property Name="SessionTimeout" Type="Edm.Int64"/>
				<NavigationProperty Name="Sessions" Type="Collection(Redfish.Session)"/>
			</EntityType>
			<EntityType Name="Session">
				<Key><PropertyRef Name="Id"/></Key>
				<Property Name="Id" Type="Edm.String" Nullable="false"/>
				<Property Name="Name" Type="Edm.String"/>
				<Property Name="UserName" Type="Edm.String"/>
			</EntityType>
			<EntityType Name="ComputerSystem">
				<Key><PropertyRef Name="Id"/></Key>
				<Property Name="Id" Type="Edm.String" Nullable="false"/>
				<Property Name="Name" Type="Edm.String"/>
				<Property Name="PowerState" Type="Edm.String"/>
			</EntityType>
			<EntityContainer Name="Service">
				<EntitySet Name="ServiceRoot" EntityType="Redfish.ServiceRoot"/>
				<EntitySet Name="SessionService" EntityType="Redfish.SessionService"/>
				<EntitySet Name="Sessions" EntityType="Redfish.Session"/>
				<EntitySet Name="Systems" EntityType="Redfish.ComputerSystem"/>
			</EntityContainer>
		</Schema>
	</edmx:DataServices>
</edmx:Edmx>`)
	})

	// SessionService endpoint
	r.GET("/SessionService", func(c *gin.Context) {
		payload := map[string]any{
			"@odata.type":    "#SessionService.v1_0_0.SessionService",
			"@odata.id":      "/redfish/v1/SessionService",
			"Id":             "SessionService",
			"Name":           "Redfish Session Service",
			"ServiceEnabled": true,
			"SessionTimeout": 30,
			"Sessions":       map[string]any{"@odata.id": "/redfish/v1/SessionService/Sessions"},
		}

		c.JSON(http.StatusOK, payload)
	})

	// Sessions collection endpoint (read-only, empty list for now)
	r.GET("/SessionService/Sessions", func(c *gin.Context) {
		payload := map[string]any{
			"@odata.type":         "#SessionCollection.SessionCollection",
			"@odata.id":           "/redfish/v1/SessionService/Sessions",
			"Name":                "Session Collection",
			"Members@odata.count": 0,
			"Members":             []any{},
		}

		c.JSON(http.StatusOK, payload)
	})

	l.Info("Registered Redfish Service Root at %s", r.BasePath())
}
