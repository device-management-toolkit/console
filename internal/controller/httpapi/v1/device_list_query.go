package v1

import (
	"cmp"
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

const (
	deviceSKUTypeField    = "skuType"
	deviceAMTVersionField = "amtVersion"
	deviceOSField         = "os"

	deviceLMSInstalledField = "lmsInstalled"
	deviceCurrentModeField  = "currentMode"
	deviceSKUTypeNone       = "none"
)

type deviceQueryError string

func (e deviceQueryError) Error() string { return string(e) }

type deviceListQuery struct {
	hostname, friendlyName, tags, tagMethod string
	managed, discovered                     bool
	filter                                  deviceFilter
	order                                   []deviceSort
}

type deviceSort struct {
	field      string
	descending bool
}

func parseDeviceListQuery(values url.Values) (deviceListQuery, error) {
	q := deviceListQuery{
		hostname: values.Get("hostname"), friendlyName: values.Get("friendlyName"),
		tags: values.Get("tags"), tagMethod: values.Get("method"),
		managed: values.Get("activated") == "true", discovered: values.Get("discovered") == "true",
	}

	if values.Has("$filter") {
		filter, err := parseDeviceFilter(values.Get("$filter"))
		if err != nil {
			return q, fmt.Errorf("invalid $filter: %w", err)
		}

		q.filter = filter
	}

	if values.Has("$orderby") {
		order, err := parseDeviceOrder(values.Get("$orderby"))
		if err != nil {
			return q, fmt.Errorf("invalid $orderby: %w", err)
		}

		q.order = order
	}

	return q, nil
}

func parseDeviceOrder(value string) ([]deviceSort, error) {
	const maxOrderParts = 2

	var order []deviceSort

	for term := range strings.SplitSeq(value, ",") {
		parts := strings.Fields(term)
		if len(parts) < 1 || len(parts) > maxOrderParts {
			return nil, deviceQueryError("expected a field and optional direction")
		}

		switch parts[0] {
		case deviceSKUTypeField, deviceOSField, deviceAMTVersionField, deviceCurrentModeField, deviceLMSInstalledField, "hostname",
			"friendlyName", "guid", "connectionStatus", "mpsInstance", "mpsusername", "dnsSuffix":
		default:
			return nil, deviceQueryError(fmt.Sprintf("unknown sort field %q", parts[0]))
		}

		sort := deviceSort{field: parts[0]}
		if len(parts) == maxOrderParts {
			switch parts[1] {
			case "asc":
			case "desc":
				sort.descending = true
			default:
				return nil, deviceQueryError(fmt.Sprintf("invalid sort direction %q", parts[1]))
			}
		}

		order = append(order, sort)
	}

	return order, nil
}

func (q deviceListQuery) expanded(count bool) bool {
	if q.filter != nil || len(q.order) > 0 {
		return true
	}

	legacy := 0

	for _, active := range []bool{q.hostname != "", q.friendlyName != "", q.tags != "", q.managed, q.discovered} {
		if active {
			legacy++
		}
	}

	return legacy > 1 || (count && legacy > 0)
}

type deviceInventory struct {
	sku, os, version, mode, lms string
}

func inventoryForList(info *dto.DeviceInfo) deviceInventory {
	inventory := deviceInventory{sku: deviceSKUTypeNone, mode: "notActivated"}
	if info == nil {
		return inventory
	}

	sku, err := strconv.ParseUint(info.FWSku, 10, 64)
	if err == nil {
		switch {
		case sku&0x10 != 0:
			inventory.sku = "ISM"
		case sku&0x08 != 0:
			inventory.sku = "vPro"
		}
	}

	inventory.os = info.OSDistro
	if strings.TrimSpace(inventory.os) == "" {
		inventory.os = info.OSName
	}

	inventory.version = strings.ToLower(info.FWVersion)

	switch mode := strings.ToLower(strings.TrimSpace(info.CurrentMode)); mode {
	case "admin", "admin control mode", "acm":
		inventory.mode = "ACM"
	case "client", "client control mode", "ccm":
		inventory.mode = "CCM"
	case "", "not activated", "pre-provisioning state":
	default:
		inventory.mode = mode
	}

	if info.LMSInstalled != nil {
		inventory.lms = strconv.FormatBool(*info.LMSInstalled)
	}

	return inventory
}

type deviceListRow struct {
	device    dto.Device
	inventory deviceInventory
}

func (r deviceListRow) sortValue(column string) string {
	switch column {
	case deviceSKUTypeField:
		return r.inventory.sku
	case deviceOSField:
		return r.inventory.os
	case deviceCurrentModeField:
		return r.inventory.mode
	case deviceLMSInstalledField:
		return r.inventory.lms
	case "hostname":
		return r.device.Hostname
	case "friendlyName":
		return r.device.FriendlyName
	case "connectionStatus":
		return strconv.FormatBool(r.device.ConnectionStatus)
	case "mpsInstance":
		return r.device.MPSInstance
	case "mpsusername":
		return r.device.MPSUsername
	case "dnsSuffix":
		return r.device.DNSSuffix
	default:
		return r.device.GUID
	}
}

// Compare decimal segments by length and then digits, so large values cannot overflow.
func compareAMTVersion(a, b string) int {
	if a == "" || b == "" {
		return cmp.Compare(a, b)
	}

	x, y := strings.Split(a, "."), strings.Split(b, ".")
	for index := range min(len(x), len(y)) {
		left, right := strings.TrimLeft(x[index], "0"), strings.TrimLeft(y[index], "0")
		if order := cmp.Compare(len(left), len(right)); order != 0 {
			return order
		}

		if order := cmp.Compare(left, right); order != 0 {
			return order
		}
	}

	return cmp.Compare(len(x), len(y))
}

func (q deviceListQuery) compare(a, b deviceListRow) int {
	for _, sort := range q.order {
		var result int
		if sort.field == deviceAMTVersionField {
			result = compareAMTVersion(a.inventory.version, b.inventory.version)
		} else {
			result = cmp.Compare(strings.ToLower(a.sortValue(sort.field)), strings.ToLower(b.sortValue(sort.field)))
		}

		if result != 0 {
			if sort.descending {
				return -result
			}

			return result
		}
	}

	return cmp.Compare(a.device.GUID, b.device.GUID)
}

func (q deviceListQuery) matchesTags(tags, selected []string) bool {
	if q.tagMethod == "AND" {
		for _, tag := range selected {
			if !slices.Contains(tags, tag) {
				return false
			}
		}

		return true
	}

	return slices.ContainsFunc(selected, func(tag string) bool { return slices.Contains(tags, tag) })
}

// listMatchingDevices shares inventory matching across the storage backends.
// Candidate reads use tenant-scoped repository methods. Pagination is applied
// to the matching, ordered result.
func (dr *deviceRoutes) listMatchingDevices(ctx context.Context, tenant string, q deviceListQuery, page OData) ([]dto.Device, int, error) {
	items := make([]dto.Device, 0)
	if q.managed && q.discovered {
		return items, 0, nil
	}

	read := dr.t.Get

	switch {
	case q.managed:
		read = dr.t.GetActivated
	case q.discovered:
		read = dr.t.GetDiscovered
	}

	matches, count, err := q.collectMatches(ctx, tenant, read, page)
	if err != nil {
		return nil, 0, err
	}

	if len(q.order) > 0 {
		slices.SortFunc(matches, q.compare)
		start := min(page.Skip, len(matches))
		end := min(start+devicePageSize(page), len(matches))
		matches = matches[start:end]
	}

	for index := range matches {
		items = append(items, matches[index].device)
	}

	return items, count, nil
}

func devicePageSize(page OData) int {
	const defaultTop = 100
	if page.Top == 0 {
		return defaultTop
	}

	return page.Top
}

func (q deviceListQuery) collectMatches(ctx context.Context, tenant string,
	read func(context.Context, int, int, string) ([]dto.Device, error), page OData,
) ([]deviceListRow, int, error) {
	selectedTags := strings.Split(q.tags, ",")
	top := devicePageSize(page)
	stopAtPage := len(q.order) == 0 && !page.Count

	var matches []deviceListRow

	count := 0

	for offset := 0; ctx.Err() == nil; offset += MaxPageSize {
		batch, err := read(ctx, MaxPageSize, offset, tenant)
		if err != nil {
			return nil, 0, err
		}

		matches, count = q.appendBatchMatches(matches, count, batch, selectedTags, page)

		pageFilled := stopAtPage && len(matches) == top
		if len(batch) < MaxPageSize || pageFilled {
			return matches, count, nil
		}
	}

	return nil, 0, ctx.Err()
}

func (q deviceListQuery) appendBatchMatches(matches []deviceListRow, count int,
	batch []dto.Device, selectedTags []string, page OData,
) (rows []deviceListRow, total int) {
	sorted := len(q.order) > 0
	top := devicePageSize(page)

	for index := range batch {
		device := &batch[index]
		inventory := inventoryForList(device.DeviceInfo)

		if !q.matchesDevice(device, inventory, selectedTags) {
			continue
		}

		count++

		withinPage := count > page.Skip && len(matches) < top
		if sorted || withinPage {
			matches = append(matches, deviceListRow{device: *device, inventory: inventory})
		}
	}

	return matches, count
}

func (q deviceListQuery) matchesDevice(device *dto.Device, inventory deviceInventory, selectedTags []string) bool {
	return (q.hostname == "" || device.Hostname == q.hostname) &&
		(q.friendlyName == "" || device.FriendlyName == q.friendlyName) &&
		(q.tags == "" || q.matchesTags(device.Tags, selectedTags)) &&
		(q.filter == nil || q.filter(inventory))
}
