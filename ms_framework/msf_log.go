package ms_framework

import (
    "fmt"
    "time"
)


var LOG_LEVEL_DEBUG int8 = 1
var LOG_LEVEL_INFO int8 = 2
var LOG_LEVEL_WARN int8 = 3
var LOG_LEVEL_ERROR int8 = 4

var gLogLevel int8 = LOG_LEVEL_DEBUG

var LOG_LEVEL_DICT map[string]int8 = map[string]int8 {
    "DEBUG": 1,
    "INFO": 2,
    "WARN": 3,
    "ERROR": 4,
}

func SetLogLevel(level string) {

    l, ok := LOG_LEVEL_DICT[level]
    if !ok {
        ERROR_LOG("error log level %s set default(DEBUG)", level)
        gLogLevel = LOG_LEVEL_DEBUG
        return
    }

    gLogLevel = l
}

func LOG(level string, format string, params ...interface{}) {
    ll, _ := LOG_LEVEL_DICT[level]
    if ll < gLogLevel {
        return
    }

    logBody := fmt.Sprintf(format, params...)

    if ll == LOG_LEVEL_DEBUG || ll == LOG_LEVEL_ERROR {
        logCh <- fmt.Sprintf("%v - [%s] - [%s:%s] - %s\n", time.Now().Format("2006-01-02 15:04:05.000000"), level, GlobalCfg.Namespace, GlobalCfg.Service, logBody)
    } else {
        logCh <- fmt.Sprintf("%v - [%s]  - [%s:%s] - %s\n", time.Now().Format("2006-01-02 15:04:05.000000"), level, GlobalCfg.Namespace, GlobalCfg.Service, logBody)
    }
}

func DEBUG_LOG(format string, params ...interface{} ) {LOG("DEBUG", format, params...)}
func INFO_LOG (format string, params ...interface{} ) {LOG("INFO",  format, params...)}
func WARN_LOG (format string, params ...interface{} ) {LOG("WARN",  format, params...)}
func ERROR_LOG(format string, params ...interface{} ) {LOG("ERROR", format, params...)}

// 交给一个协程打印，否则日志内容不一定按照时间排序
var logCh chan string = make(chan string)

func init() {
    go func() {
        for {
            select {
            case log := <- logCh:
                fmt.Printf(log)
            }
        }
    } ()
}
