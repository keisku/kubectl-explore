package main

import (
	"fmt"
	"os"

	"github.com/keisku/kubectl-explore/explore"
)

func main() {
	if err := explore.NewCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
