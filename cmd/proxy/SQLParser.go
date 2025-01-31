package main

import (
	"fmt"
	"regexp" // For regular expressions
	"strings"
)

// Simplified AST structures (you'll need to expand these)

type Select struct {
	From  string
	Where *WhereClause // Could be nil
}

type Update struct {
	Table string
	Where *WhereClause // Could be nil
	Set   []*SetClause //Example Set clauses
}

type Insert struct {
	Table   string
	Columns []string
	Values  [][]string //Example Values
}

type WhereClause struct {
	// ... (expressions, conditions, etc.)
}

type SetClause struct {
	Column string
	Value  string
}

// Very basic tokenizer (you'll need to make this much more robust)
func tokenize(query string) []string {
	return strings.Fields(query) // Split by spaces (very simplistic)
}

func parseSQL(query string) ([]interface{}, error) {
	// Remove comments (both -- and /* */ style) using regular expressions
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

// Helper function to parse a single SQL statement (from the previous 'parse' example)
func parseStatement(tokens []string) (interface{}, error) {
	// ... (This function remains the same as in the previous example)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty statement")
	}

	switch strings.ToUpper(tokens[0]) {
	case "SELECT":
		if len(tokens) < 4 {
			return nil, fmt.Errorf("invalid SELECT statement")
		}
		return &Select{From: tokens[3]}, nil // Example

	case "UPDATE":
		if len(tokens) < 4 {
			return nil, fmt.Errorf("invalid UPDATE statement")
		}
		return &Update{Table: tokens[1]}, nil // Example

	case "INSERT":
		if len(tokens) < 3 {
			return nil, fmt.Errorf("invalid INSERT statement")
		}
		return &Insert{Table: tokens[2]}, nil // Example

	default:
		return nil, fmt.Errorf("unsupported statement type: %s", tokens[0])
	}
}

func removeComments(query string) string {
	// Regex to match both types of comments (multi-line and single-line)
	commentRegex := regexp.MustCompile(`(?s)/\*.*?\*/|--.*$`) // (?s) makes . match newlines
	return commentRegex.ReplaceAllString(query, "")
}
