package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hymatrix/hymx/schema"
	arSchema "github.com/permadao/goar/schema"
)

func genModule() {
	// ex ModuleFormat: "org.type.1.0.0"
	item, _ := s.GenerateModule([]byte{}, schema.Module{
		Base:         schema.DefaultBaseModule,
		ModuleFormat: "web.vmdocker-golua-ao.v0.0.1",
		Tags: []arSchema.Tag{
			{Name: "Image-Name", Value: "chriswebber/docker-golua:v0.0.4"},
			{Name: "Image-ID", Value: "sha256:883e4583a2426e5ab49fc33d22a574201a738c4597660d42fc1cc21ccb04f54f"},
		},
	})
	bin, _ := json.Marshal(item)

	filename := fmt.Sprintf("mod-%s.json", item.Id)
	os.WriteFile(filename, bin, 0644)

}
