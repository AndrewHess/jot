package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/andrewhess/jot/internal/jot"
)

func main() {
	if err := jot.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		var usageErr *jot.UsageError
		if errors.As(err, &usageErr) {
			fmt.Fprintln(os.Stderr, usageErr.Error())
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
