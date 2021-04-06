package ms_framework

import (
	"fmt"
)


var LOG_LEVEL_DEBUG int8 = 1
var LOG_LEVEL_INFO int8 = 2
var LOG_LEVEL_WARN int8 = 3
var LOG_LEVEL_ERROR int8 = 4

var logLevel int8 = LOG_LEVEL_DEBUG

func SetLogLevel(l int8) {
	if 	l != LOG_LEVEL_DEBUG && 
		l != LOG_LEVEL_INFO && 
		l != LOG_LEVEL_WARN && 
		l != LOG_LEVEL_ERROR {
			panic(fmt.Sprintf("error log level %d", l))
		}

	logLevel = l
}


func LOG(ll int8, level string, format string, params ...interface{}) {
	if ll < logLevel {
		return
	}

	logBody := fmt.Sprintf(format, params...)
	fmt.Printf("[%s] %s\n", level, logBody)
}

func DEBUG_LOG(format string, params ...interface{} ) {LOG(1, "DEBUG", format, params...)}
func INFO_LOG(format string, params ...interface{} ) {LOG(2, "INFO", format, params...)}
func WARN_LOG(format string, params ...interface{} ) {LOG(3, "WARN", format, params...)}
func ERROR_LOG(format string, params ...interface{} ) {LOG(4, "ERROR", format, params...)}
