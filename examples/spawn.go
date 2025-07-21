package main

import (
	"fmt"

	goarSchema "github.com/permadao/goar/schema"
)

func spawn() {
	// s := sdk.New("http://127.0.0.1:8080", "../test_keyfile.json")
	res, err := s.Spawn(
		"LSjhdzBjyWuyUPe-g6PUzt8t1PUlw2FZ9SM3_hCh2Is",
		"0x972AeD684D6f817e1b58AF70933dF1b4a75bfA51",
		[]goarSchema.Tag{},
	)
	fmt.Printf("res: %#v, err: %v\n", res, err)
}
