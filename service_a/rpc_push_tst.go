package main


import (
	msf "ms_framework"
)


type RpcPushTestHandler struct {
}

func (r *RpcPushTestHandler) GetReqPtr() interface{} {return nil}
func (r *RpcPushTestHandler) GetRspPtr() interface{} {return nil}

func (r *RpcPushTestHandler) Process(session *msf.Session) {
	msf.PushClientUnsafe("client0", "push_test", int32(10), "hello")
}
