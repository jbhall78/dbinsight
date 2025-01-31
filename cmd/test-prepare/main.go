package main

import (
	"fmt"
	"log"
//	"strconv"

	"github.com/go-mysql-org/go-mysql/client"
)

func main() {
	// MySQL connection details (replace with your credentials)
	user := "test"
	pass := "test"
	host := "127.0.0.1"
	port := "3306" // Or your port
	dbName := "mysql"

	// Connect to MySQL (including database name in DSN)
	c, err := client.Connect(fmt.Sprintf("%s:%s", host, port), user, pass, dbName)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// 1. Prepare the statement
	query := "SELECT user FROM user WHERE user = ?"
	stmt, err := c.Prepare(query)
	if err != nil {
		log.Fatal("Error preparing statement:", err)
	}
	defer stmt.Close()

	// 2. Input username to check
	usernameToCheck := "root" // The username you want to check

	// 3. Execute the prepared statement
	result, err := stmt.Execute(usernameToCheck)
	if err != nil {
		log.Fatal("Error executing statement:", err)
	}
	defer result.Close()

	// 4. Process the results
	var retrievedUsername string

	// Handle resultset
	//v, _ := result.GetInt(0, 0)
	//v, _ = r.GetIntByName(0, "id")

	retrievedUsername, err = result.GetStringByName(0, "user")
	//v, err := result.GetInt(0, 0)
	//retrievedUsername = strconv.FormatInt(int64(v), 10) //base 10

	if err != nil {
		log.Fatal("result.GetValueByName failed")
	}

	// Direct access to fields
	//for _, row := range r.Values {
	//	for _, val := range row {
	//		_ = val.Value() // interface{}
	//		// or
	//		if val.Type == mysql.FieldValueTypeString {
	//			_ = val.AsFloat64() // float64
	//}

	if retrievedUsername != usernameToCheck {
		log.Fatal("retrievedUsername != usernameToCheck")
	}

	fmt.Println("Success!")
}
