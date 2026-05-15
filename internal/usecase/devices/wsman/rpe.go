package wsman

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"
)

// TLV constants for the RPE UefiBootParametersArray.
const (
	intelVendorPrefix = 0x8086 // Intel PCI vendor ID used as TLV vendor prefix
	rpeTLVValueLen    = 4      // TLV value length (uint32 bitmask)
	// rpeCSMEBit is the eraseMask bit the UI sets when the user selects
	// "Unconfigure Intel CSME Firmware". It is NOT a TLV device-bitmask target;
	// it signals that ConfigurationDataReset=true should be set in the PUT body
	// and the TLV should be omitted (the C# SDK never uses PlatformErase for CSME).
	rpeCSMEBit = 0x10000
)

var (
	// ErrRPENotEnabled is returned when RPE is not enabled by the BIOS.
	ErrRPENotEnabled = errors.New("remote platform erase is not enabled by the BIOS on this device")
	// ErrInvalidEraseMask is returned when an eraseMask is negative.
	ErrInvalidEraseMask = errors.New("eraseMask must be non-negative")
	// ErrRPEPlatformEraseNotLatched is returned when the PlatformErase flag did not latch after PUT.
	ErrRPEPlatformEraseNotLatched = errors.New("remote erase: PlatformErase did not latch; aborting to avoid reboot with erase disabled")
	// ErrRPEConfigurationDataResetNotLatched is returned when ConfigurationDataReset did not latch after PUT.
	ErrRPEConfigurationDataResetNotLatched = errors.New("remote erase: ConfigurationDataReset did not latch; aborting to avoid reboot without CSME reset")
	// ErrRPEPutAndVerifyFailed is returned when both the PUT and the verify GET fail.
	ErrRPEPutAndVerifyFailed = errors.New("remote erase: PUT failed and verify GET failed; aborting")
)

func (c *ConnectionEntry) SetRPEEnabled(enabled bool) error {
	bootData, err := c.WsmanMessages.AMT.BootSettingData.Get()
	if err != nil {
		return err
	}

	current := bootData.Body.BootSettingDataGetResponse

	_, err = c.WsmanMessages.AMT.BootSettingData.Put(boot.BootSettingDataRequest{
		ElementName:   current.ElementName,
		InstanceID:    current.InstanceID,
		OwningEntity:  current.OwningEntity,
		PlatformErase: enabled,
	})

	return err
}

func (c *ConnectionEntry) SetRemoteEraseOptions(eraseMask int) error {
	if eraseMask < 0 {
		return ErrInvalidEraseMask
	}

	// Step 0: Return boot service to idle (32768) so AMT_BootSettingData.Put is allowed.
	// Non-fatal: some firmware versions return ActionNotSupported for this state change.
	const enabledStateOCRAndRPEDisabled = 32768 // OCR disabled, RPE disabled

	_, _ = c.WsmanMessages.CIM.BootService.RequestStateChange(enabledStateOCRAndRPEDisabled)

	// Step 1: Attempt to latch PlatformErase=true while the boot service is idle.
	// Non-fatal: some firmware versions block this sparse PUT; the full PUT in step 3
	// also sets PlatformErase=true, so the erase can still succeed without this latch.
	_ = c.SetRPEEnabled(true)

	// Step 2: Read AMT_BootSettingData to check RPEEnabled and current state.
	bootData, err := c.WsmanMessages.AMT.BootSettingData.Get()
	if err != nil {
		return fmt.Errorf("BootSettingData.Get: %w", err)
	}

	current := bootData.Body.BootSettingDataGetResponse

	if !current.RPEEnabled {
		return ErrRPENotEnabled
	}

	// Separate the CSME-unconfigure signal from the hardware TLV targets early so step 1a
	// can be skipped for hardware-only operations (TPM, SSDs, BIOS NVM, ...).
	// rpeCSMEBit is NOT a valid TLV device-bitmask bit; it is a UI-level sentinel that tells
	// us to set ConfigurationDataReset=true (AMT NV provisioning wipe) in the PUT request.
	wantCSMEReset := eraseMask&rpeCSMEBit != 0
	tlvMask := eraseMask &^ rpeCSMEBit // hardware targets only

	// Step 1a (CSME path only): Clear any active boot source override before
	// configuring the CSME reset flags.  An existing boot source (e.g. from a prior OCR
	// session) will take priority and must be cleared first (equivalent to ClearBootOptions
	// in the Intel AMT C# SDK).  NOT called for hardware-only targets (TPM, SSDs, …)
	// because clearing the boot order for those operations causes undefined BIOS behavior.
	// Also skipped when hardware TLV targets are present alongside CSME: a combined mask
	// should not reach here (the UI enforces CSME-only), but if the API is called directly
	// we must not poison the hardware targets by clearing the boot order they depend on.
	if wantCSMEReset && tlvMask == 0 {
		_, _ = c.WsmanMessages.CIM.BootConfigSetting.ChangeBootOrder("")
	}

	// Step 2b: Enable RPE mode in the boot service (32770 = OCR disabled, RPE enabled).
	// This is required when the boot service is in OCR mode (32769) from a prior SetFeatures call.
	const rpeEnabledState = 32770 // OCR disabled, RPE enabled

	_, _ = c.WsmanMessages.CIM.BootService.RequestStateChange(rpeEnabledState)

	var (
		encodedParams     string
		uefiBootNumParams int
	)

	if tlvMask != 0 {
		encodedParams, uefiBootNumParams = buildRPETLVParams(tlvMask)
	}

	// Step 3: PUT the erase flags.
	// ConfigurationDataReset=true resets AMT's NV provisioning state (= AMT unprovision).
	// PlatformErase=true triggers the RPE UEFI sequence for hardware targets (TPM, SSDs, ...).
	// The two are independent: CSME unconfigure never uses PlatformErase (per Intel C# SDK).
	putReq := boot.BootSettingDataRequest{
		ElementName:             current.ElementName,
		InstanceID:              current.InstanceID,
		OwningEntity:            current.OwningEntity,
		ConfigurationDataReset:  wantCSMEReset,
		PlatformErase:           tlvMask != 0,
		UefiBootParametersArray: encodedParams,
		UefiBootNumberOfParams:  uefiBootNumParams,
	}

	if _, putErr := c.WsmanMessages.AMT.BootSettingData.Put(putReq); putErr != nil {
		return c.verifyPutLatched(putErr, wantCSMEReset, tlvMask)
	}

	// Step 4: Activate the RPE boot config role so it executes on next restart.
	if _, err = c.WsmanMessages.CIM.BootService.SetBootConfigRole("Intel(r) AMT: Boot Configuration 0", 1); err != nil {
		return fmt.Errorf("SetBootConfigRole: %w", err)
	}

	// Step 5: Restart the platform via a full hardware power cycle (S5→S0).
	// PowerCycleOffHard is required: MasterBusReset (warm reset) keeps ME power rails energised
	// so the BIOS never gets the opportunity to execute the CSME/platform erase.
	if _, err = c.WsmanMessages.CIM.PowerManagementService.RequestPowerStateChange(power.PowerCycleOffHard); err != nil {
		return fmt.Errorf("RequestPowerStateChange: %w", err)
	}

	return nil
}

// verifyPutLatched is called when BootSettingData.Put returns an error. It issues a
// follow-up GET to check whether the flag that was actually requested (PlatformErase for
// hardware targets, ConfigurationDataReset for CSME-only) latched despite the PUT error.
// If it did not latch, a sentinel error is returned to abort the sequence safely.
func (c *ConnectionEntry) verifyPutLatched(putErr error, wantCSMEReset bool, tlvMask int) error {
	verifyData, verifyErr := c.WsmanMessages.AMT.BootSettingData.Get()
	if verifyErr != nil {
		return fmt.Errorf("%w: PUT err=%w; verify GET err=%w", ErrRPEPutAndVerifyFailed, putErr, verifyErr)
	}

	v := verifyData.Body.BootSettingDataGetResponse

	if wantCSMEReset && tlvMask == 0 {
		// CSME-only path: the flag that matters is ConfigurationDataReset.
		if !v.ConfigurationDataReset {
			return fmt.Errorf("%w: PUT err=%w", ErrRPEConfigurationDataResetNotLatched, putErr)
		}

		return nil
	}

	// Hardware-target path: the flag that matters is PlatformErase.
	if !v.PlatformErase {
		return fmt.Errorf("%w: PUT err=%w", ErrRPEPlatformEraseNotLatched, putErr)
	}

	return nil
}

// buildRPETLVParams encodes the hardware-target device bitmask as a single RPE TLV entry.
// Format per Intel AMT spec: [vendor:2 LE][typeID:2 LE][length:4 LE][value:4 LE].
func buildRPETLVParams(tlvMask int) (encodedParams string, numParams int) {
	const rpeTLVLen = 12

	tlvBuf := make([]byte, rpeTLVLen)
	binary.LittleEndian.PutUint16(tlvBuf[0:], intelVendorPrefix) // Intel vendor prefix
	binary.LittleEndian.PutUint16(tlvBuf[2:], 1)                 // ParameterTypeID = 1 (device bitmask)
	binary.LittleEndian.PutUint32(tlvBuf[4:], rpeTLVValueLen)    // value length = 4 bytes
	binary.LittleEndian.PutUint32(tlvBuf[8:], uint32(tlvMask))   //nolint:gosec // non-negative, CSME bit already stripped

	return base64.StdEncoding.EncodeToString(tlvBuf), 1
}
