package main

//
//import (
//	"fmt"
//	"time"
//	wrrp2 "wireflow/pkg/wrrp"
//	"wireflow/wrrp"
//
//	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
//)
//
//func main() {
//
//	privateKey, err := wgtypes.GeneratePrivateKey()
//	if err != nil {
//		panic(err)
//	}
//
//	publicKey := privateKey.PublicKey()
//
//	sessionId := wrrp.IDFromPublicKey(publicKey[:])
//
//	client := wrrp.NewClient(sessionId, "127.0.0.1:8080")
//	if err := client.Connect(); err != nil {
//		panic(err)
//	}
//
//	go func() {
//		client.HandleFrame(func(header *wrrp2.Header, payload []byte) {
//			fmt.Println(string(payload))
//		})
//	}()
//
//	for {
//		time.Sleep(time.Second)
//		client.Send(sessionId, []byte("hello world"))
//	}
//}
