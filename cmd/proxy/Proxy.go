package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"net/http"
	_ "net/http/pprof"

	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/server"
)

type Proxy struct {
	config           *Config
	listener         net.Listener
	pools            *Backends
	shutdown         chan struct{}
	shutdownAccepter chan struct{}
	wg               sync.WaitGroup
	mgr              *server.InMemoryProvider
}

type ServerType int

const (
	ServerTypeUndefined ServerType = iota
	ServerTypeReader
	ServerTypeWriter
)

// Connection represents a managed database connection
type Connection struct {
	Conn *client.Conn
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

	p.pools = NewBackends(p.config)
	p.pools.Initialize()

	// create user database, this needs to be shared
	p.mgr = server.NewInMemoryProvider()
	for _, item := range p.config.AuthenticationMap {
		p.mgr.AddUser(item.ProxyUser, item.ProxyPassword)
	}

	listener, err := net.Listen("tcp", p.config.ListenAddress)
	if err != nil {
		log.Println(fmt.Errorf("failed to listen to [%s]: %w", p.config.ListenAddress, err))
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

func (p *Proxy) handleConnection(conn net.Conn) {
	defer p.wg.Done()
	defer conn.Close()

	//logWithGID("handleConnection()")

	// create a new server connection
	ph := NewProxyHandler(p)

	// obtain a connection from the pool
	svr, err := p.pools.GetNextReplica()
	if err != nil {
		panic(err)
	}
	ph.readServer = svr

	// obtain a connection from the pool
	svr, err = p.pools.GetWriter()
	if err != nil {
		panic(err)
	}
	ph.writeServer = svr

	host, err := server.NewCustomizedConn(conn, server.NewDefaultServer(), p.mgr, ph)
	if err != nil {
		fmt.Printf("Access denied from: %s\n", conn.RemoteAddr())
		return
	}

	//log.Println("Registered the connection with the server")

	user := host.GetUser()

	user, err = p.config.GetBackendUser(user)
	if err != nil {
		panic(err)
	}

	password, err := p.config.GetBackendPassword(user)
	if err != nil {
		panic(err)
	}
	read_key := NewUserKey(ph.readServer.address, user, password)
	//ph.key = key

	cl_conn, err := ph.readServer.GetNextConn(read_key)
	if err != nil {
		panic(err)
	}

	write_key := NewUserKey(ph.writeServer.address, user, password)
	sv_conn, err := ph.writeServer.GetNextConn(write_key)
	if err != nil {
		panic(err)
	}

	logWithGID(fmt.Sprintf("Proxy initiated connection for user '%s' from '%s' and is assigned to user '%s' on MySQL server '%s'\n", host.GetUser(), conn.RemoteAddr(), user, cl_conn.RemoteAddr()))

	ph.read_conn = cl_conn
	//defer ph.read_conn.Close()

	ph.write_conn = sv_conn
	//defer ph.write_conn.Close()

	ph.current_conn = ph.read_conn

	// if a database is specified on the initial connect() we need this
	ph.databaseName = "mysqlslap"

	// as long as the client keeps sending commands, keep handling them
	for {
		if err := host.HandleCommand(); err != nil {
			if err.Error() != "connection closed" {
				log.Printf("Received error on connection: %v\n", err)
			}
			break
		}
	}

	err = ph.readServer.PutConn(read_key, cl_conn)
	if err != nil {
		logWithGID(err.Error())
	}
	ph.writeServer.PutConn(write_key, sv_conn)
	if err != nil {
		logWithGID(err.Error())
	}

	logWithGID(fmt.Sprintf("Proxy terminated connection for user '%s' from '%s' and is assigned to user '%s' on MySQL server '%s'\n", host.GetUser(), conn.RemoteAddr(), user, cl_conn.RemoteAddr()))

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

	p.pools.Shutdown()

	p.wg.Wait()
	log.Println("Proxy stopped")
	return nil
}
