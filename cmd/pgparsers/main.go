package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"

	"github.com/ivikasavnish/postgres-test-replay/pkg"
	"go.yaml.in/yaml/v2"
)

func main() {
	// Read base docker compose file
	composeFilePath := "./docker-compose.yml"
	compose, err := ReadDockerComposeFile(composeFilePath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Docker Compose Version: %s\n", compose.Version)
	fmt.Printf("Postgres Primary Image: %s\n", compose.Services.PostgresPrimary.Image)
	fmt.Printf("Postgres Replica Image: %s\n", compose.Services.PostgresReplica.Image)
	// Discover available ports on host
	primaryPort, err := discoverAvailablePort()
	if err != nil {
		panic(err)
	}
	replicaPort, err := discoverAvailablePort()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Discovered available Primary Port: %s\n", primaryPort)
	fmt.Printf("Discovered available Replica Port: %s\n", replicaPort)

	// Compose INPUT_DSN from docker-compose configuration
	INPUT_DSN := fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable",
		compose.Services.PostgresPrimary.Environment.POSTGRESUSER,
		compose.Services.PostgresPrimary.Environment.POSTGRESPASSWORD,
		primaryPort,
		compose.Services.PostgresPrimary.Environment.POSTGRESDB,
	)

	fmt.Printf("Composed INPUT_DSN: %s\n", INPUT_DSN)

	component, err := url.Parse(INPUT_DSN)
	if err != nil {
		panic(err)
	}

	println("Scheme:", component.Scheme)
	println("User:", component.User.String())
	println("Password:", func() string {
		password, _ := component.User.Password()
		return password
	}())
	println("Host:", component.Hostname())
	println("Port:", component.Port())
	println("Path:", component.Path)
	println("RawQuery:", component.RawQuery)

	//  new designed DSN for replica - use same URL format as primary with all query params
	replicaDSNBase := fmt.Sprintf("postgres://%s:%s@localhost:%s/%s",
		component.User.Username(),
		func() string {
			password, _ := component.User.Password()
			return password
		}(),
		replicaPort,
		component.Path[1:], // remove leading '/'
	)
	// Append query parameters if they exist
	if component.RawQuery != "" {
		replicaDSNBase += "?" + component.RawQuery
	}
	designedDSN := replicaDSNBase
	println("Designed DSN:", designedDSN)

	// change postgres to version 18
	compose.Services.PostgresPrimary.Image = "postgres:18"
	compose.Services.PostgresReplica.Image = "postgres:18"
	// Print updated images
	fmt.Printf("Updated Postgres Primary Image: %s\n", compose.Services.PostgresPrimary.Image)
	fmt.Printf("Updated Postgres Replica Image: %s\n", compose.Services.PostgresReplica.Image)

	// Set discovered ports
	compose.Services.PostgresPrimary.Ports = []string{fmt.Sprintf("%s:5432", primaryPort)}
	compose.Services.PostgresReplica.Ports = []string{fmt.Sprintf("%s:5432", replicaPort)}

	// Print updated ports
	fmt.Printf("Updated Postgres Primary Ports: %v\n", compose.Services.PostgresPrimary.Ports)
	fmt.Printf("Updated Postgres Replica Ports: %v\n", compose.Services.PostgresReplica.Ports)
	// using os.WriteFile to write updated compose back to file

	//  update username and password
	compose.Services.PostgresPrimary.Environment.POSTGRESUSER = component.User.Username()
	compose.Services.PostgresPrimary.Environment.POSTGRESPASSWORD, _ = component.User.Password()
	compose.Services.PostgresReplica.Environment.POSTGRESUSER = component.User.Username()
	compose.Services.PostgresReplica.Environment.POSTGRESPASSWORD, _ = component.User.Password()

	//  change database name
	compose.Services.PostgresPrimary.Environment.POSTGRESDB = component.Path[1:]
	compose.Services.PostgresReplica.Environment.POSTGRESDB = component.Path[1:]

	// Print updated environment variables
	fmt.Printf("Updated Postgres Primary Environment: %+v\n", compose.Services.PostgresPrimary.Environment)
	fmt.Printf("Updated Postgres Replica Environment: %+v\n", compose.Services.PostgresReplica.Environment)

	// Create directories for persistent storage
	dirs := []string{
		"./data/postgres-primary",
		"./data/postgres-replica",
		"./wal/postgres-primary",
		"./wal/postgres-replica",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			panic(fmt.Sprintf("Failed to create directory %s: %v", dir, err))
		}
	}
	fmt.Println("Created persistent storage directories for data and WAL logs.")

	updatedData, err := yaml.Marshal(&compose)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(composeFilePath, updatedData, 0644)
	if err != nil {
		panic(err)
	}
	fmt.Println("Updated docker-compose.yml written successfully.")
	//  update config.yaml file
	// create config.yaml file if not exists

	configFilePath := "./config.yaml"
	fileinfo, err := os.Stat(configFilePath)
	if os.IsNotExist(err) || fileinfo.Size() == 0 {
		_, err := os.Create(configFilePath)
		if err != nil {
			panic(err)
		}
		fmt.Println("Created new config.yaml file.")
	}
	configData := struct {
		PrimaryDSN string `yaml:"primary_dsn"`
		ReplicaDSN string `yaml:"replica_dsn"`
	}{
		PrimaryDSN: INPUT_DSN,
		ReplicaDSN: designedDSN,
	}
	configYAML, err := yaml.Marshal(&configData)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(configFilePath, configYAML, 0644)
	if err != nil {
		panic(err)
	}
	fmt.Println("Updated config.yaml written successfully.")

}

func discoverAvailablePort() (string, error) {
	// Listen on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()

	// Get the assigned port
	addr := listener.Addr().(*net.TCPAddr)
	return strconv.Itoa(addr.Port), nil
}

func ReadYAMLFile(filePath string, out interface{}) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, out)
}

func ReadDockerComposeFile(filePath string) (pkg.DockerCompose, error) {
	compose := pkg.DockerCompose{}
	err := ReadYAMLFile(filePath, &compose)
	if err != nil {
		return pkg.DockerCompose{}, err
	}
	return compose, nil
}
