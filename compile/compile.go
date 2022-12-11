package compile

import (
	"errors"
	"fmt"
	"os"
)

func Compile(f *os.File) error {
	fmt.Println("start compilation")
	initSource()
	token = nextToken()
	blockBegin(FIRSTADDR)

	block(0)
	finalSource()
	i := errorN()
	if i != 0 {
		return errors.New("the number of error is %d", i)
	}
	if i < MINERROR {
		return errors.New("too many errors")
	}
	return nil
}
