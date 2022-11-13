package m_server

import (
	"github.com/baidu/go-lib/log"
	"github.com/zyong/miniproxygo/m_config"
	"strings"
)

type Account interface {
	Valid(user string, pass string) bool
	Exist(user string) bool
}

// file validate
type FileAccount struct {
	Conf m_config.AccountsConf
}

func LoadAccount(filename string) (Account, error) {
	var err error

	fa := FileAccount{}
	fa.Conf, err = m_config.AccountsConfLoad(filename)
	if err != nil {
		log.Logger.Error("load accounts file error: %v ", err)
		return nil, err
	}

	return fa, nil
}

// Valid whether user and pass are all right
func (fa FileAccount) Valid(user string, pass string) bool {
	u := strings.TrimSpace(user)
	if v, ok := fa.Conf.Users[u]; ok && v == pass {
		return true
	}
	return false
}

// Exist Whether user exists
func (fa FileAccount) Exist(user string) bool {
	if _, ok := fa.Conf.Users[user]; ok {
		return true
	}
	return false
}
