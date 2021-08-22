package main


import (
    msf "ms_framework"
    "fmt"
    "strings"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "context"
)


// MSG_G2P_RSP_LISTENADDR
type RpcRspListenAddrReq struct {
    Addr    string
}

type RpcRspListenAddrHandler struct {
    req     RpcRspListenAddrReq
}

func (r *RpcRspListenAddrHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcRspListenAddrHandler) GetRspPtr() interface{} {return nil}

func (r *RpcRspListenAddrHandler) Process(session *msf.Session) {
    gGateAddrMap[r.req.Addr] = msf.GetConnID(session.GetConn())

    msf.DEBUG_LOG("[gate find addr] %s", r.req.Addr)
}


// MSG_S2P_PUSH
const (
    PUSH_RECORD_STATE_UNARRIVE   int8 = iota    // 未到达
    PUSH_RECORD_STATE_ARRIVED                   // 已到达
)

type RpcS2PPushReq struct {
    ClientID    string
    Typ         string
    IsSafe      bool
    PushMsg     []byte
}

type RpcS2PPushHandler struct {
    req     RpcS2PPushReq
}

func (r *RpcS2PPushHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcS2PPushHandler) GetRspPtr() interface{} {return nil}

func (r *RpcS2PPushHandler) Process(session *msf.Session) {
    
    pid := []byte{}
    if r.req.IsSafe {
        // 持久化到mongo
        mongo := msf.GetMongo("myMongo")
        db := mongo.Database("push_service")
        coll := db.Collection("safe_push_records")
        doc := bson.D{{"clientID", r.req.ClientID}, {"type", r.req.Typ}, {"pushMsg", r.req.PushMsg}, {"state", PUSH_RECORD_STATE_UNARRIVE}}
        result, err := coll.InsertOne(context.Background(), doc)
        if err != nil {
            msf.ERROR_LOG("[safe push] mongo collection insert push record err %v", err)
            return
        }

        objID := result.InsertedID.(primitive.ObjectID)
        pid = objID[:] 

        msf.DEBUG_LOG("[safe push] save to mongo result: %T, %+v", result, result)
    }

    PushToGate(r.req.ClientID, r.req.Typ, r.req.PushMsg, pid)
}


// MSG_PUSH_REPLY
type RpcPushReplyReq struct {
    Pid    []byte
}

type RpcPushReplyHandler struct {
    req     RpcPushReplyReq
}

func (r *RpcPushReplyHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcPushReplyHandler) GetRspPtr() interface{} {return nil}
func (r *RpcPushReplyHandler) ClientAccess() bool {return true}

func (r *RpcPushReplyHandler) Process(session *msf.Session) {
    var objectID primitive.ObjectID
    copy(objectID[0:], r.req.Pid)

    mongo := msf.GetMongo("myMongo")
    db := mongo.Database("push_service")
    coll := db.Collection("safe_push_records")

    doc := bson.D{{"_id", objectID}}
    update := bson.D{{"$set", bson.D{{"state", PUSH_RECORD_STATE_ARRIVED}}}}
    result, err := coll.UpdateOne(context.Background(), doc, update)
    if err != nil {
        msf.ERROR_LOG("[push reply] mongo collection update %+v push record %v", objectID, err)
        return
    }

    msf.DEBUG_LOG("[push reply] update %+v result: %T, %+v", objectID, result, result)
}


// MSG_PUSH_RESTORE
type RpcPushRestoreReq struct {
    ClientID    string
}

type RpcPushRestoreHandler struct {
    req     RpcPushRestoreReq
}

func (r *RpcPushRestoreHandler) GetReqPtr() interface{} {return &(r.req)}
func (r *RpcPushRestoreHandler) GetRspPtr() interface{} {return nil}
func (r *RpcPushRestoreHandler) ClientAccess() bool {return true}

func (r *RpcPushRestoreHandler) Process(session *msf.Session) {
    mongo := msf.GetMongo("myMongo")
    db := mongo.Database("push_service")
    coll := db.Collection("safe_push_records")

    doc := bson.D{{"clientID", r.req.ClientID}, {"state", PUSH_RECORD_STATE_UNARRIVE}}
    cursor, err := coll.Find(context.Background(), doc)
    if err != nil {
        msf.ERROR_LOG("[push restore] mongo collection find %+v error %v", r.req.ClientID, err)
        return
    }

    var docs []bson.D
    err = cursor.All(context.Background(), &docs)
    if err != nil {
        msf.ERROR_LOG("[push restore] mongo collection find %+v error cursor %v", r.req.ClientID, err)
        return
    }

    msf.DEBUG_LOG("[push restore] mongo docs %+v", len(docs))
    for _, doc := range docs {
        // msf.DEBUG_LOG("[push restore] mongo doc %d, %+v", index, doc)
        objID := doc[0].Value.(primitive.ObjectID)
        clientID := doc[1].Value.(string)
        typ := doc[2].Value.(string)
        pushMsg := doc[3].Value.(primitive.Binary).Data

        PushToGate(clientID, typ, pushMsg, objID[:])
    }
}

func PushToGate(clientID string, typ string, pushMsg []byte, pid []byte) {

    key := ""
    if "server" == typ {
        key = fmt.Sprintf("s_%s", clientID)
    } else if "client" == typ {
        key = fmt.Sprintf("c_%s", clientID)
    } else {
        msf.ERROR_LOG("[safe push] error req type %s", typ)
        return
    }

    redisCluster := msf.GetRedisCluster("myRedis2")
    target, err := redisCluster.Get(key).Result()
    if err != nil {
        msf.ERROR_LOG("[safe push] redis get %s error %v", key, err)
        return
    }

    v := strings.Split(target, "/")
    gateAddr := v[0]
    clientConnID := v[1]

    connID, ok := gGateAddrMap[gateAddr]
    if !ok {
        msf.ERROR_LOG("[safe push] gate addr(%s) not exist", gateAddr)
        return
    }
    
    client := msf.GetTcpClient(msf.CONN_ID(connID))
    if nil == client {
        msf.ERROR_LOG("[safe push] gate client(%s) not exist", connID)
        return
    }

    rpc := msf.RpcEncode(msf.MSG_P2G_PUSH, clientConnID, pid, msf.GlobalCfg.Namespace, pushMsg)
    msg := msf.MessageEncode(rpc)
    msf.MessageSend(client.GetConn(), msg)
}

