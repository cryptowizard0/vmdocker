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
			{Name: "Image-Name", Value: "chriswebber/docker-openclaw:v0.0.4"},
			{Name: "Image-ID", Value: "sha256:a79268611c91b6c8cd5cf259eeb0de154fbdb152a73c28fdaeedbab596d6137b"},
			{Name: "Openclaw-Version", Value: "2026.3.1-beta.1"},
		},
	})
	if err != nil {
		fmt.Println("generate and save module failed, ", "err", err)
		return
	}
	fmt.Println("generate and save module success, ", "id", itemId)

}
