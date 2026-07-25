package components

// VisionUpdate は次フレームで必要な視界更新の強さ。値が大きいほど強い更新で、複数の要求が
// 重なったら強い方が勝つ。2つの真偽フラグに分けず、1つの段階値で「更新の強さ」を表す。
type VisionUpdate uint8

const (
	// VisionUpdateNone は更新不要。
	VisionUpdateNone VisionUpdate = iota
	// VisionUpdateRefresh は視界だけ再計算し、レイキャストキャッシュは保持する。遮蔽が変わらない
	// AIターンごとの光源・明暗の更新に使い、静止中の視線計算を毎ターン引き直さないで済ませる。
	VisionUpdateRefresh
	// VisionUpdateForce は視界再計算に加えレイキャストキャッシュも破棄する。扉開閉・フロア遷移・
	// 帯シフト・prop 出現など遮蔽が変わる操作で使う。壁配置に依存する古いレイ結果を捨てる。
	VisionUpdateForce
)

// VisionState は視界計算の一時状態を保持するシングルトン。
// 毎フレーム・視界更新のたびに再構築されるので serde 対象外にする。
type VisionState struct {
	// VisibleTiles は現在フレームで実際に見えているタイル。struct キーのため serde 不可
	VisibleTiles map[GridElement]bool
	// LightSourceCache は視界内タイルの光源情報。視界更新のたびに再構築される
	LightSourceCache map[GridElement]LightInfo
	// PendingUpdate は次フレームで要る視界更新の強さ。遮蔽が変わる操作は Force、遮蔽が変わらない
	// 操作は Refresh を RequestUpdate で要求する。強い方だけが残る。
	PendingUpdate VisionUpdate
}

// RequestUpdate は視界更新を要求する。すでにより強い更新が要求済みなら据え置き、強い方を残す。
func (vs *VisionState) RequestUpdate(u VisionUpdate) {
	if u > vs.PendingUpdate {
		vs.PendingUpdate = u
	}
}

// NewVisionState は初期化された VisionState を返す
func NewVisionState() *VisionState {
	return &VisionState{
		VisibleTiles:     make(map[GridElement]bool),
		LightSourceCache: make(map[GridElement]LightInfo),
	}
}
