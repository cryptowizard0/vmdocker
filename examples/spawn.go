package main

import (
	"fmt"

	goarSchema "github.com/permadao/goar/schema"
)

func spawn() {
	res, err := s.Spawn(
		module,
		scheduler,
		[]goarSchema.Tag{},
	)
	fmt.Printf("res: %#v, err: %v\n", res, err)
}
