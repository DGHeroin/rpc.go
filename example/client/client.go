package main

import (
	"fmt"
	"log"
	"time"

	"github.com/DGHeroin/rpc.go"
)

func main()  {
	for i := 0; i < 1; i++ {
		go runClient()
	}
	select {

	}
}

func runClient() {
	isConnected := false
	cli, _ := rpc.NewClient()
	cli.OnConnected = func() {
		//log.Println("connected")
		isConnected = true
	}
	cli.OnData = func(message *rpc.Message) {
		log.Println("收到消息", message.Tag, string(message.Payload))
	}
	cli.OnClose	= func() {
		//log.Println("连接关闭")
		isConnected = false
		cli.Reconnect(time.Second)
	}

	go func() {
		for {
			if isConnected {
				cli.Send(123, []byte("hello world!"), func(message *rpc.Message) {
					fmt.Println("收到回复:", message.Tag, string(message.Payload))
				})
				time.Sleep(time.Second)
			} else {
				time.Sleep(time.Second)
			}
		}
	}()

	if err := cli.DialAndServe("127.0.0.1:12345", nil, nil); err != nil {
		log.Println("初始化出错", err)
		return
	}

}

