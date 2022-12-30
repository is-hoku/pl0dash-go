package getsource

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"github.com/is-hoku/pl0dash-go/table"
)

const MAXLINE int = 120           // 1 行の最大文字数
const MAXERROR int = 30           // これ以上のエラーがあると終了
const MAXNAME int = 31            // 名前の最大の長さ
const MAXNUM int = 14             // 定数の最大桁数
const TAB int = 5                 // タブのスペース
const INSERT_C string = "#0000FF" // 挿入文字の色
const DELETE_C string = "#FF0000" // 削除文字の色
const TYPE_C string = "#00FF00"   // タイプエラー文字の色

// var fpi *os.File           // ソースファイル
// var scanner *bufio.Scanner // ファイルを 1 行ずつ読むスキャナ
// var fptex *os.File     // LaTex 出力ファイル
var line [MAXLINE]byte // 1 行分の入力バッファ
var lineIndex int      // 次に読む文字の位置
var ch byte            // 最後に読んだ文字
var cToken Token       // 最後に読んだトークン
var idKind table.KindT // 現トークンの種類
var spaces int         // そのトークンの前のスペースの個数
var cr int             // その前の CR の個数
var printed bool       // トークンは印字済みか
var errorNo int = 0    // 出力したエラーの数

type KeyID int // キーの文字の種類

const (
	Begin KeyID = iota
	End
	If
	Then
	While
	Do
	Ret
	Func
	Var
	Const
	Odd
	Write
	WriteLn
	End_of_KeyWd // 予約語の名前はここまで
	Plus
	Minus
	Mult
	Div
	Lparen
	Rparen
	Equal
	Lss
	Gtr
	NotEq
	LssEq
	GtrEq
	Comma
	Period
	Semicolon
	Assign
	End_of_KeySym // 演算子と区切り記号の名前はここまで
	Id
	Num
	Nul
	End_of_Token
	Letter
	Digit
	Colon
	Others
)

type IDVal struct {
	ID    [MAXNAME]byte
	Value int
}

type Token struct {
	Kind KeyID
	U    IDVal
}

// 予約語や記号と名前
type keyWd struct {
	word  string
	keyID KeyID
}

// 予約語や記号と名前の表
var keyWdT []*keyWd = []*keyWd{
	{"begin", Begin},
	{"end", End},
	{"if", If},
	{"then", Then},
	{"while", While},
	{"do", Do},
	{"return", Ret},
	{"function", Func},
	{"var", Var},
	{"const", Const},
	{"odd", Odd},
	{"write", Write},
	{"writeln", WriteLn},
	{"$dummy1", End_of_KeyWd},
	{"+", Plus},
	{"-", Minus},
	{"*", Mult},
	{"/", Div},
	{"(", Lparen},
	{")", Rparen},
	{"=", Equal},
	{"<", Lss},
	{">", Gtr},
	{"<>", NotEq},
	{"<=", LssEq},
	{">=", GtrEq},
	{",", Comma},
	{".", Period},
	{";", Semicolon},
	{":=", Assign},
	{"$dummy2", End_of_KeySym},
}

// キーは予約語か
func IsKeyWd(k KeyID) bool {
	return (k < End_of_KeyWd)
}

// キーは記号か
func IsKeySym(k KeyID) bool {
	if k < End_of_KeyWd {
		return false
	}
	return (k < End_of_KeySym)
}

// 文字の種類を示す表にする
var charClassT [256]KeyID

func initCharClassT() {
	var i int
	for i = 0; i < 256; i++ {
		charClassT[i] = Others
	}
	for i = '0'; i <= '9'; i++ {
		charClassT[i] = Digit
	}
	for i = 'A'; i <= 'Z'; i++ {
		charClassT[i] = Letter
	}
	for i = 'a'; i <= 'z'; i++ {
		charClassT[i] = Letter
	}
	charClassT['+'] = Plus
	charClassT['-'] = Minus
	charClassT['*'] = Mult
	charClassT['/'] = Div
	charClassT['('] = Lparen
	charClassT[')'] = Rparen
	charClassT['='] = Equal
	charClassT['<'] = Lss
	charClassT['>'] = Gtr
	charClassT[','] = Comma
	charClassT['.'] = Period
	charClassT[';'] = Semicolon
	charClassT[':'] = Colon
}

func OpenSource(fileName string) (*os.File, *os.File, *bufio.Scanner, error) {
	fpi, err := os.Open(fileName)
	if err != nil {
		return nil, nil, nil, err
	}
	texFileName := fileName + ".tex"
	fptex, err := os.Create(texFileName)
	if err != nil {
		return nil, nil, nil, err
	}
	scanner := bufio.NewScanner(fpi)
	return fpi, fptex, scanner, nil
}

func InitSource(fptex *os.File) {
	lineIndex = -1
	ch = '\n'
	printed = false
	initCharClassT()
	fptex.WriteString("\\documentstyle[12pt]{article}\n")
	fptex.WriteString("\\begin{document}\n")
	fptex.WriteString("\\fboxsep=Opt\n")
	fptex.WriteString("\\def\\insert#1{$\\fbox{#1}$}\n")
	fptex.WriteString("\\def\\delete#1{$\\fboxrule=.5mm\\fbox{#1}$}\n")
	fptex.WriteString("\\rm\n")
}

func FinalSource(fptex *os.File) {
	if cToken.Kind == Period {
		printcToken(fptex)
	} else {
		ErrorInsert(Period, fptex)
	}
	fptex.WriteString("\n\\end{document}\n")
}

// エラーが多いと終了 (panic)
func errorNocheck(fptex *os.File) {
	errorNo++
	if errorNo > MAXERROR {
		fptex.WriteString("too many errors\n\\end{document}\n")
		panic("abort compilation")
	}
}

// 型エラーを .tex ファイルに出力
func ErrorType(m string, fptex *os.File) {
	printSpaces(fptex)
	fptex.WriteString(fmt.Sprintf("\\(\\stackrel{\\mbox{\\scriptsize %s}}{\\mbox{", m))
	printcToken(fptex)
	fptex.WriteString("}}\\)")
	errorNocheck(fptex)
}

// keyString(k) を .tex ファイルに挿入
func ErrorInsert(k KeyID, fptex *os.File) {
	if k < End_of_KeyWd { // 予約語
		fptex.WriteString(fmt.Sprintf("\\ \\insert{{\\bf %s}}", keyWdT[k].word))
	} else { // 演算子か区切り記号
		fptex.WriteString(fmt.Sprintf("\\ \\insert{$%s$}", keyWdT[k].word))
	}
	errorNocheck(fptex)
}

// 名前が無いとのメッセージを .tex ファイルに挿入
func ErrorMissingID(fptex *os.File) {
	fptex.WriteString("\\insert{Id}")
	errorNocheck(fptex)
}

// 演算子が無いとのメッセージを .tex ファイルに挿入
func ErrorMissingOp(fptex *os.File) {
	fptex.WriteString("\\insert{$\\otimes$}")
	errorNocheck(fptex)
}

// 今読んだトークンを読み捨てる
func ErrorDelete(fptex *os.File) {
	i := cToken.Kind
	printSpaces(fptex)
	printed = true
	if i < End_of_KeyWd { // 予約語
		fptex.WriteString(fmt.Sprintf("\\delete{{\\bf %s}}", keyWdT[i].word))
	} else if i < End_of_KeySym { // 演算子か区切り記号
		fptex.WriteString(fmt.Sprintf("\\delete{$%s$}", keyWdT[i].word))
	} else if i == Id {
		fptex.WriteString(fmt.Sprintf("\\delete{%s}", cToken.U.ID))
	} else if i == Num {
		fptex.WriteString(fmt.Sprintf("\\delete{%s}", strconv.Itoa(cToken.U.Value)))
	}
}

// エラーメッセージを .tex ファイルに出力
func ErrorMessage(m string, fptex *os.File) {
	fptex.WriteString(fmt.Sprintf("$^{%s}$", m))
	errorNocheck(fptex)
}

// エラーメッセージを出力しコンパイル終了
func ErrorF(m string, fptex *os.File) {
	ErrorMessage(m, fptex)
	fptex.WriteString("fatal errors\n\\end{document}\n")
	if errorNo != 0 {
		fmt.Printf("total %d errors\n", errorNo)
	}
	panic("abort compilation\n")
}

// エラーの個数を返す
func ErrorN() int {
	return errorNo
}

// 次の 1 文字を返す
func nextChar(scanner *bufio.Scanner, fptex *os.File) byte {
	var ch byte
	if lineIndex == -1 {
		if (scanner.Scan()) && (len(scanner.Text()) <= 120) {
			copy(line[:], []byte(scanner.Text()))
			lineIndex = 0
		} else {
			ErrorF("end of file\n", fptex)
		}
	}
	if ch = line[lineIndex]; ch == '\n' {
		lineIndex = -1
		return '\n'
	}
	lineIndex++
	return ch
}

func NextToken(scanner *bufio.Scanner, fptex *os.File) Token {
	var i int = 0
	var num int
	var cc KeyID
	var temp Token
	var ident [MAXNAME]byte
	printcToken(fptex) // 前のトークンを印字
	spaces = 0
	cr = 0

	for { // 次のトークンまでの空白や改行をカウント
		if ch == ' ' {
			spaces++
		} else if ch == '\t' {
			spaces += TAB
		} else if ch == '\n' {
			spaces = 0
			cr++
		} else {
			break
		}
		ch = nextChar(scanner, fptex)
	}

	cc = charClassT[ch]
	switch cc {
	case Letter: // identifier
		for {
			if i < MAXNAME {
				ident[i] = ch
			}
			i++
			ch = nextChar(scanner, fptex)
			if charClassT[ch] == Letter || charClassT[ch] == Digit {
				continue
			}
			break
		}
		if i >= MAXNAME {
			ErrorMessage("too long token", fptex)
			i = MAXNAME - 1
		}
		for i = 0; i < int(End_of_KeyWd); i++ {
			// 予約語の場合
			if string(ident[:]) == keyWdT[i].word {
				temp.Kind = keyWdT[i].keyID
				cToken = temp
				printed = false
				return temp
			}
		}
		// ユーザの宣言した名前の場合
		temp.Kind = Id
		temp.U.ID = ident

	case Digit: // number
		num = 0
		for {
			num = 10*num + int(ch-'0')
			i++
			ch = nextChar(scanner, fptex)
			if charClassT[ch] == Digit {
				continue
			}
			break
		}
		if i > MAXNUM {
			ErrorMessage("too large number", fptex)
		}
		temp.Kind = Num
		temp.U.Value = num

	case Colon:
		if ch = nextChar(scanner, fptex); ch == '=' { // :=
			ch = nextChar(scanner, fptex)
			temp.Kind = Assign
		} else {
			temp.Kind = Nul
		}

	case Lss:
		if ch = nextChar(scanner, fptex); ch == '=' { // <=
			ch = nextChar(scanner, fptex)
			temp.Kind = LssEq
		} else if ch == '>' { // <>
			ch = nextChar(scanner, fptex)
			temp.Kind = NotEq
		} else {
			temp.Kind = Lss
		}

	case Gtr:
		if ch = nextChar(scanner, fptex); ch == '=' { // >=
			ch = nextChar(scanner, fptex)
			temp.Kind = GtrEq
		} else {
			temp.Kind = Gtr
		}

	default:
		temp.Kind = cc
		ch = nextChar(scanner, fptex)
	}

	cToken = temp
	printed = false
	return temp
}

// t.Kind == k のチェック
// t.Kind == k なら次のトークンを読んで返す
// t.Kind != k ならエラーメッセージを出す
// t, k が共に記号か予約語なら t を捨て次のトークンを読んで返す (t を k で置き換えたことになる)
// それ以外の場合、k を挿入したことにして t を返す
func CheckGet(t Token, k KeyID, scanner *bufio.Scanner, fptex *os.File) Token {
	if t.Kind == k {
		return NextToken(scanner, fptex)
	}
	if (IsKeyWd(t.Kind) && IsKeyWd(k)) || (IsKeySym(t.Kind) && IsKeySym(k)) {
		ErrorDelete(fptex)
		ErrorInsert(k, fptex)
		return NextToken(scanner, fptex)
	}
	ErrorInsert(k, fptex)
	return t
}

// 空白や改行の印字
func printSpaces(fptex *os.File) {
	for cr > 0 {
		cr--
		fptex.WriteString("\\ \\par\n")
	}
	for spaces > 0 {
		spaces--
		fptex.WriteString("\\ ")
	}
	cr = 0
	spaces = 0
}

// 現在のトークンの印字
func printcToken(fptex *os.File) {
	i := int(cToken.Kind)
	if printed {
		printed = false
		return
	}
	printed = true
	printSpaces(fptex)
	if i < int(End_of_KeyWd) { // 予約語
		fptex.WriteString(fmt.Sprintf("{\\bf %s}", keyWdT[i].word))
	} else if i < int(End_of_KeySym) { // 演算子か区切り記号
		fptex.WriteString(fmt.Sprintf("$%s$", keyWdT[i].word))
	} else if i == int(Id) { // Identfier
		switch idKind {
		case table.VarID:
			fptex.WriteString(fmt.Sprintf("%s", cToken.U.ID))
		case table.ParID:
			fptex.WriteString(fmt.Sprintf("{\\sl %s}", cToken.U.ID))
		case table.FuncID:
			fptex.WriteString(fmt.Sprintf("{\\it %s}", cToken.U.ID))
		case table.ConstID:
			fptex.WriteString(fmt.Sprintf("{\\sf %s}", cToken.U.ID))
		}
	} else if i == int(Num) {
		fptex.WriteString(fmt.Sprintf("%d", cToken.U.Value))
	}
}

func SetIdKind(k table.KindT) {
	idKind = k
}
