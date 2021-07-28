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
	Rotate      bool   `yaml:"rotate"`
	Prefix      string `yaml:"prefix"`
}

type FileLogBackend struct {
	logCfg   logCfg
	Logger   *log.Logger
	fileName string
}

func init() {
	logb := &FileLogBackend{}
	// 初始化配置并设置到logging的backend
	logb.initCfg("./config/log-dev.yml")

}

func (logb *FileLogBackend) initCfg(cfgPath string) error {
	// 固定配置文件
	cfgData, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		fmt.Print(err)
		panic(err)
	}

	yaml.Unmarshal(cfgData, &logb.logCfg)

	format := logging.MustStringFormatter(
		logb.logCfg.Formatter,
	)
	// check path
	if _, err := os.Stat(logb.logCfg.LogPath); os.IsNotExist(err) {
		err = os.MkdirAll(logb.logCfg.LogPath, 0755)
		if err != nil {
			panic(err)
		}
	}
	// set file writer
	var file string
	if logb.logCfg.Rotate {
		file, _ = logb.FormatPath()
	} else {
		file = logb.logCfg.LogPath + logb.logCfg.LogFile
	}

	logFile, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	logb.fileName = file
	logb.Logger = log.New(logFile, logb.logCfg.Prefix, log.Ldate|log.Ltime|log.Lmicroseconds|log.Llongfile)

	bFormatter := logging.NewBackendFormatter(logb, format)
	bLeveled := logging.AddModuleLevel(bFormatter)
	switch logb.logCfg.Level {
	case "INFO":
		bLeveled.SetLevel(logging.INFO, "")
	case "DEBUG":
		bLeveled.SetLevel(logging.DEBUG, "")
	}
	logging.SetBackend(bLeveled)

	return nil
}

func (logb *FileLogBackend) FormatPath() (string, error) {
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
	return logb.logCfg.LogPath + replacer.Replace(logb.logCfg.FilePattern), nil
}

func (logb *FileLogBackend) rotate() error {
	var newFile string
	// 修改rotate策略，为默认是proxy.log文件
	// 在需要切文件的时候生成proxy-xxxx.log文件
	if logb.logCfg.Rotate {
		if len(logb.logCfg.FilePattern) > 0 {
			newFile, _ = logb.FormatPath()
		} else {
			return fmt.Errorf("FileLogBackend config error in rotate item and filePattern %s", logb.logCfg.FilePattern)
		}
	} else {
		return nil
	}

	if logb.fileName == newFile {
		return nil
	}

	file, err := os.OpenFile(newFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("create tmp log file failed %v", err)
	}
	// 设置输出为新的文件
	logb.Logger.SetOutput(file)

	logb.fileName = newFile
	return nil
}

func (logb *FileLogBackend) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	// 每次输出日志的时候切换文件名
	logb.rotate()
	rec.Level = level
	return logb.Logger.Output(calldepth+2, rec.Formatted(calldepth+1))
}
