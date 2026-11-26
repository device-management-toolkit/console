package v1

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestDeviceInventoryFilters(t *testing.T) {
	t.Parallel()

	no := false
	device := inventoryForList(&dto.DeviceInfo{
		FWSku: "8", OSDistro: "Windows 11 Enterprise", OSName: "windows",
		FWVersion: "16.10.2", CurrentMode: "Admin Control Mode", LMSInstalled: &no,
	})

	for _, test := range []struct {
		expression string
		want       bool
	}{
		{"skuType eq 'VPRO'", true},
		{"skuType eq 'ISM'", false},
		{"skuType eq 'none'", false},
		{"contains(os,'WINDOWS 11')", true},
		{"contains(os,'Server')", false},
		{"startswith(amtVersion,'16')", true},
		{"startswith(amtVersion,'16.1')", true},
		{"startswith(amtVersion,'16.2')", false},
		{"currentMode eq 'acm'", true},
		{"currentMode eq 'CCM'", false},
		{"lmsInstalled eq false", true},
		{"lmsInstalled eq true", false},
		{"skuType eq 'vPro' or skuType eq 'ISM' and lmsInstalled eq true", true},
		{"(skuType eq 'vPro' or skuType eq 'ISM') and lmsInstalled eq true", false},
		{"(skuType eq 'ISM' or (currentMode eq 'acm' and lmsInstalled eq false))", true},
	} {
		t.Run(test.expression, func(t *testing.T) {
			t.Parallel()

			filter, err := parseDeviceFilter(test.expression)
			require.NoError(t, err)
			require.Equal(t, test.want, filter(device))
		})
	}

	require.Equal(t, deviceInventory{sku: "ISM", os: "linux", mode: "CCM"},
		inventoryForList(&dto.DeviceInfo{FWSku: "24", OSName: "linux", CurrentMode: "Client"}))
	require.Equal(t, deviceInventory{sku: "none", mode: "notActivated"}, inventoryForList(nil))
	require.Equal(t, "notActivated", inventoryForList(&dto.DeviceInfo{CurrentMode: "pre-provisioning state"}).mode)

	missingLMS, err := parseDeviceFilter("lmsInstalled eq false")
	require.NoError(t, err)
	require.False(t, missingLMS(inventoryForList(nil)))

	escaped, err := parseDeviceFilter("contains(os,'it''s')")
	require.NoError(t, err)
	require.True(t, escaped(deviceInventory{os: "It's a distro"}))
}

func TestDeviceListSortColumns(t *testing.T) {
	t.Parallel()

	a := deviceListRow{
		device:    dto.Device{GUID: "a", Hostname: "a", FriendlyName: "a", MPSInstance: "a", MPSUsername: "a", DNSSuffix: "a"},
		inventory: deviceInventory{sku: "ISM", os: "Ubuntu", version: "9.5", mode: "ACM", lms: "false"},
	}
	b := deviceListRow{
		device:    dto.Device{GUID: "b", Hostname: "Z", FriendlyName: "Z", MPSInstance: "Z", MPSUsername: "Z", DNSSuffix: "Z", ConnectionStatus: true},
		inventory: deviceInventory{sku: "vPro", os: "Windows", version: "16.1", mode: "CCM", lms: "true"},
	}

	for _, column := range []string{
		"skuType", "os", "amtVersion", "currentMode", "lmsInstalled",
		"hostname", "friendlyName", "guid", "connectionStatus", "mpsInstance", "mpsusername", "dnsSuffix",
	} {
		t.Run(column, func(t *testing.T) {
			t.Parallel()

			q, err := parseDeviceListQuery(url.Values{"$orderby": {column}})
			require.NoError(t, err)
			require.Negative(t, q.compare(a, b))
			q.order[0].descending = true
			require.Positive(t, q.compare(a, b))
		})
	}

	require.Positive(t, compareAMTVersion("16.10", "16.2"))
	require.Zero(t, compareAMTVersion("016.001", "16.1"))
	require.Negative(t, (deviceListQuery{order: []deviceSort{{field: "amtVersion", descending: true}}}).
		compare(a, deviceListRow{device: b.device, inventory: a.inventory}))
}

func TestDeviceListCombinedFiltersAndCount(t *testing.T) {
	t.Parallel()

	mock, engine := devicesTest(t)
	no := false
	makeDevice := func(id, host, tag, version string) dto.Device {
		return dto.Device{
			GUID: id, Hostname: host, FriendlyName: "Lab", Tags: []string{tag},
			DeviceInfo: &dto.DeviceInfo{
				FWSku: "8", OSDistro: "Windows 11", FWVersion: version,
				LMSInstalled: &no, CurrentMode: "Admin",
			},
		}
	}
	mock.EXPECT().GetDiscovered(context.Background(), MaxPageSize, 0, "").Return([]dto.Device{
		makeDevice("1", "lab-pc", "lab", "16.2"),
		makeDevice("2", "lab-pc", "lab", "16.1"),
		makeDevice("3", "lab-pc", "lab", "16.10"),
		makeDevice("4", "lab-pc", "office", "16.3"),
		makeDevice("5", "office-pc", "lab", "16.4"),
		makeDevice("6", "lab-pc", "lab", "9.5"),
	}, nil)

	query := url.Values{
		"hostname": {"lab-pc"}, "friendlyName": {"Lab"}, "tags": {"lab"}, "discovered": {"true"},
		"$filter":  {"skuType eq 'vPro' and contains(os,'Windows 11') and startswith(amtVersion,'16') and currentMode eq 'ACM' and lmsInstalled eq false"},
		"$orderby": {"amtVersion desc"}, "$count": {"true"}, "$top": {"1"}, "$skip": {"1"},
	}
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/devices?"+query.Encode(), http.NoBody))

	require.Equal(t, http.StatusOK, res.Code, res.Body.String())

	var result dto.DeviceCountResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &result))
	require.Equal(t, 3, result.Count)
	require.Len(t, result.Data, 1)
	require.Equal(t, "1", result.Data[0].GUID)
}

func TestDeviceListRejectsInvalidQuery(t *testing.T) {
	t.Parallel()

	_, engine := devicesTest(t)

	for _, test := range []struct{ key, value string }{
		{"$orderby", "invalid"},
		{"$orderby", ""},
		{"$orderby", "hostname sideways"},
		{"$orderby", "hostname,"},
		{"$orderby", "hostname asc desc"},
		{"$filter", ""},
		{"$filter", "skuType ne 'vPro'"},
		{"$filter", "not lmsInstalled eq true"},
		{"$filter", "contains(deviceInfo.osDistro,'linux')"},
		{"$filter", "os eq 'Linux'"},
		{"$filter", "startswith(os,'win')"},
		{"$filter", "lmsInstalled eq 'true'"},
		{"$filter", "(skuType eq 'vPro'"},
		{"$filter", "skuType eq 'vPro' or"},
		{"$filter", "skuType eq'vPro'"},
		{"$filter", "contains(os,'win')and lmsInstalled eq false"},
		{"$filter", "contains(os,'unterminated)"},
		{"$filter", "skuType eq 'vPro'!"},
	} {
		t.Run(test.key+"/"+test.value, func(t *testing.T) {
			t.Parallel()

			res := httptest.NewRecorder()
			query := url.Values{test.key: {test.value}}.Encode()
			engine.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/devices?"+query, http.NoBody))
			require.Equal(t, http.StatusBadRequest, res.Code)
		})
	}
}

func TestDeviceListMultiFieldOrder(t *testing.T) {
	t.Parallel()

	query, err := parseDeviceListQuery(url.Values{"$orderby": {"currentMode asc, hostname desc, amtVersion"}})
	require.NoError(t, err)

	makeRow := func(id, host, mode, version string) deviceListRow {
		return deviceListRow{device: dto.Device{GUID: id, Hostname: host}, inventory: deviceInventory{mode: mode, version: version}}
	}
	a := makeRow("a", "alpha", "ACM", "9.5")
	b := makeRow("b", "beta", "ACM", "9.5")
	c := makeRow("c", "beta", "ACM", "16.1")
	d := makeRow("d", "zulu", "CCM", "9.5")

	require.Negative(t, query.compare(b, a))
	require.Negative(t, query.compare(b, c))
	require.Negative(t, query.compare(c, d))
}

func TestDeviceListCombinedLegacyFiltersAcrossBatches(t *testing.T) {
	t.Parallel()

	mock, engine := devicesTest(t)

	first := make([]dto.Device, MaxPageSize)
	for index := range first {
		first[index] = dto.Device{Hostname: "other-host", Tags: []string{"lab", "floor2"}}
	}

	first[len(first)-1] = dto.Device{GUID: "1", Hostname: "lab-pc", Tags: []string{"lab", "floor2"}}

	mock.EXPECT().GetDiscovered(context.Background(), MaxPageSize, 0, "").Return(first, nil)
	mock.EXPECT().GetDiscovered(context.Background(), MaxPageSize, MaxPageSize, "").Return([]dto.Device{
		{GUID: "2", Hostname: "lab-pc", Tags: []string{"lab"}},
		{GUID: "3", Hostname: "lab-pc", Tags: []string{"lab", "floor2"}},
	}, nil)

	res := httptest.NewRecorder()
	engine.ServeHTTP(res, httptest.NewRequest(http.MethodGet,
		"/api/v1/devices?discovered=true&hostname=lab-pc&tags=lab,floor2&method=AND&$top=1&$skip=1&$count=true", http.NoBody))

	require.Equal(t, http.StatusOK, res.Code, res.Body.String())

	var result dto.DeviceCountResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &result))
	require.Equal(t, 2, result.Count)
	require.Len(t, result.Data, 1)
	require.Equal(t, "3", result.Data[0].GUID)
}

func TestDeviceListSQLite(t *testing.T) {
	t.Parallel()

	pool, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	defer pool.Close()

	pool.SetMaxOpenConns(1)

	_, err = pool.ExecContext(t.Context(), `
		CREATE TABLE devices (
			guid TEXT PRIMARY KEY, tenantid TEXT, discovered BOOLEAN, deviceinfo TEXT,
			hostname TEXT DEFAULT 'lab-pc', tags TEXT DEFAULT 'lab', mpsinstance TEXT DEFAULT '',
			connectionstatus BOOLEAN DEFAULT FALSE, mpsusername TEXT DEFAULT '', friendlyname TEXT DEFAULT '',
			dnssuffix TEXT DEFAULT '', username TEXT DEFAULT '', password TEXT DEFAULT '',
			usetls BOOLEAN DEFAULT FALSE, allowselfsigned BOOLEAN DEFAULT FALSE, certhash TEXT
		);
		INSERT INTO devices (guid, tenantid, discovered, deviceinfo) VALUES
			('1', 'tenant-a', TRUE, '{"osDistro":"Windows 11","fwVersion":"9.5"}'),
			('2', 'tenant-a', TRUE, '{"osDistro":"Windows 11","fwVersion":"16.1"}'),
			('3', 'tenant-a', FALSE, '{"osDistro":"Windows 11","fwVersion":"17.0"}'),
			('4', 'tenant-b', TRUE, '{"osDistro":"Windows 11","fwVersion":"18.0"}'),
			('5', 'tenant-a', TRUE, '{"osDistro":"Ubuntu","fwVersion":"19.0"}');
	`)
	require.NoError(t, err)

	log := logger.New("error")
	repo := sqldb.NewDeviceRepo(&db.SQL{Pool: pool, Builder: squirrel.StatementBuilder}, log)
	wsman := mocks.NewMockWSMAN(gomock.NewController(t))
	wsman.EXPECT().Worker().AnyTimes()
	uc := devices.New(repo, wsman, nil, log, nil)
	engine := gin.New()
	engine.Use(middleware.ResolveTenant(log))
	NewDeviceRoutes(engine.Group("/api/v1"), uc, log)

	request := httptest.NewRequest(http.MethodGet,
		"/api/v1/devices?discovered=true&tags=lab&%24filter=contains%28os%2C%27windows%27%29&%24orderby=amtVersion+desc&$top=1&$count=true", http.NoBody)
	request.Header.Set(middleware.TenantHeaderName, "tenant-a")

	res := httptest.NewRecorder()
	engine.ServeHTTP(res, request)

	require.Equal(t, http.StatusOK, res.Code, res.Body.String())

	var result dto.DeviceCountResponse
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &result))
	require.Equal(t, 2, result.Count)
	require.Len(t, result.Data, 1)
	require.Equal(t, "2", result.Data[0].GUID)
}
