package main

import (
	"fmt"
	"os"

	"promthus/internal/crypto"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/hashpwd/main.go <password>")
		os.Exit(1)
	}
	password := os.Args[1]
	hash, err := crypto.HashPassword(password)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(hash)
}
