package ipam

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"wireflow/api/v1alpha1"
)

type SubnetManager struct {
	client client.Client
}

// AllocateSubnet allocate a subnet for new network
func (m *SubnetManager) AllocateSubnet(ctx context.Context, networkName string, pool *v1alpha1.WireflowGlobalIPPool) (string, error) {
	// 1. 解析总池子
	_, ipnet, _ := net.ParseCIDR(pool.Spec.CIDR)
	mask := net.CIDRMask(pool.Spec.SubnetMask, 32)

	// 2. 迭代计算子网 (这里可以优化：先 List 所有 Allocation 建立内存位图提高效率)
	for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); ip = nextSubnet(ip, pool.Spec.SubnetMask) {
		subnetCIDR := fmt.Sprintf("%s/%d", ip.String(), pool.Spec.SubnetMask)
		allocationName := fmt.Sprintf("subnet-%s", ipToHex(ip))

		// 3. 尝试原子创建索引对象
		alloc := &v1alpha1.WireflowSubnetAllocation{
			ObjectMeta: metav1.ObjectMeta{
				Name: allocationName,
				// 设置 OwnerReference 指向 Network，这样 Network 删除时子网自动回收
				// OwnerReferences: []metav1.OwnerReference{...},
			},
			Spec: struct {
				NetworkName string `json:"networkName"`
				CIDR        string `json:"cidr"`
			}{
				NetworkName: networkName,
				CIDR:        subnetCIDR,
			},
		}

		err := m.client.Create(ctx, alloc)
		if err == nil {
			// 创建成功，意味着我们抢到了这个段
			return subnetCIDR, nil
		}

		if !errors.IsAlreadyExists(err) {
			return "", err // 发生了其他错误
		}
		// 如果 AlreadyExists，说明该段已被占用，循环继续尝试下一个
	}

	return "", fmt.Errorf("no available subnet in pool")
}

func (m *SubnetManager) FindFirstFree(ctx context.Context, pool *v1alpha1.WireflowGlobalIPPool) (net.IP, error) {
	// 1. 从 Informer 缓存获取所有现有的分配
	var allAllocations v1alpha1.WireflowSubnetAllocationList
	m.client.List(ctx, &allAllocations)

	// 2. 将已占用的 Hex 后缀存入 Map
	used := make(map[string]struct{})
	for _, a := range allAllocations.Items {
		// 假设名称格式是 subnet-0a0a0100
		hexStr := strings.TrimPrefix(a.Name, "subnet-")
		used[hexStr] = struct{}{}
	}

	// 3. 迭代计算，遇到不在 used Map 里的第一个地址就返回
	for ip := startIP; pool.Contains(ip); ip = nextSubnet(ip) {
		if _, exists := used[ipToHex(ip)]; !exists {
			return ip, nil // 找到了回收后的空洞或全新的网段
		}
	}
	return nil, errors.New("pool exhausted")
}

// 辅助函数：计算下一个子网地址
func nextSubnet(ip net.IP, maskBits int) net.IP {
	i := ipToUint32(ip)
	i += 1 << (32 - uint32(maskBits))
	return uint32ToIP(i)
}

// ipToHex 将 net.IP 转换为 8 位的十六进制字符串
func ipToHex(ip net.IP) string {
	// 确保处理的是 IPv4 的 4 字节表示
	ipv4 := ip.To4()
	if ipv4 == nil {
		return ""
	}
	// 使用 hex.EncodeToString 直接转换字节数组
	return hex.EncodeToString(ipv4)
}

// hexToIP 将 8 位十六进制字符串还原为 net.IP
func hexToIP(h string) net.IP {
	bytes, err := hex.DecodeString(h)
	if err != nil || len(bytes) != 4 {
		return nil
	}
	return net.IP(bytes)
}

// ipToUint32 将 net.IP 转换为 uint32 数字
func ipToUint32(ip net.IP) uint32 {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return 0
	}
	// 使用 BigEndian (大端序) 保证转换结果符合直觉
	// 例如 1.0.0.0 转换后大于 0.255.255.255
	return binary.BigEndian.Uint32(ipv4)
}

// uint32ToIP 将 uint32 数字还原为 net.IP
func uint32ToIP(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}
