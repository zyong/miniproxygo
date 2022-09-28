package main

import (
	"flag"
	"fmt"
	"github.com/zyong/miniproxygo/m_debug"
	"path"
	"runtime"
	"time"
)

import (
	"github.com/baidu/go-lib/log"
	"github.com/baidu/go-lib/log/log4go"
)

import (
	"github.com/zyong/miniproxygo/m_config"
	"github.com/zyong/miniproxygo/m_server"
	"github.com/zyong/miniproxygo/m_util"
)

var (
	help        = flag.Bool("h", false, "to show help")
	confRoot    = flag.String("c", "./conf", "root path of configuration")
	logPath     = flag.String("l", "./log", "dir path of log")
	stdOut      = flag.Bool("s", false, "to show log in stdout")
	showVersion = flag.Bool("v", false, "to show version of proxy")
	showVerbose = flag.Bool("V", false, "to show verbose information about proxy")
	debugLog    = flag.Bool("d", false, "to show debug log (otherwise >= info)")
)

var version string

func main() {
	var err error
	var config m_config.Conf
	var logSwitch string

	flag.Parse()
	if *help {
		flag.PrintDefaults()
		return
	}
	if *showVerbose {
		fmt.Printf("go version: %s\n", runtime.Version())
		return
	}

	// debug switch
	if *debugLog {
		logSwitch = "DEBUG"
		m_debug.DebugIsOpen = true
	} else {
		logSwitch = "INFO"
		m_debug.DebugIsOpen = false
	}

	// initialize log
	log4go.SetLogBufferLength(10000)
	log4go.SetLogWithBlocking(false)
	log4go.SetLogFormat(log4go.FORMAT_DEFAULT_WITH_PID)
	log4go.SetSrcLineForBinLog(false)

	err = log.Init("proxygo", logSwitch, *logPath, *stdOut, "midnight", 7)
	if err != nil {
		fmt.Printf("proxygo: err in log.Init():%s\n", err.Error())
		m_util.AbnormalExit()
	}

	log.Logger.Info("proxygo[version:%s] start", version)

	// load server config
	confPath := path.Join(*confRoot, "proxy.conf")
	config, err = m_config.ConfigLoad(confPath, *confRoot, m_config.SetDefaultConfig)

	if err != nil {
		log.Logger.Error("main(): in Server ConfigLoad():%s", err.Error())
		m_util.AbnormalExit()
	}

	// start and serve
	if err = m_server.Start(config, version, *confRoot); err != nil {
		log.Logger.Error("main(): server.StartUp(): %s", err.Error())
	}

	// waiting for logger finish jobs
	time.Sleep(1 * time.Second)
	log.Logger.Close()
}
