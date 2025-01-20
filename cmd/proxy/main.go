package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/go-mysql-org/go-mysql/server"

	"gopkg.in/yaml.v3" // Or your preferred YAML library
)

type ReplicaConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Config struct {
	MySQLPrimaryHost string          `yaml:"mysql_primary_host"`
	MySQLPrimaryPort int             `yaml:"mysql_primary_port"`
	MySQLUser        string          `yaml:"mysql_user"`
	MySQLPassword    string          `yaml:"mysql_password"`
	PoolCapacity     int             `yaml:"pool_capacity"`
	ListenAddress    string          `yaml:"listen_address"`
	MySQLReplicas    []ReplicaConfig `yaml:"mysql_replicas"` // A slice of ReplicaConfig
}

// isReadQuery checks if the query is a read-only query.
func isReadQuery(query string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT")
}

// handleConnection handles a single client connection.
func handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Printf("New connection from: %s\n", conn.RemoteAddr())

	// Create a connection with user root and an empty password.
	// You can use your own handler to handle command here.
	c, err := server.NewConn(conn, "root", "", server.EmptyHandler{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading packet: %v\n", err) // Print error to stderr
		os.Exit(1)
	}

	log.Println("Registered the connection with the server")

	data, err := c.ReadPacket()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading packet: %v\n", err) // Print error to stderr
		os.Exit(1)
	}
	_ = data

	// as long as the client keeps sending commands, keep handling them
	//for {
	//	if err := c.HandleCommand(); err != nil {
	//		log.Fatal(err)
	//	}
	//	fmt.Printf("handled command")
	//}
	//p := packet.NewConn(c)
	//defer p.Close()
}

func loadConfig() (*Config, error) {
	config := Config{
		MySQLPrimaryHost: "127.0.0.1",
		MySQLPrimaryPort: 3306,
		MySQLUser:        "root",
		MySQLPassword:    "password",
		PoolCapacity:     10,
		ListenAddress:    ":3306",
	}

	configFile, err := os.Open("data/config/proxy.yaml")
	if err != nil {
		// for debugging
		configFile, err = os.Open("../../data/config/proxy.yaml")
		if err != nil {
			return nil, fmt.Errorf("failed to open config file: %w", err)
		}
	}
	defer configFile.Close()

	//var config Config
	decoder := yaml.NewDecoder(configFile)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return &config, nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	cfg, err := loadConfig()
	if err != nil {
		fmt.Println("Couldn't load configuration file")
		os.Exit(1)
	}

	fmt.Println("MySQL Primary Host:", cfg.MySQLPrimaryHost)
	fmt.Println("Replicas:")
	for i, replica := range cfg.MySQLReplicas {
		fmt.Printf("  Replica %d: Host=%s, Port=%d\n", i+1, replica.Host, replica.Port)
	}

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
