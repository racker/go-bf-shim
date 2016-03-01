package observations

type Observation struct {
	Version          uint64                        `json:"version"`
	AccountId        string                        `json:"accountId"`
	Available        bool                          `json:"available"`
	CheckId          string                        `json:"checkId"`
	CheckType        *string                       `json:"checkType"`
	CollectorId      *string                       `json:"collectorId"`
	CollectorKeys    []string                      `json:"collectorKeys"`
	EntityId         string                        `json:"entityId"`
	MonitoringZoneId *string                       `json:"monitoringZoneId"`
	Period           int32                         `json:"period"`
	Status           *string                       `json:"status"`
	Target           string                        `json:"target"`
	TenantId         *string                       `json:"tenantId"`
	Timestamp        int64                         `json:"timestamp"`
	Metrics          map[string]*ObservationMetric `json:"metrics"`
}

type ObservationMetric struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
	Unit  string      `json:"unit"`
}

