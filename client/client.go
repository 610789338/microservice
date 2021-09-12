package main

import (
    msf "ms_framework"
    "clientsdk"
    "math/rand"
    "time"
    "os"
    "strconv"
    "fmt"
)

var namespace = "YJ"

type Client struct {
    gate           *clientsdk.GateProxy
    fvc            msf.FlowVelocityCounter
    testCnt        int
    interval       int
    ordered        bool
}

func (c *Client) RpcCallTestA(cb clientsdk.CallBack) {
    TestService := c.gate.CreateServiceProxy(namespace, "ServiceA")
    if c.ordered {
        TestService.RpcCallOrdered("rpc_a", rand.Int31(), rand.Float32(), "abc", 
                        map[string]interface{}{"key1": rand.Int63(), "key2": "def"}, []int32{rand.Int31(), rand.Int31()}, cb)
    } else {
        TestService.RpcCall("rpc_a", rand.Int31(), rand.Float32(), "abc", 
                        map[string]interface{}{"key1": rand.Int63(), "key2": "def"}, []int32{rand.Int31(), rand.Int31()}, cb)
    }
}

func (c *Client) RpcCallTestB(cb clientsdk.CallBack) {
    TestService := c.gate.CreateServiceProxy(namespace, "ServiceA")
    if c.ordered {
        TestService.RpcCallOrdered("rpc_b", rand.Int31(), cb)
    } else {
        TestService.RpcCall("rpc_b", rand.Int31(), cb)
    }
}

func (c *Client) RpcCallDBTest(cb clientsdk.CallBack) {
    TestService := c.gate.CreateServiceProxy(namespace, "ServiceA")
    TestService.RpcCall("rpc_db_test", cb)
}

func (c *Client) RpcCallPushTest(cb clientsdk.CallBack) {
    TestService := c.gate.CreateServiceProxy(namespace, "ServiceA")
    TestService.RpcCall("rpc_push_test", cb)
}

func (c *Client) StartTest(mode string, idx int) {
    c.gate = clientsdk.CreateGateProxy(IP, Port)
    if nil == c.gate {
        panic("gate is nil~~~")
    }

    c.gate.Login(fmt.Sprintf("client%d", idx), namespace)

    msf.INFO_LOG("client start %v", c.gate.LocalAddr())
    c.fvc = msf.FlowVelocityCounter{Counter: "client rtt"}
    c.fvc.Start()

    callback := clientsdk.CallBack(func(err string, reply map[string]interface{}) {
            if err != "" {
                msf.ERROR_LOG("[rpc call] - response: err(%v) reply(%v)", err, reply)
            } else {
                msf.DEBUG_LOG("[rpc call] - response: err(%v) reply(%v)", err, reply)
            }
            c.fvc.Count()
        })

    startTs := msf.GetNowTimestampMs()
    for i := 0; i < c.testCnt; i++ {
        if "testa" == mode {
            c.RpcCallTestA(callback)
        } else if "testb" == mode {
            c.RpcCallTestB(callback)
        } else if "dbtest" == mode {
            c.RpcCallDBTest(callback)
        }

        if c.interval != 0 {
            time.Sleep(time.Microsecond * time.Duration(c.interval))
        }
    }

    endTs := msf.GetNowTimestampMs()
    if endTs > startTs {
        msf.INFO_LOG("send avg ops %v/s", int64(c.testCnt)/(endTs - startTs)*1000)
    }

    for {
        if c.fvc.GetTotalCount() == int64(c.testCnt) {
            break
        }
        time.Sleep(time.Microsecond)
    }
    c.fvc.Stop()

    endTs = msf.GetNowTimestampMs()
    msf.INFO_LOG("rtt avg ops %v/s", int64(c.testCnt)*1000/(endTs - startTs))

    c.gate.Logoff(fmt.Sprintf("client%d", idx))
}

func (c *Client) StartPushTest(idx int) {
    c.gate = clientsdk.CreateGateProxy(IP, Port)
    if nil == c.gate {
        panic("gate is nil~~~")
    }

    c.gate.Login(fmt.Sprintf("client%d", idx), namespace)

    msf.INFO_LOG("client start %v", c.gate.LocalAddr())

    for i := 0; i < c.testCnt; i++ {
        c.RpcCallPushTest(nil)
        
        if c.interval != 0 {
            time.Sleep(time.Microsecond * time.Duration(c.interval))
        }
    }

    time.Sleep(time.Second)
    c.gate.Logoff(fmt.Sprintf("client%d", idx))
}

var IP = "127.0.0.1"
var Port int = 8886

func main() {
    var mode string = ""
    var clientCnt, testCnt, interval int = 0, 0, 0
    var ordered bool = false
    var err error
    var logLevel = "DEBUG"

    if len(os.Args) > 1 {
        for idx := 1; idx < len(os.Args); idx++ {
            switch os.Args[idx] {
            case "--help":
                idx++
                Usage()

            case "-h":
                idx++
                IP = os.Args[idx]

            case "-p":
                idx++
                if Port, err = strconv.Atoi(os.Args[idx]); err != nil {
                    panic(fmt.Sprintf("-p %v error, must be number", os.Args[idx]))
                }

            case "-m":
                idx++
                mode = os.Args[idx]
                if mode != "testa" && mode != "testb" && mode != "dbtest" && mode != "pushtest" {
                    panic(fmt.Sprintf("-m %v error must in [testa, testb, dbtest, pushtest]", mode))
                }

            case "-n":
                idx++
                if clientCnt, err = strconv.Atoi(os.Args[idx]); err != nil {
                    panic(fmt.Sprintf("-n %v error, must be number", os.Args[idx]))
                }

            case "-t":
                idx++
                if testCnt, err = strconv.Atoi(os.Args[idx]); err != nil {
                    panic(fmt.Sprintf("-t %v error, must be number", os.Args[idx]))
                }

            case "-i":
                idx++
                if interval, err = strconv.Atoi(os.Args[idx]); err != nil {
                    panic(fmt.Sprintf("-i %v error, must be number", os.Args[idx]))
                }

            case "-o":
                idx++
                arg := os.Args[idx]
                if arg == "true" {
                    ordered = true
                } else if arg == "false" {
                    ordered = false
                } else {
                    panic(fmt.Sprintf("-o %v error, must be true/false", os.Args[idx]))
                }

            case "-l":
                idx++
                logLevel = os.Args[idx]
            }
        }
    } else {
        Usage()
    }

    msf.SetLogLevel(logLevel)

    msf.INFO_LOG("client %s mode %v client - [test cnt %v, interval %v] logLevel %v", mode, clientCnt, testCnt, interval, logLevel)

    if "pushtest" == mode  {
        for i := 0; i < clientCnt ; i++ {
            c := Client{testCnt: testCnt, interval: interval, ordered: ordered}
            go c.StartPushTest(i)
        }
    } else {
        for i := 0; i < clientCnt ; i++ {
            c := Client{testCnt: testCnt, interval: interval, ordered: ordered}
            go c.StartTest(mode, i)
        }
    }

    ch := make(chan struct{})
    <- ch
}

func Usage() {

    fmt.Printf("\n")
    fmt.Printf("--help:    print Usage\n")
    fmt.Printf(fmt.Sprintf("-h    :    gate listen addr, default %s\n", IP))
    fmt.Printf(fmt.Sprintf("-p    :    gate listen port, default %d\n", Port))
    fmt.Printf("-m    :    test mode, must in [testa, testb, dbtest, pushtest]\n")
    fmt.Printf("-n    :    client cnt(goroutine), default 0\n")
    fmt.Printf("-t    :    rpc test cnt, default 0\n")
    fmt.Printf("-o    :    rpc test ordered, default false\n")
    fmt.Printf("-i    :    rpc test interval(microsecond), default 0\n")
    fmt.Printf("-l    :    log level, default DEBUG\n")

    fmt.Printf("for example:\n")
    fmt.Printf("./client -m testa -n 5 -t 1000\n")
    fmt.Printf("./client -h 10.246.13.142 -p 8886 -m pushtest -n 1 -t 10 -i 1000 -o false -l INFO\n")
    fmt.Printf("\n")

    os.Exit(0)
}
