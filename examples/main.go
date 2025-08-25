package main

import (
	"fmt"
	"os"

	"github.com/everFinance/goether"
	"github.com/hymatrix/hymx/sdk"
	"github.com/hymatrix/hymx/vmm/core/token/schema"
	"github.com/permadao/goar"
)

var (
	url = "http://127.0.0.1:8080"

	prvKey     = "0x64dd2342616f385f3e8157cf7246cf394217e13e8f91b7d208e9f8b60e25ed1b"
	signer, _  = goether.NewSigner(prvKey)
	bundler, _ = goar.NewBundler(signer)
	s          = sdk.NewFromBundler(url, bundler)

	module    = "4sX9Uo5-Qk37yUOMLCMrwnm4S3Wfu3Fp7QCSRN0oeoU"
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
	case "transfer":
		if len(os.Args) < 3 {
			fmt.Println("please provide to address for transfer")
			os.Exit(1)
		}
		toAddr := os.Args[2]
		err := transfer(s, toAddr, schema.StakeMinAmount)
		if err != nil {
			fmt.Printf("transfer err: %v\n", err)
		} else {
			fmt.Println("transfer success to ", toAddr)
		}
	case "pingpong":
		pingpong()
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
	case "ollama":
		ollama()
	case "stress":
		doTansfer()
	default:
		fmt.Printf("unknown cmd: %s\n", cmd)
		os.Exit(1)
	}
}
