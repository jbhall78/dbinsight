package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/go-mysql-org/go-mysql/client"
	// "time"
)

/*
 * connection
 *   conn
 *
 * readers[]: address
 *   pool[username].init()
 *   pool[username].getConn()
 *   rridx
 *   init()
 *   getNextReader
 * writer: address
 *  init()
 */

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
	mu       sync.Mutex
	//healthCheckShutdown chan struct{}
}

// generic struct that represents either a reader or writer
type BackendServer struct {
	pools      map[UserKey][]*client.Pool
	address    string
	serverType ServerType
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
		pools:   make(map[UserKey][]*client.Pool),
	}
}

func (pools *Backends) Initialize() error {
	pools.mu.Lock()
	defer pools.mu.Unlock()

	pools.usermap = NewUserMap(pools.config)
	pools.usermap.Initialize()

	// readers
	for _, replica := range pools.config.BackendReplicas {
		svr := NewBackendServer(fmt.Sprintf("%s:%d", replica.Host, replica.Port))
		svr.serverType = ServerTypeReader
		pools.replicas = append(pools.replicas, svr)

		for _, item := range pools.usermap.users {
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
			svr.pools[key] = append(svr.pools[key], pool)
		}
	}

	// writers
	wsvr := NewBackendServer(fmt.Sprintf("%s:%d", pools.config.BackendPrimaryHost, pools.config.BackendPrimaryPort))
	wsvr.serverType = ServerTypeWriter
	pools.primary = wsvr

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
	/*
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
	*/
	return nil
}
