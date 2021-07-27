package miniproxygo

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type server struct {
	Env           string `yaml:"env"`
	Cpunum        bool   `yaml:"cpunum`
	Addr          string `yaml:"addr"`
	Reuseport     bool   `yaml:"reuseport"`
	Goroutinepool bool   `yaml:"goroutinepool"`
}

type config struct {
	Serv server `yaml:"server"`
}

var conf config

func NewConfig(path string) (config, error) {
	if conf != (config{}) {
		return conf, nil
	}
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
