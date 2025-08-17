package main

import (
	"fmt"

	"github.com/hymatrix/hymx/sdk"
	"github.com/permadao/goar/schema"
	goarSchema "github.com/permadao/goar/schema"
)

func recover1() {
	// s := sdk.New("http://127.0.0.1:8080", "../test_keyfile.json")

	res, err := s.SpawnAndWait(
		"LSjhdzBjyWuyUPe-g6PUzt8t1PUlw2FZ9SM3_hCh2Is",
		"0x972AeD684D6f817e1b58AF70933dF1b4a75bfA51",
		[]goarSchema.Tag{},
	)
	if err != nil {
		fmt.Println("Failed to spawn: ", err)
		return
	}
	fmt.Println("spawn load ok, ", res)
	target := res.Id

	code := `
		print('Hello from lua!')
		Name = 'Hello'
		count = 0
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

	for i := 1; i <= 10; i++ {
		code = `
			count = count + 1
			print(count)
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
		fmt.Println("eval at handle: ", i, ", res: ", res)
	}
}

func recover2() {
	s := sdk.New("http://127.0.0.1:8080", "../test_keyfile.json")

	target := "BIQjCNaQgOF6QX93BXUw93BHVGZ_lyBLpYyxi0uGFpA"

	code := `
		count
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
