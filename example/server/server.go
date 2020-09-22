package main

import (
	"github.com/DGHeroin/rpc.go"
	log "github.com/sirupsen/logrus"
	"sync/atomic"
	"time"
)
var (
	count uint64 = 0
	connCount int64 = 0
)
func main() {
	qps := uint32(0)
	count := int32(0)
	server, _ := rpc.NewServer()

	server.OnNew = func(id uint64) {
		//log.Println("新连接", id)
		time.AfterFunc(time.Second*2, func() {
			atomic.AddInt32(&count, 1)
			server.Send(id, 100, []byte("你好"))
		})
	}

	server.OnClose = func(id uint64) {
		//log.Println("关闭连接", id)
		atomic.AddInt32(&count, -1)
	}
	server.OnData = func(id uint64, tag uint32, payload []byte) {
		atomic.AddUint32(&qps, 1)
		log.Println("收到请求", tag, string(payload))
		if tag == 133 {
			server.Send(id, tag, []byte("一百年只能上一次陆地"))
		}
	}
	//go func() {
	//	for {
	//		time.Sleep(time.Second)
	//		val := atomic.LoadUint32(&qps)
	//		atomic.StoreUint32(&qps, 0)
	//		if val == 0 && count == 0 {
	//			continue
	//		}
	//		log.Println("qps:", val, "count:", atomic.LoadInt32(&count))
	//	}
	//}()
	if err := server.Serve("127.0.0.1:12345", nil, nil); err != nil {
		log.Println(err)
	}
}

