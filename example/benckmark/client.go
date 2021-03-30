package main

import (
    "encoding/binary"
    "fmt"
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
    log.Println("open..")
    isConnected = true
}

func (h *clientHandler) OnMessage(message *rpc.Message) {

}

func (h *clientHandler) OnClose() {
    isConnected = false
}

var (
    isConnected = false
)

func runClient() {
    var qps uint32
    cli, _ := rpc.NewClient(&rpc.ClientOption{
        ReadTimeout:  time.Second * 10,
        WriteTimeout: time.Second * 10,
    })
    cli.AddPlugin(&clientHandler{})
    fmt.Println("start...")
    go func() {
        for {
            if isConnected {
                n := atomic.AddUint32(&qps, 1)
                data := make([]byte, 4)
                binary.BigEndian.PutUint32(data, n)
                err := cli.Request(data, func(message *rpc.Message) {
                    fmt.Println("收到回复:", message.Payload)
                })
                if err != nil {
                    log.Println("err", err)
                }
            } else {
                time.Sleep(time.Second)
            }
        }
    }()

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
    conn, _ := kcp.NewKCPDialer("127.0.0.1:12345", []byte("1234"), []byte("1234"))
    if err := cli.Serve(conn); err != nil {
        log.Println("初始化出错", err)
        return
    }

}
