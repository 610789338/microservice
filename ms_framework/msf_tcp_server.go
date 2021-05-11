package ms_framework

import (
	"net"
	"fmt"
	"time"
	"io"
	"sync"
)

type CONN_ID string

var RECV_BUF_MAX_LEN uint32 = 10*1024*1024  // 10M

func GetClientID(c net.Conn) CONN_ID {
	return GetConnID(c)
}

func GetConnID(c net.Conn) CONN_ID {
	return CONN_ID(c.RemoteAddr().String())  // ip:port
}

func GenConnIDByIPPort(ip string, port uint32) CONN_ID {
	return CONN_ID(fmt.Sprintf("%s:%d", ip, port))
}

type TcpServer struct {
	ip 				string
	port 			int
	listener 		net.Listener
	clients  		map[CONN_ID]*TcpClient
	lb 				*LoadBalancer
	stop 			bool
	mutex			sync.RWMutex
}

type TcpClient struct {
	id 				CONN_ID
	conn			net.Conn
	recvBuf 		[]byte
	remainLen 		uint32
	exit 			bool
	lastActiveTime  int64
}

func (s *TcpServer) Start() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.ip, s.port))
	if err != nil {
		panic(fmt.Sprintf("CreateTcpServer error %s:%d - %v", s.ip, s.port, err))
	}
	defer s.Stop()

	s.listener = listener
	INFO_LOG("tcp server listen at %v", s.listener.Addr())

	for !s.stop {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.stop {
				break
			}

			ERROR_LOG("tcp server accept error %v", err)
			continue
		}

		INFO_LOG("tcp server(%s:%d) accept client %v", s.ip, s.port, conn.RemoteAddr())

		connID := GetConnID(conn)

		s.mutex.Lock()

		if s.stop {
			s.mutex.Unlock()
			break
		}

		if !s.lb.AddElement(string(connID), 0) {
			s.mutex.Unlock()
			return
		}

		s.clients[connID] = &TcpClient{
			id: connID, 
			conn: conn, 
			recvBuf: make([]byte, RECV_BUF_MAX_LEN),
			remainLen: 0,
			exit: false,
			lastActiveTime: GetNowTimestampMs(),
		}
		s.mutex.Unlock()

		go s.clients[connID].HandleRead()
	}
}

func (s *TcpServer) Stop(){
	// INFO_LOG("tcp server(%s:%d) close start ... %+v", s.ip, s.port, s)

	if s.stop {
		return
	}

	s.mutex.Lock()
	for _, client := range s.clients {
		client.exit = true
	}

	s.stop = true

	if s.listener != nil {
		s.listener.Close()
	}
	s.mutex.Unlock()

	// wait all client close
	for true {
		s.mutex.Lock()
		if len(s.clients) == 0 {
			s.mutex.Unlock()
			break
		}
		
		s.mutex.Unlock()
	}

	INFO_LOG("tcp server(%s:%d) close...", s.ip, s.port)
}

func (s *TcpServer) onClientClose(c *TcpClient){
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.clients, GetClientID(c.conn))
	s.lb.DelElement(string(GetConnID(c.conn)))
}

func (c *TcpClient) HandleRead() {
	defer func() {
		c.Close()
	} ()

	for !c.exit {
		c.conn.SetReadDeadline(time.Now().Add(100*time.Millisecond))
		rLen, err := c.conn.Read(c.recvBuf[c.remainLen:])
		if err != nil {
			e, ok := err.(*net.OpError)
			if ok && e.Timeout() == true {
				// WARN_LOG("read timeout %v", err)

				// heart beat
				now := GetNowTimestampMs()
				if now - c.lastActiveTime > 10*1000 {

					if now - c.lastActiveTime > 20*1000 {
						ERROR_LOG("tcp connect heartbeat timeout %v", c.conn.RemoteAddr(), now, c.lastActiveTime)
						break
					}

					rpc := rpcMgr.RpcEncode(MSG_HEART_BEAT_REQ)
					msg := rpcMgr.MessageEncode(rpc)
					if !c.HeartBeat(msg) {
						break
					}
				}
				continue
			}

			if err != io.EOF {
				ERROR_LOG("read error %v", err)
				break
			}
		}

		if 0 == rLen {
			// remote close
			INFO_LOG("tcp connection close by remote %v %v", c.conn.RemoteAddr(), err)
			break
		}

		// INFO_LOG("tcp recv buf %v %v", rLen, c.conn.RemoteAddr())

		c.lastActiveTime = GetNowTimestampMs()

		c.remainLen += uint32(rLen)
		if c.remainLen > RECV_BUF_MAX_LEN {
			ERROR_LOG("tcp connection buff cache overflow!!! %dk > %dk", c.remainLen/1024, RECV_BUF_MAX_LEN/1024)
			break
			
		} else if c.remainLen > RECV_BUF_MAX_LEN/2 {
			WARN_LOG("tcp connection buff cache too long!!! %dk > %dk", c.remainLen/1024, RECV_BUF_MAX_LEN/2/1024)
		}

		procLen := rpcMgr.MessageDecode(c.Turn2Session(), c.recvBuf[:c.remainLen])
		c.remainLen -= procLen
		if c.remainLen < 0 {
			ERROR_LOG("c.remainLen(%d) < 0 procLen(%d) @%s", c.remainLen, procLen, c.conn.RemoteAddr())
			c.remainLen = 0
			continue
		}

		copy(c.recvBuf, c.recvBuf[procLen: procLen + c.remainLen])
	}
}

func (c *TcpClient) Close() {
	INFO_LOG("tcp client close %v", c.conn.RemoteAddr())
	tcpServer.onClientClose(c)
}

func (c *TcpClient) Write(b []byte) (n int, err error){
	n, err = c.conn.Write(b)
	return
}

func (c *TcpClient) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *TcpClient) GetClientID() CONN_ID {
	return GetClientID(c.conn)
}

func (c *TcpClient) Turn2Session() *Session {
	return &Session{typ: SessionTcpClient, id: string(c.id), conn: c.conn}
}

func (c *TcpClient) HeartBeat(msg []byte) bool {
	if len(msg) != 0 {
		wLen, err := c.conn.Write(msg)
		if err != nil {
			ERROR_LOG("send heart beat write %v error %v", c.conn.RemoteAddr(), err)
			return false
		}

		if wLen != len(msg) {
			WARN_LOG("send heart beat write len(%v) != rsp msg len(%v) @%v", wLen, len(msg), c.conn.RemoteAddr())
		}
	}

	return true
}

var tcpServer *TcpServer = nil

func CreateTcpServer(_ip string, _port int) {
	tcpServer = &TcpServer{
		ip: _ip, 
		port: _port, 
		clients: make(map[CONN_ID]*TcpClient), 
		stop: false,
		lb: &LoadBalancer{elements: make(map[string]uint32)},
	}
}

func StartTcpServer() {
	go tcpServer.Start()
}

func StopTcpServer() {
	tcpServer.Stop()
}

func GetClient(connID CONN_ID) *TcpClient {
	tcpServer.mutex.RLock()
	defer tcpServer.mutex.RUnlock()

	client, ok := tcpServer.clients[connID]
	if !ok {
		return nil
	}

	return client
}
