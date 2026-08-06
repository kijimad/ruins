package components

// UserSettings は設定画面で変更するグローバル設定を保持する。singleton componentとして管理される。
// 永続層は config.UserConfig で、これはそのランタイムミラー。InitSingleton が config からシードする。
type UserSettings struct {
	Language string // 表示言語の言語コード。"ja" / "en" など
}

// NewUserSettings は指定言語で初期化された UserSettings を返す
func NewUserSettings(lang string) *UserSettings {
	return &UserSettings{Language: lang}
}
