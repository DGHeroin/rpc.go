package main

import (
	"github.com/DGHeroin/rpc.go"
	log "github.com/sirupsen/logrus"
	"time"
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
	cli.OnData = func(tag uint32, data []byte) {
		//log.Println("收到消息", tag, string(data))
	}
	cli.OnClose	= func() {
		//log.Println("连接关闭")
		isConnected = false
		cli.Reconnect(time.Second)
	}

	go func() {
		for {
			if isConnected {
				cli.Send(123, []byte("hello world!"))
				time.Sleep(time.Microsecond)
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

