package main

import (
    "flag"
    "github.com/DGHeroin/rpc.go"
    log "github.com/sirupsen/logrus"
    "sync/atomic"
    "time"
)

var (
    count     uint64 = 0
    connCount int64  = 0
)
var (
    address = flag.String("addr", "127.0.0.1:12345", "listen address")
)

func main() {
    qps := uint32(0)
    count := int32(0)
    server, _ := rpc.NewServer()

    server.OnNew = func(id uint64) {
        //log.Println("新连接", id)
        time.AfterFunc(time.Second*2, func() {
            atomic.AddInt32(&count, 1)
            server.Send(id, 100, []byte("你好"))
        })
    }

    server.OnClose = func(id uint64) {
        //log.Println("关闭连接", id)
        atomic.AddInt32(&count, -1)
    }
    server.OnData = func(tag uint32, payload []byte) {
        atomic.AddUint32(&qps, 1)
        //log.Println("收到请求", tag, string(payload))
    }
    go func() {
        for {
            time.Sleep(time.Second)
            val := atomic.LoadUint32(&qps)
            atomic.StoreUint32(&qps, 0)
            log.Println("qps:", val, "count:", atomic.LoadInt32(&count))
        }
    }()
    server.Serve(*address, nil, nil)
}
