package main

import (
	"flag"
	"fmt"
	"github.com/shadowsocks/go-shadowsocks2/core"
	"github.com/zyong/miniproxygo/m_debug"
	"path"
	"runtime"
	"strings"
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
	server		= flag.String("server", "", "server listen address")
	client		= flag.String("client", "", "client connect address")
	cipher		= flag.String("cipher", "AEAD_CHACHA20_POLY1305", "available ciphers:" + strings.Join(core.ListCipher(), " "))
	password	= flag.String("password", "", "password")
	key			= flag.String("key", "", "base64 url-encode key (derive from password if empty)")
	keygen		= flag.Int("keygen", 0, "generate a base64 url-encoded random key of given length in byte")
	socks		= flag.String("socks", "", "(client-only) SOCKS listen address")
	stdOut      = flag.Bool("s", false, "to show log in stdout")
	showVersion = flag.Bool("v", false, "to show version of bfe")
	showVerbose = flag.Bool("V", false, "to show verbose information about bfe")
	debugLog    = flag.Bool("d", false, "to show debug log (otherwise >= info)")
)

var confpath = flag.String("confpath", "m_config/m_config-dev.yml", "m_server m_config file path")

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

	log.Logger.Info("miniproxy[version:%s] start", version)

	// if client
	if *client != "" {
		// load client config
		confPath := path.Join(*confRoot, "miniproxy-client.conf")
		config, err = m_config.ConfigLoad(confPath, *confRoot, m_config.SetDefaultClientConfig)

		if err != nil {
			log.Logger.Error("main(): in Client ConfigLoad():%s", err.Error())
			m_util.AbnormalExit()
		}

		// start and serve
		if err = m_server.StartClient(config, version, *confRoot); err != nil {
			log.Logger.Error("main(): server.StartUp(): %s", err.Error())
		}
	}

	if *server != "" {
		// load server config
		confPath := path.Join(*confRoot, "miniproxy.conf")
		config, err = m_config.ConfigLoad(confPath, *confRoot, m_config.SetDefaultServerConfig)

		if err != nil {
			log.Logger.Error("main(): in Server ConfigLoad():%s", err.Error())
			m_util.AbnormalExit()
		}

		// start and serve
		if err = m_server.StartServer(config, version, *confRoot); err != nil {
			log.Logger.Error("main(): server.StartUp(): %s", err.Error())
		}
	}






}
