package main

import (
    "log"
    "sync/atomic"
    "time"

    "github.com/DGHeroin/rpc.go"
)

var (
    count     uint64 = 0
    connCount int64  = 0
)

func main() {
    qps := uint32(0)
    count := int32(0)
    server, _ := rpc.NewServer()

    server.OnNew = func(id uint64) {
        //log.Println("新连接", id)
        time.AfterFunc(time.Second*2, func() {
            atomic.AddInt32(&count, 1)
            server.Send(id, 100, []byte("你好"), nil)
        })
    }

    server.OnClose = func(id uint64) {
        //log.Println("关闭连接", id)
        atomic.AddInt32(&count, -1)
    }
    server.OnData = func(id uint64, message *rpc.Message) {
        atomic.AddUint32(&qps, 1)
        log.Println("收到请求", message.Tag)

        if message.Tag == 123 {
            message.Reply(77, []byte("收到了"))
            server.Send(id, 66, []byte("一百年只能上一次陆地"), nil)
        }
    }
    if err := server.Serve("127.0.0.1:12345", nil, nil); err != nil {
        log.Println(err)
    }
}
