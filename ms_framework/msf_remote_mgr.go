package ms_framework

import (
	"fmt"
	"net"
	"strconv"
	"time"
	"io"
	"math/rand"
)

type REMOTE_ID string  // namespace:service

/*
 * 建立本地路由缓存
 * 本地路由缓存更新：被动更新（依赖etcd）
 * 负载均衡
 */
type RemoteMgr struct {
	remotes  	map[REMOTE_ID]map[CONN_ID]*Remote
	addChan		chan []string
	delChan		chan []string
}

type Remote struct {
	id 				REMOTE_ID
	conn			net.Conn
	rmgr 			*RemoteMgr
	recvBuf 		[]byte
	remainLen 		uint32
}

func (rmgr *RemoteMgr) Start() {
	for true {
		select {
		case add := <- rmgr.addChan:
			INFO_LOG("OnRemoteDiscover %s:%s @%s:%s", add[0], add[1], add[2], add[3])
			port, _ := strconv.Atoi(add[3])
			rmgr.ConnectRemote(add[0], add[1], add[2], uint32(port))

		case del := <- rmgr.delChan:
			remoteID, connID := REMOTE_ID(del[0]), CONN_ID(del[1])
			conns, ok := rmgr.remotes[remoteID]
			if !ok {
				ERROR_LOG("remote not exist %s", remoteID)

			} else {
				_, ok := conns[connID]
				if !ok {
					ERROR_LOG("remote conn not exist %s @%s", remoteID, connID)
				}

				INFO_LOG("OnRemoteDisappear %s:%s @%v", remoteID, connID)
				delete(conns, connID)
			}
		}
	}
}

func (rmgr *RemoteMgr) OnRemoteDiscover(namespace string, svrName string, ip string, port uint32) {
	// INFO_LOG("OnRemoteDiscover %v %v", namespace, svrName)
	rmgr.addChan <- []string{namespace, svrName, ip, fmt.Sprintf("%d", port)}
}

func (rmgr *RemoteMgr) OnRemoteDisappear(remoteID REMOTE_ID, connID CONN_ID) {
	// INFO_LOG("OnRemoteDisappear %v %v %+v", remoteID, connID, rmgr)
	rmgr.delChan <- []string{string(remoteID), string(connID)}
}

func (rmgr *RemoteMgr) ConnectRemote(namespace string, svrName string, ip string, port uint32) {
	retryCnt := 5
	for true {
		c, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), time.Second)
		if err != nil {
			ERROR_LOG("connect %s:%s @%s:%d fail %v retry(%d)...", namespace, svrName, ip, port, err, retryCnt)
			time.Sleep(time.Second)

			retryCnt -= 1
			if retryCnt <= 0 {
				break
			}

			continue
		}

		remoteID := GetRemoteID(namespace, svrName)
		_, ok := rmgr.remotes[remoteID]
		if !ok {
			rmgr.remotes[remoteID] = make(map[CONN_ID]*Remote)
		}

		rmgr.remotes[remoteID][GetConnID(c)] = &Remote{
			id: remoteID,
			conn: c,
			rmgr: rmgr,
			recvBuf: make([]byte, RECV_BUF_MAX_LEN),
			remainLen: 0,
		}

		go rmgr.remotes[remoteID][GetConnID(c)].HandleRead()
		break
	}
}

func (r *Remote) HandleRead() {
	defer func() {
		INFO_LOG("remote close %v", r.conn.RemoteAddr())
		r.rmgr.OnRemoteDisappear(r.id, GetConnID(r.conn))
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

		procLen, _ := rpcMgr.MessageDecode(nil, r.recvBuf[:r.remainLen])
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

var remoteMgr *RemoteMgr = nil

func CreateRemoteMgr() {
	remoteMgr = &RemoteMgr{
		remotes: make(map[REMOTE_ID]map[CONN_ID]*Remote),
		addChan: make(chan []string),
		delChan: make(chan []string),
	}

	go remoteMgr.Start()
}

func GetRemoteID(namespace string, svrName string) REMOTE_ID {
	return REMOTE_ID(fmt.Sprintf("%s:%s", namespace, svrName))
}

func ChoiceRemote(remoteID REMOTE_ID) *Remote {
	conns, ok := remoteMgr.remotes[remoteID]
	if !ok {
		return nil
	}

	// 负载均衡
	var remote *Remote = nil
	idx := rand.Intn(len(conns))
	for _, remote = range(conns) {
		if idx <= 0 {
			break
		}

		idx -= 1
	}

	return remote
}
