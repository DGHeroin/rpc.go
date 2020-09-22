package rpc

import (
	"crypto/sha1"
	"log"
	"net"
	"sync"
	"time"

	//log "github.com/sirupsen/logrus"

	"github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
)

type Client struct {
	address      string
	conn         net.Conn
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	//aesCipher    kcp.BlockCrypt
	password     []byte
	salt         []byte
	exitChan     chan bool
	OnConnected  func()
	OnClose      func()
	OnData       func(tag uint32, payload []byte)
	sendMutex sync.Mutex
}

func NewClient() (*Client, error) {

	cli := &Client{
		exitChan:     make(chan bool),
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}
	return cli, nil
}

func (c *Client) DialAndServe(address string, password []byte, salt []byte) error {
	var (
		conn net.Conn
		err error
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
			conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
			_, tag, payload, err := readMessage(conn)
			if err != nil {
				log.Println("client read error:", err)
				return err
			}
			lastSeen = time.Now()
			if payload == nil {
				continue
			}
			// on message
			c.OnData(tag, payload)
		}
	}
	return nil
}

func (c *Client) Close() {
	if c.exitChan != nil {
		c.exitChan <- true
	}
}

func (c *Client) Send(tag uint32, data []byte) (n int, err error) {
	c.sendMutex.Lock()
	defer c.sendMutex.Unlock()
	if c.conn == nil {
		return -1, ErrorConnectionInvalid
	}
	c.conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	return writeMessage(c.conn, MessageTypeData, tag, data)
}

func (c *Client) sendKeepAlive() {
	c.sendMutex.Lock()
	defer c.sendMutex.Unlock()
	if c.conn == nil {
		return
	}
	c.conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	writeMessage(c.conn, MessageTypePing, 0, nil)
}

func (c *Client) Reconnect(second time.Duration) {
	time.AfterFunc(second, func() {
		c.DialAndServe(c.address, c.password, c.salt)
	})
}
