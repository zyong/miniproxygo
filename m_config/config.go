package m_config

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type server struct {
	Env           string `yaml:"env"`
	Cpunum        bool   `yaml:"cpunum"`
	Addr          string `yaml:"addr"`
	Reuseport     bool   `yaml:"reuseport"`
	Goroutinepool bool   `yaml:"goroutinepool"`
}

type Conf struct {
	Serv server `yaml:"m_server"`
}

func SetDefaultConfig(conf *Conf) {

}

func ConfigLoad(path string, root string) (Conf, error) {
	var conf Conf
	SetDefaultConfig(&conf)

	cfgData, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(cfgData, &conf)
	if err != nil {
		log.Fatalf("err:%v", err)
	}
	return conf, err
}
