package options

// OutputOptions 定义结果输出的通用参数
type OutputOptions struct {
	OutputExcel string // -oe, --outputExcel
	OutputTxt   string // -ot, --outputTxt
}

// ApplyToParams 将输出参数应用到 Task 的 Params 中
func (o *OutputOptions) ApplyToParams(params map[string]interface{}) {
	if o.OutputExcel != "" {
		params["output_excel"] = o.OutputExcel
	}
	if o.OutputTxt != "" {
		params["output_txt"] = o.OutputTxt
	}
}
