package v2

type (
	HardwareInfoResults struct {
		ComputerSystemPackage CIMComputerSystemPackage
		SystemPackage         CIMSystemPackage
		Chassis               CIMChassis
		Chip                  CIMChip
		Card                  CIMCard
		BIOSElement           CIMBIOSElement
		Processor             CIMProcessor
		PhysicalMemory        CIMPhysicalMemory
		MediaAccessDevices    CIMMediaAccessDevice
		PhysicalPackage       CIMPhysicalPackage
	}
	CIMComputerSystemPackage struct {
		PlatformGUID string
	}
	CIMSystemPackage struct {
		SystemPackageItems []SystemPackageItems
	}
	CIMChassis struct {
		Version            string
		SerialNumber       string
		Model              string
		Manufacturer       string
		ElementName        string
		CreationClassName  string
		Tag                string
		OperationalStatus  []int
		PackageType        PackageType
		ChassisPackageType ChassisPackageType
	}
	CIMChip struct {
		Pull []ChipItems
		Get  ChipItems
	}
	CIMCard struct {
		Pull []CardItems
		Get  CardItems
	}
	// CIMBIOSElement struct {
	// 	Pull []BiosElement
	// 	Get  BiosElement
	// }.
	CIMBIOSElement struct {
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
	CIMProcessor struct {
		Pull []ProcessorItems
		Get  ProcessorItems
	}
	CIMPhysicalMemory struct {
		Pull []PhysicalMemory
		Get  PhysicalMemory
	}
	CIMMediaAccessDevice struct {
		Pull []MediaAccessDevice
		Get  MediaAccessDevice
	}
	CIMPhysicalPackage struct {
		PullMemoryItems []PhysicalMemory
		PullCardItems   []CardItems
	}
	SystemPackageItems struct {
		PlatformGUID string
	}

	// ChassisItem struct {
	// 	Version            string
	// 	SerialNumber       string
	// 	Model              string
	// 	Manufacturer       string
	// 	ElementName        string
	// 	CreationClassName  string
	// 	Tag                string
	// 	OperationalStatus  []int
	// 	PackageType        PackageType
	// 	ChassisPackageType ChassisPackageType
	// }.

	ChipItems struct {
		CanBeFRUed        bool   `xml:"CanBeFRUed"`        // Boolean that indicates whether this PhysicalElement can be FRUed (TRUE) or not (FALSE).
		CreationClassName string `xml:"CreationClassName"` // CreationClassName indicates the name of the class or the subclass used in the creation of an instance. When used with the other key properties of this class, this property allows all instances of this class and its subclasses to be uniquely identified.
		ElementName       string `xml:"ElementName"`
		Manufacturer      string `xml:"Manufacturer"`      // The name of the organization responsible for producing the PhysicalElement. This organization might be the entity from whom the Element is purchased, but this is not necessarily true. The latter information is contained in the Vendor property of CIM_Product.
		OperationalStatus []int  `xml:"OperationalStatus"` // Indicates the current statuses of the element.
		Tag               string `xml:"Tag"`               // An arbitrary string that uniquely identifies the Physical Element and serves as the key of the Element. The Tag property can contain information such as asset tag or serial number data. The key for PhysicalElement is placed very high in the object hierarchy in order to independently identify the hardware or entity, regardless of physical placement in or on Cabinets, Adapters, and so on. For example, a hotswappable or removable component can be taken from its containing (scoping) Package and be temporarily unused. The object still continues to exist and can even be inserted into a different scoping container. Therefore, the key for Physical Element is an arbitrary string and is defined independently of any placement or location-oriented hierarchy.
		Version           string `xml:"Version"`           // A string that indicates the version of the PhysicalElement.
	}

	CardItems struct {
		CanBeFRUed        bool        `xml:"CanBeFRUed"`        // Boolean that indicates whether this PhysicalElement can be FRUed (TRUE) or not (FALSE).
		CreationClassName string      `xml:"CreationClassName"` // CreationClassName indicates the name of the class or the subclass used in the creation of an instance.
		ElementName       string      `xml:"ElementName"`
		Manufacturer      string      `xml:"Manufacturer"`      // The name of the organization responsible for producing the PhysicalElement.
		Model             string      `xml:"Model"`             // The name by which the PhysicalElement is generally known.
		OperationalStatus []int       `xml:"OperationalStatus"` // Indicates the current statuses of the element
		PackageType       PackageType `xml:"PackageType"`       // Enumeration defining the type of the PhysicalPackage. Note that this enumeration expands on the list in the Entity MIB (the attribute, entPhysicalClass). The numeric values are consistent with CIM's enum numbering guidelines, but are slightly different than the MIB's values.
		SerialNumber      string      `xml:"SerialNumber"`      // A manufacturer-allocated number used to identify the Physical Element.
		Tag               string      `xml:"Tag"`               // An arbitrary string that uniquely identifies the Physical Element and serves as the key of the Element.
		Version           string      `xml:"Version"`           // A string that indicates the version of the PhysicalElement.
	}

	// BiosElement struct {
	// 	TargetOperatingSystem TargetOperatingSystem `xml:"TargetOperatingSystem"` // The TargetOperatingSystem property specifies the element's operating system environment.
	// 	SoftwareElementID     string                `xml:"SoftwareElementID"`     // This is an identifier for the SoftwareElement and is designed to be used in conjunction with other keys to create a unique representation of the element.
	// 	SoftwareElementState  SoftwareElementState  `xml:"SoftwareElementState"`  // The SoftwareElementState is defined in this model to identify various states of a SoftwareElement's life cycle.
	// 	Name                  string                `xml:"Name"`                  // The name used to identify this SoftwareElement.
	// 	OperationalStatus     []int                 `xml:"OperationalStatus"`     // Indicates the current statuses of the element.
	// 	ElementName           string                `xml:"ElementName"`           // A user-friendly name for the object. This property allows each instance to define a user-friendly name in addition to its key properties, identity data, and description information. Note that the Name property of ManagedSystemElement is also defined as a user-friendly name. But, it is often subclassed to be a Key. It is not reasonable that the same property can convey both identity and a user-friendly name, without inconsistencies. Where Name exists and is not a Key (such as for instances of LogicalDevice), the same information can be present in both the Name and ElementName properties. Note that if there is an associated instance of CIM_EnabledLogicalElementCapabilities, restrictions on this properties may exist as defined in ElementNameMask and MaxElementNameLen properties defined in that class.
	// 	Version               string                `xml:"Version"`               // The version of the BIOS software image.
	// 	Manufacturer          string                `xml:"Manufacturer"`          // The manufacturer of the BIOS software image.
	// 	PrimaryBIOS           bool                  `xml:"PrimaryBIOS"`           // If true, this is the primary BIOS of the ComputerSystem.
	// 	ReleaseDate           Time                  `xml:"ReleaseDate"`           // Date that this BIOS was released.
	// }.

	ProcessorItems struct {
		DeviceID                string         `xml:"DeviceID,omitempty"`                // An address or other identifying information to uniquely name the LogicalDevice.
		CreationClassName       string         `xml:"CreationClassName,omitempty"`       // CreationClassName indicates the name of the class or the subclass used in the creation of an instance. When used with the other key properties of this class, this property allows all instances of this class and its subclasses to be uniquely identified.
		SystemName              string         `xml:"SystemName,omitempty"`              // The scoping System's Name.
		SystemCreationClassName string         `xml:"SystemCreationClassName,omitempty"` // The scoping System's CreationClassName.
		ElementName             string         `xml:"ElementName,omitempty"`             // A user-friendly name for the object. This property allows each instance to define a user-friendly name in addition to its key properties, identity data, and description information. Note that the Name property of ManagedSystemElement is also defined as a user-friendly name. But, it is often subclassed to be a Key. It is not reasonable that the same property can convey both identity and a user-friendly name, without inconsistencies. Where Name exists and is not a Key (such as for instances of LogicalDevice), the same information can be present in both the Name and ElementName properties. Note that if there is an associated instance of CIM_EnabledLogicalElementCapabilities, restrictions on this properties may exist as defined in ElementNameMask and MaxElementNameLen properties defined in that class.
		OperationalStatus       []int          `xml:"OperationalStatus,omitempty"`       // Indicates the current statuses of the element.
		HealthState             HealthState    `xml:"HealthState,omitempty"`             // Indicates the current health of the element.
		EnabledState            EnabledState   `xml:"EnabledState,omitempty"`            // EnabledState is an integer enumeration that indicates the enabled and disabled states of an element.
		RequestedState          RequestedState `xml:"RequestedState,omitempty"`          // RequestedState is an integer enumeration that indicates the last requested or desired state for the element, irrespective of the mechanism through which it was requested.
		Role                    string         `xml:"Role,omitempty"`                    // A free-form string that describes the role of the Processor, for example, "Central Processor" or "Math Processor".
		Family                  int            `xml:"Family,omitempty"`                  // The Processor family type. For example, values include "Pentium(R) processor with MMX(TM) technology" (value=14) and "68040" (value=96).
		OtherFamilyDescription  string         `xml:"OtherFamilyDescription,omitempty"`  // A string that describes the Processor Family type. It is used when the Family property is set to 1 ("Other"). This string should be set to NULL when the Family property is any value other than 1.
		UpgradeMethod           UpgradeMethod  `xml:"UpgradeMethod,omitempty"`           // CPU socket information that includes data on how this Processor can be upgraded (if upgrades are supported). This property is an integer enumeration.
		MaxClockSpeed           int            `xml:"MaxClockSpeed,omitempty"`           // The maximum speed (in MHz) of this Processor.
		CurrentClockSpeed       int            `xml:"CurrentClockSpeed,omitempty"`       // The current speed (in MHz) of this Processor.
		Stepping                string         `xml:"Stepping,omitempty"`                // Stepping is a free-form string that indicates the revision level of the Processor within the Processor.Family.
		CPUStatus               CPUStatus      `xml:"CPUStatus,omitempty"`               // The CPUStatus property that indicates the current status of the Processor.
		ExternalBusClockSpeed   int            `xml:"ExternalBusClockSpeed,omitempty"`   // The speed (in MHz) of the external bus interface (also known as the front side bus).
	}

	PhysicalMemory struct {
		PartNumber                 string     `xml:"PartNumber"`        // The part number assigned by the organization that is responsible for producing or manufacturing the PhysicalElement.
		SerialNumber               string     `xml:"SerialNumber"`      // A manufacturer-allocated number used to identify the Physical Element.
		Manufacturer               string     `xml:"Manufacturer"`      // The name of the organization responsible for producing the PhysicalElement. This organization might be the entity from whom the Element is purchased, but this is not necessarily true. The latter information is contained in the Vendor property of CIM_Product.
		ElementName                string     `xml:"ElementName"`       // 'ElementName' is constant. In CIM_Chip instances its value is set to 'Managed System Memory Chip'.
		CreationClassName          string     `xml:"CreationClassName"` // CreationClassName indicates the name of the class or the subclass used in the creation of an instance. When used with the other key properties of this class, this property allows all instances of this class and its subclasses to be uniquely identified.
		Tag                        string     `xml:"Tag"`               // An arbitrary string that uniquely identifies the Physical Element and serves as the key of the Element. The Tag property can contain information such as asset tag or serial number data. The key for PhysicalElement is placed very high in the object hierarchy in order to independently identify the hardware or entity, regardless of physical placement in or on Cabinets, Adapters, and so on. For example, a hotswappable or removable component can be taken from its containing (scoping) Package and be temporarily unused. The object still continues to exist and can even be inserted into a different scoping container. Therefore, the key for Physical Element is an arbitrary string and is defined independently of any placement or location-oriented hierarchy.
		OperationalStatus          []int      `xml:"OperationalStatus"` // Indicates the current statuses of the element. Various operational statuses are defined.
		FormFactor                 int        `xml:"FormFactor,omitempty"`
		MemoryType                 MemoryType `xml:"MemoryType,omitempty"`                 // The type of PhysicalMemory. Synchronous DRAM is also known as SDRAM Cache DRAM is also known as CDRAM CDRAM is also known as Cache DRAM SDRAM is also known as Synchronous DRAM BRAM is also known as Block RAM
		Speed                      int        `xml:"Speed,omitempty"`                      // The speed of the PhysicalMemory, in nanoseconds.
		Capacity                   int        `xml:"Capacity,omitempty"`                   // The total capacity of this PhysicalMemory, in bytes.
		BankLabel                  string     `xml:"BankLabel,omitempty"`                  // A string identifying the physically labeled bank where the Memory is located - for example, 'Bank 0' or 'Bank A'.
		ConfiguredMemoryClockSpeed int        `xml:"ConfiguredMemoryClockSpeed,omitempty"` // The configured clock speed (in MHz) of PhysicalMemory.
		IsSpeedInMhz               bool       `xml:"IsSpeedInMhz,omitempty"`               // The IsSpeedInMHz property is used to indicate if the Speed property or the MaxMemorySpeed contains the value of the memory speed. A value of TRUE shall indicate that the speed is represented by the MaxMemorySpeed property. A value of FALSE shall indicate that the speed is represented by the Speed property.
		MaxMemorySpeed             int        `xml:"MaxMemorySpeed,omitempty"`             // The maximum speed (in MHz) of PhysicalMemory.
	}

	MediaAccessDevice struct {
		Capabilities            []Capabilities `xml:"Capabilities,omitempty"`  // ValueMap={0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12} Values={Unknown, Other, Sequential Access, Random Access, Supports Writing, Encryption, Compression, Supports Removeable Media, Manual Cleaning, Automatic Cleaning, SMART Notification, Supports Dual Sided Media, Predismount Eject Not Required} ArrayType=Indexed
		CreationClassName       string         `xml:"CreationClassName"`       // CreationClassName indicates the name of the class or the subclass used in the creation of an instance.
		DeviceID                string         `xml:"DeviceID"`                // An address or other identifying information to uniquely name the LogicalDevice.
		ElementName             string         `xml:"ElementName"`             // This property allows each instance to define a user-friendly name in addition to its key properties, identity data, and description information.
		EnabledDefault          EnabledDefault `xml:"EnabledDefault"`          // An enumerated value indicating an administrator's default or startup configuration for the Enabled State of an element.
		EnabledState            EnabledState   `xml:"EnabledState"`            // EnabledState is an integer enumeration that indicates the enabled and disabled states of an element. It can also indicate the transitions between these requested states.
		MaxMediaSize            uint64         `xml:"MaxMediaSize,omitempty"`  // Maximum size, in KBytes, of media supported by this Device.
		OperationalStatus       []int          `xml:"OperationalStatus"`       // Indicates the current statuses of the element.
		RequestedState          RequestedState `xml:"RequestedState"`          // RequestedState is an integer enumeration that indicates the last requested or desired state for the element, irrespective of the mechanism through which it was requested.
		Security                Security       `xml:"Security,omitempty"`      // ValueMap={1, 2, 3, 4, 5, 6, 7} Values={Other, Unknown, None, Read Only, Locked Out, Boot Bypass, Boot Bypass and Read Only}
		SystemCreationClassName string         `xml:"SystemCreationClassName"` // The scoping System's CreationClassName.
		SystemName              string         `xml:"SystemName"`              // The scoping System's Name.
	}

	Time struct {
		DateTime string `xml:"Datetime"`
	}

	// ChassisPackageType is an enumeration defining the type of the PhysicalPackage.
	ChassisPackageType int

	// OperationalStatus is the current statuses of the element.
	OperationalStatus int

	// PackageType is the type of the PhysicalPackage.
	PackageType int

	// TargetOperatingSystem is the element's operating system environment.
	TargetOperatingSystem int

	// SoftwareElementState is defined in this model to identify various states of a SoftwareElement's life cycle.
	SoftwareElementState int

	// HealthState is an enumeration of the possible values for the HealthState property.
	HealthState int

	// EnabledState is an enumeration of the possible values for the EnabledState property.
	EnabledState int

	// RequestedState is an enumeration of the possible values for the RequestedState property.
	RequestedState int

	// UpgradeMethod is an enumeration of the possible values for the UpgradeMethod property.
	UpgradeMethod int

	// CPUStatus is an enumeration of the possible values for the CPUStatus property.
	CPUStatus int

	// MemoryType is an enumeration that describes the type of memory.
	MemoryType int

	// Capabilities is an integer enumeration that indicates the various capabilities of the media access device.
	Capabilities int

	// EnabledDefault is an integer enumeration that indicates the administrator's default or startup configuration for the Enabled State of an element.
	EnabledDefault int

	// Security is an integer enumeration that indicates the security supported by the media access device.
	Security int
)
