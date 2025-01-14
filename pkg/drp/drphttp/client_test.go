package drphttp

import (
	"linkany/pkg/drp"
	"net"
	"testing"
)

func TestClient_Connect(t *testing.T) {
	node := &drp.Node{}
	c := NewClient(node)
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
	if err != nil {
		t.Fatal(err)
	}
	c.node = &drp.Node{
		IpV4Addr: addr,
	}
	_, err = c.Connect("http://127.0.0.1:8080/drp")
	if err != nil {
		t.Fatal(err)
	}
}
