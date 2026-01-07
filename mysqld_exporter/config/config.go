package config

var (
	defaultMysqlSettings = MysqlSettings{
		Host:     "127.0.0.1",
		Port:     3306,
		User:     "root",
		Password: "root",
		Protocol: "tcp",
	}
	DefaultSettings = Settings{
		Mysql: defaultMysqlSettings,
	}
)

type MysqlSettings struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Protocol string `yaml:"protocol"`
}

type Settings struct {
	Mysql MysqlSettings `yaml:"mysql"`
}
