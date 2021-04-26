package ms_framework

import (
	"net"
	"fmt"
	"time"
	"io"
)

type CLIENT_ID string  // RemoteAddr()
type CONN_ID CLIENT_ID

var RECV_BUF_MAX_LEN uint32 = 1024*1024  // 1M

type TcpServer struct {
	ip 			string
	port 		int
	listener 	net.Listener
	clients  	map[CLIENT_ID]*TcpClient
}

type TcpClient struct {
	id 				CLIENT_ID
	conn			net.Conn
	recvBuf 		[]byte
	remainLen 		uint32
	exit 			bool
	lastActiveTime  int64
}

func GetClientID(c net.Conn) CLIENT_ID {
	return CLIENT_ID(GetConnID(c))
}

func GetConnID(c net.Conn) CONN_ID {
	return CONN_ID(c.RemoteAddr().String())
}

func (s *TcpServer) Start() {
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.ip, s.port))
	if err != nil {
		panic(fmt.Sprintf("CreateTcpServer error %s:%d - %v", s.ip, s.port, err))
	}
	defer func() {
		s.Close()
	} ()

	INFO_LOG("tcp server listen at %s:%d", s.ip, s.port)

	s.listener = l

	for true {
		conn, err := s.listener.Accept()
		if err != nil {
			ERROR_LOG("tcp server accept error %v", err)
			continue
		}

		INFO_LOG("tcp server(%s:%d) accept client %v", s.ip, s.port, conn.RemoteAddr())

		cID := GetClientID(conn)
		s.clients[cID] = &TcpClient{
			id: cID, 
			conn: conn, 
			recvBuf: make([]byte, RECV_BUF_MAX_LEN),
			remainLen: 0,
			exit: false,
		}
		go s.clients[cID].HandleRead()
	}
}

func (s *TcpServer) Close(){
	INFO_LOG("tcp server(%s:%d) close...", s.ip, s.port)
	for _, client := range s.clients {
		client.exit = true
	}

	for len(s.clients) != 0 {}

	s.listener.Close()
}

func (s *TcpServer) onClientClose(c *TcpClient){
	delete(s.clients, GetClientID(c.conn))
}

func (c *TcpClient) HandleRead() {
	defer func() {
		INFO_LOG("tcp client close %v", c.conn.RemoteAddr())
		c.Close()
	} ()

	for !c.exit {
		c.conn.SetReadDeadline(time.Now().Add(100*time.Millisecond))
		rLen, err := c.conn.Read(c.recvBuf[c.remainLen:])
		if err != nil {
			e, ok := err.(*net.OpError)
			if ok && e.Timeout() == true {
				// WARN_LOG("read timeout %v", err)
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

		c.lastActiveTime = GetNowTimestamp()

		c.remainLen += uint32(rLen)
		if c.remainLen > RECV_BUF_MAX_LEN/2 {
			WARN_LOG("tcp connection buff cache too long!!! %dk > %dk", c.remainLen/1024, RECV_BUF_MAX_LEN/1024)
			
		} else if c.remainLen > RECV_BUF_MAX_LEN {
			ERROR_LOG("tcp connection buff cache overflow!!! %dk > %dk", c.remainLen/1024, RECV_BUF_MAX_LEN/1024)
			break
		}

		procLen := rpcMgr.MessageDecode(c.Turn2Session(), c.recvBuf[:c.remainLen])
		c.remainLen -= procLen
		if c.remainLen < 0 {
			ERROR_LOG("c.remainLen(%d) < 0 procLen(%d) @%s", c.remainLen, procLen, c.conn.RemoteAddr())
			c.remainLen = 0
			continue
		}

		copy(c.recvBuf, c.recvBuf[procLen: procLen + c.remainLen])
		// INFO_LOG("tcp recv buf %v", rLen)
	}
}

func (c *TcpClient) Close() {
	tcpServer.onClientClose(c)
}

func (c *TcpClient) Write(b []byte) (n int, err error){
	n, err = c.conn.Write(b)
	return
}

func (c *TcpClient) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *TcpClient) GetClientID() CLIENT_ID {
	return GetClientID(c.conn)
}

func (c *TcpClient) Turn2Session() *Session {
	return &Session{typ: SessionTcpClient, id: string(c.id), conn: c.conn}
}

// func (c *TcpClient) HeartBeat() {
	
// }

var tcpServer *TcpServer = nil

func CreateTcpServer(_ip string, _port int) {
	tcpServer = &TcpServer{ip: _ip, port: _port, clients: make(map[CLIENT_ID]*TcpClient)}
}

func TcpServerStart() {
	tcpServer.Start()
}

func GetClient(clientID CLIENT_ID) *TcpClient {
	// TODO: concurrent panic
	client, ok := tcpServer.clients[clientID]
	if !ok {
		ERROR_LOG("client %v not exist", clientID)
		return nil
	}

	return client
}
