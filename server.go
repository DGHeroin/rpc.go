package rpc

import (
	"crypto/sha1"
	"net"
	"sync"
	"time"

	"github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
)

type Server struct {
	address string
	mutex   sync.RWMutex
	//ln      net.Listener
	clientId  uint64
	sessions  map[uint64]*AcceptClient
	OnNew     func(id uint64)
	OnData    func(id uint64, tag uint32, payload []byte)
	OnClose   func(id uint64)
}
type AcceptClient struct {
	conn     net.Conn
	lastSeen time.Time
}

// 创建服务器
func NewServer() (*Server, error) {
	s := &Server{}
	s.sessions = make(map[uint64]*AcceptClient)
	return s, nil
}

func (s *Server) Serve(address string, password []byte, salt []byte) error {
	var (
		ln net.Listener
		err error
		withOptions = false
		kcpListener *kcp.Listener
	)

	if password != nil && salt != nil {
		key := pbkdf2.Key(password, salt, 1024, 32, sha1.New)
		block, err := kcp.NewAESBlockCrypt(key)
		if err != nil {
			return err
		}
		kcpListener, err = kcp.ListenWithOptions(address, block, 10, 3)
		ln = kcpListener
		withOptions = true
	} else {
		ln, err = kcp.Listen(address)
	}
	if err != nil {
		return err
	}
	//s.ln = ln
	s.address = address
	isRunning := true
	ticker := time.NewTicker(time.Second * 3)
	defer func() {
		ticker.Stop()
	}()
	// monitor
	go func() {
		for isRunning {
			<-ticker.C
			var deleteList []uint64
			s.mutex.RLock()
			for id, cli := range s.sessions {
				if time.Now().Sub(cli.lastSeen) > time.Second*10 {
					deleteList = append(deleteList, id)
				}
			}
			s.mutex.RUnlock()
			for _, id := range deleteList {
				s.removeClient(id)
			}
		}
	}()

	for isRunning {
		var (
			conn net.Conn
			err error
		)
		if withOptions {
			conn, err = kcpListener.AcceptKCP()
		} else {
			conn, err = ln.Accept()
		}
		if err != nil {
			isRunning = false
			return err
		}
		go s.handleConn(conn)
	}
	return nil
}
func (s *Server) addClient(conn net.Conn) (uint64, *AcceptClient) {
	s.mutex.Lock()
	var (
		id uint64
		ac *AcceptClient
	)
	for {
		id = s.clientId
		if _, ok := s.sessions[id]; !ok {
			ac = &AcceptClient{
				conn: conn,
				lastSeen: time.Now(),
			}
			s.sessions[id] = ac
			break
		}
		s.clientId++
	}
	s.mutex.Unlock()
	s.OnNew(id)
	return id, ac
}
func (s *Server) removeClient(id uint64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, ok := s.sessions[id]; ok {
		delete(s.sessions, id)
		s.OnClose(id)
	}
}
func (s *Server) handleConn(conn net.Conn) {
	id, cli := s.addClient(conn)
	defer s.removeClient(id)
	// on new conn
	for {
		_, tag, payload, err := readMessage(conn)
		if err != nil {
			//log.Println("server read message error", err)
			break
		}
		cli.lastSeen = time.Now()
		if payload == nil {
			continue
		}
		// on message
		s.OnData(id, tag, payload)
	}

}

func (s *Server) Send(id uint64, tag uint32, data []byte) (n int, err error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if cli, ok := s.sessions[id]; ok {
		return writeMessage(cli.conn, MessageTypeData, tag, data)
	}
	return 0, ErrorConnectIdInvalid
}
