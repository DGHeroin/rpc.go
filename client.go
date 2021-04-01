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
        requestManager  *RequestManager
        pluginContainer PluginContainer
        listM           sync.Mutex
        sendList        []*Message
        lastFlushSend   time.Time
        openOnce        *sync.Once
        sendCh          chan []byte
    }
    ClientOption struct {
        ReadTimeout  time.Duration
        WriteTimeout time.Duration
    }
)

func NewClient(opt *ClientOption) (*Client, error) {
    if opt == nil {
        opt = defaultClientOption()
    }
    cli := &Client{
        option:         *opt,
        requestManager: newRequestManager(),
        openOnce:       &sync.Once{},
        sendCh:         make(chan []byte, 10),
    }
    return cli, nil
}

func defaultClientOption() *ClientOption {
    return &ClientOption{}
}
func (c *Client) AddPlugin(p interface{}) {
    c.pluginContainer.Add(p)
}
func (c *Client) RemovePlugin(p interface{}) {
    c.pluginContainer.Remove(p)
}
func (c *Client) Serve(conn net.Conn) error {
    c.conn = conn
    var wg sync.WaitGroup
    defer func() {
        _ = c.sendClose()
        _ = conn.Close()
        c.pluginContainer.Range(func(i interface{}) {
            if p, ok := i.(ClientOnClosePlugin); ok {
                p.OnClose()
            }
        })
    }()
    // send ping
    err := c.sendKeepAlive()
    if err != nil {
        return err
    }
    wg.Add(1)
    go func() {
        defer wg.Done()
        for {
            select {
            case data := <-c.sendCh:
                _, err = conn.Write(data)
                if err != nil {
                    return
                }
            }
        }
    }()
    wg.Add(1)
    go func() {
        defer wg.Done()
        for {
            if err := c.setReadTimeout(); err != nil {
                return
            }
            var msg = NewMessage(c.sendCh)
            if err := msg.Decode(conn); err != nil {
                return
            }
            if err != nil {
                return
            }
            c.openOnce.Do(func() {
                c.pluginContainer.Range(func(i interface{}) {
                    if p, ok := i.(ClientOnOpenPlugin); ok {
                        p.OnOpen()
                    }
                })
            })
            switch msg.Type {
            case MessageTypeRequest, MessageTypeOneWay:
                // on message
                c.pluginContainer.Range(func(i interface{}) {
                    if p, ok := i.(ClientOnMessagePlugin); ok {
                        p.OnMessage(msg)
                    }
                })
            case MessageTypeResponse:
                // on reply
                c.requestManager.OnReply(msg)
            }
        }
    }()
    wg.Wait()
    return nil
}

func (c *Client) Close() {
    _ = c.conn.Close()
}

func (c *Client) Request(data []byte, cb func(*Message)) (err error) {
    msg := NewMessage(c.sendCh)
    msg.Type = MessageTypeRequest
    msg.requestId = c.requestManager.NextRequestId(cb)
    msg.Payload = data
    return c.postMessage(msg)
}

func (c *Client) Push(data []byte) (err error) {
    msg := NewMessage(c.sendCh)
    msg.Type = MessageTypeOneWay
    msg.requestId = 0
    msg.Payload = data
    return c.postMessage(msg)
}

func (c *Client) sendKeepAlive() error {
    msg := NewMessage(c.sendCh)
    msg.Type = MessageTypeKeep
    return c.postMessage(msg)
}
func (c *Client) sendClose() error {
    msg := NewMessage(c.sendCh)
    msg.Type = MessageTypeClose
    return c.postMessage(msg)
}

func (c *Client) Reconnect(second time.Duration) {
    time.AfterFunc(second, func() {
        //c.Serve(c.address, c.password, c.salt)
    })
}
func (c *Client) postMessage(msg *Message) error {
    if err := c.setWriteTimeout(); err != nil {
        return err
    }
    msg.Emit()
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
