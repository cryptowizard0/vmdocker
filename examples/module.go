package main

import (
	"fmt"

	vmdockerSchema "github.com/cryptowizard0/vmdocker/vmdocker/schema"
	"github.com/hymatrix/hymx/schema"
	arSchema "github.com/permadao/goar/schema"
)

func genModule() {
	// ex ModuleFormat: "org.type.1.0.0"
	itemId, err := s.SaveModule([]byte{}, schema.Module{
		Base:         schema.DefaultBaseModule,
		ModuleFormat: vmdockerSchema.ModuleFormat,
		Tags: []arSchema.Tag{
			{Name: "Image-Name", Value: "chriswebber/docker-openclaw:v0.0.1"},
			{Name: "Image-ID", Value: "sha256:85060d33695718db193d3e37d5d8d9c379ed76a21b6d471e96c5ae55c14dbf95"},
			{Name: "Openclaw-Version", Value: "2026.3.1-beta.1"},
		},
	})
	if err != nil {
		fmt.Println("generate and save module failed, ", "err", err)
		return
	}
	fmt.Println("generate and save module success, ", "id", itemId)

}
