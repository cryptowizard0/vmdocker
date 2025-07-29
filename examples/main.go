package main

import (
	"fmt"
	"os"

	"github.com/everFinance/goether"
	"github.com/hymatrix/hymx/sdk"
	"github.com/permadao/goar"
)

var (
	url = "http://127.0.0.1:8080"

	prvKey     = "0x64dd2342616f385f3e8157cf7246cf394217e13e8f91b7d208e9f8b60e25ed1b"
	signer, _  = goether.NewSigner(prvKey)
	bundler, _ = goar.NewBundler(signer)
	s          = sdk.NewFromBundler(url, bundler)
	s2         = sdk.New(url, "./test_keyfile2.json")

	module    = "qvsXuWo0sLardhzyNJI-9JGbWOFYqQz7PZfUD2JlgvU"
	scheduler = "0x972AeD684D6f817e1b58AF70933dF1b4a75bfA51"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("please input cmd, ex: pingpong, sendMessage, spawn, eval, eval2, receive, receive2, reply, inbox, result, checkpoint, ollama, recover1, recover2")
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "init":
		initRegistry(initToken())
	case "pingpong":
		pingpong()
	case "sendMessage":
		sendMessage()
	case "module":
		genModule()
	case "spawn":
		spawn()
	case "spawnChild":
		spawnChild()
	case "eval":
		eval()
	case "receive":
		receive()
	case "receive2":
		receive2()
	case "reply":
		reply()
	case "inbox":
		inbox()
	case "result":
		result()
	case "ollama":
		ollama()
	case "recover1":
		recover1()
	case "recover2":
		recover2()
	case "stress":
		doTansfer()
	case "checkpoint1":
		checkpoint_1()
	case "checkpoint2":
		pid := os.Args[2]
		checkpoint_2(pid)
	default:
		fmt.Printf("unknown cmd: %s\n", cmd)
		os.Exit(1)
	}
}
