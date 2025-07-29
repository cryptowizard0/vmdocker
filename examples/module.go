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
			{Name: "Image-Name", Value: "chriswebber/docker-golua:v0.0.2"},
			{Name: "Image-ID", Value: "sha256:b2e104cdcb5c09a8f213aefcadd451cbabfda1f16c91107e84eef051f807d45b"},
		},
	})
	bin, _ := json.Marshal(item)

	filename := fmt.Sprintf("mod-%s.json", item.Id)
	os.WriteFile(filename, bin, 0644)

}
