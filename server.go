package rpc

import (
    "crypto/sha1"
    "github.com/xtaci/kcp-go"
    "golang.org/x/crypto/pbkdf2"
    "sync"
    "time"
)

type Server struct {
    address string
    mutex   sync.RWMutex
    ln      *kcp.Listener
    aesCipher kcp.BlockCrypt
    clientId  uint64
    sessions  map[uint64]*AcceptClient
    OnNew     func(id uint64)
    OnData    func(tag uint32, payload []byte)
    OnClose   func(id uint64)
}
type AcceptClient struct {
    conn     *kcp.UDPSession
    lastSeen time.Time
}

// 创建服务器
func NewServer() (*Server, error) {
    s := &Server{}
    s.sessions = make(map[uint64]*AcceptClient)
    return s, nil
}

func (s *Server) Serve(address string, password []byte, salt []byte) error {
    if password != nil && salt != nil {
        key := pbkdf2.Key(password, salt, 1024, 32, sha1.New)
        block, _ := kcp.NewAESBlockCrypt(key)
        s.aesCipher = block
    }
    s.address = address
    ln, err := kcp.ListenWithOptions(s.address, s.aesCipher, 10, 3)
    if err != nil {
        return err
    }
    s.ln = ln

    isRunning := true
    ticker := time.NewTicker(time.Second * 3)
    defer ticker.Stop()
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
        conn, err := s.ln.AcceptKCP()
        if err != nil {
            isRunning = false
            return err
        }
        go s.handleConn(conn)
    }
    return nil
}
func (s *Server) addClient(conn *kcp.UDPSession) (uint64, *AcceptClient) {
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
func (s *Server) handleConn(conn *kcp.UDPSession) {
    id, cli := s.addClient(conn) // on new conn
    defer s.removeClient(id) // on close conn

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
        s.OnData(tag, payload)
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
