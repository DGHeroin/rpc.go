package main

import (
    "context"
    "github.com/DGHeroin/rpc.go/client"
    "log"
)

func main()  {
    dis, _ := client.NewPeer2PeerDiscovery("127.0.0.1:9527")
    c := client.NewClient("game.server",dis, client.DefaultOption)
    var (
        req string
        reply string
    )
    err := c.Call(context.Background(), "mul", req, &reply)
    if err != nil {
        log.Println(err)
        return
    }
    log.Println("reply:", reply)
}
