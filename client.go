package rpc

import (
    "bufio"
    "github.com/DGHeroin/rpc.go/common"
    "net"
    "sync"
    "time"
)

type (
    Client struct {
        conn            net.Conn
        option          ClientOption
        requestManager  *common.RequestManager
        pluginContainer common.PluginContainer
        listM           sync.Mutex
        sendList        []*common.Message
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
        requestManager: common.NewRequestManager(),
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

    var (
        wg sync.WaitGroup
        r  *bufio.Reader
    )
    r = bufio.NewReaderSize(conn, 16*1024)
    defer func() {
        _ = c.sendClose()
        _ = conn.Close()
        c.pluginContainer.Range(func(i interface{}) {
            if p, ok := i.(common.ClientOnClosePlugin); ok {
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
            var msg = common.NewMessage(c.sendCh)
            if err := msg.Decode(r); err != nil {
                return
            }
            if err != nil {
                return
            }
            c.openOnce.Do(func() {
                c.pluginContainer.Range(func(i interface{}) {
                    if p, ok := i.(common.ClientOnOpenPlugin); ok {
                        p.OnOpen()
                    }
                })
            })
            switch msg.Type {
            case common.MessageTypeRequest, common.MessageTypeOneWay:
                // on message
                c.pluginContainer.Range(func(i interface{}) {
                    if p, ok := i.(common.ClientOnMessagePlugin); ok {
                        p.OnMessage(msg)
                    }
                })
            case common.MessageTypeResponse:
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

func (c *Client) Request(data []byte, cb func(*common.Message)) (err error) {
    msg := common.NewMessage(c.sendCh)
    msg.Type = common.MessageTypeRequest
    msg.RequestId = c.requestManager.NextRequestId(cb)
    msg.Payload = data
    return c.postMessage(msg)
}

func (c *Client) Push(data []byte) (err error) {
    msg := common.NewMessage(c.sendCh)
    msg.Type = common.MessageTypeOneWay
    msg.RequestId = 0
    msg.Payload = data
    return c.postMessage(msg)
}

func (c *Client) sendKeepAlive() error {
    msg := common.NewMessage(c.sendCh)
    msg.Type = common.MessageTypeKeep
    return c.postMessage(msg)
}
func (c *Client) sendClose() error {
    msg := common.NewMessage(c.sendCh)
    msg.Type = common.MessageTypeClose
    return c.postMessage(msg)
}

func (c *Client) Reconnect(second time.Duration) {
    time.AfterFunc(second, func() {
        //c.Serve(c.address, c.password, c.salt)
    })
}
func (c *Client) postMessage(msg *common.Message) error {
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
