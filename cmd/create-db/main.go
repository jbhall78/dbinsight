package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-mysql-org/go-mysql/client"
)

func main() {
	// MySQL connection details
	user := "jhall"
	pass := "test"
	host := "127.0.0.1"
	port := "3306" // Or your port
	dbName := "dbinsight_test"

	// Connect to MySQL (including database name in DSN)
	fmt.Println("Connecting to server...")
	c, err := client.Connect(fmt.Sprintf("%s:%s", host, port), user, pass, dbName)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	fmt.Println("Running test...")
	_, err = c.Execute("INSERT INTO foo VALUES (1)")
	if err != nil {
		log.Printf("Couldn't insert value: %s", dbName)
	}

	fmt.Println("Dropping database...")
	_, err = c.Execute(fmt.Sprintf("DROP DATABASE %s", dbName))
	if err != nil {
		log.Printf("Couldn't drop database: %s", dbName)
	}
	// Create the database if it doesn't exist
	fmt.Println("Creating database...")
	_, err = c.Execute(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		log.Fatalf("Error creating database: %v", err)
	}

	//Select the database
	fmt.Println("Using database...")
	if _, err := c.Execute(fmt.Sprintf("USE %s", dbName)); err != nil {
		log.Fatal(err)
	}

	// Read the SQL file
	files := []string{
		"products.sql",
		"data_types_demo.sql",
	}
	var sqlBytes []byte
	for _, filename := range files {
		sqlFile := "data/schema/" + filename
		sqlBytes, err = os.ReadFile(sqlFile)
		if err != nil {
			sqlFile = "../../" + sqlFile
			sqlBytes, err = os.ReadFile(sqlFile)
			if err != nil {
				log.Fatalf("Error reading SQL file: %v", err)
			}
		}

		sql := string(sqlBytes)

		// Execute the SQL
		fmt.Println("Executing Queries: ", sql)
		_, err = c.Execute(sql)
		if err != nil {
			log.Fatalf("Error executing SQL: %v", err)
		}

		fmt.Printf("SQL from %s executed successfully.\n", sqlFile)
	}
}
