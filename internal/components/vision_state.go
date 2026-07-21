package components

// VisionState は視界計算の一時状態を保持するシングルトン。
// 毎フレーム・視界更新のたびに再構築されるので serde 対象外にする。
type VisionState struct {
	// VisibleTiles は現在フレームで実際に見えているタイル。struct キーのため serde 不可
	VisibleTiles map[GridElement]bool
	// LightSourceCache は視界内タイルの光源情報。視界更新のたびに再構築される
	LightSourceCache map[GridElement]LightInfo
	// NeedsForceUpdate は次フレームで視界を強制再計算するフラグ。扉開閉やフロア遷移で立てる
	NeedsForceUpdate bool
}

// NewVisionState は初期化された VisionState を返す
func NewVisionState() *VisionState {
	return &VisionState{
		VisibleTiles:     make(map[GridElement]bool),
		LightSourceCache: make(map[GridElement]LightInfo),
	}
}
