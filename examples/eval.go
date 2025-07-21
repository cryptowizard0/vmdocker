package main

import (
	"fmt"

	"github.com/hymatrix/hymx/sdk"
	"github.com/permadao/goar/schema"
	goarSchema "github.com/permadao/goar/schema"
)

func eval() {
	// s := sdk.New("http://127.0.0.1:8080", "../test_keyfile.json")

	res, err := s.SpawnAndWait(
		"pQUiSWVtGTIP8isNyGMlYLpknEnpFA2SWtiTDNAecIo",
		"0x972AeD684D6f817e1b58AF70933dF1b4a75bfA51",
		[]goarSchema.Tag{},
	)
	if err != nil {
		fmt.Println("Failed to spawn: ", err)
		return
	}

	target := res.Id
	fmt.Println("spawn ok, pid: ", res.Id)

	code := `
		print('Hello from lua!')
		Name = 'Hello'
		Cache({Name = 'World'})
		Cache({Name2 = 'World2'})
	`

	res, err = s.SendMessageAndWait(target, code,
		[]schema.Tag{
			{Name: "Action", Value: "Eval"},
			{Name: "Target", Value: target},
			{Name: "Module", Value: "0x84534"},
			{Name: "Block-Height", Value: "100000"},
			{Name: "Data", Value: code},
		})
	if err != nil {
		fmt.Println("handler error: ", err)
		return
	}
	fmt.Println("target1 load ok, ", res)
}

func eval2() {
	s := sdk.New("http://127.0.0.1:8080", "../test_keyfile.json")

	target := "LEQKWcbvuWdQUIO4sWt5lgIK9x3nE8oFg6cTxSxtfn8"

	code := `
		print(Name)
	`

	res, err := s.SendMessageAndWait(target, code,
		[]schema.Tag{
			{Name: "Action", Value: "Eval"},
			{Name: "Target", Value: target},
			{Name: "Module", Value: "0x84534"},
			{Name: "Block-Height", Value: "100000"},
			{Name: "Data", Value: code},
		})
	if err != nil {
		fmt.Println("handler error: ", err)
		return
	}
	fmt.Println("target1 load ok, ", res)
}
