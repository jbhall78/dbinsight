package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-mysql-org/go-mysql/client"
)

// top level pools struct holds references to the readers/writers and all connections
type Pools struct {
	readerPool          []*PoolServer
	writerPool          *PoolServer
	config              *Config
	mu                  sync.Mutex
	healthCheckShutdown chan struct{}
}

// generic struct that represents either a reader or writer
type PoolServer struct {
	connections       []*Connection
	connectionsToKeep int
	address           string
	serverType        ServerType
}

func NewPools(config *Config) *Pools {
	return &Pools{
		config:              config,
		healthCheckShutdown: make(chan struct{}),
	}
}

func NewPoolServer(address string) *PoolServer {
	return &PoolServer{
		address: address,
	}
}

func (pools *Pools) Initialize() error {
	pools.mu.Lock()
	defer pools.mu.Unlock()

	// readers
	for _, replica := range pools.config.MySQLReplicas {
		for i := 0; i < pools.config.ReplicaPoolCapacity; i++ {
			ps := NewPoolServer(fmt.Sprintf("%s:%d", replica.Host, replica.Port))
			ps.connectionsToKeep = pools.config.ReplicaPoolCapacity
			ps.serverType = ServerTypeReader
			pools.readerPool = append(pools.readerPool, ps)
		}
	}

	// writers
	ws := NewPoolServer(fmt.Sprintf("%s:%d", pools.config.MySQLPrimaryHost, pools.config.MySQLPrimaryPort))
	ws.serverType = ServerTypeWriter
	ws.connectionsToKeep = pools.config.PrimaryPoolCapacity
	pools.writerPool = ws

	// start health check thread
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := pools.CheckHealth(); err != nil {
					log.Println("Health check failed:", err)
				}
			case <-pools.healthCheckShutdown: // Receive shutdown signal
				return // Exit the goroutine
			}
		}
	}()

	return nil
}

func (pools *Pools) CheckServerHealth(ps *PoolServer) error {
	// first check that we have enough connections
	if len(ps.connections) < ps.connectionsToKeep {
		for i := 0; i < ps.connectionsToKeep-len(ps.connections); i++ {
			conn, err := client.Connect(ps.address, pools.config.MySQLUser, pools.config.MySQLPassword, "")
			if err != nil {
				return fmt.Errorf("failed to connect to MySQL: %w", err)
			}
			ps.connections = append(ps.connections, &Connection{Conn: conn, serverType: ps.serverType})
			if ps.serverType == ServerTypeReader {
				log.Printf("Connected to MySQL server: %s as a reader\n", ps.address)
			} else {
				log.Printf("Connected to MySQL server: %s as a writer\n", ps.address)
			}
		}
	}

	// now check connections
	for i, c := range ps.connections {
		if c.Conn == nil {
			conn, err := client.Connect(ps.address, pools.config.MySQLUser, pools.config.MySQLPassword, "")
			if err != nil {
				return fmt.Errorf("failed to connect to MySQL: %w", err)
			}
			c.Conn = conn
		}
		if !checkConnection(c) {
			log.Printf("Connection [%d] to MySQL server %s unhealthy", i, ps.address)
		}
	}

	return nil
}

func (pools *Pools) CheckHealth() error {
	pools.mu.Lock()
	defer pools.mu.Unlock()

	for _, ps := range pools.readerPool {
		err := pools.CheckServerHealth(ps)
		if err != nil {
			return err
		}
	}
	pools.CheckServerHealth(pools.writerPool)

	return nil
}

func checkConnection(c *Connection) bool {
	for i := 0; i < 3; i++ { // Try 3 pings
		err := c.Conn.Ping()
		if err == nil {
			return true // Connection is healthy
		}
		time.Sleep(200 * time.Millisecond) // Wait before retrying
	}
	return false // Connection is considered down after 3 failed attempts
}
