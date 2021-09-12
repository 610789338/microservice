package main

import (
    msf "ms_framework"
)


var push_fvc = msf.FlowVelocityCounter{Counter: "push fvc"}
// 
type RpcPushTest struct {
    Arg1     uint32
    Arg2     string
}

type RpcPushTestHandler struct {
    req     RpcPushTest
}

func (r *RpcPushTestHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcPushTestHandler) GetRspPtr() interface{} {return nil}

func (r *RpcPushTestHandler) Process(session *msf.Session) {
    msf.DEBUG_LOG("rpc push test %v, %v", r.req.Arg1, r.req.Arg2)
    push_fvc.Count()
}

func init() {
    msf.RegistRpcHandlerForce("push_test",     func() msf.RpcHandler {return new(RpcPushTestHandler)})
    push_fvc.Start()
}
