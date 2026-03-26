package ecs

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/volcengine/volcengine-go-sdk/service/vpc"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
)

type stepConfigVolcengineSg struct {
	sgId                string
	VolcengineEcsConfig *VolcengineEcsConfig
}

func (s *stepConfigVolcengineSg) Run(ctx context.Context, stateBag multistep.StateBag) multistep.StepAction {
	ui := stateBag.Get("ui").(packer.Ui)
	client := stateBag.Get("client").(*VolcengineClientWrapper)
	if s.VolcengineEcsConfig.SecurityGroupId != "" {
		//valid
		input := vpc.DescribeSecurityGroupsInput{
			SecurityGroupIds: volcengine.StringSlice([]string{s.VolcengineEcsConfig.SecurityGroupId}),
		}
		out, err := client.VpcClient.DescribeSecurityGroupsWithContext(ctx, &input)
		if err != nil || len(out.SecurityGroups) == 0 {
			return Halt(stateBag, err, fmt.Sprintf("Error query SecurityGroup with id %s", s.VolcengineEcsConfig.SecurityGroupId))
		}
		if *out.SecurityGroups[0].VpcId != s.VolcengineEcsConfig.VpcId {
			return Halt(stateBag, fmt.Errorf("SecurityGroup id %s vpc not match",
				s.VolcengineEcsConfig.SecurityGroupId), "")
		}
		ui.Say(fmt.Sprintf("Using existing SecurityGroup id is %s", s.VolcengineEcsConfig.SecurityGroupId))
		stateBag.Put("skipSecurityGroupRule", true)
		return multistep.ActionContinue
	}
	//create new SecurityGroup
	if s.VolcengineEcsConfig.SecurityGroupName == "" {
		s.VolcengineEcsConfig.SecurityGroupName = defaultSecurityGroupName
	}
	ui.Say(fmt.Sprintf("Creating new SecurityGroup with name %s", s.VolcengineEcsConfig.SecurityGroupName))

	input := vpc.CreateSecurityGroupInput{
		VpcId:             volcengine.String(s.VolcengineEcsConfig.VpcId),
		SecurityGroupName: volcengine.String(s.VolcengineEcsConfig.SecurityGroupName),
	}
	output, err := client.VpcClient.CreateSecurityGroupWithContext(ctx, &input)
	if err != nil {
		return Halt(stateBag, err, "Error creating new SecurityGroup")
	}
	s.sgId = *output.SecurityGroupId
	s.VolcengineEcsConfig.SecurityGroupId = *output.SecurityGroupId

	_, err = WaitVpcStatus(stateBag, s.VolcengineEcsConfig.VpcId, "Available")
	if err != nil {
		return Halt(stateBag, err, "Error creating new SecurityGroup")
	}

	// Add security group rules for SSH (22) and RDP (3389)
	ui.Say(fmt.Sprintf("Authorizing SecurityGroup %s Rules", s.VolcengineEcsConfig.SecurityGroupId))

	// SSH rule (TCP 22)
	sshInput := vpc.AuthorizeSecurityGroupIngressInput{
		SecurityGroupId: volcengine.String(s.VolcengineEcsConfig.SecurityGroupId),
		Protocol:        volcengine.String("tcp"),
		PortStart:       volcengine.Int64(22),
		PortEnd:         volcengine.Int64(22),
		CidrIp:          volcengine.String("0.0.0.0/0"),
	}
	_, err = client.VpcClient.AuthorizeSecurityGroupIngressWithContext(ctx, &sshInput)
	if err != nil {
		return Halt(stateBag, err, "Error authorizing SSH security group rule")
	}

	// RDP rule (TCP 3389)
	rdpInput := vpc.AuthorizeSecurityGroupIngressInput{
		SecurityGroupId: volcengine.String(s.VolcengineEcsConfig.SecurityGroupId),
		Protocol:        volcengine.String("tcp"),
		PortStart:       volcengine.Int64(3389),
		PortEnd:         volcengine.Int64(3389),
		CidrIp:          volcengine.String("0.0.0.0/0"),
	}
	_, err = client.VpcClient.AuthorizeSecurityGroupIngressWithContext(ctx, &rdpInput)
	if err != nil {
		return Halt(stateBag, err, "Error authorizing RDP security group rule")
	}

	return multistep.ActionContinue
}

func (s *stepConfigVolcengineSg) Cleanup(stateBag multistep.StateBag) {
	if s.sgId != "" {
		ui := stateBag.Get("ui").(packer.Ui)
		client := stateBag.Get("client").(*VolcengineClientWrapper)
		ui.Say(fmt.Sprintf("Deleting SecurityGroup with Id %s ", s.sgId))
		err := WaitSgClean(stateBag, s.sgId)
		if err != nil {
			ui.Error(fmt.Sprintf("Error delete SecurityGroup %s", err))
			return
		}
		input := vpc.DeleteSecurityGroupInput{
			SecurityGroupId: volcengine.String(s.sgId),
		}
		_, err = client.VpcClient.DeleteSecurityGroup(&input)
		if err != nil {
			ui.Error(fmt.Sprintf("Error delete SecurityGroup %s", err))
		}
	}
}
