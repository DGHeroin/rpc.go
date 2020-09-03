package rpc

import (
    "crypto/sha1"
    "github.com/xtaci/kcp-go"
    "golang.org/x/crypto/pbkdf2"
    "time"
)

type Client struct {
    address     string
    conn        *kcp.UDPSession
    lastSeen    time.Time
    aesCipher   kcp.BlockCrypt
    password    []byte
    salt        []byte
    OnConnected func()
    OnClose     func()
    OnData      func(tag uint32, payload []byte)
}

func NewClient() (*Client, error) {
    cli := &Client{}
    return cli, nil
}

func (c *Client) DialAndServe(address string, password []byte, salt []byte) error {
    if password != nil && salt != nil {
        key := pbkdf2.Key(password, salt, 1024, 32, sha1.New)
        block, _ := kcp.NewAESBlockCrypt(key)
        c.aesCipher = block
    }
    c.address = address
    c.password = password
    c.salt = salt
    conn, err := kcp.DialWithOptions(address, c.aesCipher, 10, 3)
    if err != nil {
        return err
    }
    c.conn = conn
    c.OnConnected()
    ticker := time.NewTicker(time.Second * 3)
    isRunning := true
    c.lastSeen = time.Now()
    defer func() {
        isRunning = false
        ticker.Stop()
        c.Close()
    }()

    go func() {
        for isRunning {
            <-ticker.C
            go c.sendKeepAlive()
            if time.Now().Sub(c.lastSeen) > time.Second*10 {
                isRunning = false
                c.Close()
                return
            }
        }
    }()

    for isRunning {
        msgType, tag, payload, err := readMessage(conn)
        if err != nil {
            return err
        }
        c.lastSeen = time.Now()
        if msgType == MessageTypeData {
            c.OnData(tag, payload)
        }
    }
    return nil
}

func (c *Client) Close() {
    c.OnClose()
    if c.conn != nil {
        _ = c.conn.Close()
    }
}

func (c *Client) Send(tag uint32, data []byte) (n int, err error) {
    if c.conn == nil {
        return -1, ErrorConnectionInvalid
    }
    return writeMessage(c.conn, MessageTypeData, tag, data)
}

func (c *Client) sendKeepAlive() error {
    if c.conn == nil {
        return ErrorConnectionInvalid
    }
    _, err := writeMessage(c.conn, MessageTypePing, 0, nil)
    return err
}

func (c *Client) Reconnect(second time.Duration) {
    time.AfterFunc(second, func() {
        _ = c.DialAndServe(c.address, c.password, c.salt)
    })
}
