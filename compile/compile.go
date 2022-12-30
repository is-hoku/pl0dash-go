package compile

import (
	"bufio"
	"errors"
	"fmt"
	"os"

	"github.com/is-hoku/pl0dash-go/codegen"
	"github.com/is-hoku/pl0dash-go/getsource"
	"github.com/is-hoku/pl0dash-go/table"
)

const MINERROR int = 3    // エラーがこれ以下なら実行
const FIRSTADDR int = 2   // 各ブロックの最初の変数のアドレス
var token getsource.Token // 次のトークンを入れておく

func Compile(fptex *os.File, scanner *bufio.Scanner) error {
	fmt.Println("start compilation")
	getsource.InitSource(fptex)                 // getsource の初期設定
	token = getsource.NextToken(scanner, fptex) // 最初のトークン
	table.BlockBegin(FIRSTADDR)                 // これ以後の宣言は新しいブロックのもの
	block(0, scanner, fptex)                    // 0 はダミー (主ブロックの関数名はない)
	getsource.FinalSource(fptex)
	i := getsource.ErrorN() // エラーメッセージの個数
	if i != 0 {
		return errors.New(fmt.Sprintf("the number of error is %d", i))
	}
	if i < MINERROR {
		return errors.New("too many errors")
	}
	return nil
}

// pIndex はこのブロックの関数名のインデックス
func block(pIndex int, scanner *bufio.Scanner, fptex *os.File) {
	backP := codegen.GenCodeV(codegen.Jmp, 0, fptex) // 内部関数を飛び越す命令、後でバックパッチ
	for {
		switch token.Kind { // 宣言部のコンパイルをくりかえす
		case getsource.Const: // 定数宣言部
			token = getsource.NextToken(scanner, fptex)
			constDecl(scanner, fptex)
			continue
		case getsource.Var: //変数宣言部
			token = getsource.NextToken(scanner, fptex)
			varDecl(scanner, fptex)
			continue
		case getsource.Func: // 関数宣言部
			token = getsource.NextToken(scanner, fptex)
			funcDecl(scanner, fptex)
			continue
		default:
			break
		}
		break
	}
	codegen.BackPatch(backP)                       // 内部関数を飛び越す命令にパッチ
	changeV(pIndex, codegen.NextCode())            // この関数の開始番地を修正
	codegen.GenCodeV(codegen.Ict, frameL(), fptex) // このブロックの実行時の必要記憶域を取る命令
	statement()                                    // このブロックの主文
	genCodeR()                                     // リターン命令
	blockEnd()                                     // ブロックが終わったことを table に連絡
}

// 定数宣言
func constDecl(scanner *bufio.Scanner, fptex *os.File) {
	var temp getsource.Token
	for {
		if token.Kind == getsource.Id {
			getsource.SetIdKind(table.ConstID) // 印字のための情報セット
			temp = token
			token = getsource.CheckGet(getsource.NextToken(scanner, fptex), getsource.Equal, scanner, fptex) // 名前の次は = のはず
			if token.Kind == getsource.Num {
				table.EnterTconst(temp.U.ID, token.U.Value, fptex) // 定数名と値をテーブルに
			} else {
				getsource.ErrorType("number", fptex)
			}
			token = getsource.NextToken(scanner, fptex)
		} else {
			getsource.ErrorMissingID(fptex)
		}
		if token.Kind != getsource.Comma { // 次がコンマなら定数宣言が続く
			if token.Kind == getsource.Id { // 次が名前ならコンマを忘れたことにする
				getsource.ErrorInsert(getsource.Comma, fptex)
				continue
			} else {
				break
			}
		}
		token = getsource.NextToken(scanner, fptex)
	}
	token = getsource.CheckGet(token, getsource.Semicolon, scanner, fptex) // 最後は ; のはず
}

// 変数宣言
func varDecl(scanner *bufio.Scanner, fptex *os.File) {
	for {
		if token.Kind == getsource.Id {
			getsource.SetIdKind(table.VarID)   // 印字のための情報セット
			table.EnterTvar(token.U.ID, fptex) // 変数名をテーブルに、番地は table が決める
			token = getsource.NextToken(scanner, fptex)
		} else {
			getsource.ErrorMissingID(fptex)
		}
		if token.Kind != getsource.Comma { // 次がコンマなら変数宣言が続く
			if token.Kind == getsource.Id { // 次が名前ならコンマを忘れたことにする
				getsource.ErrorInsert(getsource.Comma, fptex)
				continue
			} else {
				break
			}
		}
		token = getsource.NextToken(scanner, fptex)
	}
	token = getsource.CheckGet(token, getsource.Semicolon, scanner, fptex) // 最後は ; のはず
}

// 関数宣言のコンパイル
func funcDecl(scanner *bufio.Scanner, fptex *os.File) {
	if token.Kind == getsource.Id {
		getsource.SetIdKind(table.FuncID) // 印字のための情報セット
		// 関数名をテーブルに登録
		// その先頭番地は次のコードの番地 NextCode()
		fIndex := table.EnterTfunc(token.U.ID, codegen.NextCode(), fptex)
		token = getsource.CheckGet(getsource.NextToken(scanner, fptex), getsource.Lparen, scanner, fptex)
		blockBegin(FIRSTADDR) // パラメタ名のレベルは関数のブロックと同じ
		for {
			if token.Kind == getsource.Id { // パラメタ名がある場合
				getsource.SetIdKind(table.ParID)   // 印字のための情報セット
				table.EnterTpar(token.U.ID, fptex) // パラメタ名をテーブルに登録
				token = getsource.NextToken(scanner, fptex)
			} else {
				break
			}
			if token.Kind != getsource.Comma { // 次がコンマならパラメタ名が続く
				if token.Kind == getsource.Id { // 次が名前ならコンマを忘れたことに
					getsource.ErrorInsert(getsource.Comma, fptex)
					continue
				} else {
					break
				}
			}
			token = getsource.NextToken(scanner, fptex)
		}
		token = getsource.CheckGet(token, getsource.Rparen, scanner, fptex) // 最後は ) のはず
		table.Endpar()                                                      // パラメタ部が終わったことをテーブルに連絡
		if token.Kind == getsource.Semicolon {
			getsource.ErrorDelete(fptex)
			token = getsource.NextToken(scanner, fptex)
		}
		block(fIndex, scanner, fptex)                                          // ブロックのコンパイル、その関数名のインデックスを渡す
		token = getsource.CheckGet(token, getsource.Semicolon, scanner, fptex) // 最後は ; のはず
	} else {
		getsource.ErrorMissingID(fptex) // 関数名がない
	}
}

// 文のコンパイル
func statement() {
}
