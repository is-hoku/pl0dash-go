package table

type KindT int // Identifier の種類

const (
	VarID KindT = iota
	FuncID
	ParID
	ConstID
)
