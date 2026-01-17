package gamelog

var (
	// FieldLog はフィールド用ログ
	FieldLog = NewSafeSlice(FieldLogMaxSize)
	// SceneLog は会話シーンでステータス変化を通知する用ログ
	SceneLog = NewSafeSlice(SceneLogMaxSize)
)
