package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/go-mysql-org/go-mysql/client"
)

// WriterPool manages a pool of connections for write operations
type WriterPool struct {
	pool   []*Connection
	mu     sync.Mutex
	config *Config
}

func NewWriterPool(config *Config) *WriterPool {
	return &WriterPool{
		pool:   []*Connection{},
		config: config,
	}
}

func (wp *WriterPool) Start() error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	wp.pool = make([]*Connection, 0, wp.config.PrimaryPoolCapacity)

	for i := 0; i < wp.config.PrimaryPoolCapacity; i++ {
		conn, err := client.Connect(fmt.Sprintf("%s:%d", wp.config.MySQLPrimaryHost, wp.config.MySQLPrimaryPort), wp.config.MySQLUser, wp.config.MySQLPassword, "")
		if err != nil {
			return fmt.Errorf("failed to connect to MySQL: %w", err)
		}
		log.Printf("[%d] Connected to MySQL Primary", i)

		wp.pool = append(wp.pool, &Connection{Conn: conn})
	}
	return nil

}

/*
// GetConnection retrieves a connection from the pool.
func (wp *WriterPool) GetConnection() (*Connection, error) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if len(wp.pool) == 0 {
		return nil, fmt.Errorf("no writer connections available")
	}

	conn := wp.pool[0]
	wp.pool = wp.pool[1:]
	return conn, nil
}

// ReleaseConnection returns a connection to the pool.
func (wp *WriterPool) ReleaseConnection(conn *Connection) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	wp.pool = append(wp.pool, conn)
}
*/

/*
// NewWriterPool creates a new WriterPool.

// AddConnection adds a connection to the pool.
func (wp *WriterPool) AddConnection() error {
        wp.mu.Lock()
        defer wp.mu.Unlock()

        if len(wp.pool) >= wp.capacity {
                return fmt.Errorf("writer pool at capacity: %d", wp.capacity)
        }

        conn, err := client.Connect(fmt.Sprintf("%s:%d", wp.host, 3306), wp.user, wp.password)
        if err != nil {
                return fmt.Errorf("failed to create writer connection: %w", err)
        }
        wp.pool = append(wp.pool, &Connection{Conn: conn})
        return nil
}


// CloseAll closes all connections in the pool.
func (wp *WriterPool) CloseAll() {
        wp.mu.Lock()
        defer wp.mu.Unlock()

        for _, conn := range wp.pool {
                if conn.Conn != nil {
                        conn.Conn.Close()
                }
        }
        wp.pool = nil
}

/*
func main() {
    writerPool := NewWriterPool(5, "root", "password", "127.0.0.1")
    err := writerPool.AddConnection()
    if err != nil {
        log.Fatal(err)
    }
    conn, err := writerPool.GetConnection()
    if err != nil {
        log.Fatal(err)
    }
    writerPool.ReleaseConnection(conn)
    writerPool.CloseAll()
}
*/
