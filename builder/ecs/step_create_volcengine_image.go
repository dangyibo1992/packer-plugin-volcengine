package ecs

import (
	"context"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/volcengine/volcengine-go-sdk/service/ecs"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
)

type stepCreateVolcengineImage struct {
	VolcengineEcsConfig *VolcengineEcsConfig
}

func (s *stepCreateVolcengineImage) Run(ctx context.Context, stateBag multistep.StateBag) multistep.StepAction {
	ui := stateBag.Get("ui").(packer.Ui)
	client := stateBag.Get("client").(*VolcengineClientWrapper)
	instanceId := stateBag.Get("instanceId").(string)
	ui.Say("create new image ")
	input := ecs.CreateImageInput{
		InstanceId:       volcengine.String(instanceId),
		ImageName:        volcengine.String(s.VolcengineEcsConfig.TargetImageName),
		CreateWholeImage: volcengine.Bool(false), // 不创建整机镜像，只创建系统盘镜像
		NeedDetection:    volcengine.Bool(false), // 不进行镜像检测
	}
	output, err := client.EcsClient.CreateImageWithContext(ctx, &input)
	if err != nil {
		return Halt(stateBag, err, "Error create image")
	}
	_, err = WaitImageStatus(stateBag, *output.ImageId, "available")
	if err != nil {
		return Halt(stateBag, err, "Error waiting for image")
	}
	stateBag.Put("TargetImageId", *output.ImageId)
	ui.Say("image created successfully: " + *output.ImageId)

	// share image to specified accounts
	if len(s.VolcengineEcsConfig.ImageShareAccounts) > 0 {
		ui.Say("sharing image to accounts: " + *s.VolcengineEcsConfig.ImageShareAccounts[0])
		shareInput := &ecs.ModifyImageSharePermissionInput{
			ImageId:        volcengine.String(*output.ImageId),
			AddAccounts:    s.VolcengineEcsConfig.ImageShareAccounts,
			RemoveAccounts: nil,
		}
		_, err = client.EcsClient.ModifyImageSharePermissionWithContext(ctx, shareInput)
		if err != nil {
			return Halt(stateBag, err, "Error sharing image")
		}
		ui.Say("image shared successfully")
	}

	return multistep.ActionContinue
}

func (stepCreateVolcengineImage) Cleanup(bag multistep.StateBag) {

}
