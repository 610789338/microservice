package clientsdk

import (
	"net"
	"fmt"
	msf "ms_framework"
	// "time"
	"io"
	"reflect"
	"sync"
)

var rpcMgr *msf.SimpleRpcMgr = nil

type CallBack func(err string, result map[string]interface{})

func Init() {
	msf.CreateSimpleRpcMgr()
	
	rpcMgr = msf.GetRpcMgr()
	rpcMgr.RegistRpcHandler(msf.MSG_G2C_RPC_RSP, func() msf.RpcHandler {return new(RpcG2CRpcRspHandler)})
}

type GateProxy struct {
	ip 				string
	port 			int
	conn			net.Conn
	recvBuf 		[]byte
	remainLen 		uint32
}

func (g *GateProxy) Start() {
	go g.HandleRead()
}

func (g *GateProxy) HandleRead() {
	defer func() {
		msf.INFO_LOG("tcp client close %v", g.conn.RemoteAddr())
		g.conn.Close()
	} ()

	for true {
		len, err := g.conn.Read(g.recvBuf[g.remainLen:])
		if err != nil {
			if err != io.EOF {
				msf.ERROR_LOG("read error %v", err)
				break
			}
		}

		if 0 == len {
			// remote close
			msf.INFO_LOG("tcp connection close by remote %v %v", g.conn.RemoteAddr(), err)
			break
		}

		g.remainLen += uint32(len)
		if g.remainLen > msf.RECV_BUF_MAX_LEN/2 {
			msf.WARN_LOG("tcp connection buff cache too long!!! %dk > %dk", g.remainLen/1024, msf.RECV_BUF_MAX_LEN/1024)
			
		} else if g.remainLen > msf.RECV_BUF_MAX_LEN {
			msf.ERROR_LOG("tcp connection buff cache overflow!!! %dk > %dk", g.remainLen/1024, msf.RECV_BUF_MAX_LEN/1024)
			break
		}

		procLen := rpcMgr.MessageDecode(g.Turn2Session(), g.recvBuf[:g.remainLen])
		g.remainLen -= procLen
		if g.remainLen < 0 {
			msf.ERROR_LOG("g.remainLen(%d) < 0 procLen(%d) @%s", g.remainLen, procLen, g.conn.RemoteAddr())
			g.remainLen = 0
			continue
		}

		copy(g.recvBuf, g.recvBuf[procLen: procLen + g.remainLen])
	}
}

func (g *GateProxy) RpcCall(rpcName string, args ...interface{}) {
	rpc := rpcMgr.RpcEncode(rpcName, args...)
	msg := rpcMgr.MessageEncode(rpc)

	wLen, err := g.conn.Write(msg)
	if err != nil {
		msf.ERROR_LOG("write %v error %v", g.conn.RemoteAddr(), err)
	}

	if wLen != len(msg) {
		msf.WARN_LOG("write len(%v) != msg len(%v) @%v", wLen, len(msg), g.conn.RemoteAddr())
	}
}

func (g *GateProxy) Turn2Session() *msf.Session {
	return msf.CreateSession(msf.SessionTcpClient, fmt.Sprintf("%s:%d", g.ip, g.port), g.conn)
}

func CreateGateProxy(_ip string, _port int) *GateProxy {
	g, err := net.Dial("tcp", fmt.Sprintf("%s:%d", _ip, _port))
	if err != nil {
		msf.ERROR_LOG("connect %s:%d error %v", _ip, _port, err)
		return nil
	}

	msf.INFO_LOG("connect %s:%d success %v", _ip, _port, g)

	gp := &GateProxy {
		ip:	_ip,
		port: _port,
		conn: g, 
		recvBuf: make([]byte, msf.RECV_BUF_MAX_LEN), 
		remainLen: 0,
	}

	gp.Start()

	return gp
}

func (g *GateProxy) CreateServiceProxy(namespace string, serviceName string) *ServiceProxy {
	return &ServiceProxy{Gp: g, Namespace: namespace, ServiceName: serviceName}
}


type ServiceProxy struct {
	Gp 				*GateProxy
	Namespace 		string
	ServiceName		string
}

// c2s的rpc调用，最后一个参数若是Func，则建立rid<->callback的缓存
func (g *ServiceProxy) RpcCall(rpcName string, args ...interface{}) {

	var rid uint32 = 0
	if len(args) > 0 {
		lastArg := args[len(args)-1]
		t := reflect.TypeOf(lastArg)
		if t.Kind() == reflect.Func {
			rid = GenGid()
			msf.AddCallBack(rid, []interface{}{lastArg.(CallBack)})
			args = args[:len(args)-1]
		}
	}

	// msf.DEBUG_LOG("rpc call %s args %v", rpcName, args)

	innerRpc := rpcMgr.RpcEncode(rpcName, args...)
	g.Gp.RpcCall(msf.MSG_C2G_RPC_ROUTE, g.Namespace, g.ServiceName, rid, innerRpc)
}

// 单个实例唯一的gid
var globalGID uint32 = 0
var gidMutex sync.Mutex

func GenGid() uint32 {
	var ret uint32
	gidMutex.Lock()
	globalGID += 1
	ret = globalGID
	gidMutex.Unlock()

	return ret
}
