package dto

type CIMResponse struct {
	Response  interface{}   `json:"response,omitempty"`
	Responses []interface{} `json:"responses,omitempty"`
	Status    int           `json:"status,omitempty"`
}

type HardwareInfo struct {
	CIMComputerSystemPackage CIMResponse `json:"CIM_ComputerSystemPackage,omitempty"`
	CIMSystemPackaging       CIMResponse `json:"CIM_SystemPackaging,omitempty"`
	CIMChassis               CIMResponse `json:"CIM_Chassis,omitempty"`
	CIMChip                  CIMResponse `json:"CIM_Chip,omitempty"`
	CIMCard                  CIMResponse `json:"CIM_Card,omitempty"`
	CIMBIOSElement           CIMResponse `json:"CIM_BIOSElement,omitempty"`
	CIMProcessor             CIMResponse `json:"CIM_Processor,omitempty"`
	CIMPhysicalMemory        CIMResponse `json:"CIM_PhysicalMemory,omitempty"`
}

type DiskInfo struct {
	CIMMediaAccessDevice CIMResponse `json:"CIM_MediaAccessDevice,omitempty"`
	CIMPhysicalPackage   CIMResponse `json:"CIM_PhysicalPackage,omitempty"`
}

type GeneralSettings struct {
	Header interface{} `json:"header,omitempty"`
	Body   interface{} `json:"body,omitempty"`
}
