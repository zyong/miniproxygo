package miniproxygo

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/op/go-logging"
	"gopkg.in/yaml.v2"
)

type logCfg struct {
	Formatter   string `yaml:"formatter"`
	LogFile     string `yaml:"logFile"`
	FilePattern string `yaml:"filePattern"`
	LogPath     string `yaml:"path"`
	Level       string `yaml:"level"`
}

func init() {
	// 固定配置文件
	cfgData, err := ioutil.ReadFile("./config/log-dev.yml")
	if err != nil {
		fmt.Print(err)
		panic(err)
	}

	var cfg logCfg
	yaml.Unmarshal(cfgData, &cfg)

	format := logging.MustStringFormatter(
		cfg.Formatter,
	)

	file := cfg.LogPath + cfg.LogFile
	if len(cfg.FilePattern) > 0 {
		t := time.Now()
		m := t.Month()
		y := t.Year()
		d := t.Day()
		// replace year month day
		replacer := strings.NewReplacer(
			"MM", fmt.Sprintf("%d", m),
			"dd", fmt.Sprintf("%d", d),
			"yyyy", fmt.Sprintf("%d", y),
			"%H", fmt.Sprintf("%d", t.Hour()),
			"%M", fmt.Sprintf("%d", t.Minute()),
			"%S", fmt.Sprintf("%d", t.Second()),
		)
		file = replacer.Replace(cfg.FilePattern)
	}

	// b := logging.NewLogBackend(os.Stderr, "", 0)
	// 程序是常驻内存的，init只会执行一次，所以需要在log写日志里面来处理rotate
	logFile, _ := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	b := logging.NewLogBackend(logFile, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Llongfile)

	bFormatter := logging.NewBackendFormatter(b, format)
	bLeveled := logging.AddModuleLevel(bFormatter)
	switch cfg.Level {
	case "INFO":
		bLeveled.SetLevel(logging.INFO, "")
	case "DEBUG":
		bLeveled.SetLevel(logging.DEBUG, "")
	}

	logging.SetBackend(bLeveled)

}
