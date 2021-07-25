package miniproxy

import (
	"log"
	"os"

	"github.com/op/go-logging"
)

func init() {
	format := logging.MustStringFormatter(
		` %{shortfile} %{shortfunc} ▶ %{level:.4s} %{message}`,
	)

	// b := logging.NewLogBackend(os.Stderr, "", 0)
	logFile, _ := os.OpenFile("./log.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	b := logging.NewLogBackend(logFile, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Llongfile)

	bFormatter := logging.NewBackendFormatter(b, format)
	bLeveled := logging.AddModuleLevel(bFormatter)
	if debug {
		bLeveled.SetLevel(logging.DEBUG, "")
	} else {
		bLeveled.SetLevel(logging.INFO, "")
	}
	logging.SetBackend(bLeveled)

}
