package table

import (
	"bytes"
	"os"

	"github.com/is-hoku/pl0dash-go/getsource"
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
var tfIndex int                // 名前表の関数を保持しているインデックス (一時)
var localAddr int              // 現在のブロックの最後の変数の番地
// 引数付き関数は引数、関数、関数内の変数の順番で実行時にスタックされるため、ブロックのデータ領域には退避領域、 RetAdr, a, b, c の順でスタックされることを考慮すると top (スタックの最後尾)  が指すところから 2 番地目から変数がある

const (
	VarID KindT = iota
	FuncID
	ParID
	ConstID
)

// ブロックの始まり (最初の変数の番地) で呼ばれる
func BlockBegin(firstAddr int, fptex *os.File) {
	if level == -1 { // 主ブロックの初期設定
		localAddr = firstAddr
		tIndex = 0
		level++
		return
	}
	if level == MAXLEVEL-1 {
		getsource.ErrorF("too many nested blocks", fptex)
	}
	index[level] = tIndex // 今までのブロックの情報を格納
	addr[level] = localAddr
	localAddr = firstAddr // 新しいブロックの最初の変数の番地
	level++               // 新しいブロックのレベル
	return
}

// ブロックの終わりで呼ばれる
func BlockEnd() {
	level--
	tIndex = index[level] // 一つ外側のブロックの情報を回復
	localAddr = addr[level]
}

// 現ブロックのレベルを返す
func BLevel() int {
	return level
}

// 現ブロックの関数のパラメタ数を返す
func FPars() int {
	return nameTable[index[level-1]].u.f.pars
}
func enterT(id []byte, fptex *os.File) { // 名前表に名前を登録
	if tIndex < MAXTABLE {
		copy(id, nameTable[tIndex].name[:])
		tIndex++
	} else {
		getsource.ErrorF("too many names", fptex)
	}
}

// 名前表に関数名と先頭番地を登録
func EnterTfunc(id []byte, v int, fptex *os.File) int {
	enterT(id, fptex)
	nameTable[tIndex].kind = FuncID
	nameTable[tIndex].u.f.raddr.level = level
	nameTable[tIndex].u.f.raddr.addr = v // 関数の先頭番地 (目的コード)
	nameTable[tIndex].u.f.pars = 0       // パラメタ数の初期値
	tfIndex = tIndex                     // 関数名のインデックスを一時保持
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
	nameTable[tIndex].u.raddr.addr = localAddr // localAddr はブロックの最初の変数の番地 (はじめは 2)
	localAddr++
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

// 名前表 [ti] の値 (関数の先頭番地) の変更
func ChangeV(ti int, newVal int) {
	nameTable[ti].u.f.raddr.addr = newVal
}

func SearchT(id []byte, k KindT, fptex *os.File) int {
	var i int
	copy(id, nameTable[0].name[:]) // 番兵を立てる
	for i = tIndex; bytes.Equal(id, nameTable[i].name[:]) == false; i-- {
	}
	if i != 0 { // 名前があった
		return i
	} else { // 名前がなかった
		getsource.ErrorType("undef", fptex)
		if k == VarID {
			return EnterTvar(id, fptex) // 変数名の時は仮登録
		}
		return 0
	}
}

// 名前表 [i] の種類を返す
func RetKindT(i int) KindT {
	return nameTable[i].kind
}

// 名前表 [ti] のアドレスを返す
func RetRelAddr(ti int) RelAddr {
	return nameTable[ti].u.raddr
}

// 名前表 [ti] の value を返す
func RetVal(ti int) int {
	return nameTable[ti].u.value
}

// 名前表 [ti] の関数のパラメタ数を返す
func RetPars(ti int) int {
	return nameTable[ti].u.f.pars
}

// そのブロックで実行時に必要とするメモリ容量
func RetFrameL() int {
	return localAddr
}
