package ecs

import (
	"fmt"
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/ecs"
	"github.com/volcengine/volcengine-go-sdk/service/vpc"
)

const (
	defaultVpcName           = "volcengine_packer_vpc"
	defaultVpcCidr           = "172.20.0.0/16"
	defaultSubnetName        = "volcengine_packer_subnet"
	defaultSubnetCidr        = "172.20.0.0/24"
	defaultSecurityGroupName = "volcengine_packer_security_group"

	defaultRetryInterval = 10 * time.Second
	defaultRetryTimes    = 10

	defaultEcsNamePrefix = "volcengine_packer_ecs"
)

// getDefaultEcsName generates a unique ECS instance name with timestamp
func getDefaultEcsName() string {
	return fmt.Sprintf("%s_%s", defaultEcsNamePrefix, time.Now().Format("20060102150405"))
}

type VolcengineClientWrapper struct {
	EcsClient *ecs.ECS
	VpcClient *vpc.VPC
}

type WaitForParam struct {
	Call          func() (interface{}, error)
	Process       func(interface{}, error) ProcessResult
	RetryInterval time.Duration
	RetryTimes    int
}

type ProcessResult struct {
	Complete  bool
	StopRetry bool
}

var (
	WaitForSuccess = ProcessResult{
		Complete:  true,
		StopRetry: true,
	}

	WaitForRetry = ProcessResult{
		Complete:  false,
		StopRetry: false,
	}
)
