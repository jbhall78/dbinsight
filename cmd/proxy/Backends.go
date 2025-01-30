package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/go-mysql-org/go-mysql/client"
	// "time"
)

type UserKey struct {
	Username string
	Host     string
	Password string
}

// top level pools struct holds references to the readers/writers and all connections
type Backends struct {
	replicas []*BackendServer
	primary  *BackendServer
	usermap  *UserMap
	config   *Config
	rr_index int // server to use for round robin load balancing
	mu       sync.RWMutex
	//healthCheckShutdown chan struct{}
}

// generic struct that represents either a reader or writer
type BackendServer struct {
	pools      map[UserKey]*client.Pool
	address    string
	serverType ServerType
	mu         sync.RWMutex // Add a read/write mutex
}

func NewUserKey(host string, user string, pass string) UserKey {
	return UserKey{
		Username: user,
		Host:     host,
		Password: pass,
	}
}

func NewBackends(config *Config) *Backends {
	return &Backends{
		config: config,
		//healthCheckShutdown: make(chan struct{}),
	}
}

func NewBackendServer(address string) *BackendServer {
	return &BackendServer{
		address: address,
		pools:   make(map[UserKey]*client.Pool),
	}
}

func (be *Backends) Initialize() error {
	be.mu.Lock()
	defer be.mu.Unlock()

	be.usermap = NewUserMap(be.config)
	be.usermap.Initialize()

	// readers
	for _, replica := range be.config.BackendReplicas {
		svr := NewBackendServer(fmt.Sprintf("%s:%d", replica.Host, replica.Port))
		svr.serverType = ServerTypeReader
		be.replicas = append(be.replicas, svr)

		for _, item := range be.usermap.users {
			pool, err := client.NewPoolWithOptions(
				svr.address,
				item.backend_user,
				item.backend_pass,
				"",
				client.WithLogFunc(log.Printf), // Or your logging function
				client.WithPoolLimits(10, 100, 5),
				client.WithConnOptions(), // No connection options
			)
			if err != nil {
				panic(err)
			}
			key := NewUserKey(svr.address, item.backend_user, item.backend_pass)
			//svr.pools[key] = append(svr.pools[key], pool)
			svr.AddPool(key, pool)
		}
	}

	// writer
	wsvr := NewBackendServer(fmt.Sprintf("%s:%d", be.config.BackendPrimaryHost, be.config.BackendPrimaryPort))
	wsvr.serverType = ServerTypeWriter
	be.primary = wsvr
	for _, item := range be.usermap.users {
		pool, err := client.NewPoolWithOptions(
			wsvr.address,
			item.backend_user,
			item.backend_pass,
			"",
			client.WithLogFunc(log.Printf), // Or your logging function
			client.WithPoolLimits(10, 100, 5),
			client.WithConnOptions(), // No connection options
		)
		if err != nil {
			panic(err)
		}
		key := NewUserKey(wsvr.address, item.backend_user, item.backend_pass)
		wsvr.AddPool(key, pool)
	}

	// start health check thread
	/*go func() {
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
	}()*/

	return nil
}

func (be *Backends) GetNextConn(key UserKey) (*client.Conn, error) {
	be.mu.Lock()         // Acquire a read lock
	defer be.mu.Unlock() // Release the read lock

	pool, ok := be.replicas[be.rr_index].pools[key]
	if !ok {
		return nil, fmt.Errorf("no pool available")
	}

	ctx := context.Background()
	conn, err := pool.GetConn(ctx)

	be.rr_index = (be.rr_index + 1) % len(be.replicas)

	return conn, err
}

func (bs *BackendServer) AddPool(key UserKey, pool *client.Pool) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.pools[key] = pool
}

func (bs *BackendServer) DeletePool(key UserKey) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	delete(bs.pools, key)
}

//func (pools *Backends) CheckServerHealth(be *BackendServer) error {
//fmt.Println("called:", be.connectionsToKeep, " ", be.address, "wtf: ", len(be.connections))
// first check that we have enough connections
/*	if len(be.connections) < be.connectionsToKeep {
		connections := len(be.connections)
		for i := 0; i < be.connectionsToKeep-connections; i++ {
			//fmt.Println("Trying to connect")
			conn, err := client.Connect(be.address, be.user, be.password, "")
			if err != nil {
				fmt.Printf("failed to connect to MySQL: %s\n", err)
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
*/
/*
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
*/
//	return nil
//}

/*
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
*/

/*
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
*/

func (pools *Backends) Shutdown() error {
	//close(pools.healthCheckShutdown)
	pools.mu.Lock()
	defer pools.mu.Unlock()

	// readers
	for _, ps := range pools.replicas {
		for _, pool := range ps.pools {
			pool.Close()
		}
	}

	// writers
	for _, pool := range pools.primary.pools {
		pool.Close()
	}

	return nil
}
