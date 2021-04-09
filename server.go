package rpc

import (
    "bufio"
    "github.com/DGHeroin/rpc.go/common"
    "log"
    "net"
    "sync"
    "time"
)

type (
    Server struct {
        address         string
        mutex           sync.RWMutex
        clientId        uint64
        sessions        map[uint64]chan []byte
        requestManager  *common.RequestManager
        option          ServerOption
        pluginContainer common.PluginContainer
        exitChan        chan bool
    }
    ServerOption struct {
        ReadTimeout  time.Duration
        WriteTimeout time.Duration
    }
)

// 创建服务器
func NewServer(opt *ServerOption) (*Server, error) {
    if opt == nil {
        opt = defaultServerOption()
    }
    s := &Server{
        option:         *opt,
        requestManager: common.NewRequestManager(),
        exitChan:       make(chan bool),
    }
    s.sessions = make(map[uint64]chan []byte)
    return s, nil
}

func defaultServerOption() *ServerOption {
    return &ServerOption{
    }
}
func (s *Server) AddPlugin(p interface{}) {
    s.pluginContainer.Add(p)
}
func (s *Server) RemovePlugin(p interface{}) {
    s.pluginContainer.Remove(p)
}
func (s *Server) Serve(ln net.Listener) error {
    for {
        conn, err := ln.Accept()
        if err != nil {
            return err
        }
        go s.handleConn(conn)

    }
}
func (s *Server) addClient() (uint64, chan []byte) {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    var id uint64
    for {
        id = s.clientId
        if _, ok := s.sessions[id]; !ok {
            ch := make(chan []byte)
            s.sessions[id] = ch
            s.pluginContainer.Range(func(i interface{}) {
                if p, ok2 := i.(common.ServerOnAcceptPlugin); ok2 {
                    p.OnAccept(id)
                }
            })
            return id, ch
        }
        s.clientId++
    }
}
func (s *Server) removeClient(id uint64) {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    if _, ok := s.sessions[id]; ok {
        delete(s.sessions, id)
        s.pluginContainer.Range(func(i interface{}) {
            if p, ok2 := i.(common.ServerOnClosePlugin); ok2 {
                p.OnClose(id)
            }
        })
    }
}
func (s *Server) handleConn(conn net.Conn) {
    var (
        err error
        msg *common.Message
        wg  sync.WaitGroup
        r *bufio.Reader
    )
    r = bufio.NewReaderSize(conn, 16*1024)
    exitCh := make(chan bool)
    id, sendCh := s.addClient()
    defer func() {
        s.removeClient(id)
    }()
    wg.Add(1)
    go func() {
        defer func() {
            wg.Done()
            _ = conn.Close()
        }()
        for {
            select {
            case <-exitCh:
                return
            case data := <-sendCh:
                if data == nil {
                    return
                }
                if err = s.setWriteTimeout(conn); err != nil {
                    return
                }
                _, err = conn.Write(data)
                if err != nil {
                    return
                }
            }
        }
    }()
    wg.Add(1)
    go func() {
        defer func() {
            wg.Done()
            sendCh <- nil
            _ = conn.Close()
            close(sendCh)
            close(exitCh)
        }()
        for {
            select {
            case <-exitCh:
                return
            default:
                if err = s.setReadTimeout(conn); err != nil {
                    log.Println(err)
                    return
                }
                msg = common.NewMessage(sendCh)
                err = msg.Decode(r)
                if err != nil {
                    log.Println(err)
                    return
                }
                err = s.handleMessage(id, msg)
                if err != nil {
                    log.Println(err)
                    return
                }
            }
        }
    }()
    wg.Wait()
}
func (s *Server) handleMessage(id uint64, msg *common.Message) error {
    if msg == nil {
        return nil
    }
    switch msg.Type {
    case common.MessageTypeRequest, common.MessageTypeOneWay:
        // on message
        s.pluginContainer.Range(func(i interface{}) {
            if p, ok := i.(common.ServerOnMessagePlugin); ok {
                p.OnMessage(id, msg)
            }
        })
    case common.MessageTypeResponse:
        // on reply
        s.requestManager.OnReply(msg)
        return nil
    case common.MessageTypeKeep:
        return s.sendKeepAlive(id)
    default:
        return common.ErrorMessageTypeInvalid
    }
    return nil
}

func (s *Server) sendKeepAlive(id uint64) error {
    ch := s.getSendChannel(id)
    if ch == nil {
        return common.ErrorConnectionInvalid
    }
    msg := common.NewMessage(ch)
    msg.Type = common.MessageTypeKeep
    msg.Emit()
    return nil
}
func (s *Server) getSendChannel(id uint64) chan []byte {
    s.mutex.Lock()
    ch, ok := s.sessions[id]
    s.mutex.Unlock()
    if ok {
        return ch
    }
    return nil
}
func (s *Server) Request(id uint64, tag uint32, data []byte, cb func(*common.Message)) (n int, err error) {
    ch := s.getSendChannel(id)
    if ch == nil {
        return 0, common.ErrorConnectionInvalid
    }

    msg := common.NewMessage(ch)
    msg.Type = common.MessageTypeRequest
    msg.RequestId = s.requestManager.NextRequestId(cb)
    msg.Payload = data
    msg.Emit()

    return 0, nil
}

func (s *Server) Push(id uint64, tag uint32, data []byte) (n int, err error) {
    ch := s.getSendChannel(id)
    if ch == nil {
        return 0, common.ErrorConnectionInvalid
    }
    msg := common.NewMessage(ch)
    msg.Type = common.MessageTypeOneWay
    msg.RequestId = 0
    msg.Payload = data

    msg.Emit()
    return 0, nil
}
func (s *Server) setReadTimeout(conn net.Conn) error {
    if s.option.ReadTimeout == 0 {
        return nil
    }
    return conn.SetReadDeadline(time.Now().Add(s.option.ReadTimeout))
}
func (s *Server) setWriteTimeout(conn net.Conn) error {
    if s.option.WriteTimeout == 0 {
        return nil
    }
    return conn.SetWriteDeadline(time.Now().Add(s.option.WriteTimeout))
}
