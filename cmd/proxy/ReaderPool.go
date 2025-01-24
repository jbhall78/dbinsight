package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-mysql-org/go-mysql/client"
)

// Connection represents a managed database connection
type ReaderPool struct {
	readers []*ReadServer
	mu      sync.Mutex
	config  *Config
	done    chan struct{}
}

type ReadServer struct {
	pool    []*Connection
	address string
}

func NewReadServer(address string) *ReadServer {
	return &ReadServer{
		address: address,
	}
}

func (rs *ReadServer) checkConnection(c *Connection) bool {
	for i := 0; i < 3; i++ { // Try 3 pings
		err := c.Conn.Ping()
		if err == nil {
			return true // Connection is healthy
		}
		time.Sleep(200 * time.Millisecond) // Wait before retrying
	}
	return false // Connection is considered down after 3 failed attempts
}

func NewReaderPool(config *Config) *ReaderPool {
	return &ReaderPool{
		readers: []*ReadServer{},
		config:  config,
	}
}

func (rp *ReaderPool) CheckHealth() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	for _, rs := range rp.readers {
		for _, c := range rs.pool {
			if !rs.checkConnection(c) {
				log.Printf("Connection to MySQL Replica %s unhealthy", rs.address)
			}
		}
	}
	return nil
}

func (rp *ReaderPool) Connect(rs *ReadServer) error {
	conn, err := client.Connect(rs.address, rp.config.MySQLUser, rp.config.MySQLPassword, "")
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	log.Printf("%s Connected to MySQL Replica ", rs.address)

	rs.pool = append(rs.pool, &Connection{Conn: conn, serverType: ServerTypeReader})

	return nil
}

func (rp *ReaderPool) Start() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	for _, replica := range rp.config.MySQLReplicas {
		rs := NewReadServer(fmt.Sprintf("%s:%d", replica.Host, replica.Port))

		for i := 0; i < rp.config.ReplicaPoolCapacity; i++ {
			rp.Connect(rs)
		}
	}
	rp.done = make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := rp.CheckHealth(); err != nil {
					log.Println("Health check failed:", err)
				}
			case <-rp.done: // Receive shutdown signal
				return // Exit the goroutine
			}
		}
	}()
	return nil
}

func (rp *ReaderPool) DeleteServer(rs *ReadServer) error {
	for _, rs := range rp.readers {
		for _, c := range rs.pool {
			err := c.Conn.Close()
			if err != nil {
				return fmt.Errorf("error closing MySQL connection[%s]: %w", c.Conn.RemoteAddr(), err)
			}
		}
	}
	return nil
}

func (rp *ReaderPool) Stop() error {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	close(rp.done)

	for _, rs := range rp.readers {
		err := rp.DeleteServer(rs)
		if err != nil {
			fmt.Println(err)
		}
	}

	return nil
}

/*
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
*/

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
