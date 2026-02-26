package cache

import "fmt"

// Cache key prefixes.
const (
	PrefixFeatures   = "features:"
	PrefixPowerState = "power:"
	PrefixKVMDisplay = "kvm:display:"
	PrefixKVMInit    = "kvm:init:"
	PrefixGeneral    = "general:"
)

// MakeFeaturesKey creates a cache key for device features.
func MakeFeaturesKey(guid string) string {
	return fmt.Sprintf("%s%s", PrefixFeatures, guid)
}

// MakePowerStateKey creates a cache key for power state.
func MakePowerStateKey(guid string) string {
	return fmt.Sprintf("%s%s", PrefixPowerState, guid)
}

// MakeKVMDisplayKey creates a cache key for KVM displays.
func MakeKVMDisplayKey(guid string) string {
	return fmt.Sprintf("%s%s", PrefixKVMDisplay, guid)
}

// MakeGeneralSettingsKey creates a cache key for general settings.
func MakeGeneralSettingsKey(guid string) string {
	return fmt.Sprintf("%s%s", PrefixGeneral, guid)
}

// MakeKVMInitKey creates a cache key for KVM initialization data.
func MakeKVMInitKey(guid string) string {
	return fmt.Sprintf("%s%s", PrefixKVMInit, guid)
}

// InvalidateDeviceCache removes all cached data for a device.
func InvalidateDeviceCache(c *Cache, guid string) {
	c.Delete(MakeFeaturesKey(guid))
	c.Delete(MakePowerStateKey(guid))
	c.Delete(MakeKVMDisplayKey(guid))
	c.Delete(MakeKVMInitKey(guid))
	c.Delete(MakeGeneralSettingsKey(guid))
}
