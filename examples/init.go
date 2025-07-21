package main

import (
	"fmt"

	"github.com/permadao/goar/schema"
)

func initToken() string {
	res, err := s.SpawnAndWait(
		"Hw3VRVfOJjtLy-ll7JLkt6tMAQH-riYPJ210Gxgn-34",
		s.GetAddress(),
		[]schema.Tag{})
	fmt.Println(res, err)
	return res.Id
}

func initRegistry(hmpid string) {
	res, err := s.SpawnAndWait(
		"MVTil0kn5SRiJELW7W2jLZ6cBr3QUGj1nJ67I2Wi4Ps",
		s.GetAddress(),
		[]schema.Tag{
			{Name: "Hm-Pid", Value: hmpid},
		})
	fmt.Println(res, err)
}
