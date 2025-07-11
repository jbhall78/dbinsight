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

	deferBegin    bool
	inTransaction bool

	preparedStmts map[uint32]*client.Stmt
	stmtCounter   uint32
	stmtMutex     sync.Mutex
}

type Transaction struct {
	stmt        *client.Stmt
	beginResult *mysql.Result
	beginError  error
}

func NewProxyHandler(proxy *Proxy, readServer *BackendServer, writeServer *BackendServer) *ProxyHandler {
	// NOTE: most of the initialization code for this struct is
	//       handled in handleConnection()

	return &ProxyHandler{
		p:             proxy,
		readServer:    readServer,
		writeServer:   writeServer,
		preparedStmts: make(map[uint32]*client.Stmt),
		stmtCounter:   1, // Start counter from 1
	} // Initialize any internal state here
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

func isReplicationError(err error) bool {
	str := err.Error()

	re1 := regexp.MustCompile(`^ERROR (\d+) \((.+)\): (.+)$`) // Capturing groups

	match := re1.FindStringSubmatch(str)
	if match != nil {
		errorCode := match[1]
		//sqlState := match[2]
		//errorString := match[3]
		//fmt.Println("Match found!")
		//fmt.Println("Error Code:", errorCode)

		switch errorCode {
		case "1146":
			fallthrough
		case "1049":
			return true
		default:
			return false
		}
	}

	return false
}

func (ph *ProxyHandler) ExecuteUseQuery(query string) (*mysql.Result, error) {
	dbName, err := extractDatabaseName(query)
	if err != nil {
		return nil, err
	}
	q := "USE " + dbName + ";"

	var wg sync.WaitGroup
	wg.Add(2) // We're waiting for two operations

	var res *mysql.Result
	go func() {
		defer wg.Done()
		res, _ = ph.read_conn.Execute(q)
	}()

	go func() {
		defer wg.Done()
		_, _ = ph.write_conn.Execute(q)
	}()

	wg.Wait() // Wait for both goroutines to finish

	res.Resultset = nil // force an OK packet to be sent by go-mysql
	return res, nil
}

func (ph *ProxyHandler) ExecuteReadQuery(query string) (*mysql.Result, error) {
	if ph.p.config.LogQueries {
		logWithGID(fmt.Sprintf("executing read-only query: %s: %s\n", query, ph.current_conn.RemoteAddr()))
	}
	var res *mysql.Result
	var err error

	delay := 1.0

	//for i := 0; i < 180; i++ {
	for {
		res, err = ph.current_conn.Execute(query)
		if err == nil {
			return res, nil
		}

		// check to see if it is a replication error first.
		if !isReplicationError(err) {
			break
		}

		time.Sleep(time.Duration(delay))
		delay = math.Min(delay*2, 10000) // Double the delay, up to max
	}
	return nil, err
}

func (ph *ProxyHandler) ExecuteWriteQuery(query string) (*mysql.Result, error) {
	//query = trimTrailingNull(query)
	if ph.p.config.LogQueries {
		logWithGID(fmt.Sprintf("executing write query: %s -- database: %s: server: %s\n", query, ph.write_conn.GetDB(), ph.write_conn.RemoteAddr()))
	}
	res, err := ph.write_conn.Execute(query)
	if err != nil {
		logWithGID(fmt.Sprintf("error: %s", err.Error()))
		return nil, err
	}
	// go-mysql/client returns ResultSet
	// go-mysql/server only sends and OK packet when ResultSet is nil
	res.Resultset = nil // force an OK packet to be sent by go-mysql
	// this will have to be removed in the future when this is fixed in go-mysql
	return res, nil
}

func (ph *ProxyHandler) ExecuteQuery(query string) (*mysql.Result, error) {
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
			return ph.ExecuteUseQuery(query)

		// read-only statements
		case Select:
			fallthrough
		case Show:
			fallthrough
		case Desc:
			fallthrough
		case Describe:
			return ph.ExecuteReadQuery(query)

		// write statements
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
			return ph.ExecuteWriteQuery(query)

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
			return ph.ExecuteWriteQuery(query)

		default:
			log.Panicf("Found an unknown statement type: %s\n", query)
		}
	}

	return nil, fmt.Errorf("empty statements")
}

func (ph *ProxyHandler) HandleQuery(query string) (*mysql.Result, error) {
	//log.Println("HandleQuery called with:", query)
	return ph.ExecuteQuery(query)
}

// COM_FIELD_LIST is deprecated so this doesn't need to be implemented
func (ph *ProxyHandler) HandleFieldList(table string, fieldWildcard string) ([]*mysql.Field, error) {
	return nil, nil
}

func (ph *ProxyHandler) HandleStmtPrepare(query string) (params int, columns int, context interface{}, err error) {
	log.Println("HandleStmtPrepare called with query:", query)
	sqlStatements, err := parseSQL(query)
	if err != nil {
		log.Println(err.Error())
		return 0, 0, nil, fmt.Errorf("error parsing sql: %s", err.Error())
	}

	if len(sqlStatements) != 1 {
		return 0, 0, nil, fmt.Errorf("error parsing sql: wrong number of SQL statements for prepare command")
	}

	if !ph.useCalled {
		ph.UseDB(ph.databaseName)
		ph.useCalled = true
	}

	//var ctx *Transaction = &Transaction{}
	var stmt *client.Stmt

	for _, sqlStatement := range sqlStatements {
		switch sqlStatement {

		case Begin:
			if !ph.connectionLocked {
				log.Println("locking connection to write server")
				ph.current_conn = ph.write_conn
				ph.connectionLocked = true
			}
			ph.inTransaction = true
			//ctx.beginResult, ctx.beginError = ph.current_conn.Execute(query)
			return 0, 0, nil, nil
		case Commit:
			ph.inTransaction = false
			//_, err = ph.current_conn.Execute(query)
			return 0, 0, nil, nil
		case Rollback:
			ph.inTransaction = false
			//_, err = ph.current_conn.Execute(query)
			return 0, 0, nil, nil

		case Select:
			fallthrough
		case Show:
			fallthrough
		case Desc:
			fallthrough
		case Describe:
			fallthrough

			// write statements
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
			// 1. Prepare the statement on the backend server
			stmt, err = ph.current_conn.Prepare(query)
			if err != nil {
				logWithGID(fmt.Sprintf("error preparing statement on backend: %s", err.Error()))
				return 0, 0, nil, fmt.Errorf("error preparing statement on backend: %w", err)
			}
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
				ph.current_conn = ph.write_conn
				ph.connectionLocked = true
			}
			// 1. Prepare the statement on the backend server
			stmt, err = ph.current_conn.Prepare(query)
			if err != nil {
				logWithGID(fmt.Sprintf("error preparing statement on backend: %s", err.Error()))
				return 0, 0, nil, fmt.Errorf("error preparing statement on backend: %w", err)
			}
		default:
			log.Panicf("Found an unknown statement type: %s\n", query)
		}
	}

	// 2. Get the number of parameters and columns
	params = stmt.ParamNum()
	columns = stmt.ColumnNum()

	// Store the prepared statement and generate a unique key
	ph.stmtMutex.Lock()
	stmtKey := ph.stmtCounter
	ph.stmtCounter++
	ph.preparedStmts[stmtKey] = stmt
	ph.stmtMutex.Unlock()

	// Pass the key as context
	context = stmtKey

	return params, columns, context, nil
}

func (ph *ProxyHandler) HandleStmtExecute(context interface{}, query string, args []interface{}) (*mysql.Result, error) {
	log.Println("HandleStmtExecute called with query:", query, "and args:", args)

	// Handle BEGIN, COMMIT, ROLLBACK (context is nil)
	if context == nil {
		result, err := ph.current_conn.Execute(query)
		if err != nil {
			return nil, fmt.Errorf("error executing BEGIN/COMMIT/ROLLBACK: %w", err)
		}
		result.Resultset = nil // Force an OK packet
		return result, nil
	}

	// Retrieve the key from the context (for prepared statements)
	stmtKey, ok := context.(uint32)
	if !ok {
		return nil, fmt.Errorf("invalid context: expected statement key (uint32)")
	}

	// Retrieve the prepared statement from the map
	ph.stmtMutex.Lock()
	stmt, ok := ph.preparedStmts[stmtKey]
	ph.stmtMutex.Unlock()

	if !ok {
		return nil, fmt.Errorf("prepared statement not found for key: %d", stmtKey)
	}

	// Execute the prepared statement
	result, err := stmt.Execute(args...)
	if err != nil {
		return nil, err
	}

	logWithGID(fmt.Sprintf("Executed statement: %d", stmtKey))
	return result, nil

	/*
	   // 1. Retrieve the prepared statement from the context
	   ctx, ok := context.(*Transaction) // Type assertion to *client.Stmt

	   	if !ok {
	   		return nil, fmt.Errorf("invalid context: expected *Transaction")
	   	}

	   var err error

	   sqlStatements, err := parseSQL(query)

	   	if err != nil {
	   		log.Println(err.Error())
	   		return nil, fmt.Errorf("error parsing sql: %s", err.Error())
	   	}

	   	if len(sqlStatements) != 1 {
	   		return nil, fmt.Errorf("error parsing sql: wrong number of SQL statements for prepare command")
	   	}

	   	for _, sqlStatement := range sqlStatements {
	   		switch sqlStatement {
	   		case Begin:
	   			fallthrough
	   		case Rollback:
	   			fallthrough
	   		case Commit:
	   			result, err := ph.current_conn.Execute(query)
	   			if err != nil {
	   				log.Printf("Cannot execute begin/commit/rollback in prepared query: %s", err.Error())
	   				return nil, err
	   			}
	   			result.Resultset = nil
	   			return result, nil
	   		default:
	   			// 2. Execute the prepared statement on the backend server
	   			fmt.Printf("Context: %v\n", ctx.stmt)
	   			result, err := ctx.stmt.Execute(args...)
	   			if err != nil {
	   				return nil, fmt.Errorf("error executing prepared statement: %w", err)
	   			}
	   			//fmt.Printf("Arg type: %T\n", args[0])
	   			fmt.Println("Executed statement")
	   			fmt.Printf("Response type %T\n", result)
	   			return result, nil
	   		}
	   	}

	   return nil, fmt.Errorf("unreached code")
	*/
}

func (ph *ProxyHandler) HandleStmtClose(context interface{}) error {
	//log.Println("HandleStmtClose called with context:", context)
	ctx, ok := context.(*Transaction) // Type assertion to *client.Stmt
	if !ok {
		return fmt.Errorf("invalid context: expected *Transaction")
	}

	ctx.stmt.Close()

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
