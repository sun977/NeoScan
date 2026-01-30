package adapter

// TopLevelResult 对应 TaskResult.Result
type TopLevelResult struct {
	Attributes interface{}       `json:"attributes"`
	Evidence   map[string]string `json:"evidence,omitempty"`
}

// CommonSummary 通用摘要
type CommonSummary struct {
	StartTime int64 `json:"start_time,omitempty"`
	EndTime   int64 `json:"end_time,omitempty"`
	ElapsedMs int64 `json:"elapsed_ms,omitempty"`
}
