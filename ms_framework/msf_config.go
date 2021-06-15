package ms_framework

import (
    "io/ioutil"
    "encoding/json"
    "os"
    "fmt"
)

type GlobalConfig struct {
    Namespace       string
    Service         string
    Port            int
    LogLevel        string
    Etcd            []string
    Mongo           string
    Redis           string
    RedisCluster    []string
}

var GlobalCfg GlobalConfig

func LoadConfig(filename string, v interface{}) {
    data, err := ioutil.ReadFile(filename)
    if err != nil {
        panic(fmt.Sprintf("LoadConfig ReadFile error %v %v", filename, err))
    }

    err = json.Unmarshal(data, v)
    if err != nil {
        panic(fmt.Sprintf("LoadConfig json.Unmarshal error %v", err))
    }

    DEBUG_LOG("global config %+v", GlobalCfg)
}

func ParseArgs() {

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
}
