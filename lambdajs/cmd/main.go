package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/dop251/goja"
)

func readSource(filename string) ([]byte, error) {
	if filename == "" || filename == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(filename) //nolint: gosec
}

func main() {
	flag.Parse()

	err := func() error {
		src, err := readSource(flag.Arg(0))
		if err != nil {
			return err
		}

		vm := goja.New()
		RegisterFunctions(vm)

		value, err := vm.RunString(string(src))
		if err != nil {
			return err
		}

		fmt.Printf("run script value: %s\n", value)
		return nil
	}()

	if err != nil {
		var oErr *goja.Exception
		if errors.As(err, &oErr) {
			fmt.Fprint(os.Stderr, oErr.String())
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(64)
	}
}
