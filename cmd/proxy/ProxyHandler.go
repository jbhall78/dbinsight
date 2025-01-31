package main

import (
	"fmt"
	"log"

	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/mysql" // Import the mysql package
)

type ProxyHandler struct {
	p               *Proxy
	conn            *client.Conn
	svr             *BackendServer
	key             UserKey
	initialDatabase string
}

func NewProxyHandler(proxy *Proxy) *ProxyHandler {
	return &ProxyHandler{p: proxy} // Initialize any internal state here
}

func (ph *ProxyHandler) UseDB(dbName string) error {
	log.Println("UseDB called with:", dbName)

	if ph.conn == nil {
		ph.initialDatabase = dbName
		return nil //fmt.Errorf("called with no connection")
	}

	// Your implementation to handle COM_INIT_DB
	return ph.conn.UseDB(dbName)
}

func (ph *ProxyHandler) HandleQuery(query string) (*mysql.Result, error) {
	log.Println("HandleQuery called with:", query)

	// Your implementation to handle COM_QUERY
	return ph.conn.Execute(query)
}

// COM_FIELD_LIST is deprecated so this doesn't need to be implemented
func (ph *ProxyHandler) HandleFieldList(table string, fieldWildcard string) ([]*mysql.Field, error) {

	return nil, nil
}

func (ph *ProxyHandler) HandleStmtPrepare(query string) (params int, columns int, context interface{}, err error) {
	log.Println("HandleStmtPrepare called with query:", query)

	// 1. Prepare the statement on the backend server
	stmt, err := ph.conn.Prepare(query)
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
