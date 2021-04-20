package clientsdk

import (
	"net"
	"fmt"
	msf "ms_framework"
	"time"
	"io"
	"reflect"
)

// 单个实例唯一的gid
var globalGID uint32 = 0
var rpcMgr *msf.SimpleRpcMgr = nil

type CallBack func(err string, result map[string]interface{})
var gCbMap map[uint32] CallBack
var gCbMapMaxSize = 100
var gCbChan chan []interface{}

// 利用time.After实现callback的超时控制，避免gCbMap被撑爆
func CallBackTimeOut(rid uint32) {
	select {
	case <- time.After(time.Second * 20):
		gCbChan <- []interface{}{"get&del", rid, nil}
	}
}

func CallBackMgr() {

	for true {
		elem := <- gCbChan
		oper := elem[0].(string)

		if "add" == oper {
			rid := elem[1].(uint32)
			cb := CallBack(elem[2].(func(err string, result map[string]interface{})))

			if len(gCbMap) >= gCbMapMaxSize {
				msf.ERROR_LOG("call back cache size %v > %v", len(gCbMap), gCbMapMaxSize)
				return
			}

			gCbMap[rid] = cb

			go CallBackTimeOut(rid)

		} else if "get&del" == oper {
			rid := elem[1].(uint32)
			cb, ok := gCbMap[rid]

			cbChan := elem[2]
			if ok {
				if cbChan != nil {
					cbChan.(chan interface{}) <- cb
				}
				delete(gCbMap, rid)

			} else {
				if cbChan != nil {
					cbChan.(chan interface{}) <- nil
					msf.ERROR_LOG("call back get error %v", rid)
				}
			}
		}
	}
}

func CallBackMgrStart() {
	gCbMap = make(map[uint32] CallBack)
	gCbChan = make(chan []interface{})
	go CallBackMgr()
}

func Init() {
	CallBackMgrStart()

	msf.CreateSimpleRpcMgr()
	
	rpcMgr = msf.GetRpcMgr()
	rpcMgr.RegistRpcHandler(msf.MSG_COMMON_RSP, func() msf.RpcHandler {return new(RpcCommonRspHandler)})
}

type GateProxy struct {
	ip 				string
	port 			int
	conn			net.Conn
	recvBuf 		[]byte
	remainLen 		uint32
}

func (c *GateProxy) Start() {
	go c.HandleRead()
}

func (c *GateProxy) HandleRead() {
	defer func() {
		msf.INFO_LOG("tcp client close %v", c.conn.RemoteAddr())
		c.conn.Close()
	} ()

	for true {
		len, err := c.conn.Read(c.recvBuf[c.remainLen:])
		if err != nil {
			if err != io.EOF {
				msf.ERROR_LOG("read error %v", err)
				break
			}
		}

		if 0 == len {
			// remote close
			msf.INFO_LOG("tcp connection close by remote %v %v", c.conn.RemoteAddr(), err)
			break
		}

		c.remainLen += uint32(len)
		if c.remainLen > msf.RECV_BUF_MAX_LEN/2 {
			msf.WARN_LOG("tcp connection buff cache too long!!! %dk > %dk", c.remainLen/1024, msf.RECV_BUF_MAX_LEN/1024)
			
		} else if c.remainLen > msf.RECV_BUF_MAX_LEN {
			msf.ERROR_LOG("tcp connection buff cache overflow!!! %dk > %dk", c.remainLen/1024, msf.RECV_BUF_MAX_LEN/1024)
			break
		}

		procLen, _ := rpcMgr.MessageDecode(nil, c.recvBuf[:c.remainLen])
		c.remainLen -= procLen
		if c.remainLen < 0 {
			msf.ERROR_LOG("c.remainLen(%d) < 0 procLen(%d) @%s", c.remainLen, procLen, c.conn.RemoteAddr())
			c.remainLen = 0
			continue
		}

		copy(c.recvBuf, c.recvBuf[procLen: procLen + c.remainLen])
	}
}

func (c *GateProxy) RpcCall(rpcName string, args ...interface{}) {
	rpc := rpcMgr.RpcEncode(rpcName, args...)
	msg := rpcMgr.MessageEncode(rpc)

	wLen, err := c.conn.Write(msg)
	if err != nil {
		msf.ERROR_LOG("write %v error %v", c.conn.RemoteAddr(), err)
	}

	if wLen != len(msg) {
		msf.WARN_LOG("write len(%v) != msg len(%v) @%v", wLen, len(msg), c.conn.RemoteAddr())
	}
}

func CreateGateProxy(_ip string, _port int) *GateProxy {
	c, err := net.Dial("tcp", fmt.Sprintf("%s:%d", _ip, _port))
	if err != nil {
		msf.ERROR_LOG("connect %s:%d error %v", _ip, _port, err)
		return nil
	}

	msf.INFO_LOG("connect %s:%d success %v", _ip, _port, c)

	gp := &GateProxy {
		ip:	_ip,
		port: _port,
		conn: c, 
		recvBuf: make([]byte, msf.RECV_BUF_MAX_LEN), 
		remainLen: 0,
	}

	gp.Start()

	return gp
}

func (c *GateProxy) CreateServiceProxy(namespace string, serviceName string) *ServiceProxy {
	return &ServiceProxy{Gp: c, Namespace: namespace, ServiceName: serviceName}
}


type ServiceProxy struct {
	Gp 				*GateProxy
	Namespace 		string
	ServiceName		string
}

// c2s的rpc调用，最后一个参数若是Func，则建立rid<->callback的缓存
func (c *ServiceProxy) RpcCall(rpcName string, args ...interface{}) {

	msf.DEBUG_LOG("rpc call %s args %v", rpcName, args)

	var rid uint32 = 0
	if len(args) > 0 {
		lastArg := args[len(args)-1]
		t := reflect.TypeOf(lastArg)
		if t.Kind() == reflect.Func {
			rid = GenGid()
			gCbChan <- []interface{}{"add", rid, lastArg}
			args = args[:len(args)-1]
		}
	}

	innerRpc := rpcMgr.RpcEncode(rpcName, args...)
	c.Gp.RpcCall(msf.MSG_C2G_RPC_ROUTE, c.Namespace, c.ServiceName, rid, innerRpc)
}

func GenGid() uint32 {
	globalGID += 1
	return globalGID
}
