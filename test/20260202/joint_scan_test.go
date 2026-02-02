package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Config
const (
	masterPort = 8123
	agentPort  = 8321
	masterURL  = "http://localhost:8123/api/v1"
	dsn        = "root:ROOT@tcp(localhost:3306)/neoscan_dev?charset=utf8mb4&parseTime=True&loc=Local"
)

func TestJointScan(t *testing.T) {
	// 1. Setup Environment
	rootDir := setupEnv(t)
	// defer cleanup(rootDir)

	// 2. Check/Start Master
	if isPortOpen(masterPort) {
		log.Printf("Master port %d is already open, skipping start.", masterPort)
	} else {
		buildMaster(t, rootDir)
		masterCmd := startProcess(t, filepath.Join(rootDir, "bin", "neoMaster.exe"), filepath.Join(rootDir, "neoMaster"), "server")
		defer killProcess(masterCmd)
		if !waitForPort(masterPort, 30*time.Second) {
			t.Fatal("Master failed to start")
		}
	}

	// 3. Check/Start Agent
	if isPortOpen(agentPort) {
		log.Printf("Agent port %d is already open, skipping start.", agentPort)
	} else {
		buildAgent(t, rootDir)
		// Start Agent
		// Change cwd to neoAgent directory so it can find configs/config.yaml
		agentCmd := startProcess(t, filepath.Join(rootDir, "bin", "neoAgent.exe"), filepath.Join(rootDir, "neoAgent"), "server")
		defer func() {
			killProcess(agentCmd)
		}()
		if !waitForPort(agentPort, 30*time.Second) {
			t.Fatal("Agent failed to start")
		}
	}

	// 4. Execute Scan Logic
	// Authenticate first
	authToken = registerAndLogin(t)
	executeScanWorkflow(t)
}

var authToken string

func registerAndLogin(t *testing.T) string {
	// 1. Register
	username := "testuser_" + time.Now().Format("20060102150405")
	email := username + "@example.com"
	password := "TestPass123!"

	regPayload := map[string]interface{}{
		"username": username,
		"email":    email,
		"password": password,
	}
	// We use a temporary helper that doesn't require auth token yet
	sendRequestNoAuth(t, "POST", "/auth/register", regPayload)
	log.Printf("Registered user: %s", username)

	// 2. Login
	loginPayload := map[string]interface{}{
		"username": username,
		"password": password,
	}
	resp := sendRequestNoAuth(t, "POST", "/auth/login", loginPayload)

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Login response missing data: %v", resp)
	}

	token, ok := data["access_token"].(string)
	if !ok {
		// Try just "token"
		token, ok = data["token"].(string)
		if !ok {
			t.Fatalf("Login response missing access_token:Keys: %v", data)
		}
	}
	log.Printf("Logged in successfully")
	return token
}

func sendRequestNoAuth(t *testing.T, method, endpoint string, payload interface{}) map[string]interface{} {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(method, masterURL+endpoint, body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		// Don't fatal here for register conflict (409) if we retry, but here we use unique user
		t.Fatalf("API Error %d: %s", resp.StatusCode, string(b))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	return result
}

func verifyDB(t *testing.T, projectID uint64) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	// Check Project
	var name, status string
	err = db.QueryRow("SELECT name, status FROM projects WHERE id = ?", projectID).Scan(&name, &status)
	if err != nil {
		t.Fatalf("Failed to query project %d: %v", projectID, err)
	}
	t.Logf("DB Verification - Project: ID=%d, Name=%s, Status=%s", projectID, name, status)

	// Check Tasks
	rows, err := db.Query("SELECT id, status, tool_name FROM agent_tasks WHERE project_id = ?", projectID)
	if err != nil {
		t.Fatalf("Failed to query tasks: %v", err)
	}
	defer rows.Close()

	taskCount := 0
	for rows.Next() {
		var id int
		var status, tool string
		rows.Scan(&id, &status, &tool)
		t.Logf("DB Verification - Task: ID=%d, Status=%s, Tool=%s", id, status, tool)
		taskCount++
	}
	if taskCount == 0 {
		t.Errorf("No tasks found for project %d in DB!", projectID)
	}

	// Check Agents
	// We expect at least one agent to be registered
	agentRows, err := db.Query("SELECT agent_id, hostname, status, token FROM agents")
	if err != nil {
		t.Fatalf("Failed to query agents: %v", err)
	}
	defer agentRows.Close()

	agentCount := 0
	for agentRows.Next() {
		var aid, hostname, status, token string
		agentRows.Scan(&aid, &hostname, &status, &token)
		t.Logf("DB Verification - Agent: ID=%s, Hostname=%s, Status=%s, TokenPrefix=%s...", aid, hostname, status, token[:10])
		agentCount++
	}
	if agentCount == 0 {
		t.Errorf("No agents found in DB! Agent registration failed.")
	}
}

func setupEnv(t *testing.T) string {
	// Get project root
	_, filename, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(filename)
	// Assuming test/20260202 -> root is ../../
	rootDir := filepath.Clean(filepath.Join(testDir, "..", ".."))

	// Create bin dir
	binDir := filepath.Join(rootDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin dir: %v", err)
	}
	return rootDir
}

func buildMaster(t *testing.T, rootDir string) {
	log.Println("Building neoMaster...")
	cmd := exec.Command("go", "build", "-o", filepath.Join(rootDir, "bin", "neoMaster.exe"), "./cmd/master")
	cmd.Dir = filepath.Join(rootDir, "neoMaster")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build neoMaster: %v", err)
	}
}

func buildAgent(t *testing.T, rootDir string) {
	log.Println("Building neoAgent...")
	cmd := exec.Command("go", "build", "-o", filepath.Join(rootDir, "bin", "neoAgent.exe"), "./cmd/agent")
	cmd.Dir = filepath.Join(rootDir, "neoAgent")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build neoAgent: %v", err)
	}
}

func startProcess(t *testing.T, binPath, cwd string, args ...string) *exec.Cmd {
	log.Printf("Starting %s...", filepath.Base(binPath))
	cmd := exec.Command(binPath, args...)
	cmd.Dir = cwd

	// Redirect output for debugging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start %s: %v", binPath, err)
	}
	return cmd
}

func killProcess(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
	}
}

func isPortOpen(port int) bool {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/health", port))
	if err == nil {
		resp.Body.Close()
		return true
	}
	// Try root just in case health check is different
	resp, err = http.Get(fmt.Sprintf("http://localhost:%d/", port))
	if err == nil {
		resp.Body.Close()
		return true
	}
	return false
}

func waitForPort(port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isPortOpen(port) {
			return true
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

func executeScanWorkflow(t *testing.T) {
	// A. Create Project
	projectID := createProject(t)
	log.Printf("Created Project ID: %d", projectID)

	// B. Create Workflow
	workflowID := createWorkflow(t)
	log.Printf("Created Workflow ID: %d", workflowID)

	// C. Create Stage
	createStage(t, workflowID)
	log.Println("Created Scan Stage")

	// D. Add Workflow to Project
	addWorkflowToProject(t, projectID, workflowID)
	log.Println("Added Workflow to Project")

	// E. Start Project (Trigger Scheduler)
	startProject(t, projectID)
	log.Println("Started Project")

	// F. Monitor
	// Wait for task generation and execution
	// In a real test, we would poll the task status API
	log.Println("Waiting for scan execution (30s)...")
	time.Sleep(30 * time.Second)

	// Verify (Optional: Check if task exists)
	// checkTasks(t, projectID)
}

// API Helpers
func createProject(t *testing.T) uint64 {
	payload := map[string]interface{}{
		"name":         "Joint_Test_Project_" + time.Now().Format("20060102150405"),
		"description":  "Automated joint test",
		"target_scope": "10.44.96.1/24,10.44.96.183",
	}
	resp := sendRequest(t, "POST", "/orchestrator/projects", payload)
	// Adjust based on actual response structure: {"code": 200, "data": {"id": 1, ...}}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid response data format: %v", resp)
	}
	id, ok := data["id"].(float64)
	if !ok {
		t.Fatalf("Invalid ID format: %v", data)
	}
	return uint64(id)
}

func createWorkflow(t *testing.T) uint64 {
	payload := map[string]interface{}{
		"name":        "Joint_Test_Workflow_" + time.Now().Format("20060102150405"),
		"description": "Workflow for joint test",
	}
	resp := sendRequest(t, "POST", "/orchestrator/workflows", payload)
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid response data format: %v", resp)
	}
	id, ok := data["id"].(float64)
	if !ok {
		t.Fatalf("Invalid ID format: %v", data)
	}
	return uint64(id)
}

func createStage(t *testing.T, workflowID uint64) {
	payload := map[string]interface{}{
		"workflow_id": workflowID,
		"stage_name":  "Nmap_Fast_Scan",
		"tool_name":   "nmap",
		"tool_params": "-F",
		"execution_policy": map[string]interface{}{
			"priority": 1,
		},
	}
	sendRequest(t, "POST", "/orchestrator/stages", payload)
}

func addWorkflowToProject(t *testing.T, projectID, workflowID uint64) {
	payload := map[string]interface{}{
		"workflow_id": workflowID,
		"sort_order":  1,
	}
	sendRequest(t, "POST", fmt.Sprintf("/orchestrator/projects/%d/workflows", projectID), payload)
}

func startProject(t *testing.T, projectID uint64) {
	payload := map[string]interface{}{
		"status": "running",
	}
	sendRequest(t, "PUT", fmt.Sprintf("/orchestrator/projects/%d", projectID), payload)
}

func sendRequest(t *testing.T, method, endpoint string, payload interface{}) map[string]interface{} {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(method, masterURL+endpoint, body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("API Error %d: %s", resp.StatusCode, string(b))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	return result
}
