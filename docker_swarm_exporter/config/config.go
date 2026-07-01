package config

var (
	DefaultSettings = Settings{
		SwarmHost: "defaultMysqlSettings",
	}
)

type Settings struct {
	SwarmHost string `yaml:"swarm_host"`
}
