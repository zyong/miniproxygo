package m_config

import (
	"encoding/json"
	"log"
	"os"
)

import (
	gcfg "gopkg.in/gcfg.v1"
)

type ConfigBase struct {
	Local       bool
	Port        int
	MonitorPort int
	Cipher      string

	// settings of communicate with http client
	ClientReadTimeout       int // read timeout, in seconds
	ClientWriteTimeout      int // read timeout, in seconds
	GracefulShutdownTimeout int // graceful shutdown timeout, in seconds
}

type ConfigServer struct {
	MaxIdle      int
	MaxCpus      int // use cpu cores
	AccountsConf string
}

type ConfigClient struct {
	RemoteServer string
	Username     string
	Password     string
}

type Conf struct {
	Base   ConfigBase
	Server ConfigServer
	Client ConfigClient
}

func (cfg *Conf) SetDefaultConfig() {
	cfg.Base.ClientReadTimeout = 60
	cfg.Base.ClientWriteTimeout = 60
	cfg.Base.GracefulShutdownTimeout = 10
}

func SetDefaultConfig(conf *Conf) {
	conf.SetDefaultConfig()
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

type AccountsConf struct {
	Users map[string]string
}

// load accounts config info
func AccountsConfLoad(filename string) (AccountsConf, error) {
	var conf AccountsConf
	// open the file
	file, err := os.Open(filename)
	if err != nil {
		return conf, err
	}
	defer file.Close()

	// decode the file
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&conf); err != nil {
		return conf, err
	}
	return conf, nil
}
