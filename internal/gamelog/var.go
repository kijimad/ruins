package gamelog

var (
	// FieldLog はフィールド用ログ
	FieldLog = NewSafeSlice(FieldLogMaxSize)
)
