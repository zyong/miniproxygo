package m_config

import (
	"log"
)

import (
	gcfg "gopkg.in/gcfg.v1"
)

type ConfigServer struct {
	Local        bool
	Port         int
	RemoteServer string
	MonitorPort  int

	Cipher   string
	Username string
	Password string

	// settings of communicate with http client
	ClientReadTimeout       int // read timeout, in seconds
	ClientWriteTimeout      int // read timeout, in seconds
	GracefulShutdownTimeout int // graceful shutdown timeout, in seconds

	MaxIdle int
	MaxCpus int
}

type Conf struct {
	Server ConfigServer
}

func (cfg *ConfigServer) SetDefaultConfig() {
	cfg.ClientReadTimeout = 60
	cfg.ClientWriteTimeout = 60
	cfg.GracefulShutdownTimeout = 10
}

func SetDefaultConfig(conf *Conf) {
	conf.Server.SetDefaultConfig()
}

func ConfigLoad(path string, root string, f func(conf *Conf)) (Conf, error) {
	var cfg Conf
	var err error

	f(&cfg)

	err = gcfg.ReadFileInto(&cfg, path)
	if err != nil {
		log.Fatalf("err:%v", err)
	}

	return cfg, err
}
