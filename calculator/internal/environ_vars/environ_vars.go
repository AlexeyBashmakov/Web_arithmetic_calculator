package environ_vars

import (
	"fmt"
	"os"
	"runtime"
	"strconv"

	"web_arithmetic_calculator/internal/constants"
)

const (
	time_default    = "10000"
	expired_default = "100"
	http_port       = "7777"
	rpc_port        = "5000"
)

var env_vars = []string{
	constants.TimeAdd,
	constants.TimeSub,
	constants.TimeMult,
	constants.TimeDiv,
	constants.TimeRequ,
	constants.TimeTokenExpired,
	constants.CompPow,
	constants.HTTP_PORT,
	constants.RPC_PORT,
}

// функция проверяет, установлены-ли переменные окружения
func CheckEnvironmentVariables() bool {
	exist := true
	for _, k := range env_vars {
		_, exist = os.LookupEnv(k)
	}
	return exist
}

// функция устанавливает переменные окружения в значения по умолчанию
func SetEnvironmentVariables() error {
	env_vars := map[string]string{
		constants.TimeAdd:          time_default,
		constants.TimeSub:          time_default,
		constants.TimeMult:         time_default,
		constants.TimeDiv:          time_default,
		constants.TimeRequ:         time_default,
		constants.TimeTokenExpired: expired_default,
		constants.CompPow:          fmt.Sprintf("%d", runtime.NumCPU()/2),
		constants.HTTP_PORT:        http_port,
		constants.RPC_PORT:         rpc_port,
	}
	var err error
	for k, v := range env_vars {
		err = os.Setenv(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetValue(name string) string {
	exist := false
	for _, constant := range env_vars {
		if name == constant {
			exist = true
			break
		}
	}
	if exist {
		// if (name == constants.TimeAdd) || (name == constants.TimeSub) ||
		// 	(name == constants.TimeMult) || (name == constants.TimeDiv) ||
		// 	(name == constants.CompPow) || (name == constants.PORT) {
		val, exist := os.LookupEnv(name)
		if !exist {
			switch name {
			case constants.TimeAdd, constants.TimeSub, constants.TimeMult, constants.TimeDiv, constants.TimeRequ:
				err := os.Setenv(name, time_default)
				if err != nil {
					return ""
				}
				return time_default
			case constants.TimeTokenExpired:
				err := os.Setenv(name, expired_default)
				if err != nil {
					return ""
				}
				return expired_default
			case constants.CompPow:
				err := os.Setenv(name, fmt.Sprintf("%d", runtime.NumCPU()/2))
				if err != nil {
					return ""
				}
				return fmt.Sprintf("%d", runtime.NumCPU()/2)
			case constants.HTTP_PORT:
				err := os.Setenv(name, http_port)
				if err != nil {
					return ""
				}
				return http_port
			case constants.RPC_PORT:
				err := os.Setenv(name, rpc_port)
				if err != nil {
					return ""
				}
				return rpc_port
			}
		} else {
			return val
		}
	}

	return ""
}

func GetValueInt(name string) (int, bool) {
	r := GetValue(name)
	if r == "" {
		return 0, false
	} else {
		result, err := strconv.Atoi(r)
		if err != nil {
			return 0, false
		} else {
			return result, true
		}
	}
}
