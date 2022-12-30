package table

import (
	"fmt"
	"os"
)

const MAXNAME int = 31   // 名前の最大の長さ
const MAXLEVEL int = 5   // ブロックの最大深さ
const MAXTABLE int = 100 // 名前表の最大長さ

type KindT int        // Identifier の種類
type RelAddr struct { // 変数・パラメタ・関数のアドレスの型
	level int
	addr  int
}
type tableE struct {
	kind KindT         // 名前の種類
	name [MAXNAME]byte // 名前のつづり
	u    struct {
		value int // 定数の場合：値
		f     struct {
			raddr RelAddr // 関数の場合：先頭アドレス
			pars  int     // 関数の場合：パラメタ数
		}
		raddr RelAddr // 変数・パラメタの場合：アドレス
	}
}

var nameTable [MAXTABLE]tableE // 名前表
var tIndex int = 0             // 名前表のインデックス
var level int = -1             // 現在のブロックレベル
var index [MAXLEVEL]int        // index[i] にはブロックレベル i の最後のインデックス
var addr [MAXLEVEL]int         // addr[i] にはブロックレベル i の最後の変数の番地
var tfIndex int
var localAddr int // 現在のブロックの最後の変数の番地

const (
	VarID KindT = iota
	FuncID
	ParID
	ConstID
)

func enterT(id []byte, fptex *os.File) { // 名前表に名前を登録
	if tIndex < MAXTABLE {
		tIndex++
		copy(id, nameTable[tIndex].name[:])
	} else {
		errorF("too many names", fptex)
	}
}

// 名前表に関数名と先頭番地を登録
func EnterTfunc(id []byte, v int, fptex *os.File) int {
	enterT(id, fptex)
	nameTable[tIndex].kind = FuncID
	nameTable[tIndex].u.f.raddr.level = level
	nameTable[tIndex].u.f.raddr.addr = v // 関数の先頭番地
	nameTable[tIndex].u.f.pars = 0       // パラメタ数の初期値
	tfIndex = tIndex
	return tIndex
}

// 名前表にパラメタ名を登録
func EnterTpar(id []byte, fptex *os.File) int {
	enterT(id, fptex)
	nameTable[tIndex].kind = ParID
	nameTable[tIndex].u.raddr.level = level
	nameTable[tfIndex].u.f.pars++ // 関数のパラメタ数のカウント
	return tIndex
}

// 名前表に変数名を登録
func EnterTvar(id []byte, fptex *os.File) int {
	enterT(id, fptex)
	nameTable[tIndex].kind = VarID
	nameTable[tIndex].u.raddr.level = level
	localAddr++
	nameTable[tIndex].u.raddr.addr = localAddr
	return tIndex
}

// 名前表に定数名とその値を登録
func EnterTconst(id []byte, v int, fptex *os.File) int {
	enterT(id, fptex)
	nameTable[tIndex].kind = ConstID
	nameTable[tIndex].u.value = v
	return tIndex
}

// パラメタ宣言部の最後で呼ばれる
func Endpar() {
	pars := nameTable[tfIndex].u.f.pars // 関数のパラメタ数
	if pars == 0 {
		return
	}
	for i := 1; i <= pars; i++ { // 各パラメタの番地を求める
		nameTable[tfIndex+i].u.raddr.addr = i - 1 - pars
	}
}

func BlockBegin(firstAddr int) { // ブロックの始まり (最初の変数の番地) で呼ばれる
	if level == -1 {

	}
}

// エラーが多いと終了 (panic)
func errorNocheck(fptex *os.File) {
	errorNo++
	if errorNo > MAXERROR {
		fptex.WriteString("too many errors\n\\end{document}\n")
		panic("abort compilation")
	}
}

var errorNo int = 0     // 出力したエラーの数
const MAXERROR int = 30 // これ以上のエラーがあると終了

// エラーメッセージを .tex ファイルに出力
func errorMessage(m string, fptex *os.File) {
	fptex.WriteString(fmt.Sprintf("$^{%s}$", m))
	errorNocheck(fptex)
}

// エラーメッセージを出力しコンパイル終了
func errorF(m string, fptex *os.File) {
	errorMessage(m, fptex)
	fptex.WriteString("fatal errors\n\\end{document}\n")
	if errorNo != 0 {
		fmt.Printf("total %d errors\n", errorNo)
	}
	panic("abort compilation\n")
}
