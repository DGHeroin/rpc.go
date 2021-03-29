package rpc

import (
    "net"
    "sync"
    "time"
)

type (
    Server struct {
        address         string
        mutex           sync.RWMutex
        clientId        uint64
        sessions        map[uint64]*AcceptClient
        requestManager  *RequestManager
        option          ServerOption
        pluginContainer PluginContainer
    }
    AcceptClient struct {
        conn net.Conn
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
    }
    s.sessions = make(map[uint64]*AcceptClient)
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
        var (
            conn net.Conn
            err  error
        )

        conn, err = ln.Accept()

        if err != nil {
            return err
        }
        go s.handleConn(conn)
    }
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
            }
            s.sessions[id] = ac
            break
        }
        s.clientId++
    }
    s.mutex.Unlock()
    s.pluginContainer.Range(func(i interface{}) {
        if p, ok := i.(ServerOnAcceptPlugin); ok {
            p.OnAccept(id)
        }
    })
    return id, ac
}
func (s *Server) removeClient(id uint64) {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    if _, ok := s.sessions[id]; ok {
        delete(s.sessions, id)
        s.pluginContainer.Range(func(i interface{}) {
            if p, ok := i.(ServerOnClosePlugin); ok {
                p.OnClose(id)
            }
        })
    }
}
func (s *Server) handleConn(conn net.Conn) {
    id, _ := s.addClient(conn)
    defer s.removeClient(id)
    // on new conn
    for {
        if s.option.ReadTimeout != 0 {
            if err := conn.SetReadDeadline(time.Now().Add(s.option.ReadTimeout)); err != nil {
                break
            }
        }
        msg, err := readMessage(conn)
        if err != nil {
            break
        }
        if msg != nil {
            switch msg.Type {
            case MessageTypeRequest, MessageTypeOneWay:
                // on message
                s.pluginContainer.Range(func(i interface{}) {
                    if p, ok := i.(ServerOnMessagePlugin); ok {
                        p.OnMessage(id, msg)
                    }
                })
                FreeMessage(msg)
            case MessageTypeResponse:
                // on reply
                s.requestManager.OnReply(msg)
                FreeMessage(msg)
            }
        }
    }

}

func (s *Server) Request(id uint64, tag uint32, data []byte, cb func(*Message)) (n int, err error) {
    s.mutex.RLock()
    cli, ok := s.sessions[id]
    s.mutex.RUnlock()
    if !ok {
        return 0, nil
    }
    if s.option.WriteTimeout!=0{
        if err = cli.conn.SetWriteDeadline(time.Now().Add(s.option.WriteTimeout)); err != nil {
            return -1, err
        }
    }

    msg := NewMessage()
    msg.Type = MessageTypeRequest
    msg.requestId = s.requestManager.NextRequestId(cb)
    msg.Payload = data
    msg.Tag = tag
    if _, err = writeMessage(cli.conn, msg); err != nil {
        s.removeClient(id)
    }

    return 0, nil
}

func (s *Server) Push(id uint64, tag uint32, data []byte, cb func(*Message)) (n int, err error) {
    s.mutex.RLock()
    cli, ok := s.sessions[id]
    s.mutex.RUnlock()
    if !ok {
        return 0, nil
    }
    if s.option.WriteTimeout!=0{
        if err = cli.conn.SetWriteDeadline(time.Now().Add(s.option.WriteTimeout)); err != nil {
            return -1, err
        }
    }

    msg := NewMessage()
    msg.Type = MessageTypeOneWay
    msg.requestId = 0
    msg.Payload = data
    msg.Tag = tag
    if _, err = writeMessage(cli.conn, msg); err != nil {
        s.removeClient(id)
    }
    return 0, nil
}
func (c *Server) setReadTimeout(conn net.Conn) error {
    if c.option.ReadTimeout != 0 {
        return nil
    }
    return conn.SetReadDeadline(time.Now().Add(c.option.ReadTimeout))
}
func (c *Server) setWriteTimeout(conn net.Conn) error {
    if c.option.WriteTimeout != 0 {
        return nil
    }
    return conn.SetWriteDeadline(time.Now().Add(c.option.WriteTimeout))
}
