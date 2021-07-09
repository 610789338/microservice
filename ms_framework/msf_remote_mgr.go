package ms_framework

import (
    "fmt"
    "net"
    "time"
    "io"
    "sync"
)

type REMOTE_ID string  // namespace:service

/*
 * 用于gate服务发现
 * 依赖etcd建立本地路由缓存
 * 根据缓存做路由负载均衡
 */
type RemoteMgr struct {
    remotes          map[CONN_ID]*Remote
    lbs              map[REMOTE_ID]*LoadBalancer
    mutex            sync.RWMutex
}

type Remote struct {
    id               REMOTE_ID
    conn             net.Conn
    recvBuf          []byte
    remainLen        uint32
}

func (rmgr *RemoteMgr) OnRemoteDiscover(namespace string, svrName string, ip string, port uint32) {

    connID := GenConnIDByIPPort(ip, port)
    _, ok := rmgr.remotes[connID]
    if ok {
        WARN_LOG("remote %s:%s@%v already exist", namespace, svrName, connID)

    } else {
        INFO_LOG("OnRemoteDiscover %s:%s@%v", namespace, svrName, connID)

        retryCnt := 5
        for retryCnt > 0 {
            err := rmgr.ConnectRemote(namespace, svrName, ip, port)
            if err != nil {
                ERROR_LOG("connect %s:%s @%s:%d fail %v retry(%d)...", namespace, svrName, ip, port, err, retryCnt)
                time.Sleep(time.Second)

                retryCnt -= 1
                continue
            }

            break
        }
    }
}

func (rmgr *RemoteMgr) OnRemoteDisappear(remoteID REMOTE_ID, connID CONN_ID) {

    rmgr.mutex.Lock()
    defer rmgr.mutex.Unlock()

    _, ok := rmgr.remotes[connID]
    if !ok {
        // ERROR_LOG("remote not exist %s", remoteID)
    } else {
        INFO_LOG("OnRemoteDisappear %s@%v", remoteID, connID)
        
        delete(rmgr.remotes, connID)
        rmgr.lbs[remoteID].DelElement(string(connID))
    }
}

func (rmgr *RemoteMgr) ConnectRemote(namespace string, svrName string, ip string, port uint32) error {
    c, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), 5*time.Second)
    if err != nil {
        return err
    }
    
    INFO_LOG("connect %s:%s @%s:%d success", namespace, svrName, ip, port)

    rmgr.mutex.Lock()
    defer rmgr.mutex.Unlock()

    connID := GenConnIDByIPPort(ip, port)
    _, ok := rmgr.remotes[connID]
    if ok {
        ERROR_LOG("repeate remote %s", connID)
        return nil
    }

    remoteID := GetRemoteID(namespace, svrName)
    _, ok = rmgr.lbs[remoteID]
    if !ok {
        rmgr.lbs[remoteID] = &LoadBalancer{}
    }

    if !rmgr.lbs[remoteID].AddElement(string(connID)) {
        return nil
    }

    rmgr.remotes[connID] = &Remote{
        id: remoteID,
        conn: c,
        recvBuf: make([]byte, RECV_BUF_MAX_LEN),
        remainLen: 0,
    }
    
    go rmgr.remotes[connID].HandleRead()

    // 上报身份
    rpc := RpcEncode(MSG_G2S_IDENTITY_REPORT, GetServerIdentity())
    msg := MessageEncode(rpc)
    MessageSend(c, msg)

    return nil
}

func (r *Remote) HandleRead() {
    defer func() {
        INFO_LOG("remote close %v", r.conn.RemoteAddr())
        remoteMgr.OnRemoteDisappear(r.id, GetConnID(r.conn))
        r.conn.Close()
    } ()

    for true {
        len, err := r.conn.Read(r.recvBuf[r.remainLen:])
        // INFO_LOG("remote read %v %v", len, err)
        if err != nil {
            if err != io.EOF {
                ERROR_LOG("read error %v", err)
                break
            }
        }

        if 0 == len {
            // remote close
            INFO_LOG("tcp connection close by remote %v %v", r.conn.RemoteAddr(), err)
            break
        }

        r.remainLen += uint32(len)
        if r.remainLen > RECV_BUF_MAX_LEN/2 {
            WARN_LOG("tcp connection buff cache too long!!! %dk > %dk", r.remainLen/1024, RECV_BUF_MAX_LEN/1024)
            
        } else if r.remainLen > RECV_BUF_MAX_LEN {
            ERROR_LOG("tcp connection buff cache overflow!!! %dk > %dk", r.remainLen/1024, RECV_BUF_MAX_LEN/1024)
            break
        }

        procLen := rpcMgr.MessageDecode(r.Turn2Session(), r.recvBuf[:r.remainLen])
        r.remainLen -= procLen
        if r.remainLen < 0 {
            ERROR_LOG("r.remainLen(%d) < 0 procLen(%d) @%s", r.remainLen, procLen, r.conn.RemoteAddr())
            r.remainLen = 0
            continue
        }

        copy(r.recvBuf, r.recvBuf[procLen: procLen + r.remainLen])
    }
}

func (r *Remote) Write(b []byte) (n int, err error){
    n, err = r.conn.Write(b)
    return
}

func (r *Remote) RemoteAddr() net.Addr {
    return r.conn.RemoteAddr()
}

func (r *Remote) Turn2Session() *Session {
    return &Session{typ: SessionRemote, conn: r.conn}
}

func (r *Remote) GetConn() net.Conn {
    return r.conn
}

var remoteMgr *RemoteMgr = nil

func CreateRemoteMgr() {
    remoteMgr = &RemoteMgr{
        remotes: make(map[CONN_ID]*Remote),
        lbs:     make(map[REMOTE_ID]*LoadBalancer),
    }
}

func GetRemoteID(namespace string, svrName string) REMOTE_ID {
    return REMOTE_ID(fmt.Sprintf("%s:%s", namespace, svrName))
}

func ChoiceRemote(remoteID REMOTE_ID) *Remote {
    lbs, ok := remoteMgr.lbs[remoteID]
    if !ok {
        return nil
    }

    connID := CONN_ID(lbs.LoadBalance())
    remote, ok := remoteMgr.remotes[connID]
    if !ok {
        return nil
    }

    return remote
}

func GetRemote(connID CONN_ID) *Remote {
    remoteMgr.mutex.RLock()
    defer remoteMgr.mutex.RUnlock()

    remote, ok := remoteMgr.remotes[connID]
    if !ok {
        return nil
    }

    return remote
}

func OnRemoteDiscover(namespace string, svrName string, ip string, port uint32) {
    go remoteMgr.OnRemoteDiscover(namespace, svrName, ip, port)
}

func OnRemoteDisappear(namespace string, svrName string, ip string, port uint32) {
    remoteID := GetRemoteID(namespace, svrName)
    connID := GenConnIDByIPPort(ip, port)
    go remoteMgr.OnRemoteDisappear(remoteID, connID)
}
