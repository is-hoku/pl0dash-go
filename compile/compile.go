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
	table.BlockBegin(FIRSTADDR, fptex)          // これ以後の宣言は新しいブロックのもの
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
	codegen.BackPatch(backP)                                // 内部関数を飛び越す命令にパッチ
	table.ChangeV(pIndex, codegen.NextCode())               // この関数の開始番地を修正
	codegen.GenCodeV(codegen.Ict, table.RetFrameL(), fptex) // このブロックの実行時の必要記憶域を取る命令
	statement(scanner, fptex)                               // このブロックの主文
	codegen.GenCodeR(fptex)                                 // リターン命令
	table.BlockEnd()                                        // ブロックが終わったことを table に連絡
}

// 定数宣言
func constDecl(scanner *bufio.Scanner, fptex *os.File) {
	var temp getsource.Token
	for {
		if token.Kind == getsource.Id {
			getsource.SetIdKind(getsource.ConstID) // 印字のための情報セット
			temp = token
			token = getsource.CheckGet(getsource.NextToken(scanner, fptex), getsource.Equal, scanner, fptex) // 名前の次は = のはず
			if token.Kind == getsource.Num {
				table.EnterTconst(temp.U.ID[:], token.U.Value, fptex) // 定数名と値をテーブルに
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
			getsource.SetIdKind(getsource.VarID)  // 印字のための情報セット
			table.EnterTvar(token.U.ID[:], fptex) // 変数名をテーブルに、番地は table が決める
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
		getsource.SetIdKind(getsource.FuncID) // 印字のための情報セット
		// 関数名をテーブルに登録
		// その先頭番地は次のコードの番地 NextCode()
		fIndex := table.EnterTfunc(token.U.ID[:], codegen.NextCode(), fptex)
		token = getsource.CheckGet(getsource.NextToken(scanner, fptex), getsource.Lparen, scanner, fptex)
		table.BlockBegin(FIRSTADDR, fptex) // パラメタ名のレベルは関数のブロックと同じ
		for {
			if token.Kind == getsource.Id { // パラメタ名がある場合
				getsource.SetIdKind(getsource.ParID)  // 印字のための情報セット
				table.EnterTpar(token.U.ID[:], fptex) // パラメタ名をテーブルに登録
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
func statement(scanner *bufio.Scanner, fptex *os.File) {
	var tIndex int
	var k getsource.KindT
	var backP, backP2 int // バックパッチ用
	for {
		switch token.Kind {
		case getsource.Id: // 代入文のコンパイル
			tIndex = table.SearchT(token.U.ID[:], getsource.VarID, fptex)
			k = table.RetKindT(tIndex)
			getsource.SetIdKind(k)                                // 印字のための情報セット
			if (k != getsource.VarID) && (k != getsource.ParID) { // 変数名かパラメタ名のはず
				getsource.ErrorType("var/par", fptex)
			}
			token = getsource.CheckGet(getsource.NextToken(scanner, fptex), getsource.Assign, scanner, fptex) // := のはず
			expression(scanner, fptex)                                                                        // 式のコンパイル
			codegen.GenCodeT(codegen.Sto, tIndex, fptex)                                                      // 左辺への代入命令
			return
		case getsource.If: // if 文のコンパイル
			token = getsource.NextToken(scanner, fptex)
			condition(scanner, fptex)                                         // 条件式のコンパイル
			token = getsource.CheckGet(token, getsource.Then, scanner, fptex) // then のはず
			backP = codegen.GenCodeV(codegen.Jpc, 0, fptex)                   // jpc 命令
			statement(scanner, fptex)                                         // 文のコンパイル
			codegen.BackPatch(backP)                                          // 上の jpc 命令にバックパッチ
			return
		case getsource.Ret: // return 文のコンパイル
			token = getsource.NextToken(scanner, fptex)
			expression(scanner, fptex) // 式のコンパイル
			codegen.GenCodeR(fptex)    // ret 命令
			return
		case getsource.Begin:
			token = getsource.NextToken(scanner, fptex)
			for {
				statement(scanner, fptex) // 文のコンパイル
				for {
					if token.Kind == getsource.Semicolon { // 次が ; なら文が続く
						token = getsource.NextToken(scanner, fptex)
						break
					}
					if token.Kind == getsource.End { // 次が end なら終わり
						token = getsource.NextToken(scanner, fptex)
						return
					}
					if isStBeginKey(token) == 1 { // 次が文の先頭記号なら
						getsource.ErrorInsert(getsource.Semicolon, fptex) // ; を忘れたことにする
						break
					}
					getsource.ErrorDelete(fptex) // それ以外ならエラーとして読み捨てる
					token = getsource.NextToken(scanner, fptex)
				}
			}
		case getsource.While: // while 文のコンパイル
			token = getsource.NextToken(scanner, fptex)
			backP2 = codegen.NextCode()                                     // while 文の最後の jmp 命令の飛び先
			condition(scanner, fptex)                                       // 条件式のコンパイル
			token = getsource.CheckGet(token, getsource.Do, scanner, fptex) // do のはず
			backP = codegen.GenCodeV(codegen.Jpc, 0, fptex)                 // 条件式が偽の時飛び出す jpc 命令
			statement(scanner, fptex)                                       // 文のコンパイル
			codegen.GenCodeV(codegen.Jmp, backP2, fptex)                    // while 文の先頭へのジャンプ命令
			codegen.BackPatch(backP)                                        // 偽の時飛び出す jpc 命令へのバックパッチ
			return
		case getsource.Write: // write 文のコンパイル
			token = getsource.NextToken(scanner, fptex)
			codegen.GenCodeO(codegen.Wrl, fptex) // 改行を出力する wrl 命令
			return
		case getsource.End: // 空文を読んだことにして終わり
			return
		case getsource.Semicolon: // 空文を読んだことにして終わり
			return
		default: // 文の先頭のキーまで読み捨てる
			getsource.ErrorDelete(fptex) // 今読んだトークンを読み捨てる
			token = getsource.NextToken(scanner, fptex)
			continue
		}
	}
}

// トークン t は文の先頭のキーか？
func isStBeginKey(t getsource.Token) int {
	switch t.Kind {
	case getsource.If:
		fallthrough
	case getsource.Begin:
		fallthrough
	case getsource.Ret:
		fallthrough
	case getsource.While:
		fallthrough
	case getsource.Write:
		fallthrough
	case getsource.WriteLn:
		return 1
	default:
		return 0
	}
}

// 式のコンパイル
func expression(scanner *bufio.Scanner, fptex *os.File) {
	k := token.Kind
	if k == getsource.Plus || k == getsource.Minus {
		token = getsource.NextToken(scanner, fptex)
		term(scanner, fptex)
		if k == getsource.Minus {
			codegen.GenCodeO(codegen.Neg, fptex)
		}
	} else {
		term(scanner, fptex)
	}
	k = token.Kind
	for k == getsource.Plus || k == getsource.Minus {
		token = getsource.NextToken(scanner, fptex)
		term(scanner, fptex)
		if k == getsource.Minus {
			codegen.GenCodeO(codegen.Sub, fptex)
		} else {
			codegen.GenCodeO(codegen.Add, fptex)
		}
		k = token.Kind
	}
}

// 式の項のコンパイル
func term(scanner *bufio.Scanner, fptex *os.File) {
	factor(scanner, fptex)
	k := token.Kind
	for k == getsource.Mult || k == getsource.Div {
		token = getsource.NextToken(scanner, fptex)
		factor(scanner, fptex)
		if k == getsource.Mult {
			codegen.GenCodeO(codegen.Mul, fptex)
		} else {
			codegen.GenCodeO(codegen.Div, fptex)
		}
		k = token.Kind
	}
}

// 式の因子のコンパイル
func factor(scanner *bufio.Scanner, fptex *os.File) {
	var tIndex, i int
	var k getsource.KindT
	if token.Kind == getsource.Id {
		tIndex = table.SearchT(token.U.ID[:], getsource.VarID, fptex)
		k = table.RetKindT(tIndex)
		getsource.SetIdKind(table.RetKindT(tIndex)) // 印字のための情報セット
		switch k {
		case getsource.VarID:
			fallthrough
		case getsource.ParID: // 変数名かパラメタ名
			codegen.GenCodeT(codegen.Lod, tIndex, fptex)
			token = getsource.NextToken(scanner, fptex)
			break
		case getsource.ConstID: // 定数名
			codegen.GenCodeV(codegen.Lit, table.RetVal(tIndex), fptex)
			token = getsource.NextToken(scanner, fptex)
			break
		case getsource.FuncID: // 関数呼び出し
			token = getsource.NextToken(scanner, fptex)
			if token.Kind == getsource.Lparen {
				i = 0 // i は実引数の個数
				token = getsource.NextToken(scanner, fptex)
				if token.Kind != getsource.Rparen {
					for {
						expression(scanner, fptex)
						i++ // 実引数のコンパイル
						if token.Kind == getsource.Comma {
							token = getsource.NextToken(scanner, fptex)
							continue
						}
						token = getsource.CheckGet(token, getsource.Rparen, scanner, fptex)
						break
					}
				} else {
					token = getsource.NextToken(scanner, fptex)
				}
				if table.RetPars(tIndex) != i { // RetPars(tIndex) は仮引数の個数
					getsource.ErrorMessage("\\#par", fptex)
				}
			} else {
				getsource.ErrorInsert(getsource.Lparen, fptex)
				getsource.ErrorInsert(getsource.Rparen, fptex)
			}
			codegen.GenCodeT(codegen.Cal, tIndex, fptex) // call 命令
			break
		}
	} else if token.Kind == getsource.Num { // 定数
		codegen.GenCodeV(codegen.Lit, token.U.Value, fptex)
		token = getsource.NextToken(scanner, fptex)
	} else if token.Kind == getsource.Lparen { // (, 因子, )
		token = getsource.NextToken(scanner, fptex)
		expression(scanner, fptex)
		token = getsource.CheckGet(token, getsource.Rparen, scanner, fptex)
	}
	switch token.Kind { // 因子の後がまた因子ならエラー
	case getsource.Id:
		fallthrough
	case getsource.Num:
		fallthrough
	case getsource.Lparen:
		getsource.ErrorMissingOp(fptex)
		factor(scanner, fptex)
	default:
		return
	}
}

// 条件式のコンパイル
func condition(scanner *bufio.Scanner, fptex *os.File) {
	var k getsource.KeyID
	if token.Kind == getsource.Odd {
		token = getsource.NextToken(scanner, fptex)
		expression(scanner, fptex)
		codegen.GenCodeO(codegen.Odd, fptex)
	} else {
		expression(scanner, fptex)
		k = token.Kind
		switch k {
		case getsource.Equal:
			fallthrough
		case getsource.Gtr:
			fallthrough
		case getsource.NotEq:
			fallthrough
		case getsource.LssEq:
			fallthrough
		case getsource.GtrEq:
			break
		default:
			getsource.ErrorType("rel-op", fptex)
			break
		}
	}
	token = getsource.NextToken(scanner, fptex)
	expression(scanner, fptex)
	switch k {
	case getsource.Equal:
		codegen.GenCodeO(codegen.Eq, fptex)
	case getsource.Lss:
		codegen.GenCodeO(codegen.Ls, fptex)
	case getsource.Gtr:
		codegen.GenCodeO(codegen.Gr, fptex)
	case getsource.NotEq:
		codegen.GenCodeO(codegen.Neq, fptex)
	case getsource.LssEq:
		codegen.GenCodeO(codegen.Lseq, fptex)
	case getsource.GtrEq:
		codegen.GenCodeO(codegen.Greq, fptex)
	}
}
