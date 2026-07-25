package components

// VisionState は視界計算の一時状態を保持するシングルトン。
// 毎フレーム・視界更新のたびに再構築されるので serde 対象外にする。
type VisionState struct {
	// VisibleTiles は現在フレームで実際に見えているタイル。struct キーのため serde 不可
	VisibleTiles map[GridElement]bool
	// LightSourceCache は視界内タイルの光源情報。視界更新のたびに再構築される
	LightSourceCache map[GridElement]LightInfo
	// PendingUpdate は次フレームで視界を再計算するか。遮蔽が変わる操作も変わらない操作も
	// 一律に再計算を要求する。更新の強さを段階に分けず常に作り直すことで、更新種別の取り違えで
	// 古い遮蔽が残り幽霊影が出る不具合を構造的に無くす。
	PendingUpdate bool
}

// RequestUpdate は次フレームの視界再計算を要求する。
func (vs *VisionState) RequestUpdate() {
	vs.PendingUpdate = true
}

// NewVisionState は初期化された VisionState を返す
func NewVisionState() *VisionState {
	return &VisionState{
		VisibleTiles:     make(map[GridElement]bool),
		LightSourceCache: make(map[GridElement]LightInfo),
	}
}
