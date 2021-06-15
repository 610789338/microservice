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
	gate 		*clientsdk.GateProxy
	fvc			msf.FlowVelocityCounter
	testCnt 	int
}

func (c *Client) RpcCallTestA(cb clientsdk.CallBack) {
	TestService := c.gate.CreateServiceProxy(namespace, "ServiceA")
	TestService.RpcCall("rpc_a", rand.Int31(), rand.Float32(), "abc", 
						map[string]interface{}{"key1": rand.Int63(), "key2": "def"}, []int32{rand.Int31(), rand.Int31()}, cb)
}

func (c *Client) RpcCallTestB(cb clientsdk.CallBack) {
	TestService := c.gate.CreateServiceProxy(namespace, "ServiceA")
	TestService.RpcCall("rpc_b", rand.Int31(), cb)
}

func (c *Client) RpcCallDBTest(cb clientsdk.CallBack) {
	TestService := c.gate.CreateServiceProxy(namespace, "ServiceA")
	TestService.RpcCall("rpc_db_test", cb)
}

func (c *Client) Start() {
	c.gate = clientsdk.CreateGateProxy("10.246.13.142", 8886)
	if nil == c.gate {
		panic("gate is nil~~~")
	}

	msf.INFO_LOG("client start %v", c.gate.LocalAddr())
	c.fvc = msf.FlowVelocityCounter{Counter: "client rtt"}
	c.fvc.Start()

	startTs := msf.GetNowTimestampMs()
	for i := 0; i < c.testCnt; i++ {
		c.RpcCallDBTest(clientsdk.CallBack(func(err string, reply map[string]interface{}) {
			if err != "" {
				msf.ERROR_LOG("[rpc call] - response: err(%v) reply(%v)", err, reply)
				return
			}
			c.fvc.Count()
		}))
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
	msf.INFO_LOG("rtt avg ops %v/s", int64(c.testCnt)/(endTs - startTs)*1000)
}

func main() {
	var clientCnt, testCnt int = 0, 0
	var err error

    if len(os.Args) > 1 {
	    for idx := 1; idx < len(os.Args); idx++ {
	        switch os.Args[idx] {
	        case "-c":
	            idx++
	            if clientCnt, err = strconv.Atoi(os.Args[idx]); err != nil {
	            	panic(fmt.Sprintf("-c %v error, must be number", os.Args[idx]))
	            }

	        case "-t":
	            idx++
	            if testCnt, err = strconv.Atoi(os.Args[idx]); err != nil {
	            	panic(fmt.Sprintf("-t %v error, must be number", os.Args[idx]))
	            }
	        }
	    }
    }

    msf.INFO_LOG("client cnt %v, test cnt %v", clientCnt, testCnt)

	for i := 0; i < clientCnt ; i++ {
		c := Client{testCnt: testCnt}
		go c.Start()
	}

	ch := make(chan struct{})
	<- ch
}
