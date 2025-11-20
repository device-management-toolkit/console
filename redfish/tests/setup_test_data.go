package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Get database path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dbPath := filepath.Join(homeDir, ".config", "device-management-toolkit", "console.db")

	log.Printf("Opening database: %s\n", dbPath)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Insert test device
	deviceInfo := `{"BuildNumber":"3000","FWBuild":"1234","FWVersion":"16.0.0","FQDN":"test-device-1.local","IPConfiguration":{"IPAddress":"192.168.1.100","MACAddress":"00:11:22:33:44:55"},"OS":"Windows 10","SKU":"vPro","UUID":"12345678-1234-1234-1234-123456789012","features":{"AMT":true,"KVM":true,"SOL":true}}`

	query := `INSERT OR REPLACE INTO devices (
        guid, hostname, tags, mpsinstance, connectionstatus, 
        mpsusername, tenantid, friendlyname, dnssuffix, deviceinfo
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	ctx := context.Background()

	_, err = db.ExecContext(ctx, query,
		"test-system-1",
		"test-device-1.local",
		"[]",
		"",
		true,
		"admin",
		"",
		"Test System 1",
		"local",
		deviceInfo,
	)
	if err != nil {
		return err
	}

	log.Println("✓ Test device inserted successfully")
	log.Println("  System ID: test-system-1")
	log.Println("  Hostname: test-device-1.local")
	log.Println("  Friendly Name: Test System 1")

	return nil
}
