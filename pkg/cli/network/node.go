package network

import (
	"context"
	"encoding/json"
	"wireflow/internal/grpc"
)

// 指定网络里添加节点
func (n *networkManager) AddOrRmNode(ctx context.Context, networkId, action string, nodeIds []string) error {
	params := &NetworkParams{
		Name: networkId,
	}

	params.AppIds = append(params.AppIds, nodeIds...)

	bs, err := json.Marshal(params)
	if err != nil {
		return err
	}

	message := &grpc.ManagementMessage{
		Body: bs,
	}

	switch action {
	case "add":
		message.Type = grpc.Type_MessageTypeNetworkAddNode
	case "rm":
		message.Type = grpc.Type_MessageTypeNetworkRemoveNode
	}
	_, err = n.client.Request(ctx, message)
	if err != nil {
		return err
	}

	//fmt.Fprintln(os.Stdout, string(resp.Body))
	return nil
}
