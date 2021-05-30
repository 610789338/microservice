package ms_framework

import (
	"fmt"
	"time"
)


var LOG_LEVEL_DEBUG int8 = 1
var LOG_LEVEL_INFO int8 = 2
var LOG_LEVEL_WARN int8 = 3
var LOG_LEVEL_ERROR int8 = 4

var logLevel int8 = LOG_LEVEL_DEBUG

var LOG_LEVEL_DICT map[string]int8 = map[string]int8 {
	"DEBUG": 1,
	"INFO": 2,
	"WARN": 3,
	"ERROR": 4,
}

func SetLogLevel(level string) {

	l := LOG_LEVEL_DICT[level]

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

	if ll == LOG_LEVEL_DEBUG || ll == LOG_LEVEL_ERROR {
		fmt.Printf("%v - [%s] - [%s:%s] - %s\n", time.Now().Format("2006-01-02 15:04:05.000000"), level, GlobalCfg.Namespace, GlobalCfg.Service, logBody)
	} else {
		fmt.Printf("%v - [%s]  - [%s:%s] - %s\n", time.Now().Format("2006-01-02 15:04:05.000000"), level, GlobalCfg.Namespace, GlobalCfg.Service, logBody)
	}
}

func DEBUG_LOG(format string, params ...interface{} ) {LOG(1, "DEBUG", format, params...)}
func INFO_LOG(format string, params ...interface{} ) {LOG(2, "INFO", format, params...)}
func WARN_LOG(format string, params ...interface{} ) {LOG(3, "WARN", format, params...)}
func ERROR_LOG(format string, params ...interface{} ) {LOG(4, "ERROR", format, params...)}
