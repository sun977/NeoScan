/**
 * Web爬虫数据处理器 - 资产清洗引擎
 * @author: sun977
 * @date: 2025.01.06
 * @description: 专门处理 Web 爬虫产生的非结构化/大体积数据 (如 HTML 源码, 截图, JS 文件等)，
 * 提取关键信息 (Title, Fingerprint, Favicon) 并将原始数据归档。
 */
package etl

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"neomaster/internal/config"
	orcModel "neomaster/internal/model/orchestrator"
	"neomaster/internal/pkg/logger"
)

// WebCrawlerDataHandler Web爬虫数据处理器接口
// 职责: 处理 Web 爬虫产生的非结构化/大体积数据 (如 HTML 源码, 截图, JS 文件等)
type WebCrawlerDataHandler interface {
	// Handle 处理爬虫数据
	// result: 包含爬虫结果的 StageResult
	// 返回值:
	// - processedData: 处理后的关键数据 (如提取的 Title, Fingerprint, Favicon Hash)
	// - error: 处理错误
	Handle(ctx context.Context, result *orcModel.StageResult) (*ProcessedWebData, error)
}

// ProcessedWebData 处理后的 Web 数据
type ProcessedWebData struct {
	Title        string            `json:"title"`
	Headers      map[string]string `json:"headers"`
	TechStack    []string          `json:"tech_stack"` // 识别到的技术栈
	ScreenshotID string            `json:"screenshot_id,omitempty"`
	HTMLHash     string            `json:"html_hash,omitempty"`
}

// webCrawlerDataHandler 默认实现
type webCrawlerDataHandler struct {
	storageDir string // 本地存储目录 (e.g. "./data/web_evidence")
}

// NewWebCrawlerDataHandler 创建 Web 爬虫数据处理器
func NewWebCrawlerDataHandler() WebCrawlerDataHandler {
	// 优先使用配置中的路径，如果未配置则使用默认值
	storageDir := "./data/web_evidence"
	if cfgPath := config.GetConfig().App.Master.WebCrawler.StoragePath; cfgPath != "" {
		storageDir = cfgPath
	}

	if err := os.MkdirAll(storageDir, 0755); err != nil {
		logger.LogBusinessError(err, "", 0, "", "etl.NewWebCrawlerDataHandler", "INIT", nil)
	}
	return &webCrawlerDataHandler{
		storageDir: storageDir,
	}
}

// CrawlerOutput 定义爬虫工具的标准输出结构 (与 Agent 端 Output Normalizer 对齐)
// 假设爬虫工具输出的 JSON 结构如下
type CrawlerOutput struct {
	URL        string            `json:"url"`
	Title      string            `json:"title"`
	Headers    map[string]string `json:"headers"`
	TechStack  []string          `json:"tech_stack"`
	HTML       string            `json:"html"`       // 可能很大
	Screenshot string            `json:"screenshot"` // Base64 编码
}

// Handle 实现接口
func (h *webCrawlerDataHandler) Handle(ctx context.Context, result *orcModel.StageResult) (*ProcessedWebData, error) {
	// 1. 解析 result.Output (Attributes 字段通常存储的是结构化摘要，Evidence 或 raw output 存储大体积数据)
	// 在 StageResult 模型中，Attributes 是结构化的 JSON，Evidence 是原始证据。
	// 对于爬虫，Agent 可能会把 Base64 截图放在 Evidence 里，或者直接在 Attributes 里。
	// 这里假设关键数据在 Attributes 中 (如果是大体积数据，建议 Agent 上传到对象存储后只传 URL，但目前假设是 Base64)

	var output CrawlerOutput
	// 尝试从 Attributes 解析 (假设 Mapper 已经做了一层转换，或者直接解析原始数据)
	// 注意：StageResult.Attributes 是 JSON 字符串
	if err := json.Unmarshal([]byte(result.Attributes), &output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal crawler attributes: %w", err)
	}

	processed := &ProcessedWebData{
		Title:     output.Title,
		Headers:   output.Headers,
		TechStack: output.TechStack,
	}

	// 2. 处理截图 (Base64 -> 文件)
	if output.Screenshot != "" {
		// 生成文件名: taskID_timestamp.png
		filename := fmt.Sprintf("%s_%d.png", result.TaskID, time.Now().UnixNano())
		filePath := filepath.Join(h.storageDir, filename)

		// 解码 Base64
		// 某些前缀可能包含 "data:image/png;base64,"，需要去除
		rawBase64 := output.Screenshot
		if idx := len("data:image/png;base64,"); len(rawBase64) > idx && rawBase64[:idx] == "data:image/png;base64," {
			rawBase64 = rawBase64[idx:]
		}

		data, err := base64.StdEncoding.DecodeString(rawBase64)
		if err != nil {
			// 仅记录错误但不中断流程，因为截图是非关键数据
			logger.LogBusinessError(err, "decode_screenshot", 0, "", "etl.web_crawler.Handle", "DECODE", map[string]interface{}{
				"task_id": result.TaskID,
			})
		} else {
			if err := os.WriteFile(filePath, data, 0644); err != nil {
				logger.LogBusinessError(err, "save_screenshot", 0, "", "etl.web_crawler.Handle", "SAVE", map[string]interface{}{
					"path": filePath,
				})
			} else {
				// 记录相对路径或 ID
				processed.ScreenshotID = filename
				logger.LogInfo("Screenshot saved", "", 0, "", "etl.web_crawler.Handle", "", map[string]interface{}{
					"path": filePath,
				})
			}
		}
	}

	// 3. 处理 HTML (计算 Hash 用于去重)
	if output.HTML != "" {
		hash := md5.Sum([]byte(output.HTML))
		processed.HTMLHash = hex.EncodeToString(hash[:])
		// 可选：将 HTML 归档
		// 如果需要归档 HTML，可以在这里写入文件，类似截图处理
	}

	return processed, nil
}
