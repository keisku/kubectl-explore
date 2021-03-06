package main

import (
	"fmt"
	"os"
)

func main() {
	if err := NewCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
