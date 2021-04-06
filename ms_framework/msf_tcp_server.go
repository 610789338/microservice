package ms_framework

import (
	"net"
	"fmt"
	// "time"
	"io"
)

type CLIENT_ID string

// type NetClient interface {
// 	handleRead()
// 	Close()
// }

var RECV_BUF_MAX_LEN uint32 = 1024*1024  // 1M

type TcpServer struct{
	ip 			string
	port 		int
	listener 	net.Listener
	clients  	map[CLIENT_ID]*TcpClient
}

type TcpClient struct{
	id 				CLIENT_ID
	conn			net.Conn
	server 			*TcpServer
	recvBuf 		[]byte
	remainLen 		uint32
	exit 			bool
}

func GetClientID(c net.Conn) CLIENT_ID {
	return CLIENT_ID(c.RemoteAddr().String())
}

func (s *TcpServer) Start(){
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
			server: s,
			recvBuf: make([]byte, RECV_BUF_MAX_LEN),
			remainLen: 0,
			exit: false,
		}
		go s.clients[cID].handleRead()
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

func (c *TcpClient) handleRead() {
	defer func() {
		INFO_LOG("tcp client close %v", c.conn.RemoteAddr())
		c.Close()
	} ()

	for !c.exit {
		// c.conn.SetReadDeadline(time.Now().Add(100*time.Millisecond))
		len, err := c.conn.Read(c.recvBuf[c.remainLen:])
		if err != nil && err != io.EOF {
			ERROR_LOG("read error %v", err)
			c.exit = true
			continue
		}

		if 0 == len {
			// remote close
			INFO_LOG("tcp connection close by remote %v %v", c.conn.RemoteAddr(), err)
			c.exit = true
			continue
		}

		c.remainLen += uint32(len)
		if c.remainLen > RECV_BUF_MAX_LEN/2 {
			WARN_LOG("tcp connection buff cache too long!!! %dk > %dk", c.remainLen/1024, RECV_BUF_MAX_LEN/1024)
			
		} else if c.remainLen > RECV_BUF_MAX_LEN {
			ERROR_LOG("tcp connection buff cache overflow!!! %dk > %dk", c.remainLen/1024, RECV_BUF_MAX_LEN/1024)
			c.exit = true
			continue
		}

		dlen := rpcMgr.RpcParse(c.recvBuf[:c.remainLen])
		c.remainLen -= dlen
		if c.remainLen < 0 {
			ERROR_LOG("c.remainLen(%d) < 0 dlen(%d)", c.remainLen, dlen)
		}

		copy(c.recvBuf, c.recvBuf[dlen:dlen+c.remainLen])
	}
}

func (c *TcpClient) Close() {
	c.server.onClientClose(c)
}

func CreateTcpServer(ip string, port int) *TcpServer {
	tcpServer := TcpServer{ip: "127.0.0.1", port: 6666, clients: make(map[CLIENT_ID]*TcpClient)}

	return &tcpServer
}
