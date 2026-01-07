package etl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orcModel "neomaster/internal/model/orchestrator"

	"github.com/stretchr/testify/assert"
)

func TestWebCrawlerDataHandler_Handle(t *testing.T) {
	// Setup temporary directory
	tmpDir, err := os.MkdirTemp("", "crawler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	handler := &webCrawlerDataHandler{
		storageDir: tmpDir,
	}

	// Mock data
	screenshotData := []byte("fake-image-data")
	encodedScreenshot := base64.StdEncoding.EncodeToString(screenshotData)
	htmlContent := "<html><body>Test</body></html>"

	output := CrawlerOutput{
		URL:        "http://example.com",
		Title:      "Example",
		Screenshot: encodedScreenshot,
		HTML:       htmlContent,
	}
	outputJSON, _ := json.Marshal(output)

	result := &orcModel.StageResult{
		TaskID:     "task-123",
		Attributes: string(outputJSON),
	}

	// Execute
	processed, err := handler.Handle(context.Background(), result)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, processed)
	assert.Equal(t, "Example", processed.Title)
	assert.NotEmpty(t, processed.ScreenshotID)
	assert.NotEmpty(t, processed.HTMLHash)

	// Verify file saved
	savedFile := filepath.Join(tmpDir, processed.ScreenshotID)
	content, err := os.ReadFile(savedFile)
	assert.NoError(t, err)
	assert.Equal(t, screenshotData, content)
}

func TestWebCrawlerDataHandler_Handle_InvalidBase64(t *testing.T) {
	// Setup temporary directory
	tmpDir, err := os.MkdirTemp("", "crawler_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	handler := &webCrawlerDataHandler{
		storageDir: tmpDir,
	}

	output := CrawlerOutput{
		URL:        "http://example.com",
		Screenshot: "invalid-base64-string!@#",
	}
	outputJSON, _ := json.Marshal(output)

	result := &orcModel.StageResult{
		TaskID:     "task-124",
		Attributes: string(outputJSON),
	}

	// Execute
	processed, err := handler.Handle(context.Background(), result)

	// Verify - Should not fail entire process, just log error
	assert.NoError(t, err)
	assert.NotNil(t, processed)
	assert.Empty(t, processed.ScreenshotID) // No screenshot ID should be set
}
