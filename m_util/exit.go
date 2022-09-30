package m_util

import (
	"os"
)

import (
	"github.com/baidu/go-lib/log"
)

func AbnormalExit() {
	// waiting for logger finish jobs
	log.Logger.Close()
	// exit
	os.Exit(1)
}