// 文件处理工具包
// 提供文件读取、写入、判断等操作
package utils

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

// 从文件读取内容返回列表
// filePath: 文件路径
// 返回: 内容列表, 错误信息
func ReadFileLines(filePath string) ([]string, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("文件不存在: %s", filePath)
	}

	// 读取文件内容
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件内容失败: %v", err)
	}

	// 按行分割内容，处理不同操作系统的换行符
	lines := strings.Split(string(content), "\n")

	// 移除可能的空行和包含仅换行符的内容
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		// 移除行末的回车符（Windows换行符 \r\n）
		line = strings.TrimSuffix(line, "\r")
		// 只添加非空行或包含非空白字符的行
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") { // 增加对注释行(#开头)的过滤
			result = append(result, line)
		}
	}

	return result, nil
}

// 从csv文件读取指定列返回列表
// filePath: csv文件路径
// column: 列索引(从0开始)
// 返回: 列内容列表, 错误信息
func ReadCSVColumn(filePath string, column int) ([]string, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("文件不存在: %s", filePath)
	}

	// 使用标准库的 csv 包来处理 CSV 格式
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// 提取指定列内容
	columnValues := make([]string, 0)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("读取CSV行失败: %v", err)
		}

		// 检查列索引是否有效
		if column >= 0 && column < len(record) {
			columnValues = append(columnValues, strings.TrimSpace(record[column]))
		}
	}

	return columnValues, nil
}
