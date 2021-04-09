package main

import (
    "github.com/DGHeroin/rpc.go/common"
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

func (h *serverHandler) OnMessage(id uint64, message *common.Message) {
    atomic.AddUint32(&qps, 1)
    err := message.Reply(message.Payload)
    if err != nil {
        log.Println(err)
    }
}

func (h *serverHandler) OnClose(id uint64) {}

func main() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)
    server, _ := rpc.NewServer(&rpc.ServerOption{
        ReadTimeout:  time.Second * 5,
        WriteTimeout: time.Second * 5,
    })
    server.AddPlugin(&serverHandler{})
    go func() {
        for {
            time.Sleep(time.Second)
            n := atomic.LoadUint32(&qps)
            atomic.StoreUint32(&qps, 0)
            if n != 0 {
                log.Println("qps:", n)
            }
        }
    }()
    ln, _ := kcp.NewKCPListenerServe("127.0.0.1:12345", []byte("1234"), []byte("1234"))
    if err := server.Serve(ln); err != nil {
        log.Println(err)
    }
}
