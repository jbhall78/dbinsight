package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/go-mysql-org/go-mysql/client"
)

// Configuration
var (
	primaryAddr = "mysql1:3306"
	replicaAddr = "mysql2:3307"
	user        = "root"
	password    = "password"
)

// GetConnection returns a connection to the appropriate MySQL server.
func getConnection(isReadQuery bool) (*client.Conn, error) {
	addr := primaryAddr
	if isReadQuery {
		addr = replicaAddr
	}
	conn, err := client.Connect(addr, user, password, "")
	if err != nil {
		return nil, fmt.Errorf("error connecting to MySQL server (%s): %v", addr, err)
	}
	return conn, nil
}

// isReadQuery checks if the query is a read-only query.
func isReadQuery(query string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT")
}

// handleConnection handles a single client connection.
func handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("New connection from: %s\n", conn.RemoteAddr())

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("Connection read error: %v", err)
			break
		}

		query := strings.TrimSpace(string(buf[:n]))
		log.Printf("Received query: %s", query)

		dbConn, err := getConnection(isReadQuery(query))
		if err != nil {
			log.Printf("Error getting database connection: %v", err)
			conn.Write([]byte(fmt.Sprintf("Error: %v\n", err)))
			continue
		}
		defer dbConn.Close()

		result, err := dbConn.Execute(query)
		if err != nil {
			log.Printf("Error executing query: %v", err)
			conn.Write([]byte(fmt.Sprintf("Query error: %v\n", err)))
			continue
		}

		conn.Write([]byte(fmt.Sprintf("Result: %v\n", result)))
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	listener, err := net.Listen("tcp", ":3306")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listening on port 3306: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println("Listening on port 3306...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go handleConnection(conn)
	}
}

/*package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"strings"

	client "github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/packet"
)

// Configuration (replace with actual values)
var (
	primaryAddr  = "mysql1:3306"
	replicaAddr  = "mysql2:3307"
	user         = "root"
	password     = "password"
	poolCapacity = 10
)

// Connection pools
var (
	primaryPool *client.Pool
	replicaPool *client.Pool
)

func initPools() error {
	var err error

	primaryPool, err = client.NewPool(primaryAddr, user, password, "", "", poolCapacity)
	if err != nil {
		return fmt.Errorf("error creating primary pool: %v", err)
	}

	replicaPool, err = client.NewPool(replicaAddr, user, password, "", "", poolCapacity)
	if err != nil {
		return fmt.Errorf("error creating replica pool: %v", err)
	}

	return nil
}

func releasePools() {
	if primaryPool != nil {
		primaryPool.Close()
	}
	if replicaPool != nil {
		replicaPool.Close()
	}
}

func getConnection(isReadQuery bool) (*client.Conn, error) {
	if isReadQuery {
		if replicaPool == nil {
			return nil, fmt.Errorf("replica pool is not initialized")
		}
		return replicaPool.Get()
	}
	if primaryPool == nil {
		return nil, fmt.Errorf("primary pool is not initialized")
	}
	return primaryPool.Get()
}

func releaseConnection(conn *client.Conn) {
	if conn != nil {
		conn.Close()
	}
}

func isReadQuery(query string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT")
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Printf("New connection from: %s\n", conn.RemoteAddr())

	for {
		data, err := packet.ReadPacket(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error: %v", err)
			}
			break
		}

		cmd := data[0]

		switch cmd {
		case client.COM_QUERY:
			query := string(data[1:])
			dbConn, err := getConnection(isReadQuery(query))
			if err != nil {
				log.Printf("Error getting connection: %v", err)
				// Send error packet to client
				errPacket := packet.NewErrPacket(client.ER_UNKNOWN_ERROR, client.Message("Error getting connection"))
				if err := packet.WritePacket(conn, errPacket.Dump()); err != nil {
					log.Printf("Error writing error packet: %v", err)
				}
				continue
			}
			defer releaseConnection(dbConn)

			result, err := dbConn.Execute(query)
			if err != nil {
				log.Printf("Error executing query: %v", err)
				errPacket := packet.NewErrPacket(client.ER_UNKNOWN_ERROR, client.Message(err.Error()))
				if err := packet.WritePacket(conn, errPacket.Dump()); err != nil {
					log.Printf("Error writing error packet: %v", err)
				}
				continue
			}

			// Collect visualization data here (placeholder)
			// ...

			err = packet.WritePacket(conn, result.Packet)
			if err != nil {
				log.Printf("Error writing packet: %v", err)
				break
			}

		case client.COM_PING:
			err := packet.WritePacket(conn, packet.OKPacket([]byte{}))
			if err != nil {
				log.Printf("Error writing pong: %v", err)
				break
			}
		default:
			log.Printf("Unhandled command: %d", cmd)
			errPacket := packet.NewErrPacket(client.ER_UNKNOWN_ERROR, client.Message(fmt.Sprintf("Unhandled command: %d", cmd)))
			if err := packet.WritePacket(conn, errPacket.Dump()); err != nil {
				log.Printf("Error writing error packet: %v", err)
			}
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	err := initPools()
	if err != nil {
		log.Fatal(err)
	}
	defer releasePools()

	listener, err := net.Listen("tcp", ":3306")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listening: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println("Listening on port 3306...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accepting connection: %v\n", err)
			continue
		}

		go handleConnection(conn)
	}
}
*/
