package main

import (
	"fmt"
	"time"

	"github.com/permadao/goar/schema"
	goarSchema "github.com/permadao/goar/schema"
)

func eval() {
	// s := sdk.New("http://127.0.0.1:8080", "../test_keyfile.json")

	res, err := s.SpawnAndWait(
		module,
		scheduler,
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

	time.Sleep(2000 * time.Millisecond)

	// s2 := sdk.NewFromBundler("http://127.0.0.1:8081", bundler)
	res, err = s.SendMessageAndWait(target, code,
		[]schema.Tag{
			{Name: "Action", Value: "Eval"},
			{Name: "Target", Value: target},
			{Name: "Data", Value: code},
		})
	if err != nil {
		fmt.Println("handler error: ", err)
		return
	}
	fmt.Println("target1 load ok, ", res)
}
