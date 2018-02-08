package depot

import (
	"os"
	"time"

	"github.com/choueric/jconfig"
)

type Config struct {
	ServerAddr  string `json:"server_addr"`
	ServerPort  int    `json:"server_port"`
	ControlPort int    `json:"control_port"`
	TunnelPort  int    `json:"tunnel_port"`
	WebPort     int    `json:"web_port"`
	UserName    string `json:"user_name"`
	Password    string `json:"password"`
	Timeout     int    `json:"timeout"` // unit: second
	Debug       bool   `json:"debug"`
	// internal
	jc   interface{} // must be interface{}, otherwise panic
	path string      // path of the configuration file
}

const (
	defaultConfig = `{
	"server_addr": "127.0.0.1",
	"server_port": 8864,
	"control_port": 8964,
	"tunnel_port": 9064,
	"web_port": 8888,
	"timeout": 600,
	"user_name": "user",
	"password": "password",
	"debug": false
} `
	VERSION = "0.0.2"
)

var readTimeout time.Duration

func GetDefaultConfigPath() string {
	return os.Getenv("HOME") + "/.depot/config.json"
}

func GetDefaultConfigDir() string {
	return os.Getenv("HOME") + "/.depot"
}

func GetConfig(filepath string) (*Config, error) {
	jc := jconfig.New(filepath, Config{})

	if _, err := jc.Load(defaultConfig); err != nil {
		return nil, err
	}

	config := jc.Data().(*Config)
	config.jc = jc
	config.path = jc.FilePath()

	readTimeout = time.Duration(config.Timeout) * time.Second

	return config, nil
}
