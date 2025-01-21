package main

import (
	"crypto/sha256"
	"encoding/hex"

	//	"fmt"
	"log"
	//	"sync"
)

/*
// ClientConnectionPool maps hashed addresses to connections
type ClientConnectionPool struct {
	pool []*PooledConnection
	mu   sync.Mutex
}

func NewClientConnectionPool() *ClientConnectionPool {
	return &ClientConnectionPool{}
}

func (ccp *ClientConnectionPool) AddConnection(conn *client.Conn, addr string) {
	ccp.mu.Lock()
	defer ccp.mu.Unlock()

	hashedAddr := HashAddress(addr)
	pooledConn := &PooledConnection{Conn: conn, HashedAddr: hashedAddr}
	ccp.pool = append(ccp.pool, pooledConn)
}

func (ccp *ClientConnectionPool) GetConnection(hashedAddr string) *client.Conn {
	ccp.mu.Lock()
	defer ccp.mu.Unlock()

	if len(ccp.pool) == 0 {
		return nil
	}

	for _, pooledConn := range ccp.pool {
		if pooledConn.HashedAddr == hashedAddr {
			return pooledConn.Conn
		}
	}

	return nil // No matching connection found
}

func (ccp *ClientConnectionPool) CloseAll() {
	ccp.mu.Lock()
	defer ccp.mu.Unlock()

	for _, pooledConn := range ccp.pool {
		if pooledConn != nil && pooledConn.Conn != nil {
			pooledConn.Conn.Close()
		}
	}
	ccp.pool = nil
}
*/

// HashAddress generates a SHA256 hash of the IP address and port.
func HashAddress(addr string) string {
	h := sha256.Sum256([]byte(addr))
	return hex.EncodeToString(h[:])
}

// ConnectionPool manages the overall connection pooling logic
type ConnectionPool struct {
	connections map[string]*Connection
	readerPool  *ReaderPool
	writerPool  *WriterPool
	config      *Config
}

// NewConnectionPool creates a new ConnectionPool
func NewConnectionPool(config *Config) *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[string]*Connection),
		config:      config,
	}
}

// Start initializes the reader and writer pools
func (cp *ConnectionPool) Start() error {
	log.Print("Creating connection pools...")
	cp.readerPool = NewReaderPool(cp.config)
	cp.writerPool = NewWriterPool(cp.config)
	err := cp.writerPool.Start()
	return err
}

/*
// AssignReader assigns a connection from the reader pool to the client
func (cp *ConnectionPool) AssignReader(hashedAddr string, addr string) (*client.Conn, error) {
	conn, err := cp.readerPool.GetConnection()
	if err != nil {
		return nil, fmt.Errorf("no reader connections available: %w", err)
	}

	cp.addConnection(hashedAddr, conn.Conn, addr)
	cp.readerPool.ReleaseConnection(conn)
	return conn.Conn, nil
}
*/
/*
// UpgradeToWriter upgrades the connection to a writer connection from the writer pool
func (cp *ConnectionPool) UpgradeToWriter(hashedAddr string) (*client.Conn, error) {
	pool := cp.clientPools[hashedAddr]
	if pool == nil {
		return nil, fmt.Errorf("pool not found for hash: %s", hashedAddr)
	}

	conn := pool.GetConnection(hashedAddr)
	if conn == nil {
		return nil, fmt.Errorf("connection not found in pool for hash: %s", hashedAddr)
	}

	cp.removeConnection(hashedAddr)
	pool.CloseAll()

	writeConn, err := cp.writerPool.GetConnection()
	if err != nil {
		return nil, fmt.Errorf("no writer connections available: %w", err)
	}

	cp.addConnection(hashedAddr, writeConn.Conn, "")
	cp.writerPool.ReleaseConnection(writeConn)
	return writeConn.Conn, nil
}

func (cp *ConnectionPool) addConnection(hashedAddr string, conn *client.Conn, addr string) {
	if _, ok := cp.clientPools[hashedAddr]; ok {
		cp.clientPools[hashedAddr].AddConnection(conn, addr)
		fmt.Println("Connection from", hashedAddr, "added to existing pool")
	} else {
		pool := NewClientConnectionPool()
		pool.AddConnection(conn, addr)
		cp.clientPools[hashedAddr] = pool
		fmt.Println("Connection from", hashedAddr, "created new pool")
	}
}

func (cp *ConnectionPool) removeConnection(hashedAddr string) {
	delete(cp.clientPools, hashedAddr)
}
*/

/*
func main() {
    readerPool := NewReaderPool(5, "root", "password", "127.0.0.1")
    writerPool := NewWriterPool(5, "root", "password", "127.0.0.1")
    poolManager := NewConnectionPool(5, "root", "password", "127.0.0.1", readerPool, writerPool)
    err := poolManager.Start()
    if err != nil {
        log.Fatal(err)
    }
    hash := HashAddress("test")
    conn, err := poolManager.AssignReader(hash, "test")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Reader: ", conn)
    conn, err = poolManager.UpgradeToWriter(hash)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Writer: ", conn)
}
*/

/* package main

import (
        "crypto/sha256"
        "encoding/hex"
        "fmt"
        "log"
        "sync"

        "github.com/go-mysql-org/go-mysql/client"
)

// PooledConnection associates a connection with its hashed address
type PooledConnection struct {
        Conn      *client.Conn
        HashedAddr string
}

// GenericConnectionPool manages a pool of generic connections
type GenericConnectionPool struct {
        pool     []*client.Conn
        mu       sync.Mutex
        nextConn int
}

func NewGenericConnectionPool() *GenericConnectionPool {
        return &GenericConnectionPool{}
}

func (gcp *GenericConnectionPool) AddConnection(conn *client.Conn) {
        gcp.mu.Lock()
        defer gcp.mu.Unlock()
        gcp.pool = append(gcp.pool, conn)
}

func (gcp *GenericConnectionPool) GetConnection() *client.Conn {
        gcp.mu.Lock()
        defer gcp.mu.Unlock()

        if len(gcp.pool) == 0 {
                return nil
        }

        conn := gcp.pool[gcp.nextConn]
        gcp.nextConn = (gcp.nextConn + 1) % len(gcp.pool)
        return conn
}

func (gcp *GenericConnectionPool) ReleaseConnection(conn *client.Conn) {
    gcp.mu.Lock()
    defer gcp.mu.Unlock()
    gcp.pool = append(gcp.pool, conn)
}

func (gcp *GenericConnectionPool) CloseAll() {
        gcp.mu.Lock()
        defer gcp.mu.Unlock()

        for _, conn := range gcp.pool {
                if conn != nil {
                        conn.Close()
                }
        }
        gcp.pool = nil
        gcp.nextConn = 0
}

// ClientConnectionPool maps hashed addresses to connections
type ClientConnectionPool struct {
        pool     []*PooledConnection
        mu       sync.Mutex
}

func NewClientConnectionPool() *ClientConnectionPool {
        return &ClientConnectionPool{}
}

func (ccp *ClientConnectionPool) AddConnection(conn *client.Conn, addr string) {
        ccp.mu.Lock()
        defer ccp.mu.Unlock()

        hashedAddr := HashAddress(addr)
        pooledConn := &PooledConnection{Conn: conn, HashedAddr: hashedAddr}
        ccp.pool = append(ccp.pool, pooledConn)
}

func (ccp *ClientConnectionPool) GetConnection(hashedAddr string) *client.Conn {
        ccp.mu.Lock()
        defer ccp.mu.Unlock()

        if len(ccp.pool) == 0 {
                return nil
        }

        for _, pooledConn := range ccp.pool {
                if pooledConn.HashedAddr == hashedAddr {
                        return pooledConn.Conn
                }
        }

        return nil // No matching connection found
}

func (ccp *ClientConnectionPool) CloseAll() {
        ccp.mu.Lock()
        defer ccp.mu.Unlock()

        for _, pooledConn := range ccp.pool {
                if pooledConn != nil && pooledConn.Conn != nil {
                        pooledConn.Conn.Close()
                }
        }
        ccp.pool = nil
}

// HashAddress generates a SHA256 hash of the IP address and port.
func HashAddress(addr string) string {
        h := sha256.Sum256([]byte(addr))
        return hex.EncodeToString(h[:])
}

// ConnectionPool manages the overall connection pooling logic
type ConnectionPool struct {
        clientPools map[string]*ClientConnectionPool
        genericPool GenericConnectionPool
        capacity    int
        user        string
        password    string
        host        string
}

// NewConnectionPool creates a new ConnectionPool
func NewConnectionPool(capacity int, user, password, host string) *ConnectionPool {
        return &ConnectionPool{
                clientPools: make(map[string]*ClientConnectionPool),
                genericPool: NewGenericConnectionPool(),
                capacity:    capacity,
                user:        user,
                password:    password,
                host:        host,
        }
}

// Start initializes the generic connection pool
func (cp *ConnectionPool) Start() error {
        for i := 0; i < cp.capacity; i++ {
                conn, err := client.Connect(fmt.Sprintf("%s:%d", cp.host, 3306), cp.user, cp.password)
                if err != nil {
                        return fmt.Errorf("failed to create generic mysql connection %d: %w", i, err)
                }
                cp.genericPool.AddConnection(conn)
        }

        return nil
}

// AssignReader assigns a connection from the generic pool to the client
func (cp *ConnectionPool) AssignReader(hashedAddr string, addr string) (*client.Conn, error) {
        conn := cp.genericPool.GetConnection()
        if conn == nil {
                return nil, fmt.Errorf("no connections available in pool")
        }

        cp.addConnection(hashedAddr, conn, addr)
    cp.genericPool.ReleaseConnection(conn)
        return conn, nil
}

// UpgradeToWriter upgrades the connection to a writer connection from the generic pool
func (cp *ConnectionPool) UpgradeToWriter(hashedAddr string) (*client.Conn, error) {
        pool := cp.clientPools[hashedAddr]
        if pool == nil {
                return nil, fmt.Errorf("pool not found for hash: %s", hashedAddr)
        }

        conn := pool.GetConnection(hashedAddr)
        if conn == nil {
                return nil, fmt.Errorf("connection not found in pool for hash: %s", hashedAddr)
        }

    pool.CloseAll()
    cp.removeConnection(hashedAddr)

    writeConn := cp.genericPool.GetConnection()
    if writeConn == nil {
        return nil, fmt.Errorf("no connections available in generic pool to upgrade to writer")
    }

    cp.addConnection(hashedAddr, writeConn, "")
    cp.genericPool.ReleaseConnection(writeConn)
    return writeConn, nil
}

func (cp *ConnectionPool) addConnection(hashedAddr string, conn *client.Conn, addr string) {
        if _, ok := cp.clientPools[hashedAddr]; ok {
                cp.clientPools[hashedAddr].AddConnection(conn, addr)
                fmt.Println("Connection from", hashedAddr, "added to existing pool")
        } else {
                pool := NewClientConnectionPool()
                pool.AddConnection(conn, addr)
                cp.clientPools[hashedAddr] = pool
                fmt.Println("Connection from", hashedAddr, "created new pool")
        }
}

func (cp *ConnectionPool) removeConnection(hashedAddr string) {
        delete(cp.clientPools, hashedAddr)
}

func main() {
    poolManager := NewConnectionPool(5, "root", "password", "127.0.0.1")
    err := poolManager.Start()
    if err != nil {
        log.Fatal(err)
    }
    hash := HashAddress("test")
    conn, err := poolManager.AssignReader(hash, "test")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Reader: ", conn)
    conn, err = poolManager.UpgradeToWriter(hash)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Writer: ", conn)
}

*/

/*package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sync"

	"github.com/go-mysql-org/go-mysql/client"
)

type PooledConnection struct {
	Conn       *client.Conn
	HashedAddr string
}

type GenericConnectionPool struct {
	pool     []*client.Conn
	mu       sync.Mutex
	nextConn int
}

func NewGenericConnectionPool() *GenericConnectionPool {
	return &GenericConnectionPool{}
}

func (gcp *GenericConnectionPool) AddConnection(conn *client.Conn) {
	gcp.mu.Lock()
	defer gcp.mu.Unlock()
	gcp.pool = append(gcp.pool, conn)
}

func (gcp *GenericConnectionPool) GetConnection() *client.Conn {
	gcp.mu.Lock()
	defer gcp.mu.Unlock()

	if len(gcp.pool) == 0 {
		return nil
	}

	conn := gcp.pool[gcp.nextConn]
	gcp.nextConn = (gcp.nextConn + 1) % len(gcp.pool)
	return conn
}

func (gcp *GenericConnectionPool) CloseAll() {
	gcp.mu.Lock()
	defer gcp.mu.Unlock()

	for _, conn := range gcp.pool {
		if conn != nil {
			conn.Close()
		}
	}
	gcp.pool = nil
	gcp.nextConn = 0
}

type ClientConnectionPool struct {
	pool []*PooledConnection
	mu   sync.Mutex
}

func NewClientConnectionPool() *ClientConnectionPool {
	return &ClientConnectionPool{}
}

func (ccp *ClientConnectionPool) AddConnection(conn *client.Conn, addr string) {
	ccp.mu.Lock()
	defer ccp.mu.Unlock()

	hashedAddr := HashAddress(addr)
	pooledConn := &PooledConnection{Conn: conn, HashedAddr: hashedAddr}
	ccp.pool = append(ccp.pool, pooledConn)
}

func (ccp *ClientConnectionPool) GetConnection(hashedAddr string) *client.Conn {
	ccp.mu.Lock()
	defer ccp.mu.Unlock()

	if len(ccp.pool) == 0 {
		return nil
	}

	for _, pooledConn := range ccp.pool {
		if pooledConn.HashedAddr == hashedAddr {
			return pooledConn.Conn
		}
	}

	return nil // No matching connection found
}

func (ccp *ClientConnectionPool) CloseAll() {
	ccp.mu.Lock()
	defer ccp.mu.Unlock()

	for _, pooledConn := range ccp.pool {
		if pooledConn != nil && pooledConn.Conn != nil { // Check for nil PooledConnection and Conn
			pooledConn.Conn.Close()
		}
	}
	ccp.pool = nil
}

// HashAddress generates a SHA256 hash of the IP address and port.
func HashAddress(addr string) string {
	h := sha256.Sum256([]byte(addr))
	return hex.EncodeToString(h[:])
}

type ConnectionPool struct {
	clientPools map[string]*ClientConnectionPool // Map of hashed address to ClientConnectionPool
	readPool    GenericConnectionPool
	writePool   GenericConnectionPool
	config      *Config
}

func NewConnectionPool(config *Config) *ConnectionPool {
	return &ConnectionPool{
		clientPools: make(map[string]*ClientConnectionPool),
		config:      config,
	}
}

func (cp *ConnectionPool) Start() error {
	for i := 0; i < 5; i++ { // 5 read connections
		conn, err := client.Connect(fmt.Sprintf("%s:%d", host, port), user, password)
		if err != nil {
			return fmt.Errorf("failed to create generic read mysql connection %d: %w", i, err)
		}
		cp.readPool.AddConnection(conn)
	}
	for i := 0; i < 5; i++ { // 5 write connections
		conn, err := client.Connect(fmt.Sprintf("%s:%d", host, port), user, password)
		if err != nil {
			return fmt.Errorf("failed to create generic write mysql connection %d: %w", i, err)
		}
		cp.writePool.AddConnection(conn)
	}
	return nil
}

func (cp *ConnectionPool) AssignReader(hashedAddr string, addr string) *client.Conn {
	readConn := cp.readPool.GetConnection()
	if readConn == nil {
		log.Println("No read connections available")
		return nil
	}
	cp.addConnection(hashedAddr, readConn, addr)
	return readConn
}

func (cp *ConnectionPool) UpgradeToWriter(hashedAddr string) *client.Conn {
	pool := cp.clientPools[hashedAddr]
	if pool == nil {
		log.Println("Pool not found")
		return nil
	}
	cp.removeConnection(hashedAddr)
	writeConn := cp.writePool.GetConnection()
	if writeConn == nil {
		log.Println("No write connections available")
		return nil
	}
	cp.addConnection(hashedAddr, writeConn, "")
	return writeConn
}

func (cp *ConnectionPool) addConnection(hashedAddr string, conn *client.Conn, addr string) {
	if _, ok := cp.clientPools[hashedAddr]; ok {
		cp.clientPools[hashedAddr].AddConnection(conn, addr)
		fmt.Println("Connection from", hashedAddr, "added to existing pool")
	} else {
		pool := NewClientConnectionPool()
		pool.AddConnection(conn, addr)
		cp.clientPools[hashedAddr] = pool
		fmt.Println("Connection from", hashedAddr, "created new pool")
	}
}

func (cp *ConnectionPool) removeConnection(hashedAddr string) {
	delete(cp.clientPools, hashedAddr)
}
*/
