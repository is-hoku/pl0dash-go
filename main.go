package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/is-hoku/pl0dash-go/codegen"
	"github.com/is-hoku/pl0dash-go/compile"
	"github.com/is-hoku/pl0dash-go/getsource"
)

func main() {
	if len(os.Args) != 2 {
		err := errors.New("invalid argument length")
		fmt.Println(err)
		return
	}
	fileName := os.Args[1]
	f, tex, scanner, err := getsource.OpenSource(fileName)
	if err != nil {
		err := errors.New(fmt.Sprintf("cannot open the file: %s", err))
		fmt.Println(err)
		return
	}
	defer tex.Close()
	defer f.Close()
	if err := compile.Compile(f, scanner); err != nil {
		codegen.Execute(tex)
	}
	if err != nil {
		err := errors.New(fmt.Sprintf("Error: %s", err))
		fmt.Println(err)
		return
	}
}
