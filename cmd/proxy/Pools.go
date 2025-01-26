package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-mysql-org/go-mysql/client"
)

// top level pools struct holds references to the readers/writers and all connections
type Backends struct {
	readerPool          []*BackendServer
	writerPool          *BackendServer
	config              *Config
	mu                  sync.Mutex
	healthCheckShutdown chan struct{}
}

// generic struct that represents either a reader or writer
type BackendServer struct {
	connections       []*Connection
	connectionsToKeep int
	address           string
	serverType        ServerType
	user              string
	password          string
}

func NewBackends(config *Config) *Backends {
	return &Backends{
		config:              config,
		healthCheckShutdown: make(chan struct{}),
	}
}

func NewBackendServer(address string, user string, password string) *BackendServer {
	return &BackendServer{
		address:  address,
		user:     user,
		password: password,
	}
}

func (pools *Backends) Initialize() error {
	pools.mu.Lock()
	defer pools.mu.Unlock()

	// readers
	for _, replica := range pools.config.MySQLReplicas {
		svr := NewBackendServer(fmt.Sprintf("%s:%d", replica.Host, replica.Port), replica.User, replica.Password)
		svr.connectionsToKeep = pools.config.ReplicaPoolCapacity
		svr.serverType = ServerTypeReader
		pools.readerPool = append(pools.readerPool, svr)
	}

	// writers
	wsvr := NewBackendServer(fmt.Sprintf("%s:%d", pools.config.MySQLPrimaryHost, pools.config.MySQLPrimaryPort), pools.config.MySQLPrimaryUser, pools.config.MySQLPrimaryPassword)
	wsvr.serverType = ServerTypeWriter
	wsvr.connectionsToKeep = pools.config.PrimaryPoolCapacity
	pools.writerPool = wsvr

	// start health check thread
	go func() {
		ticker := time.NewTicker(time.Duration(pools.config.HealthCheckDelay) * time.Second)
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

func (pools *Backends) CheckServerHealth(be *BackendServer) error {
	// first check that we have enough connections
	if len(be.connections) < be.connectionsToKeep {
		connections := len(be.connections)
		for i := 0; i < be.connectionsToKeep-connections; i++ {
			conn, err := client.Connect(be.address, be.user, be.password, "")
			if err != nil {
				return fmt.Errorf("failed to connect to MySQL: %w", err)
			}
			be.connections = append(be.connections, &Connection{Conn: conn, serverType: be.serverType})
			str := ""
			if be.serverType == ServerTypeReader {
				str = "reader"
			} else {
				str = "writer"
			}
			logWithGID(fmt.Sprintf("Connected to MySQL server: %s as a %s [%d]\n", be.address, str, i))
		}
	}

	// now check connections
	for i, c := range be.connections {
		if c.Conn == nil {
			conn, err := client.Connect(be.address, be.user, be.password, "")
			if err != nil {
				return fmt.Errorf("failed to connect to MySQL: %w", err)
			}
			c.Conn = conn
		}
		if !checkConnection(c) {
			log.Printf("Connection [%d] to MySQL server %s unhealthy", i, be.address)
		}
	}

	return nil
}

func (pools *Backends) CheckHealth() error {
	pools.mu.Lock()
	defer pools.mu.Unlock()
	//log.Println("CheckHealth called")

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

func (pools *Backends) Shutdown() error {
	close(pools.healthCheckShutdown)
	pools.mu.Lock()
	defer pools.mu.Unlock()

	// readers
	for _, ps := range pools.readerPool {
		for _, c := range ps.connections {
			c.Conn.Close()
		}
	}

	// writers
	for _, c := range pools.writerPool.connections {
		c.Conn.Close()
	}

	return nil
}
