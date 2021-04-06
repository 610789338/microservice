package main

import (
	"net"
	"fmt"
	msf "ms_framework"
	"time"
)


var rpcMgr = msf.CreateSimpleRpcMgr()

func rpcCall(c net.Conn, rpcName string, args ...interface{}) {

	b := rpcMgr.RpcEncode(rpcName, args...)

	len := uint32(len(b))
	ret := make([]byte, msf.PACKAGE_SIZE_LEN + len)
	msf.WritePacketLen(ret, len)

	copy(ret[msf.PACKAGE_SIZE_LEN:], b)

	wLen, err := c.Write(ret)
	if err != nil {
		msf.ERROR_LOG("write %v error %v", c.RemoteAddr(), err)
	}

	msf.INFO_LOG("write %s success %d", rpcName, wLen)
}

func main() {
	ip := "127.0.0.1"
	port := 6666
	c, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		msf.ERROR_LOG("connect %s:%d error %v", ip, port, err)
		return
	}

	msf.INFO_LOG("connect %s:%d success %v", ip, port, c)

	for true {
		m := make(map[string]interface{})
		m["key1"] = 10
		m["key2"] = "def"

		l := make([]int32)
		l = l.append(123)

		rpcCall(c, "rpc_test", 10, float32(9.9), "abc", m, l)
		time.Sleep(time.Second)
	}
	// rpcCall(c, "rpc_test", 666)
	// time.Sleep(time.Second)
}
