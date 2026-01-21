package options

// OutputOptions 定义结果输出的通用参数
type OutputOptions struct {
	OutputCsv  string // -oc, --outputCsv
	OutputJson string // -oj, --outputJson
}

// ApplyToParams 将输出参数应用到 Task 的 Params 中
func (o *OutputOptions) ApplyToParams(params map[string]interface{}) {
	if o.OutputCsv != "" {
		params["output_csv"] = o.OutputCsv
	}
	if o.OutputJson != "" {
		params["output_json"] = o.OutputJson
	}
}
