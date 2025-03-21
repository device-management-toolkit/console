package dto

type (
	HardwareInfoResults struct {
		CIMComputerSystemPackage CIMComputerSystemPackage
		CIMSystemPackage         CIMSystemPackage
		CIMChassis               CIMChassis
		CIMChip                  CIMChips `json:"CIMChip" binding:"required"`
		CIMCard                  CIMCard
		CIMBIOSElement           CIMBIOSElement
		CIMProcessor             CIMProcessor
		CIMPhysicalPackage       CIMPhysicalPackage
		CIMPhysicalMemory        CIMPhysicalMemory
		CIMMediaAccessDevice     CIMMediaAccessDevice
	}

	CIMBIOSElement struct {
		Response  CIMBIOSElementResponse `json:"response"`
		Responses []any                  `json:"responses"`
	}

	CIMCard struct {
		Response  CIMCardResponseGet `json:"response"`
		Responses []any              `json:"responses"`
	}

	CIMChassis struct {
		Response  CIMChassisResponse `json:"response"`
		Responses []any              `json:"responses"`
	}

	CIMChips struct {
		Responses []CIMChipGet `json:"responses"`
	}

	CIMComputerSystemPackage struct {
		// PlatformGUID string
		Response  string `json:"response"`
		Responses string `json:"responses"`
	}

	CIMMediaAccessDevices struct {
		Responses []CIMMediaAccessDevice `json:"responses"`
	}

	CIMPhysicalMemory struct {
		Responses []CIMPhysicalMemoryResponse `json:"responses"`
	}

	CIMPhysicalPackage struct {
		Responses []CIMPhysicalPackageResponses `json:"responses"`
	}

	CIMProcessor struct {
		Responses []CIMProcessorResponse `json:"responses"`
	}

	CIMSystemPackage struct {
		Responses []CIMSystemPackagingResponses `json:"responses"`
	}

	CIMSystemPackagingResponses struct {
		SystemPackageItems any
	}

	CIMChassisResponse struct {
		Version            string
		SerialNumber       string
		Model              string
		Manufacturer       string
		ElementName        string
		CreationClassName  string
		Tag                string
		OperationalStatus  []int
		PackageType        int
		ChassisPackageType int
	}

	CIMBIOSElementResponse struct {
		TargetOperatingSystem TargetOperatingSystem
		SoftwareElementID     string
		SoftwareElementState  SoftwareElementState
		Name                  string
		OperationalStatus     []int
		ElementName           string
		Version               string
		Manufacturer          string
		PrimaryBIOS           bool
		ReleaseDate           Time
	}

	CIMChip struct {
		Pull []any
		Get  CIMChipGet
	}

	CIMChipGet struct {
		CanBeFRUed        bool
		CreationClassName string
		ElementName       string
		Manufacturer      string
		OperationalStatus []int
		Tag               string
		Version           string
	}

	CIMProcessorResponse struct {
		DeviceID                string
		CreationClassName       string
		SystemName              string
		SystemCreationClassName string
		ElementName             string
		OperationalStatus       []int
		HealthState             int
		EnabledState            int
		RequestedState          int
		Role                    string
		Family                  int
		OtherFamilyDescription  string
		UpgradeMethod           int
		MaxClockSpeed           int
		CurrentClockSpeed       int
		Stepping                string
		CPUStatus               int
		ExternalBusClockSpeed   int
	}

	CIMMediaAccessDevice struct {
		Pull []any
		Get  struct {
			Capabilities            []int
			CreationClassName       string
			DeviceID                string
			ElementName             string
			EnabledDefault          int
			EnabledState            int
			MaxMediaSize            int
			OperationalStatus       []int
			RequestedState          int
			Security                int
			SystemCreationClassName string
			SystemName              string
		}
	}

	CIMPhysicalMemoryResponse struct {
		PartNumber                 string
		SerialNumber               string
		Manufacturer               string
		ElementName                string
		CreationClassName          string
		Tag                        string
		OperationalStatus          []int
		FormFactor                 int
		MemoryType                 int
		Speed                      int
		Capacity                   int
		BankLabel                  string
		ConfiguredMemoryClockSpeed int
		IsSpeedInMhz               bool
		MaxMemorySpeed             int
	}

	CIMPhysicalPackageResponses struct {
		PullMemoryItems []any
		PullCardItems   []any
	}

	CIMCardResponse struct {
		Pull []any
		Get  CIMCardResponseGet
	}

	CIMCardResponseGet struct {
		CanBeFRUed        bool
		CreationClassName string
		ElementName       string
		Manufacturer      string
		Model             string
		OperationalStatus []int
		PackageType       int
		SerialNumber      string
		Tag               string
		Version           string
	}

	Time struct {
		DateTime string `xml:"Datetime"`
	}

	// ChassisPackageType is an enumeration defining the type of the PhysicalPackage.
	ChassisPackageType int

	// PackageType is the type of the PhysicalPackage.
	PackageType int

	// TargetOperatingSystem is the element's operating system environment.
	TargetOperatingSystem int

	// SoftwareElementState is defined in this model to identify various states of a SoftwareElement's life cycle.
	SoftwareElementState int
)
