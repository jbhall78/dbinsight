package main

import (
	"fmt"
	"os"
	"runtime"

	"gopkg.in/yaml.v3" // Or your preferred YAML library
)

type ReplicaConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type Config struct {
	MySQLPrimaryHost    string          `yaml:"mysql_primary_host"`
	MySQLPrimaryPort    int             `yaml:"mysql_primary_port"`
	MySQLUser           string          `yaml:"mysql_user"`
	MySQLPassword       string          `yaml:"mysql_password"`
	PrimaryPoolCapacity int             `yaml:"primary_pool_capacity"`
	ReplicaPoolCapacity int             `yaml:"replica_pool_capacity"`
	ListenAddress       string          `yaml:"listen_address"`
	HealthCheckDelay    int             `yaml:"health_check_delay"`
	MySQLReplicas       []ReplicaConfig `yaml:"mysql_replicas"` // A slice of ReplicaConfig
}

func loadConfig() (*Config, error) {
	config := Config{
		MySQLPrimaryHost:    "127.0.0.1",
		MySQLPrimaryPort:    3306,
		MySQLUser:           "root",
		MySQLPassword:       "password",
		PrimaryPoolCapacity: 10,
		ReplicaPoolCapacity: 10,
		ListenAddress:       ":3306",
		HealthCheckDelay:    5,
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

	p, err := NewProxy(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting proxy: %v", err)
		os.Exit(1)
	}

	p.Start()
}
