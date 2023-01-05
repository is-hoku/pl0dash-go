package table

import (
	"os"

	"github.com/is-hoku/pl0dash-go/getsource"
)

const MAXLEVEL int = 5   // ブロックの最大深さ
const MAXTABLE int = 100 // 名前表の最大長さ

var nameTable [MAXTABLE]getsource.TableE // 名前表
var tIndex int = 0                       // 名前表のインデックス
var level int = -1                       // 現在のブロックレベル
var index [MAXLEVEL]int                  // index[i] にはブロックレベル i の最後のインデックス
var addr [MAXLEVEL]int                   // addr[i] にはブロックレベル i の最後の変数の番地
var tfIndex int                          // 名前表の関数を保持しているインデックス (一時)
var localAddr int                        // 現在のブロックの最後の変数の番地
// 引数付き関数は引数、関数、関数内の変数の順番で実行時にスタックされるため、ブロックのデータ領域には退避領域、 RetAdr, a, b, c の順でスタックされることを考慮すると top (スタックの最後尾)  が指すところから 2 番地目から変数がある

//const (
//	VarID getsource.KindT = iota
//	FuncID
//	ParID
//	ConstID
//)

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
	if level == 0 {
		tIndex = 0
		localAddr = 0
		return
	}
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
	if level == 0 {
		return 0 // 主ブロックにはパラメタがない
	}
	return nameTable[index[level-1]].U.F.Pars
}

func enterT(id string, fptex *os.File) { // 名前表に名前を登録
	if tIndex < MAXTABLE {
		tIndex++
		nameTable[tIndex].Name = id
	} else {
		getsource.ErrorF("too many names", fptex)
	}
}

// 名前表に関数名と先頭番地を登録
func EnterTfunc(id string, v int, fptex *os.File) int {
	enterT(id, fptex)
	nameTable[tIndex].Kind = getsource.FuncID
	nameTable[tIndex].U.F.Raddr.Level = level
	nameTable[tIndex].U.F.Raddr.Addr = v // 関数の先頭番地 (目的コード)
	nameTable[tIndex].U.F.Pars = 0       // パラメタ数の初期値
	tfIndex = tIndex                     // 関数名のインデックスを一時保持
	//fmt.Println("nameTable", tIndex, nameTable[tIndex].Kind, nameTable[tIndex-1].Name, nameTable[tIndex].U.F.Raddr.Level, nameTable[tIndex].U.F.Raddr.Addr)
	return tIndex
}

// 名前表にパラメタ名を登録
func EnterTpar(id string, fptex *os.File) int {
	enterT(id, fptex)
	nameTable[tIndex].Kind = getsource.ParID
	nameTable[tIndex].U.Raddr.Level = level
	nameTable[tfIndex].U.F.Pars++ // 関数のパラメタ数のカウント
	//fmt.Println("nameTable", tIndex, nameTable[tIndex].Kind, nameTable[tIndex-1].Name, nameTable[tIndex].U.F.Raddr.Level, nameTable[tfIndex].U.F.Pars)
	return tIndex
}

// 名前表に変数名を登録
func EnterTvar(id string, fptex *os.File) int {
	enterT(id, fptex)
	nameTable[tIndex].Kind = getsource.VarID
	nameTable[tIndex].U.Raddr.Level = level
	nameTable[tIndex].U.Raddr.Addr = localAddr // localAddr はブロックの最初の変数の番地 (はじめは 2)
	localAddr++
	return tIndex
}

// 名前表に定数名とその値を登録
func EnterTconst(id string, v int, fptex *os.File) int {
	enterT(id, fptex)
	nameTable[tIndex].Kind = getsource.ConstID
	nameTable[tIndex].U.Value = v
	return tIndex
}

// パラメタ宣言部の最後で呼ばれる
func Endpar() {
	pars := nameTable[tfIndex].U.F.Pars // 関数のパラメタ数
	if pars == 0 {
		return
	}
	for i := 1; i <= pars; i++ { // 各パラメタの番地を求める
		nameTable[tfIndex+i].U.Raddr.Addr = i - 1 - pars
	}
}

// 名前表 [ti] の値 (関数の先頭番地) の変更
func ChangeV(ti int, newVal int) {
	nameTable[ti].U.F.Raddr.Addr = newVal
}

func SearchT(id string, k getsource.KindT, fptex *os.File) int {
	var i int
	nameTable[0].Name = id // 番兵を立てる
	for i = tIndex; id != nameTable[i].Name; i-- {
	}
	if i != 0 { // 名前があった
		return i
	} else { // 名前がなかった
		getsource.ErrorType("undef", fptex)
		if k == getsource.VarID {
			return EnterTvar(id, fptex) // 変数名の時は仮登録
		}
		return 0
	}
}

// 名前表 [i] の種類を返す
func RetKindT(i int) getsource.KindT {
	return nameTable[i].Kind
}

// 名前表 [ti] のアドレスを返す
func RetRelAddr(ti int) getsource.RelAddr {
	switch nameTable[ti].Kind {
	case getsource.VarID:
		return nameTable[ti].U.Raddr
	case getsource.FuncID:
		return nameTable[ti].U.F.Raddr
	case getsource.ParID:
		return nameTable[ti].U.Raddr
	case getsource.ConstID:
		return nameTable[ti].U.Raddr
	default:
		return nameTable[ti].U.Raddr
	}
}

// 名前表 [ti] の value を返す
func RetVal(ti int) int {
	return nameTable[ti].U.Value
}

// 名前表 [ti] の関数のパラメタ数を返す
func RetPars(ti int) int {
	return nameTable[ti].U.F.Pars
}

// そのブロックで実行時に必要とするメモリ容量
func RetFrameL() int {
	return localAddr
}
