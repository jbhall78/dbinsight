package main

import (
	"fmt"
	"os"
	"runtime"

	"gopkg.in/yaml.v3" // Or your preferred YAML library
)

type ReplicaConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type AuthenticationMapItem struct {
	ProxyUser       string `yaml:"proxy_user"`
	ProxyPassword   string `yaml:"proxy_password"`
	BackendUser     string `yaml:"backend_user"`
	BackendPassword string `yaml:"backend_password"`
}

type Config struct {
	ProxyUser              string                  `yaml:"proxy_user"`
	ProxyPassword          string                  `yaml:"proxy_password"`
	BackendPrimaryHost     string                  `yaml:"backend_primary_host"`
	BackendPrimaryPort     int                     `yaml:"backend_primary_port"`
	BackendPrimaryUser     string                  `yaml:"backend_primary_user"`
	BackendPrimaryPassword string                  `yaml:"backend_primary_password"`
	PrimaryPoolCapacity    int                     `yaml:"primary_pool_capacity"`
	ReplicaPoolCapacity    int                     `yaml:"replica_pool_capacity"`
	ListenAddress          string                  `yaml:"listen_address"`
	HealthCheckDelay       int                     `yaml:"health_check_delay"`
	BackendReplicas        []ReplicaConfig         `yaml:"backend_replicas"` // A slice of ReplicaConfig
	AuthenticationMap      []AuthenticationMapItem `yaml:"authentication_map"`
}

func (c *Config) GetBackendUser(user string) (string, error) {
	for _, item := range c.AuthenticationMap {
		if item.ProxyUser == user {
			return item.BackendUser, nil
		}
	}
	return "", fmt.Errorf("no backend user found for user: %s", user)
}

func (c *Config) GetBackendPassword(user string) (string, error) {
	for _, item := range c.AuthenticationMap {
		if item.BackendUser == user {
			return item.BackendPassword, nil
		}
	}
	return "", fmt.Errorf("no password found for user: %s", user)
}

func loadConfig() (*Config, error) {
	config := Config{
		ProxyUser:              "root",
		ProxyPassword:          "changeme",
		BackendPrimaryHost:     "127.0.0.1",
		BackendPrimaryPort:     3306,
		BackendPrimaryUser:     "root",
		BackendPrimaryPassword: "password",
		PrimaryPoolCapacity:    10,
		ReplicaPoolCapacity:    10,
		ListenAddress:          ":3306",
		HealthCheckDelay:       5,
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

	fmt.Println("MySQL Primary Host:", cfg.BackendPrimaryHost)
	fmt.Println("Replicas:")
	for i, replica := range cfg.BackendReplicas {
		fmt.Printf("  Replica %d: Host=%s, Port=%d\n", i+1, replica.Host, replica.Port)
	}

	p, err := NewProxy(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting proxy: %v", err)
		os.Exit(1)
	}

	p.Start()
}
