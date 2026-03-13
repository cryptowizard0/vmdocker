package main

import (
	"fmt"
	"os"

	vmdockerSchema "github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
	"github.com/hymatrix/hymx/schema"
	arSchema "github.com/permadao/goar/schema"
)

func genModule() {
	imageName := os.Getenv("VMDOCKER_SANDBOX_IMAGE_NAME")
	imageID := os.Getenv("VMDOCKER_SANDBOX_IMAGE_ID")
	if imageName == "" || imageID == "" {
		imageName = "chriswebber/docker-openclaw-sandbox:fix-test"
		imageID = "sha256:4daa6b51a12f41566bca09c2ca92a4982263db47f40d20d11c8f83f6ae85bc0e"
	}

	// ex ModuleFormat: "org.type.1.0.0"
	itemId, err := s.SaveModule([]byte{}, schema.Module{
		Base:         schema.DefaultBaseModule,
		ModuleFormat: vmdockerSchema.ModuleFormat,
		Tags: []arSchema.Tag{
			{Name: "Runtime-Backend", Value: "sandbox"},
			{Name: "Image-Name", Value: imageName},
			{Name: "Image-ID", Value: imageID},
			{Name: "Sandbox-Agent", Value: "shell"},
			{Name: "Openclaw-Version", Value: "2026.3.1-beta.1"},
		},
	})
	if err != nil {
		fmt.Println("generate and save module failed, ", "err", err)
		return
	}
	fmt.Println("generate and save module success, ", "id", itemId)

}
