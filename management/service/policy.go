package service

import (
	"context"
	"strings"
	"wireflow/api/v1alpha1"
	"wireflow/internal/log"
	"wireflow/management/dto"
	"wireflow/management/resource"
	"wireflow/management/vo"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PolicyService interface {
	CreateOrUpdatePolicy(ctx context.Context, policyDto *dto.PolicyDto) (*vo.PolicyVo, error)
	ListPolicy(ctx context.Context, pageParam *dto.PageRequest) (*dto.PageResult[vo.PolicyVo], error)
}

type policyService struct {
	log    *log.Logger
	client *resource.Client
}

func (p policyService) ListPolicy(ctx context.Context, pageParam *dto.PageRequest) (*dto.PageResult[vo.PolicyVo], error) {
	var (
		policyList v1alpha1.WireflowPolicyList
		err        error
	)
	err = p.client.GetAPIReader().List(ctx, &policyList, client.InNamespace(pageParam.Namespace))

	if err != nil {
		return nil, err
	}

	// 2. 获取全量数据（模拟）
	allPolicies := []*vo.PolicyVo{ /* ... 很多数据 ... */ }

	for _, n := range policyList.Items {
		allPolicies = append(allPolicies, &vo.PolicyVo{
			Name:               n.Name,
			Action:             n.Annotations["action"],
			Description:        n.Annotations["description"],
			WireflowPolicySpec: &n.Spec,
		})
	}

	// 3. 逻辑过滤（搜索）
	var filteredPolicies []*vo.PolicyVo
	if pageParam.Search != "" {
		for _, n := range allPolicies {

			policyType := n.Action
			description := n.Description

			if strings.Contains(n.Name, pageParam.Search) || strings.Contains(policyType, pageParam.Search) || strings.Contains(description, pageParam.Search) {
				filteredPolicies = append(filteredPolicies, n)
			}
		}
	} else {
		filteredPolicies = allPolicies
	}

	// 4. 执行内存切片分页
	total := len(filteredPolicies)
	start := (pageParam.Page - 1) * pageParam.PageSize
	end := start + pageParam.PageSize

	// 防止切片越界越界
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	// 截取
	data := filteredPolicies[start:end]
	var res []*vo.PolicyVo
	for _, n := range data {
		res = append(res, n)
	}

	var vos []vo.PolicyVo
	for _, n := range res {
		vos = append(vos, *n)
	}

	return &dto.PageResult[vo.PolicyVo]{
		Page:     pageParam.Page,
		PageSize: pageParam.PageSize,
		Total:    int64(len(allPolicies)),
		List:     vos,
	}, nil
}

func (p *policyService) CreateOrUpdatePolicy(ctx context.Context, policyDto *dto.PolicyDto) (*vo.PolicyVo, error) {
	newPolicy := &v1alpha1.WireflowPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "wireflowcontroller.wireflow.run/v1alpha1",
			Kind:       "WireflowPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyDto.Name, // 强制使用 DTO 外层的名字
			Namespace: "default",      // 或者从上下文获取
			Labels: map[string]string{
				"action":      policyDto.Action,
				"description": policyDto.Description,
			},
		},
		// 关键点：直接把嵌入的指针赋值给 Spec
		Spec: policyDto.WireflowPolicySpec,
	}

	// 使用SSA模式
	manager := client.FieldOwner("wireflow-controller-manager")

	err := p.client.Patch(ctx, newPolicy, client.Apply, manager)
	if err != nil {
		return nil, err
	}
	policyVo := vo.PolicyVo{
		Name:               newPolicy.Name,
		Action:             newPolicy.Spec.Action,
		Description:        policyDto.Description,
		Namespace:          policyDto.Namespace,
		WireflowPolicySpec: &newPolicy.Spec,
	}

	return &policyVo, nil
}

func NewPolicyService(client *resource.Client) PolicyService {
	return &policyService{
		log:    log.GetLogger("policy-service"),
		client: client,
	}
}

func buildPolicyFromArgs(namespace, name string, peerSelector metav1.LabelSelector, IngressRule []v1alpha1.IngressRule, EgressRule []v1alpha1.EgressRule, action string) v1alpha1.WireflowPolicy {
	return v1alpha1.WireflowPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "WireflowNetwork",
			APIVersion: "wireflowcontroller.wireflow.run/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: v1alpha1.WireflowPolicySpec{
			PeerSelector: peerSelector,
			Ingress:      IngressRule,
			Egress:       EgressRule,
			Action:       action,
			Network:      "wireflow-default-net",
		},
	}
}
