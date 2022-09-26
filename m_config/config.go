package m_config

import (
	"log"
)

import (
	gcfg "gopkg.in/gcfg.v1"
)

type ConfigBasic struct {
	HttpPort    int
	MonitorPort int
	MaxCpus     int

	RemoteServer string
	Cipher	string
	Key		string
	Password string

	// settings of communicate with http client
	TLSHandShakeTimeout     int  // tls handshake timeout, in seconds
	ClientReadTimeout       int  // read timeout, in seconds
	ClientWriteTimeout      int  // read timeout, in seconds
	GracefulShutdownTimeout int  // graceful shutdown timeout, in seconds
	MaxHeaderBytes          int  // max header length inbytes in request
	MaxHeaderUriBytes       int  // max URI (in header) length in bytes in request
	MaxProxyHeaderBytes     int  // max header length in bytes in Proxy protocol
	KeepAliveEnabled        bool // if false, client connection is shutdown disregard of http headers

	Modules []string // modules to load

	DebugServHttp bool // whether open server http debug log

	ConnectionTimeout int
	ReadTimeout       int
	WriteTimeout      int

	MaxIdle             int
	TlsHandshakeTimeout int
}

type Conf struct {
	Server ConfigBasic `yaml:"m_server"`
}

func (cfg *ConfigBasic) SetDefaultConfig() {
	cfg.HttpPort = 8080
	cfg.MonitorPort = 8421
	cfg.MaxCpus = 0

	cfg.TLSHandShakeTimeout = 30
	cfg.ClientReadTimeout = 60
	cfg.ClientWriteTimeout = 60
	cfg.GracefulShutdownTimeout = 10
	cfg.MaxHeaderBytes = 1048576
	cfg.MaxHeaderUriBytes = 8192
	cfg.KeepAliveEnabled = true

}

func SetDefaultConfig(conf *Conf) {
	conf.Server.SetDefaultConfig()
}

func ConfigLoad(path string, root string) (Conf, error) {
	var cfg Conf
	var err error

	SetDefaultConfig(&cfg)

	err = gcfg.ReadFileInto(&cfg, path)
	if err != nil {
		log.Fatalf("err:%v", err)
	}

	return cfg, err
}
