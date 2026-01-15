package main

import "wireflow/wrrp"

func main() {
	server := wrrp.NewServer()
	if err := server.Start(); err != nil {
		panic(err)
	}
}
