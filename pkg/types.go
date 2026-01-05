package pkg

type DockerCompose struct {
	Version  string `yaml:"version"`
	Services struct {
		PostgresPrimary struct {
			Image         string `yaml:"image"`
			ContainerName string `yaml:"container_name"`
			Environment   struct {
				POSTGRESUSER     string `yaml:"POSTGRES_USER"`
				POSTGRESPASSWORD string `yaml:"POSTGRES_PASSWORD"`
				POSTGRESDB       string `yaml:"POSTGRES_DB"`
			} `yaml:"environment"`
			Ports    []string `yaml:"ports"`
			Volumes  []string `yaml:"volumes"`
			Command  string   `yaml:"command"`
			Networks []string `yaml:"networks"`
		} `yaml:"postgres-primary"`
		PostgresReplica struct {
			Image         string `yaml:"image"`
			ContainerName string `yaml:"container_name"`
			Environment   struct {
				POSTGRESUSER     string `yaml:"POSTGRES_USER"`
				POSTGRESPASSWORD string `yaml:"POSTGRES_PASSWORD"`
				POSTGRESDB       string `yaml:"POSTGRES_DB"`
			} `yaml:"environment"`
			Ports     []string `yaml:"ports"`
			Volumes   []string `yaml:"volumes"`
			Command   string   `yaml:"command"`
			Networks  []string `yaml:"networks"`
			DependsOn []string `yaml:"depends_on"`
		} `yaml:"postgres-replica"`
	} `yaml:"services"`
	Volumes struct {
		PostgresPrimaryData interface{} `yaml:"postgres-primary-data"`
		PostgresReplicaData interface{} `yaml:"postgres-replica-data"`
	} `yaml:"volumes"`
	Networks struct {
		PostgresNetwork struct {
			Driver string `yaml:"driver"`
		} `yaml:"postgres-network"`
	} `yaml:"networks"`
}
