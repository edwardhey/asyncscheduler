package utils

import "golib/log"
import "fmt"

func DebugWithLableAndFormat(label string, format string, v ...interface{}) {
	log.Debug(fmt.Sprintf("[%s] ", label) + fmt.Sprintf(format, v...))
}

func InfoWithLableAndFormat(label string, format string, v ...interface{}) {
	log.Info(fmt.Sprintf("[%s] ", label) + fmt.Sprintf(format, v...))
}

func ErrorWithLableAndFormat(label string, format string, v ...interface{}) {
	log.Error(fmt.Sprintf("[%s] ", label) + fmt.Sprintf(format, v...))
}

func InfoWithLable(label string, v ...interface{}) {
	log.InfoEx(fmt.Sprintf("[%s]", label), v)
}

func DebugWithLable(label string, v ...interface{}) {
	log.DebugEx(fmt.Sprintf("[%s]", label), v)
}

func ErrorWithLable(label string, v ...interface{}) {
	log.ErrorEx(fmt.Sprintf("[%s]", label), v)
}
