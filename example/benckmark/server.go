package main

import (
    "github.com/DGHeroin/rpc.go/kcp"
    "log"
    "sync/atomic"
    "time"

    "github.com/DGHeroin/rpc.go"
)
var (
    qps uint32
)
type serverHandler struct {

}

func (h *serverHandler) OnAccept(id uint64) {

}

func (h *serverHandler) OnMessage(id uint64, message *rpc.Message) {
    atomic.AddUint32(&qps, 1)
    message.Reply(133, []byte("hello world"))
}

func (h *serverHandler) OnClose(id uint64) {
}

func main() {
    server, _ := rpc.NewServer( nil)
    server.AddPlugin(&serverHandler{})
    go func() {
        for {
            time.Sleep(time.Second)
            n:=atomic.LoadUint32(&qps)
            atomic.StoreUint32(&qps, 0)
            log.Println("qps:", n)
        }
    }()
    ln, _ := kcp.NewKCPListenerServe("127.0.0.1:12345", nil, nil)
    if err := server.Serve(ln); err != nil {
        log.Println(err)
    }
}
