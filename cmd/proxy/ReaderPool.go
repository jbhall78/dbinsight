package main

import (
	"fmt"
	//        "log"
	"sync"
	// "github.com/go-mysql-org/go-mysql/client"
)

// Connection represents a managed database connection
type ReaderPool struct {
	pool   []*Connection
	mu     sync.Mutex
	config *Config
}

func NewReaderPool(config *Config) *ReaderPool {
	return &ReaderPool{
		//		pool:     make([]*Connection, 0, capacity),
		pool:   []*Connection{},
		config: config,
	}
}

func (rp *ReaderPool) GetConnection() (*Connection, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if len(rp.pool) == 0 {
		return nil, fmt.Errorf("no reader connections available")
	}

	conn := rp.pool[0]
	rp.pool = rp.pool[1:]
	return conn, nil
}

func (rp *ReaderPool) ReleaseConnection(conn *Connection) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	rp.pool = append(rp.pool, conn)
}

/*
import (
        "fmt"
        "log"
        "sync"

        "github.com/go-mysql-org/go-mysql/client"
)


// ReaderPool manages a pool of connections for read operations
type ReaderPool struct {
        pool     []*Connection
        mu       sync.Mutex
        capacity int
        user string
        password string
        host string
}

func NewReaderPool(capacity int, user, password, host string) *ReaderPool {
        return &ReaderPool{
                pool:     make([]*Connection, 0, capacity),
                capacity: capacity,
                user: user,
                password: password,
                host: host,
        }
}

func (rp *ReaderPool) AddConnection(host string, port int) error {
        rp.mu.Lock()
        defer rp.mu.Unlock()

        if len(rp.pool) >= rp.capacity {
                return fmt.Errorf("reader pool at capacity: %d", rp.capacity)
        }

        conn, err := client.Connect(fmt.Sprintf("%s:%d", host, port), rp.user, rp.password)
        if err != nil {
                return fmt.Errorf("failed to create reader connection: %w", err)
        }
        rp.pool = append(rp.pool, &Connection{Conn: conn})
        return nil
}



func (rp *ReaderPool) CloseAll() {
        rp.mu.Lock()
        defer rp.mu.Unlock()

        for _, conn := range rp.pool {
                if conn.Conn != nil {
                        conn.Conn.Close()
                }
        }
        rp.pool = nil
}

/*
func main() {
    readerPool := NewReaderPool(5, "root", "password", "127.0.0.1")
    err := readerPool.AddConnection("127.0.0.1", 3306)
    if err != nil {
        log.Fatal(err)
    }
    conn, err := readerPool.GetConnection()
    if err != nil {
        log.Fatal(err)
    }
    readerPool.ReleaseConnection(conn)
    readerPool.CloseAll()
}
*/
