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
			{Name: "Image-Name", Value: "chriswebber/docker-testrt:v0.0.1"},
			{Name: "Image-ID", Value: "sha256:00501e9a7d5310e245eeb0ca5224ea5ce9ba76fd7f9b5de219b1636675b65c33"},
		},
	})
	if err != nil {
		fmt.Println("generate and save module failed, ", "err", err)
		return
	}
	fmt.Println("generate and save module success, ", "id", itemId)

}
