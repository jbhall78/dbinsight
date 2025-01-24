package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	//	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/server"
)

type Proxy struct {
	config           *Config
	listener         net.Listener
	connectionPool   *ConnectionPool
	shutdown         chan struct{}
	shutdownAccepter chan struct{}
	wg               sync.WaitGroup
}

type ServerType int

const (
	ServerTypeUndefined ServerType = iota
	ServerTypeReader
	ServerTypeWriter
)

// Connection represents a managed database connection
type Connection struct {
	Conn       *client.Conn
	inUse      bool
	serverType ServerType
	dbName     string
	mu         sync.RWMutex // Mutex for protecting the connection
}

func NewProxy(config *Config) (*Proxy, error) {
	return &Proxy{
		config:           config,
		shutdown:         make(chan struct{}),
		shutdownAccepter: make(chan struct{}),
	}, nil
}

func (p *Proxy) Start() error {
	// start debug server
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

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

	// Example software shutdown, replace with your actual logic
	//go func() {
	//	// Example: Shutdown after 10 seconds (replace with your condition)
	//	time.Sleep(10 * time.Second)
	//	close(p.shutdown)
	//}()

	select {
	case <-sigchan:
		log.Println("Received Unix signal, initiating shutdown...")
	case <-p.shutdown:
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
			case <-p.shutdownAccepter:
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

/*
	func isClosedError(err error) bool {
		if err == nil {
			return false
		}

		if strings.Contains(err.Error(), "use of closed network connection") {
			return true
		}

		opErr, ok := err.(*net.OpError)
		if !ok {
			return false
		}

		sysErr, ok := opErr.Err.(*os.SyscallError)
		if !ok {
			return false
		}

		if sysErr.Err == syscall.EPIPE || sysErr.Err == syscall.ECONNRESET {
			return true
		}

		return false
	}
*/
func (p *Proxy) handleConnection(conn net.Conn) {
	defer p.wg.Done()
	defer conn.Close()

	cl, err := p.connectionPool.writerPool.GetConnection()
	if err != nil {
		fmt.Println(fmt.Errorf("cannot assign connection to a MySQL server"))
		os.Exit(1)
	}
	fmt.Printf("Proxy received connection from '%s' and is assigned to a MySQL server '%s'\n", conn.RemoteAddr().String(), cl.Conn.RemoteAddr())

	err = cl.Conn.Ping()
	if err != nil {
		fmt.Println("ping error: ", err)
		return
	}
	fmt.Println("Ping OK")

	// Create a connection with user root and an empty password.
	// You can use your own handler to handle command here.
	//host, err := server.NewConn(conn, "root", "", server.EmptyHandler{})
	host, err := server.NewConn(conn, "root", "", NewProxyHandler())
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Registered the connection with the server")

	// as long as the client keeps sending commands, keep handling them
	for {
		if err := host.HandleCommand(); err != nil {
			log.Fatal(err)
		}
	}

	/*
	   // Bidirectional copy

	   	go func() {
	   		_, err := io.Copy(server.Conn, conn)
	   		if err != nil && !isClosedError(err) {
	   			fmt.Printf("Error copying from client to server: %v\n", err)
	   		}
	   	}()

	   _, err = io.Copy(conn, server.Conn)

	   	if err != nil && !isClosedError(err) {
	   		fmt.Printf("Error copying from server to client: %v\n", err)
	   	}
	*/
}

func (p *Proxy) Stop() error {
	close(p.shutdownAccepter)

	if p.listener != nil {
		if err := p.listener.Close(); err != nil {
			return err
		}
	}

	p.connectionPool.Stop()

	p.wg.Wait()
	log.Println("Proxy stopped")
	return nil
}
