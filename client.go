package rpc

import (
    "net"
    "sync"
    "time"
)

type (
    Client struct {
        conn            net.Conn
        option          ClientOption
        exitChan        chan bool
        requestManager  *RequestManager
        pluginContainer PluginContainer
        listM           sync.Mutex
        sendList        []*Message
        lastFlushSend   time.Time
    }
    ClientOption struct {
        ReadTimeout   time.Duration
        WriteTimeout  time.Duration
        FlushDuration time.Duration
    }
)

func NewClient(opt *ClientOption) (*Client, error) {
    if opt == nil {
        opt = defaultClientOption()
    }
    cli := &Client{
        exitChan:       make(chan bool),
        option:         *opt,
        requestManager: newRequestManager(),
    }
    return cli, nil
}

func defaultClientOption() *ClientOption {
    return &ClientOption{
        //FlushDuration: time.Millisecond * 100,
    }
}
func (s *Client) AddPlugin(p interface{}) {
    s.pluginContainer.Add(p)
}
func (s *Client) RemovePlugin(p interface{}) {
    s.pluginContainer.Remove(p)
}
func (c *Client) DialAndServe(conn net.Conn) error {
    c.conn = conn
    c.pluginContainer.Range(func(i interface{}) {
        if p, ok := i.(ClientOnOpenPlugin); ok {
            p.OnOpen()
        }
    })
    defer func() {
        go close(c.exitChan)
        c.pluginContainer.Range(func(i interface{}) {
            if p, ok := i.(ClientOnClosePlugin); ok {
                p.OnClose()
            }
        })
        _ = conn.Close()
    }()

    for {
        select {
        case <-c.exitChan:
            return nil
        default:
            if err := c.setReadTimeout(); err != nil {
                return err
            }
            msg, err := readMessage(conn)
            if err != nil {
                return err
            }
            if msg != nil {
                switch msg.Type {
                case MessageTypeRequest, MessageTypeOneWay:
                    // on message
                    c.pluginContainer.Range(func(i interface{}) {
                        if p, ok := i.(ClientOnMessagePlugin); ok {
                            p.OnMessage(msg)
                        }
                    })
                    FreeMessage(msg)
                case MessageTypeResponse:
                    // on reply
                    c.OnReply(msg)
                    FreeMessage(msg)
                }
            }
        }
    }
}
func (c *Client) OnReply(msg *Message) {
    c.requestManager.OnReply(msg)

}
func (c *Client) Close() {
    close(c.exitChan)
}

func (c *Client) Request(tag uint32, data []byte, cb func(*Message)) (err error) {
    if c.conn == nil {
        return ErrorConnectionInvalid
    }

    msg := NewMessage()
    msg.Type = MessageTypeRequest
    msg.requestId = c.requestManager.NextRequestId(cb)
    msg.Tag = tag
    msg.Payload = data
    return c.postMessage(msg)
}

func (c *Client) flushSendList() error {
    c.listM.Lock()

    c.lastFlushSend = time.Now()
    oldList := c.sendList
    c.sendList = []*Message{}
    c.listM.Unlock()

    if err := c.setWriteTimeout(); err != nil {
        return err
    }
    if _, err := writeMessage(c.conn, oldList...); err != nil {
        return err
    }
    return nil
}

func (c *Client) Push(tag uint32, data []byte, cb func(*Message)) (err error) {
    if c.conn == nil {
        return ErrorConnectionInvalid
    }
    msg := NewMessage()
    msg.Type = MessageTypeOneWay
    msg.requestId = 0
    msg.Tag = tag
    msg.Payload = data
    return c.postMessage(msg)
}

func (c *Client) sendKeepAlive() error {
    if c.conn == nil {
        return nil
    }

    msg := NewMessage()
    msg.Type = MessageTypeKeep
    return c.postMessage(msg)
}

func (c *Client) Reconnect(second time.Duration) {
    time.AfterFunc(second, func() {
        //c.DialAndServe(c.address, c.password, c.salt)
    })
}
func (c *Client) postMessage(msg *Message) error {
    if c.option.FlushDuration == 0 {
        if err := c.setWriteTimeout(); err != nil {
            return err
        }
        _, err := writeMessage(c.conn, msg)
        return err
    }

    c.listM.Lock()
    c.sendList = append(c.sendList, msg)
    c.listM.Unlock()
    if time.Now().Sub(c.lastFlushSend) >= c.option.FlushDuration {
        return c.flushSendList()
    }
    return nil

}
func (c *Client) setReadTimeout() error {
    if c.option.ReadTimeout == 0 {
        return nil
    }
    return c.conn.SetReadDeadline(time.Now().Add(c.option.ReadTimeout))
}
func (c *Client) setWriteTimeout() error {
    if c.option.WriteTimeout == 0 {
        return nil
    }
    return c.conn.SetWriteDeadline(time.Now().Add(c.option.WriteTimeout))
}
