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

type Proxy struct {
	config         *Config
	listener       net.Listener
	connectionPool *ConnectionPool
	shutdown       chan struct{}
	wg             sync.WaitGroup
}

// Connection represents a managed database connection
type Connection struct {
	Conn *client.Conn
}

func NewProxy(config *Config) (*Proxy, error) {
	return &Proxy{
		config:   config,
		shutdown: make(chan struct{}),
	}, nil
}

func (p *Proxy) Start() error {
	// initialize the connection pools
	p.connectionPool = NewConnectionPool(p.config)
	p.connectionPool.Start()

	listener, err := net.Listen("tcp", p.config.ListenAddress)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to listen to [%s]: %w", p.config.ListenAddress, err))
		os.Exit(1)
	}
	p.listener = listener

	log.Printf("Proxy listening on %s", p.config.ListenAddress)

	p.wg.Add(1)
	go p.acceptConnections()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Software shutdown channel
	shutdownChan := make(chan struct{})

	// Example software shutdown, replace with your actual logic
	//go func() {
	//	// Example: Shutdown after 10 seconds (replace with your condition)
	//	time.Sleep(10 * time.Second)
	//	close(shutdownChan)
	//}()

	select {
	case <-sigchan:
		log.Println("Received Unix signal, initiating shutdown...")
	case <-shutdownChan:
		log.Println("Software shutdown initiated...")
	}

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

	server, err := p.connectionPool.writerPool.GetConnection()
	if err != nil {
		fmt.Println(fmt.Errorf("cannot assign connection to a MySQL server"))
	}
	fmt.Printf("Proxy received connection from '%s' and is assigned to a MySQL server '%s'\n", conn.RemoteAddr().String(), server.Conn.RemoteAddr())
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

	p.wg.Wait()
	log.Println("Proxy stopped")
	return nil
}
