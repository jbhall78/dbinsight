package main

import (
	"fmt"
	"log"
	"regexp" // For regular expressions
	"strconv"
	"strings"
)

// Simplified AST structures (you'll need to expand these)

type SQLCommand int

const (
	Set = iota

	// read only commands
	Select
	Show
	Use
	Desc
	Describe

	// write commands
	Insert
	Update
	Delete
	Create
	Alter
	Drop
	Truncate
	Rename
	Grant
	Revoke
)

// Very basic tokenizer (you'll need to make this much more robust)
func tokenize(query string) []string {
	return strings.Fields(query) // Split by spaces (very simplistic)
}

func parseSQL(query string) ([]interface{}, error) {
	statements := splitAndProcessStatements(query, "8.0.33")

	parsedStatements := make([]interface{}, 0)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt) // Remove leading/trailing whitespace

		if stmt == "" { //Skip empty statements
			continue
		}

		tokens := tokenize(stmt)
		if len(tokens) == 0 {
			continue // Skip empty statements
		}

		parsedStmt, err := parseStatement(tokens) // Parse each individual statement
		if err != nil {
			return nil, err // Return error immediately if any statement fails
		}
		parsedStatements = append(parsedStatements, parsedStmt)
	}
	return parsedStatements, nil
}

func splitAndProcessStatements(query, mysqlVersion string) []string {
	var statements []string
	re := regexp.MustCompile(`(?s)/\*!\s*(\d+)(.*?)\*/|([^;]+);?`) // Updated regex

	matches := re.FindAllStringSubmatch(query, -1)
	for _, match := range matches {
		if match[1] != "" { // Conditional comment
			versionRequired, err := strconv.Atoi(match[1])
			if err != nil {
				log.Println("Error parsing version number:", err)
				continue // Skip invalid conditional comment
			}

			// Simplified version comparison (you might need a more robust one)
			versionParts := strings.Split(mysqlVersion, ".")
			major, _ := strconv.Atoi(versionParts[0])

			if major >= versionRequired/10000 { // Execute conditional command
				conditionalStmt := strings.TrimSpace(match[2])
				if conditionalStmt != "" {
					statements = append(statements, conditionalStmt)
				}
			}
		} else if match[3] != "" { // Regular statement
			statement := strings.TrimSpace(match[3])
			if statement != "" {
				statements = append(statements, statement)
			}
		}
	}

	return statements
}

/*
func parseSQL(query string) ([]interface{}, error) {
	// Remove comments using regular expressions
	query = removeComments(query)

	// Split the query into individual statements by ';'
	statements := strings.Split(query, ";")

	parsedStatements := make([]interface{}, 0)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt) // Remove leading/trailing whitespace

		if stmt == "" { //Skip empty statements
			continue
		}

		tokens := tokenize(stmt)
		if len(tokens) == 0 {
			continue // Skip empty statements
		}

		parsedStmt, err := parseStatement(tokens) // Parse each individual statement
		if err != nil {
			return nil, err // Return error immediately if any statement fails
		}
		parsedStatements = append(parsedStatements, parsedStmt)
	}

	return parsedStatements, nil
}
*/

// Helper function to parse a single SQL statement (from the previous 'parse' example)
func parseStatement(tokens []string) (int, error) {
	// ... (This function remains the same as in the previous example)
	if len(tokens) == 0 {
		return 0, fmt.Errorf("empty statement")
	}

	if len(tokens) < 2 {
		return 0, fmt.Errorf("invalid statement")
	}

	switch strings.ToUpper(tokens[0]) {
	case "SET":
		return Set, nil
	// read only commands
	case "SELECT":
		return Select, nil
	case "SHOW":
		return Show, nil
	case "USE":
		return Use, nil
	case "DESC":
		return Desc, nil
	case "DESCRIBE":
		return Describe, nil
	// write commands
	case "INSERT":
		return Insert, nil
	case "UPDATE":
		return Update, nil
	case "DELETE":
		return Delete, nil
	case "CREATE":
		return Create, nil
	case "ALTER":
		return Alter, nil
	case "DROP":
		return Drop, nil
	case "TRUNCATE":
		return Truncate, nil
	case "RENAME":
		return Rename, nil
	case "GRANT":
		return Grant, nil
	case "REVOKE":
		return Revoke, nil
	default:
		return 0, fmt.Errorf("unsupported statement type: %s", tokens[0])
	}
}

func removeComments(query string) string {
	// Regex to match both types of comments (multi-line and single-line)
	commentRegex := regexp.MustCompile(`(?s)/\*.*?\*/|--.*$`) // (?s) makes . match newlines
	return commentRegex.ReplaceAllString(query, "")
}

// example usage:

//func main() {
//    mysqlVersion := "8.0.33" // Example MySQL version (you would get this from the connection)
//    sqlQuery := `
//        SELECT id, name FROM users WHERE status = 1; -- This is a comment
//        /*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
//        UPDATE products SET price = 10.99 WHERE category = 'electronics'; /* Multi-line
//        comment */ /*!50000 INSERT INTO orders (user_id, product_id) VALUES (1, 2) */; # Another comment
//        SELECT * from users;
//    `
//
//   parsedStmts, err := parse(sqlQuery, mysqlVersion) // Pass MySQL version
//    if err != nil {
//        log.Fatal(err)
//    }
// ... (Rest of the main function remains the same)
//}
