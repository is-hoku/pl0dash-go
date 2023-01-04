package codegen

import (
	"fmt"
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

// 目的コード (命令語) の実行
func Execute(fptex *os.File) {
	var stack [MAXMEM]int     // 実行時スタック
	var display [MAXLEVEL]int // 現在見える各ブロックの先頭番地のディスプレイ
	var pc, top, lev int
	var i inst // 実行する命令語
	fmt.Println("start execution")
	top = 0        // 次にスタックに入れる場所
	pc = 0         // 命令語のカウンタ
	stack[0] = 0   // stack[top] は callee で壊すディスプレイの退避場所
	stack[1] = 0   // stack[top+1] は caller への戻り番地
	display[0] = 0 // 主ブロックの先頭番地は 0
	for {
		i = code[pc] // これから実行する命令語
		pc++
		switch i.opCode {
		case Lit:
			stack[top] = i.u.value
			top++
		case Lod:
			stack[top] = stack[display[i.u.addr.Level]+i.u.addr.Addr]
		case Sto:
			top--
			stack[display[i.u.addr.Level]+i.u.addr.Addr] = stack[top]
		case Cal:
			// i.u.addr.Level は callee の名前のレベル、 callee のブロックのレベル lev はそれに +1 したもの
			lev = i.u.addr.Level + 1
			stack[top] = display[lev] // display[lev] の退避
			stack[top+1] = pc
			display[lev] = top // 現在の top が callee のブロックの先頭番地
			pc = i.u.addr.Addr
		case Ret:
			top--
			temp := stack[top]                   // スタックのトップにあるものが返す値
			top = display[i.u.addr.Level]        // top を呼ばれたときの値に戻す
			display[i.u.addr.Level] = stack[top] // 壊したディスプレイの回復
			pc = stack[top+1]
			top -= i.u.addr.Addr // 実引数の分だけ top を戻す
			stack[top] = temp    // 返す値をスタックの top へ
			top++
		case Ict:
			top += i.u.value
			if top >= MAXMEM-MAXREG {
				getsource.ErrorF("stack overflow", fptex)
			}
		case Jmp:
			pc = i.u.value
		case Opr:
			switch i.u.optr {
			case Neg:
				stack[top-1] = -stack[top-1]
				continue
			case Add:
				top--
				stack[top-1] += stack[top]
				continue
			case Sub:
				top--
				stack[top-1] -= stack[top]
				continue
			case Mul:
				top--
				stack[top-1] *= stack[top]
				continue
			case Div:
				top--
				stack[top-1] /= stack[top]
				continue
			case Odd:
				stack[top-1] = stack[top-1] & 1 // 奇数判定のための論理積演算
				continue
			case Eq:
				top--
				stack[top-1] = bool2int(stack[top-1] == stack[top])
				continue
			case Ls:
				top--
				stack[top-1] = bool2int(stack[top-1] < stack[top])
				continue
			case Gr:
				top--
				stack[top-1] = bool2int(stack[top-1] > stack[top])
				continue
			case Neq:
				top--
				stack[top-1] = bool2int(stack[top-1] != stack[top])
				continue
			case Lseq:
				top--
				stack[top-1] = bool2int(stack[top-1] <= stack[top])
				continue
			case Greq:
				top--
				stack[top-1] = bool2int(stack[top-1] >= stack[top])
				continue
			case Wrt:
				top--
				fmt.Printf("%d ", stack[top])
				continue
			case Wrl:
				fmt.Println("")
				continue
			}
		}
		if pc == 0 {
			break
		}
	}
}

func bool2int(b bool) int {
	if b {
		return 1
	}
	return 0
}
