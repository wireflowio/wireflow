package node

import (
	"context"
	"fmt"
	"wireflow/internal/core/domain"
	"wireflow/pkg/cli/network"

	"github.com/spf13/cobra"
)

// start cmd
func NewNodeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node <sub-command>",
		Short: "manager nodes in the given network",
		Long:  `该命令将一个或者多个已存在的节点(node_id)授权并加入或移除指定的网络(network_id)`,
		Example: `  # 添加单个节点
  wfctl network node add prod-net node-01
  
  # 批量添加多个节点
  wfctl network node add prod-net node-01 node-02 node-03`,
		Args: cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(newNodeAddCommand())
	cmd.AddCommand(newNodeRemoveCommand())

	return cmd
}

func newNodeAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <network_id> <node_id>...",
		Short: "将节点加入特定网络",
		// Long 字段可以用来详细解释这些参数是什么
		Long: `该命令将一个或者多个已存在的节点(node_id)授权并加入到指定的网络(network_id)中。
    
参数说明:
  network_id    目标网络的唯一标识符或名称
  node_id       待加入节点的唯一标识符或名称`,
		Example: `  # 添加单个节点
  wfctl network node add prod-net node-01
  
  # 批量添加多个节点
  wfctl network node add prod-net node-01 node-02 node-03`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 参数获取
			networkID := args[0]
			nodeIDs := args[1:]

			fmt.Printf("目标网络: %s\n", networkID)
			fmt.Printf("准备添加 %d 个节点...\n", len(nodeIDs))

			fmt.Printf(" >> 正在处理节点: %s\n", nodeIDs)

			return addNodeToNetwork(networkID, nodeIDs)

		},
	}

	return cmd
}

func addNodeToNetwork(networkId string, nodeIds []string) error {
	manager, err := network.NewNetworkManager(domain.ServerUrl)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return manager.AddOrRmNode(ctx, networkId, "add", nodeIds)
}

func newNodeRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm <network_id> <node_id>...",
		Short: "将节点离开特定网络",
		// Long 字段可以用来详细解释这些参数是什么
		Long: `该命令将一个或者多个已存在的节点(node_id)授权并离开已经加入到的指定的网络(network_id)中。
    
参数说明:
  network_id    目标网络的唯一标识符或名称
  node_id       待加入节点的唯一标识符或名称`,
		Example: `  # 移除单个节点
  wfctl network node rm prod-net node-01
  
  # 批量移除多个节点
  wfctl network node rm prod-net node-01 node-02 node-03`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 参数获取
			networkID := args[0]
			nodeIDs := args[1:]

			fmt.Printf("目标网络: %s\n", networkID)
			fmt.Printf("准备添加 %d 个节点...\n", len(nodeIDs))

			fmt.Printf(" >> 正在处理节点: %s\n", nodeIDs)

			return rmNodeToNetwork(networkID, nodeIDs)

		},
	}

	return cmd
}

func rmNodeToNetwork(networkId string, nodeIds []string) error {
	manager, err := network.NewNetworkManager(domain.ServerUrl)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return manager.AddOrRmNode(ctx, networkId, "rm", nodeIds)
}
