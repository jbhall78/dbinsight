package main

import (
	"crypto/sha256"
	"encoding/hex"

	//	"fmt"
	"log"
	//	"sync"
)

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

func (cp *ConnectionPool) Stop() error {
	log.Print("Releasing connection pools...")
	err := cp.writerPool.Stop()
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
