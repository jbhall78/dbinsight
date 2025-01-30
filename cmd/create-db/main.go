package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-mysql-org/go-mysql/client"
)

func main() {
	// MySQL connection details
	user := "test"
	pass := "test"
	host := "127.0.0.1"
	port := "3306" // Or your port
	dbName := "dbinsight_test"

	// Connect to MySQL (including database name in DSN)
	c, err := client.Connect(fmt.Sprintf("%s:%s", host, port), user, pass, "")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// Create the database if it doesn't exist
	_, err = c.Execute(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName))
	if err != nil {
		log.Fatalf("Error creating database: %v", err)
	}

	//Select the database
	if _, err := c.Execute(fmt.Sprintf("USE %s", dbName)); err != nil {
		log.Fatal(err)
	}

	// Read the SQL file
	sqlFile := "data/schema/create_tables.sql"
	sqlBytes, err := os.ReadFile(sqlFile)
	if err != nil {
		log.Fatalf("Error reading SQL file: %v", err)
	}

	sql := string(sqlBytes)

	// Execute the SQL
	_, err = c.Execute(sql)
	if err != nil {
		log.Fatalf("Error executing SQL: %v", err)
	}

	fmt.Printf("SQL from %s executed successfully.\n", sqlFile)
}
