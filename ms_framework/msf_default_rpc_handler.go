package ms_framework

import (
	"fmt"
	"github.com/vmihailenco/msgpack"
	"bytes"
	"reflect"
	"net"
)


var MSG_C2G_RPC_ROUTE 			= "a"  // client to gate rpc route (client include mservice/gameserver/gameclient)
var MSG_G2S_RPC_CALL 			= "b"  // gate to service rpc call
var MSG_S2G_RPC_RSP 			= "c"  // service to gate rpc response
var MSG_G2C_RPC_RSP 			= "d"  // gate to client rpc response

var MSG_HEART_BEAT_REQ 			= "e"  // heart beat request
var MSG_HEART_BEAT_RSP 			= "f"  // heart beat response

var MSG_G2S_IDENTITY_REPORT		= "g"  // gate to service identity report (cluster gate or client gate ?)

type C2GRouteCbInfo struct {
	rid 			uint32
	connID 			CONN_ID
	nameSpace	 	string
	service 		string
	rpcName			string
	createTime		int64
}

// ***************************  client->gate  ***************************
// ***************************  gate->service ***************************
// ***************************  service->gate ***************************
// ***************************  gate->client  ***************************

// MSG_C2G_RPC_ROUTE
type RpcC2GRpcRouteReq struct {
	NameSpace	 	string
	Service 		string
	Rid 			uint32
	InnerRpc		[]byte
}

type RpcC2GRpcRouteHandler struct {
	req 	RpcC2GRpcRouteReq
}

func (r *RpcC2GRpcRouteHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcC2GRpcRouteHandler) GetRspPtr() interface{} {return nil}

func (r *RpcC2GRpcRouteHandler) Process(session *Session) {
	/*
	 * 消息路由，根据namespace:service:method从本地ip缓存中找到对应service的tcp连接，然后将消息路由过去
	 * 从B里面解析出Rid
	 * if Rid != 0
	 *   生成GRid，并建立GRid <-> clientID:Rid的对应关系
	 * 往service发送MSG_G2S_RPC_CALL请求
	 */

	RpcFvcCount()
	
	remoteID := GetRemoteID(r.req.NameSpace, r.req.Service)
	remote := ChoiceRemote(remoteID)

	if remote != nil {

		// DEBUG_LOG("[RpcC2GRpcRouteHandler] - SERVICE - [%s:%s] rid[%v] response[nil]", r.req.NameSpace, r.req.Service, r.req.Rid)

		grid := uint32(0)
		if r.req.Rid != 0 {
			grid = GenGid()
		}

		if r.req.Rid != 0 {

			var rpcName string
			decoder := msgpack.NewDecoder(bytes.NewBuffer(r.req.InnerRpc))
			decoder.Decode(&rpcName)

			cbInfo := C2GRouteCbInfo {
				rid: r.req.Rid, 
				connID: CONN_ID(session.GetID()), 
				nameSpace: r.req.NameSpace, 
				service: r.req.Service, 
				rpcName: rpcName, 
				createTime: GetNowTimestampMs(),
			}

			// timeoutCb := func() {
			// 	error := fmt.Sprintf("rpc call %s:%s:%s gate time out", r.req.NameSpace, r.req.Service, rpcName)
			// 	INFO_LOG("[rpc route] - [%s:%s] rid[%v] response[%v]", r.req.NameSpace, r.req.Service, r.req.Rid, error)

			// 	rpc := rpcMgr.RpcEncode(MSG_G2C_RPC_RSP, r.req.Rid, error, nil)
			// 	msg := rpcMgr.MessageEncode(rpc)
			// 	MessageSend(session.conn, msg)
			// }

			// 超时时间最好大于client cb的超时时间
			AddCallBack(grid, []interface{}{cbInfo}, 101, nil)
		}

		rpc := RpcEncode(MSG_G2S_RPC_CALL, grid, r.req.InnerRpc)
		msg := MessageEncode(rpc)
		MessageSend(remote.conn, msg)

	} else {
		
		error := fmt.Sprintf("service %s:%s not exist", r.req.NameSpace, r.req.Service)
		INFO_LOG("[rpc route] - [%s:%s] rid[%v] response[%v]", r.req.NameSpace, r.req.Service, r.req.Rid, error)

		// error response
		rpc := rpcMgr.RpcEncode(MSG_G2C_RPC_RSP, r.req.Rid, error, nil)
		msg := rpcMgr.MessageEncode(rpc)
		MessageSend(session.conn, msg)
	}
}


// MSG_G2S_RPC_CALL
type RpcG2SRpcCallReq struct {
	GRid 			uint32
	InnerRpc		[]byte  // 
}

type RpcG2SRpcCallHandler struct {
	req 	RpcG2SRpcCallReq
}

func (r *RpcG2SRpcCallHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcG2SRpcCallHandler) GetRspPtr() interface{} {return nil}

func (r *RpcG2SRpcCallHandler) Process(session *Session) {
	
	RpcFvcCount()
	
	error, reply := r.rpc_handler(session)

	if r.req.GRid != 0 {

		// response
		rpc := rpcMgr.RpcEncode(MSG_S2G_RPC_RSP, r.req.GRid, error, reply)
		msg := rpcMgr.MessageEncode(rpc)
		MessageSend(session.conn, msg)
	}

	if error != "" {
		ERROR_LOG("[rpc call] - %s", error)
	}
}

func (r *RpcG2SRpcCallHandler) rpc_handler(session *Session) (string, map[string]interface{}){

	decoder := msgpack.NewDecoder(bytes.NewBuffer(r.req.InnerRpc))

	var rpcName string
	decoder.Decode(&rpcName)

	f, ok := rpcMgr.GetRpcHanderGenerator(rpcName)
	if !ok {
		return fmt.Sprintf("rpc %s not exist", rpcName), nil
	}

	handler := f()
	reqPtr := reflect.ValueOf(handler.GetReqPtr())
	if handler.GetReqPtr() != nil && !reqPtr.IsNil() {
		stValue := reqPtr.Elem()
		for i := 0; i < stValue.NumField(); i++ {
			nv := reflect.New(stValue.Field(i).Type())
			if err := decoder.Decode(nv.Interface()); err != nil {
				return fmt.Sprint("rpc %s arg %s(%v) decode error: %v", rpcName, stValue.Type().Field(i).Name, nv.Type(), err), nil
			}

			stValue.Field(i).Set(nv.Elem())
		}
	}

	handler.Process(session)

	if r.req.GRid != 0 {
		rspPtr := reflect.ValueOf(handler.GetRspPtr())
		if handler.GetRspPtr() == nil || rspPtr.IsNil() {
			if r.req.GRid != 0 {
				return fmt.Sprint("rpc %s need response but get nil", rpcName), nil
			}
			return "", nil
		}

		// for response
		stMap := make(map[string]interface{})
		stValue := rspPtr.Elem()
		for i := 0; i < stValue.NumField(); i++ {
			stMap[stValue.Type().Field(i).Name] = stValue.Field(i).Interface()
		}

		err := GetResponseErr(handler.GetRspPtr())
		INFO_LOG("[rpc call] - [%s] args[%v] err[%v] reply[%v]", rpcName, handler.GetReqPtr(), err, stMap)
		return err, stMap
	}

	INFO_LOG("[rpc call] - [%s] args[%v] reply[nil]", rpcName, handler.GetReqPtr())
	return "", nil
}

// MSG_S2G_RPC_RSP
type RpcS2GRpcRsp struct {
	GRid	uint32
	Error   string
	Reply  	map[string]interface{}
}

type RpcS2GRpcRspHandler struct {
	req 	RpcS2GRpcRsp
}

func (r *RpcS2GRpcRspHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcS2GRpcRspHandler) GetRspPtr() interface{} {return nil}

func (r *RpcS2GRpcRspHandler) Process(session *Session) {
	// 根据GRid找到clientID:Rid，将GRid替换成Rid，然后把Error和Reply透传给client

	cbs := GetCallBack(r.req.GRid)
	if nil == cbs {
		return
	}

	cbInfo := cbs[0].(C2GRouteCbInfo)

	rid, connID := cbInfo.rid, cbInfo.connID

	var conn net.Conn
	if client := GetClient(connID); client != nil {
		conn = client.conn
	} else if remote := GetRemote(connID); remote != nil {
		conn = remote.conn
	}

	if nil == conn {
		ERROR_LOG("[rpc route] - response error: connID[%s] not exist", connID)
		return
	}

	INFO_LOG("[rpc route] - [%s:%s:%s] rid[%v] err[%s] reply[%v] timeCost[%vms]", 
		cbInfo.nameSpace, cbInfo.service, cbInfo.rpcName, rid, r.req.Error, r.req.Reply, GetNowTimestampMs() - cbInfo.createTime)

	rpc := RpcEncode(MSG_G2C_RPC_RSP, rid, r.req.Error, r.req.Reply)
	msg := MessageEncode(rpc)
	MessageSend(conn, msg)
}

// MSG_G2C_RPC_RSP
type RpcG2CRpcRspReq struct {
	Rid 	uint32
	Error   string
	Reply   map[string]interface{}
}

type RpcG2CRpcRspHandler struct {
	req 	RpcG2CRpcRspReq
}

func (r *RpcG2CRpcRspHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcG2CRpcRspHandler) GetRspPtr() interface{} {return nil}

func (r *RpcG2CRpcRspHandler) Process(session *Session) {

	cbs := GetCallBack(r.req.Rid)
	if nil == cbs {
		ERROR_LOG("RpcG2CRpcRspHandler GetCallBack error %v", r.req.Rid)
		return
	}

	ch := cbs[0].(chan []interface{})
	ch <- []interface{}{r.req.Error, r.req.Reply}
}

// ***************************  heart beat ***************************

// MSG_HEART_BEAT_REQ
type RpcHeartBeatReqHandler struct {
}

func (r *RpcHeartBeatReqHandler) GetReqPtr() interface{} {return nil}
func (r *RpcHeartBeatReqHandler) GetRspPtr() interface{} {return nil}

func (r *RpcHeartBeatReqHandler) Process(session *Session) {
	// DEBUG_LOG("heart beat request from %v", GetConnID(session.conn))

	rpc := RpcEncode(MSG_HEART_BEAT_RSP)
	msg := MessageEncode(rpc)
	MessageSend(session.conn, msg)
}

// MSG_HEART_BEAT_RSP
type RpcHeartBeatRspHandler struct {
}

func (r *RpcHeartBeatRspHandler) GetReqPtr() interface{} {return nil}
func (r *RpcHeartBeatRspHandler) GetRspPtr() interface{} {return nil}

func (r *RpcHeartBeatRspHandler) Process(session *Session) {
	// DEBUG_LOG("heart beat response from %v", GetConnID(session.conn))
}

// ***************************  identity report ***************************

// MSG_G2S_IDENTITY_REPORT
type RpcG2SIdentityReportReq struct {
	Identity 	int8
}

type RpcG2SIdentityReportHandler struct {
	req 	RpcG2SIdentityReportReq
}

func (r *RpcG2SIdentityReportHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcG2SIdentityReportHandler) GetRspPtr() interface{} {return nil}

func (r *RpcG2SIdentityReportHandler) Process(session *Session) {
	identityStr, ok := IdentityMap[r.req.Identity]
	if !ok {
		ERROR_LOG("error identity report %d from %v", r.req.Identity, GetConnID(session.conn))
	} else {
		DEBUG_LOG("identity report %s from %v", identityStr, GetConnID(session.conn))
		tcpServer.onClientIdentityReport(session.conn, r.req.Identity)
	}
}
