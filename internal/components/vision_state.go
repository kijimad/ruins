package components

// VisionState は視界計算の一時状態を保持するシングルトン。
// 毎フレーム・視界更新のたびに再構築されるので serde 対象外にする。
type VisionState struct {
	// VisibleTiles は現在フレームで実際に見えているタイル。struct キーのため serde 不可
	VisibleTiles map[GridElement]bool
	// LightSourceCache は視界内タイルの光源情報。視界更新のたびに再構築される
	LightSourceCache map[GridElement]LightInfo
	// NeedsForceUpdate は次フレームで視界を強制再計算するフラグ。扉開閉・フロア遷移・帯シフト・
	// prop 出現など遮蔽が変わる操作で立てる。壁配置に依存するレイキャストキャッシュも破棄させる
	NeedsForceUpdate bool
	// NeedsVisionRefresh は遮蔽が変わらないまま視界だけ再計算させたいときのフラグ。
	// AIターンごとの光源・明暗の更新に使う。レイキャストキャッシュは破棄せず、静止中の
	// 視線計算を毎ターン引き直さないで済ませる。壁が変わる操作は NeedsForceUpdate を使う
	NeedsVisionRefresh bool
}

// NewVisionState は初期化された VisionState を返す
func NewVisionState() *VisionState {
	return &VisionState{
		VisibleTiles:     make(map[GridElement]bool),
		LightSourceCache: make(map[GridElement]LightInfo),
	}
}
