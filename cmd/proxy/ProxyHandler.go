package main

import (
	"fmt"
	"log"

	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/mysql" // Import the mysql package
)

type ProxyHandler struct {
	p                *Proxy
	read_conn        *client.Conn
	write_conn       *client.Conn
	current_conn     *client.Conn
	readServer       *BackendServer
	writeServer      *BackendServer
	databaseName     string
	initialDatabase  string
	connectionLocked bool
}

func NewProxyHandler(proxy *Proxy) *ProxyHandler {
	// NOTE: most of the initialization code for this struct is
	//       handled in handleConnection()

	return &ProxyHandler{p: proxy} // Initialize any internal state here
}

func (ph *ProxyHandler) UseDB(dbName string) error {
	log.Println("UseDB called with:", dbName)

	if ph.current_conn == nil {
		ph.initialDatabase = dbName
		ph.databaseName = dbName
		return nil //fmt.Errorf("called with no connection")
	}
	ph.databaseName = dbName

	// Your implementation to handle COM_INIT_DB
	return ph.current_conn.UseDB(dbName)
}

func (ph *ProxyHandler) HandleQuery(query string) (*mysql.Result, error) {
	log.Println("HandleQuery called with:", query)

	stmts, err := parseSQL(query)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	for _, stmt := range stmts {
		switch stmt {

		case Use:
			ph.current_conn.Execute(query)
			return nil, nil

		// read-only statements
		case Select:
			fallthrough
		case Show:
			fallthrough

		case Desc:
			fallthrough
		case Describe:
			logWithGID(fmt.Sprintf("executing read-only query: %s", query))
			return ph.current_conn.Execute(query)

		case Insert:
			if !ph.connectionLocked {
				log.Println("locking connection to write server")
				//ph.write_conn.Sequence = ph.read_conn.Sequence
				ph.current_conn = ph.write_conn

			}
			ph.current_conn.UseDB(ph.databaseName)
			log.Println("executing write query: ", query)
			res, err := ph.current_conn.Execute(query)
			if err != nil {
				return nil, err
			}
			log.Println("got result: ", res)
			res.Resultset = nil
			return res, nil

		// write statements
		case Update:
			fallthrough

		case Delete:
			fallthrough
		case Create:
			fallthrough
		case Alter:
			fallthrough
		case Drop:
			fallthrough
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
				ph.write_conn.Sequence = ph.read_conn.Sequence
				ph.current_conn = ph.write_conn

			}
			ph.current_conn.UseDB(ph.databaseName)
			log.Println("executing write query: ", query)
			res, err := ph.current_conn.Execute(query)
			if err != nil {
				return nil, err
			}
			log.Println("got result: ", res)

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
