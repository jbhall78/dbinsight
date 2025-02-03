package main

import (
	"fmt"
	"log"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/mysql" // Import the mysql package
)

type ProxyHandler struct {
	p            *Proxy
	read_conn    *client.Conn
	write_conn   *client.Conn
	current_conn *client.Conn
	readServer   *BackendServer
	writeServer  *BackendServer
	databaseName string
	//	initialDatabase  string
	connectionLocked bool
	useCalled        bool
	//mu               sync.RWMutex
}

func NewProxyHandler(proxy *Proxy) *ProxyHandler {
	// NOTE: most of the initialization code for this struct is
	//       handled in handleConnection()

	return &ProxyHandler{p: proxy} // Initialize any internal state here
}

func (ph *ProxyHandler) UseDB(dbName string) error {
	//log.Println("UseDB called with:", dbName)

	//ph.mu.Lock()
	//defer ph.mu.Unlock()

	if ph.current_conn == nil {
		ph.databaseName = dbName
		return nil
	}
	ph.databaseName = dbName
	//logWithGID(fmt.Sprintf("switching database to: '%s': %s\n", dbName, ph.current_conn.RemoteAddr()))
	// Your implementation to handle COM_INIT_DB
	//err := ph.current_conn.UseDB(dbName)

	query := "USE " + dbName + ";"

	var wg sync.WaitGroup
	wg.Add(2) // We're waiting for two operations

	go func() {
		defer wg.Done()
		_, _ = ph.read_conn.Execute(query)
	}()

	go func() {
		defer wg.Done()
		_, _ = ph.write_conn.Execute(query)
	}()

	wg.Wait() // Wait for both goroutines to finish

	//if err != nil {
	//		logWithGID(fmt.Sprintf("error received switching to database: %s\n", dbName))
	//	}

	return nil
}

/*
func trimTrailingNull(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\x00' {
		return s[:len(s)-1]
	}
	return s
}
*/

func extractDatabaseName(query string) (string, error) {
	// 1. Trim whitespace and convert to lowercase for case-insensitivity
	query = strings.TrimSpace(strings.ToLower(query))

	// 2. Regular expression for matching "USE database_name"
	re := regexp.MustCompile(`^use\s+([a-zA-Z0-9_]+);?$`) // Improved regex

	match := re.FindStringSubmatch(query)

	if match == nil {
		return "", fmt.Errorf("invalid USE statement: %s", query)
	}

	return match[1], nil // Return the captured database name
}

func (ph *ProxyHandler) HandleQuery(query string) (*mysql.Result, error) {
	//log.Println("HandleQuery called with:", query)

	//ph.mu.Lock()
	//defer ph.mu.Unlock()

	stmts, err := parseSQL(query)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if !ph.useCalled {
		ph.UseDB(ph.databaseName)
		ph.useCalled = true
	}

	for _, stmt := range stmts {
		switch stmt {

		case Use:
			dbName, err := extractDatabaseName(query)
			if err != nil {
				continue
			}
			query := "USE " + dbName + ";"

			var wg sync.WaitGroup
			wg.Add(2) // We're waiting for two operations

			var res *mysql.Result
			go func() {
				defer wg.Done()
				res, _ = ph.read_conn.Execute(query)
			}()

			go func() {
				defer wg.Done()
				_, _ = ph.write_conn.Execute(query)
			}()

			wg.Wait() // Wait for both goroutines to finish

			res.Resultset = nil // force an OK packet to be sent by go-mysql
			return res, nil

		// read-only statements
		case Select:
			fallthrough
		case Show:
			fallthrough

		case Desc:
			fallthrough
		case Describe:
			//logWithGID(fmt.Sprintf("executing read-only query: %s: %s\n", query, ph.current_conn.RemoteAddr()))
			var res *mysql.Result
			var err error

			delay := 5.0

			for i := 0; i < 90; i++ {
				res, err = ph.current_conn.Execute(query)
				if err == nil {
					return res, nil
				}
				// check to see if it is a replication error first.
				time.Sleep(time.Duration(delay))
				delay = math.Min(delay*2, 100) // Double the delay, up to max
			}
			return nil, err

		case Create:
			fallthrough
		case Alter:
			fallthrough
		case Drop:
			fallthrough
		case Delete:
			fallthrough
		case Update:
			fallthrough
		case Insert:
			//query = trimTrailingNull(query)
			//logWithGID(fmt.Sprintf("executing write query: %s -- database: %s: server: %s\n", query, ph.current_conn.GetDB(), ph.current_conn.RemoteAddr()))
			res, err := ph.write_conn.Execute(query)
			if err != nil {
				logWithGID(fmt.Sprintf("error: %s", err.Error()))
				return nil, err
			}
			res.Resultset = nil // force an OK packet to be sent by go-mysql
			return res, nil

		// write statements

		case Truncate:
			fallthrough
		case Rename:
			fallthrough
		case Grant:
			fallthrough
		case Revoke:
			fallthrough
		case Set:
			if !ph.connectionLocked {
				log.Println("locking connection to write server")
				//ph.write_conn.Sequence = ph.read_conn.Sequence
				ph.current_conn = ph.write_conn
				ph.connectionLocked = true
			}
			//ph.current_conn.UseDB(ph.databaseName)
			//logWithGID(fmt.Sprintf("executing write query: %s\n", query))
			res, err := ph.write_conn.Execute(query)
			if err != nil {
				return nil, err
			}

			return res, nil

		default:
			log.Panicf("Found an unknown statement type: %T\n", stmt)
		}
	}

	return nil, fmt.Errorf("empty statements")
}

// COM_FIELD_LIST is deprecated so this doesn't need to be implemented
func (ph *ProxyHandler) HandleFieldList(table string, fieldWildcard string) ([]*mysql.Field, error) {

	return nil, nil
}

func (ph *ProxyHandler) HandleStmtPrepare(query string) (params int, columns int, context interface{}, err error) {
	log.Println("HandleStmtPrepare called with query:", query)

	// 1. Prepare the statement on the backend server
	stmt, err := ph.current_conn.Prepare(query)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("error preparing statement on backend: %w", err)
	}

	// 2. Get the number of parameters and columns
	params = stmt.ParamNum()
	columns = stmt.ColumnNum()

	// 3. Store the prepared statement in the context
	context = stmt // Store the *backend* prepared statement

	return params, columns, context, nil
}

func (ph *ProxyHandler) HandleStmtExecute(context interface{}, query string, args []interface{}) (*mysql.Result, error) {
	log.Println("HandleStmtExecute called with query:", query, "and args:", args)

	// 1. Retrieve the prepared statement from the context
	backendStmt, ok := context.(*client.Stmt) // Type assertion to *client.Stmt
	if !ok {
		return nil, fmt.Errorf("invalid context: expected *client.Stmt")
	}

	// 2. Execute the prepared statement on the backend server
	result, err := backendStmt.Execute(args...)
	if err != nil {
		return nil, fmt.Errorf("error executing prepared statement: %w", err)
	}

	return result, nil
}

func (ph *ProxyHandler) HandleStmtClose(context interface{}) error {
	log.Println("HandleStmtClose called with context:", context)

	// 1. Retrieve the prepared statement from the context
	backendStmt, ok := context.(*client.Stmt) // Type assertion to *client.Stmt
	if !ok {
		return fmt.Errorf("invalid context: expected *client.Stmt")
	}

	backendStmt.Close()

	// Your implementation to handle COM_STMT_CLOSE
	return nil
}

func (ph *ProxyHandler) HandleOtherCommand(cmd byte, data []byte) error {
	log.Printf("HandleOtherCommand called with cmd: %d and data: %v\n", cmd, data)
	// Your implementation to handle other commands
	return nil
}

/*
func main() {
        handler := NewProxyHandler()
        handler.UseDB("test_db")
        _, _ = handler.HandleQuery("SELECT * FROM users")
        _, _ = handler.HandleFieldList("users", "*")
        _, _, _, _ = handler.HandleStmtPrepare("SELECT * FROM users WHERE id = ?")
        _, _ = handler.HandleStmtExecute(nil, "SELECT * FROM users WHERE id = ?", []interface{}{1})
        _ = handler.HandleStmtClose(nil)
        _ = handler.HandleOtherCommand(0x03, []byte{0x01, 0x02})
}
*/
