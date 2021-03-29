package main

import (
    "github.com/DGHeroin/rpc.go/kcp"
    "log"
    "sync/atomic"
    "time"

    "github.com/DGHeroin/rpc.go"
)

func main() {
    for i := 0; i < 1; i++ {
        go runClient()
    }
    select {}
}

type clientHandler struct {

}

func (h *clientHandler) OnOpen() {
    isConnected = true
}

func (h *clientHandler) OnMessage( message *rpc.Message) {

}

func (h *clientHandler) OnClose() {

}
var (
    isConnected = false
)

func runClient() {
    var qps uint32
    cli, _ := rpc.NewClient( nil)
    cli.AddPlugin(&clientHandler{})
    go func() {
        for {
            if isConnected {
                atomic.AddUint32(&qps, 1)
                cli.Request(123, []byte("hello world!"), func(message *rpc.Message) {
                    //fmt.Println("收到回复:", message.Tag, string(message.Payload))
                })
            } else {
                time.Sleep(time.Second)
            }
        }
    }()

    go func() {
        for {
            if isConnected {
                atomic.AddUint32(&qps, 1)
                cli.Request(123, []byte("hello world!"), func(message *rpc.Message) {
                    //fmt.Println("收到回复:", message.Tag, string(message.Payload))
                })
            } else {
                time.Sleep(time.Second)
            }
        }
    }()

    go func() {
        for {
            time.Sleep(time.Second)
            n:=atomic.LoadUint32(&qps)
            atomic.StoreUint32(&qps, 0)
            log.Println("qps:", n)
        }
    }()
    conn, _ := kcp.NewKCPDialer("127.0.0.1:12345", nil, nil)
    if err := cli.DialAndServe(conn); err != nil {
        log.Println("初始化出错", err)
        return
    }

}
