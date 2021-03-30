package rpc

import (
    "log"
    "sync"
)

type RequestManager struct {
    mutex      sync.Mutex
    requestId  uint32
    requestMap map[uint32]func(*Message)
}

func (m *RequestManager) OnReply(msg *Message) {
    id := msg.requestId
    m.mutex.Lock()
    cb, ok := m.requestMap[id]
    m.mutex.Unlock()
    if !ok {
        log.Println("不存在", id)
        return
    }
    cb(msg)
}

func (c *RequestManager) NextRequestId(cb func(*Message)) uint32 {
    for {
        if c.requestId == 0 {
            c.requestId++
        }
        c.mutex.Lock()
        _, ok := c.requestMap[c.requestId]
        c.mutex.Unlock()
        if !ok {
            c.mutex.Lock()
            c.requestMap[c.requestId] = cb
            c.mutex.Unlock()
            return c.requestId
        }
        c.requestId++
    }
}
func newRequestManager() *RequestManager {
    return &RequestManager{
        requestId:  0,
        requestMap: map[uint32]func(*Message){},
    }
}
