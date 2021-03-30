package rpc

import (
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
        requestManager  *RequestManager
        option          ServerOption
        pluginContainer PluginContainer
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
        requestManager: newRequestManager(),
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
                if p, ok2 := i.(ServerOnAcceptPlugin); ok2 {
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
            if p, ok2 := i.(ServerOnClosePlugin); ok2 {
                p.OnClose(id)
            }
        })
    }
}
func (s *Server) handleConn(conn net.Conn) {
    var (
        err error
        msg *Message
        wg  sync.WaitGroup
    )
    id, sendCh := s.addClient()
    log.Printf("conn liv|%p", conn)
    defer func() {
        log.Printf("conn die|%p", conn)
        s.removeClient(id)
    }()
    wg.Add(1)
    go func() {
        defer func() {
            log.Println("write end")
            wg.Done()
            _ = conn.Close()
        }()
        for {
            data := <-sendCh
            if data == nil {
                return
            }
            if err = s.setWriteTimeout(conn); err != nil {
                return
            }
            log.Println("服务器send", data)
            _, err = conn.Write(data)
            if err != nil {
                return
            }

        }
    }()
    wg.Add(1)
    go func() {
        defer func() {
            log.Println("read end")
            wg.Done()
            sendCh <- nil
            close(sendCh)
        }()
        for {
            if err = s.setReadTimeout(conn); err != nil {
                log.Println(err)
                return
            }
            msg = NewMessage(sendCh)
            err = msg.Decode(conn)
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
    }()
    wg.Wait()
}
func (s *Server) handleMessage(id uint64, msg *Message) error {
    if msg == nil {
        return nil
    }
    switch msg.Type {
    case MessageTypeRequest, MessageTypeOneWay:
        // on message
        s.pluginContainer.Range(func(i interface{}) {
            if p, ok := i.(ServerOnMessagePlugin); ok {
                p.OnMessage(id, msg)
            }
        })
    case MessageTypeResponse:
        // on reply
        s.requestManager.OnReply(msg)
        return nil
    case MessageTypeKeep:
        return s.sendKeepAlive(id)
    default:
        log.Println(">>>", msg)
        return ErrorMessageTypeInvalid
    }
    return nil
}

func (s *Server) sendKeepAlive(id uint64) error {
    ch := s.getSendChannel(id)
    if ch == nil {
        return ErrorConnectionInvalid
    }
    msg := NewMessage(ch)
    msg.Type = MessageTypeKeep
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
func (s *Server) Request(id uint64, tag uint32, data []byte, cb func(*Message)) (n int, err error) {
    ch := s.getSendChannel(id)
    if ch == nil {
        return 0, ErrorConnectionInvalid
    }

    msg := NewMessage(ch)
    msg.Type = MessageTypeRequest
    msg.requestId = s.requestManager.NextRequestId(cb)
    msg.Payload = data
    msg.Emit()

    return 0, nil
}

func (s *Server) Push(id uint64, tag uint32, data []byte) (n int, err error) {
    ch := s.getSendChannel(id)
    if ch == nil {
        return 0, ErrorConnectionInvalid
    }
    msg := NewMessage(ch)
    msg.Type = MessageTypeOneWay
    msg.requestId = 0
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
