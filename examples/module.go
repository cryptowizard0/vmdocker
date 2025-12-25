package main

import (
	"fmt"

	"github.com/hymatrix/hymx/schema"
	arSchema "github.com/permadao/goar/schema"
)

func genModule() {
	// ex ModuleFormat: "org.type.1.0.0"
	itemId, err := s.SaveModule([]byte{}, schema.Module{
		Base:         schema.DefaultBaseModule,
		ModuleFormat: "web.vmdocker-golua-ao.v0.0.1",
		Tags: []arSchema.Tag{
			{Name: "Image-Name", Value: "chriswebber/docker-golua:v0.0.4"},
			{Name: "Image-ID", Value: "sha256:883e4583a2426e5ab49fc33d22a574201a738c4597660d42fc1cc21ccb04f54f"},
		},
	})
	if err != nil {
		fmt.Println("generate and save module failed, ", "err", err)
		return
	}
	fmt.Println("generate and save module success, ", "id", itemId)

}
