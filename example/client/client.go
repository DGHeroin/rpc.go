package main

import (
    "flag"
    "github.com/DGHeroin/rpc.go"
    log "github.com/sirupsen/logrus"
    "sync"
    "time"
)

var (
    address = flag.String("addr", "127.0.0.1:12345", "dial address")
    n = flag.Int("n", 500, "thread num")
)
var wg sync.WaitGroup

func main()  {
    flag.Parse()

    for i := 0; i < *n; i++ {
        wg.Add(1)
        go runClient()
    }
    wg.Wait()
}

func runClient() {
    defer wg.Done()
    isConnected := false
    cli, _ := rpc.NewClient()
    cli.OnConnected = func() {
        isConnected = true

    }
    cli.OnData = func(tag uint32, data []byte) {
        //log.Println("收到消息", tag, string(data))
    }
    cli.OnClose	= func() {
        isConnected = false
        cli.Reconnect(time.Second)
    }

    go func() {
        for {
            if isConnected {
                cli.Send(123, []byte("hello world!"))
               time.Sleep(time.Nanosecond)
            } else {
                time.Sleep(time.Second)
            }
        }
    }()

    if err := cli.DialAndServe(*address, nil, nil); err != nil {
        log.Println("初始化出错", err)
        return
    }

}

