package ms_framework

import (
    "io/ioutil"
    "encoding/json"
    "os"
    "fmt"
)

type EtcdConfig struct {
    Host        string
    Port        int
}

type GlobalConfig struct {
    Namespace   string
    Service     string
    Port        int
    LogLevel    string
    Etcd        []string
}

var GlobalCfg GlobalConfig

func LoadConfig(filename string, v interface{}) {
    data, err := ioutil.ReadFile(filename)
    if err != nil {
        ERROR_LOG("LoadConfig ReadFile error %v %v", filename, err)
        return
    }

    err = json.Unmarshal(data, v)
    if err != nil {
        ERROR_LOG("LoadConfig json.Unmarshal error %v", err)
        return
    }

    DEBUG_LOG("global config %+v", GlobalCfg)
}

func ParseArgs() {

    DEBUG_LOG("args %v", os.Args)

    if len(os.Args) == 1 {
        return
    }

    for idx := 1; idx < len(os.Args); idx++ {
        switch os.Args[idx] {
        case "-c":
            idx++
            if idx >= len(os.Args) {
                panic(fmt.Sprintf("args parse error -c need follow config filename"))
            }
            LoadConfig(os.Args[idx], &GlobalCfg)
        }
    }

    SetLogLevel(GlobalCfg.LogLevel)
}
