package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "root:ROOT@tcp(localhost:3306)/neoscan_dev?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to neoscan_dev")

	// 1. Check Projects
	rows, err := db.Query("SELECT id, name, status, created_at FROM projects ORDER BY id DESC LIMIT 5")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("\n--- Recent Projects ---")
	for rows.Next() {
		var id int
		var name, status string
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &status, &createdAt); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ID: %d, Name: %s, Status: %s, CreatedAt: %s\n", id, name, status, createdAt)
	}

	// 2. Check Tasks
	rows2, err := db.Query("SELECT id, project_id, status, tool_name, created_at FROM agent_tasks ORDER BY id DESC LIMIT 5")
	if err != nil {
		log.Fatal(err)
	}
	defer rows2.Close()

	fmt.Println("\n--- Recent Tasks ---")
	for rows2.Next() {
		var id, projectID int
		var status, toolName string
		var createdAt time.Time
		if err := rows2.Scan(&id, &projectID, &status, &toolName, &createdAt); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ID: %d, ProjectID: %d, Status: %s, Tool: %s, CreatedAt: %s\n", id, projectID, status, toolName, createdAt)
	}

    // 3. Check Assets
    rows3, err := db.Query("SELECT id, ip, created_at FROM asset_hosts ORDER BY id DESC LIMIT 5")
    if err != nil {
        log.Fatal(err)
    }
    defer rows3.Close()

    fmt.Println("\n--- Recent Assets ---")
    for rows3.Next() {
        var id int
        var ip string
        var createdAt time.Time
        if err := rows3.Scan(&id, &ip, &createdAt); err != nil {
            log.Fatal(err)
        }
        fmt.Printf("ID: %d, IP: %s, CreatedAt: %s\n", id, ip, createdAt)
    }
}
