package dto

// EventLog defines the structure of an event log.
type EventLogs struct {
	Records        []EventLog `json:"records"`
	HasMoreRecords bool       `json:"hasMoreRecords"`
}

type EventLog struct {
	DeviceAddress   int    `json:"DeviceAddress"`
	EventSensorType int    `json:"EventSensorType"`
	EventType       int    `json:"EventType"`
	EventOffset     int    `json:"EventOffset"`
	EventSourceType int    `json:"EventSourceType"`
	EventSeverity   string `json:"EventSeverity"`
	SensorNumber    int    `json:"SensorNumber"`
	Entity          string `json:"Entity"`
	EntityInstance  int    `json:"EntityInstance"`
	EventData       []int  `json:"EventData"`
	Time            string `json:"Time"`
	EntityStr       string `json:"EntityStr"`
	Description     string `json:"Desc"`
	EventTypeDesc   string `json:"eventTypeDesc"`
}
