package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-mysql-org/go-mysql/client"
)

type ConnectionPool struct {
	pool     []*client.Conn
	mu       sync.Mutex
	nextConn int
}

type Proxy struct {
	config       *Config
	listener     net.Listener
	primaryPool  *ConnectionPool
	replicaPools []*ConnectionPool
	shutdown     chan struct{}
	wg           sync.WaitGroup
}

func NewProxy(config *Config) (*Proxy, error) {
	primaryPool, err := NewConnectionPool(fmt.Sprintf("%s:%d", config.MySQLPrimaryHost, config.MySQLPrimaryPort), config.MySQLUser, config.MySQLPassword, "", config.PoolCapacity)
	if err != nil {
		return nil, fmt.Errorf("failed to create primary pool: %w", err)
	}

	replicaPools, err := initializeReplicaPools(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize replica pools: %w", err)
	}

	return &Proxy{
		config:       config,
		primaryPool:  primaryPool,
		replicaPools: replicaPools,
		shutdown:     make(chan struct{}),
	}, nil
}

func initializeReplicaPools(config *Config) ([]*ConnectionPool, error) {
	replicaPools := make([]*ConnectionPool, len(config.MySQLReplicas))
	for i, replicaConfig := range config.MySQLReplicas {
		pool, err := NewConnectionPool(fmt.Sprintf("%s:%d", replicaConfig.Host, replicaConfig.Port), config.MySQLUser, config.MySQLPassword, "", config.PoolCapacity)
		if err != nil {
			return nil, fmt.Errorf("failed to create replica pool %d: %w", i+1, err)
		}
		replicaPools[i] = pool
	}
	return replicaPools, nil
}

func NewConnectionPool(addr, user, password, dbName string, poolSize int) (*ConnectionPool, error) {
	pool := &ConnectionPool{
		pool: make([]*client.Conn, poolSize),
	}
	for i := 0; i < poolSize; i++ {
		conn, err := client.Connect(addr, user, password, dbName)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to mysql: %w", err)
		}
		pool.pool[i] = conn
	}
	return pool, nil
}

func (p *Proxy) Start() error {
	listener, err := net.Listen("tcp", p.config.ListenAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	p.listener = listener

	log.Printf("Proxy listening on %s", p.config.ListenAddress)

	p.wg.Add(1)
	go p.acceptConnections()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	<-sigchan

	return p.Stop()
}

func (p *Proxy) acceptConnections() {
	defer p.wg.Done()
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			select {
			case <-p.shutdown:
				return
			default:
				log.Printf("accept error: %v", err)
			}
			continue
		}

		p.wg.Add(1)
		go p.handleConnection(conn)
	}
}

func (p *Proxy) handleConnection(conn net.Conn) {
	defer p.wg.Done()
	defer conn.Close()

	for {
		time.Sleep(1 * time.Second) // Simulate some work
		/*data, err := packet.ReadPacket(conn)
		if err != nil {
			log.Printf("read packet error: %v", err)
			return
		}
		fmt.Printf("Received packet: %x\n", data)*/
		// Implement proxy logic here (dispatching, etc.)
	}
}

func (p *Proxy) Stop() error {
	close(p.shutdown)
	if p.listener != nil {
		if err := p.listener.Close(); err != nil {
			return err
		}
	}
	p.primaryPool.Close()
	for _, replicaPool := range p.replicaPools {
		replicaPool.Close()
	}
	p.wg.Wait()
	log.Println("Proxy stopped")
	return nil
}

func (p *ConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, conn := range p.pool {
		conn.Close()
	}
	p.pool = nil
}
