package codegen

import (
	"os"

	"github.com/is-hoku/pl0dash-go/getsource"
	"github.com/is-hoku/pl0dash-go/table"
)

const MAXCODE int = 200 // 目的コードの最大長さ
const MAXMEM int = 2000 // 実行時スタックの最大長さ
const MAXREG int = 20   // 演算レジスタスタックの最大長さ
const MAXLEVEL int = 5  // ブロックの最大長さ

var cIndex int = -1 // 最後に生成した命令語のインデックス

func NextCode() int {
	return cIndex + 1
}

type OpCode int // 命令語のコード
const (
	Lit OpCode = iota
	Opr
	Lod
	Sto
	Cal
	Ret
	Ict
	Jmp
	Jpc
)

type Operator int // 演算命令のコード
const (
	Neg Operator = iota
	Add
	Sub
	Mul
	Div
	Odd
	Eq
	Ls
	Gr
	Neq
	Lseq
	Greq
	Wrt
	Wrl
)

// 命令語の型
type inst struct {
	opCode OpCode
	u      struct {
		addr  getsource.RelAddr
		value int
		optr  Operator
	}
}

var code [MAXCODE]inst // 目的コードが入る

// 命令語の生成、アドレス部に v
func GenCodeV(op OpCode, v int, fptex *os.File) int {
	checkMax(fptex)
	code[cIndex].opCode = op
	code[cIndex].u.value = v
	return cIndex
}

// 命令語の生成、アドレス部に演算命令
func GenCodeT(op OpCode, ti int, fptex *os.File) int {
	checkMax(fptex)
	code[cIndex].opCode = op
	code[cIndex].u.addr = table.RetRelAddr(ti)
	return cIndex
}

// 命令語の生成、アドレス部に演算命令
func GenCodeO(p Operator, fptex *os.File) int {
	checkMax(fptex)
	code[cIndex].opCode = Opr
	code[cIndex].u.optr = p
	return cIndex
}

// ret 命令語の生成
func GenCodeR(fptex *os.File) int {
	if code[cIndex].opCode == Ret { // 直前が ret なら生成せず
		return cIndex
	}
	checkMax(fptex)
	code[cIndex].opCode = Ret
	code[cIndex].u.addr.Level = table.BLevel()
	code[cIndex].u.addr.Addr = table.FPars() // パラメタ数 (実行スタックの解放用)
	return cIndex
}

// 目的コードのインデックスの増加とチェック
func checkMax(fptex *os.File) {
	cIndex++
	if cIndex < MAXCODE {
		return
	}
	getsource.ErrorF("too many code", fptex)
}

// 命令語のバックパッチ (次の番地を)
func BackPatch(i int) {
	code[i].u.value = cIndex + 1
}
