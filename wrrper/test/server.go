package main

import "wireflow/wrrp"

func main() {
	server := wrrper.NewServer()
	if err := server.Start(); err != nil {
		panic(err)
	}
}
