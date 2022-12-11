package main

import (
	"fmt"
	"os"

	"github.com/is-hoku/go-pl0dash/compile"
	"github.com/is-hoku/go-pl0dash/getsource"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Errorf("invalid argument length")
		return
	}
	fileName := os.Args[1]
	f, tex, err := getsource.OpenSource(fileName)
	if err != nil {
		fmt.Errorf("cannot open the file: %s", err)
		return
	}
	defer tex.Close()
	defer f.Close()
	if err := compile.Compile(f); err != nil {
		err = codegen.Execute(f)
	}
	if err != nil {
		fmt.Errorf("Error: %s", err)
	}
}
