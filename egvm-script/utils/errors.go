package utils

import "github.com/dop251/goja"

var (
	// function error
	IncorrectArgumentCount = goja.NewSymbol("incorrect argument count")

	// ordered map error
	EmptyKeyString = goja.NewSymbol("Empty key string")
)
