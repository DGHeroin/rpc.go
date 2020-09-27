package rpc

import (
    "crypto/sha1"
    "net"
    "sync"
    "time"

    "github.com/xtaci/kcp-go"
    "golang.org/x/crypto/pbkdf2"
)

type Client struct {
    address      string
    conn         net.Conn
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
    password     []byte
    salt         []byte
    exitChan     chan bool
    OnConnected  func()
    OnClose      func()
    OnData       func(message *Message)
    sendMutex    sync.Mutex
    requestMutex sync.Mutex
    requestId    uint32
    requestMap   map[uint32]func(*Message)
}

func NewClient() (*Client, error) {
    cli := &Client{
        exitChan:     make(chan bool),
        ReadTimeout:  time.Second * 10,
        WriteTimeout: time.Second * 10,
        requestMap:   make(map[uint32]func(*Message)),
    }
    return cli, nil
}

func (c *Client) DialAndServe(address string, password []byte, salt []byte) error {
    var (
        conn net.Conn
        err  error
    )
    if password != nil && salt != nil {
        key := pbkdf2.Key(password, salt, 1024, 32, sha1.New)
        block, err := kcp.NewAESBlockCrypt(key)
        if err != nil {
            return err
        }
        conn, err = kcp.DialWithOptions(address, block, 10, 3)
    } else {
        conn, err = kcp.Dial(address)
    }
    if err != nil {
        return err
    }
    c.address = address
    c.password = password
    c.salt = salt
    c.conn = conn

    c.OnConnected()
    ticker := time.NewTicker(time.Second * 3)
    isRunning := true
    lastSeen := time.Now()
    defer func() {
        isRunning = false
        ticker.Stop()
        c.exitChan = nil
        c.OnClose()
        conn.Close()
    }()

    // monitor
    go func() {
        for isRunning {
            select {
            case <-ticker.C:
                c.sendKeepAlive()
                if isRunning == false {
                    return
                }
                if time.Now().Sub(lastSeen) > time.Second*10 {
                    isRunning = false
                    c.exitChan <- true
                    c.Close()
                    return
                }
            }
        }
    }()

    for isRunning {
        select {
        case <-c.exitChan:
            return nil
        default:
            _ = conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
            msg, err := readMessage(conn)
            if err != nil {
                return err
            }
            lastSeen = time.Now()
            if msg != nil {
                switch msg.Type {
                case MessageTypeData:
                    // on message
                    c.OnData(msg)
                case MessageTypeReply:
                    // on reply
                    c.OnReply(msg)
                }
            }
        }
    }
    return nil
}
func (c *Client)OnReply(msg *Message)  {
    c.requestMutex.Lock()
    defer c.requestMutex.Unlock()
    cb, ok := c.requestMap[msg.requestId]
    if ok {
        delete(c.requestMap, msg.requestId)
    }

    if ok {
        cb(msg)
    }
}
func (c *Client) Close() {
    if c.exitChan != nil {
        c.exitChan <- true
    }
}

func (c *Client) Send(tag uint32, data []byte, cb func(*Message)) (err error) {
    c.sendMutex.Lock()
    defer c.sendMutex.Unlock()
    if c.conn == nil {
        return ErrorConnectionInvalid
    }
    c.conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
    go writeMessage(c.conn, &Message{
        Type:      MessageTypeData,
        requestId: c.NextRequestId(cb),
        Tag:       tag,
        Payload:   data,
    })
    return nil
}

func (c *Client) sendKeepAlive() {
    c.sendMutex.Lock()
    defer c.sendMutex.Unlock()
    if c.conn == nil {
        return
    }
    c.conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
    go writeMessage(c.conn, &Message{Type: MessageTypeKeep})
}

func (c *Client) Reconnect(second time.Duration) {
    time.AfterFunc(second, func() {
        c.DialAndServe(c.address, c.password, c.salt)
    })
}

func (c *Client) NextRequestId(cb func(*Message)) uint32 {
    if cb == nil {
        return 0
    }
    for {
        if c.requestId == 0 {
            c.requestId++
        }
        c.requestMutex.Lock()
        _, ok := c.requestMap[c.requestId]
        c.requestMutex.Unlock()
        if !ok {
            c.requestMap[c.requestId] = cb
            return c.requestId
        }
        c.requestId++
    }
}
