package states

// Configurable はステート設定を提供するオプショナルインターフェース
// このインターフェースを実装しないステートはデフォルト動作となる
// デフォルト動作: BlurBackground=trueとして扱う
type Configurable interface {
	StateConfig() StateConfig
}

// StateConfig はステートの共通設定
type StateConfig struct {
	// BlurBackground は背景にブラー効果を適用するかどうか
	BlurBackground bool
}
