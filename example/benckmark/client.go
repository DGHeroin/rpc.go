package main

import (
    "encoding/binary"
    "github.com/DGHeroin/rpc.go/common"
    "github.com/DGHeroin/rpc.go/kcp"
    "log"
    "sync/atomic"
    "time"

    "github.com/DGHeroin/rpc.go"
)

func main() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)
    for i := 0; i < 5; i++ {
        go runClient()
    }
    go func() {
        for {
            time.Sleep(time.Second)
            n := atomic.LoadUint32(&clientQPS)
            atomic.StoreUint32(&clientQPS, 0)
            if n != 0 {
                log.Println("clientQPS:", n)
            }
        }
    }()
    select {}
}

type clientHandler struct {
}

func (h *clientHandler) OnOpen() {
    isConnected = true
}

func (h *clientHandler) OnMessage(message *common.Message) {

}

func (h *clientHandler) OnClose() {
    isConnected = false
}

var (
    isConnected = false
    clientQPS uint32
)

func runClient() {

    cli, _ := rpc.NewClient(&rpc.ClientOption{
        ReadTimeout:  time.Second * 10,
        WriteTimeout: time.Second * 10,
    })
    cli.AddPlugin(&clientHandler{})
    go func() {
        for {
            if isConnected {
                n := atomic.AddUint32(&clientQPS, 1)
                data := make([]byte, 4)
                binary.BigEndian.PutUint32(data, n)
                err := cli.Request(data, func(message *common.Message) {
                   // fmt.Println("收到回复:", message.Payload)
                })
                if err != nil {
                    log.Println("err", err)
                }
            } else {
                time.Sleep(time.Second)
            }
        }
    }()


    conn, _ := kcp.NewKCPDialer("127.0.0.1:12345", []byte("1234"), []byte("1234"))
    if err := cli.Serve(conn); err != nil {
        log.Println("初始化出错", err)
        return
    }

}
