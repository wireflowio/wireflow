package client

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"linkany/internal"
	pb "linkany/management/grpc/mgt"
	"linkany/pkg/config"
	"testing"
)

func TestNewGrpcClient(t *testing.T) {
	t.Run("TestGrpcClient_List", TestGrpcClient_List)
	t.Run("TestGrpcClient_Watch", TestGrpcClient_Watch)
}

func TestGrpcClient_List(t *testing.T) {
	client, err := NewGrpcClient(&GrpcConfig{Addr: internal.ManagementDomain + ":50051"})
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := config.GetLocalConfig()
	if err != nil {
		t.Fatal(err)
	}

	requset := &pb.Request{
		AppId:  cfg.AppId,
		Token:  cfg.Token,
		PubKey: "a+BYvXq6/xrvsnKbgORSL6lwFzqtfXV0VnTzwdo+Vnw=",
	}

	body, err := proto.Marshal(requset)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	resp, err := client.List(ctx, &pb.ManagementMessage{
		PubKey: "a+BYvXq6/xrvsnKbgORSL6lwFzqtfXV0VnTzwdo+Vnw=",
		Body:   body,
	})

	if err != nil {
		t.Fatal(err)
	}

	t.Log(resp)
}

func TestGrpcClient_Watch(t *testing.T) {
	client, err := NewGrpcClient(&GrpcConfig{Addr: internal.ManagementDomain + ":50051"})
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := config.GetLocalConfig()
	if err != nil {
		t.Fatal(err)
	}

	requset := &pb.Request{
		AppId:  cfg.AppId,
		Token:  cfg.Token,
		PubKey: "a+BYvXq6/xrvsnKbgORSL6lwFzqtfXV0VnTzwdo+Vnw=",
	}

	body, err := proto.Marshal(requset)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	err = client.Watch(ctx, &pb.ManagementMessage{
		PubKey: "a+BYvXq6/xrvsnKbgORSL6lwFzqtfXV0VnTzwdo+Vnw=",
		Body:   body,
	}, func(networkMap pb.NetworkMap) error {
		fmt.Println(networkMap)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

}
