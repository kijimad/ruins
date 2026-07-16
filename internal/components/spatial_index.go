package components

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/mlange-42/ark/ecs"
)

// SpatialIndex はターン内で再利用可能な空間インデックスを保持する。
// 壁・キャラクターの位置をキャッシュし、O(1)で判定できるようにする。
// すべてターン開始時に1回構築し、ターン終了時に無効化する
type SpatialIndex struct {
	MapWidth, MapHeight int
	// 静的障害物の位置。壁やドアなどBlockPassコンポーネントを持つPropが対象
	BlockPass map[GridElement]bool
	// キャラクター位置のインデックス。プレイヤー・敵・隊員・中立NPCの位置
	Characters map[GridElement]ecs.Entity
	// プレイヤーエンティティのキャッシュ。プレイヤーが存在しない場合はnil
	PlayerEntity *ecs.Entity
	// 構築済みフラグ。falseの場合は初回アクセス時に構築する
	Built bool
	// BuildCount は累積の再構築回数。移動ごとの無効化→再構築チャーンを回帰テストで検知するための観測用。
	// Invalidate ではリセットしない（累積カウンタ）
	BuildCount int
}

// NewSpatialIndex は未構築の空インデックスを作成する
func NewSpatialIndex() *SpatialIndex {
	return &SpatialIndex{}
}

// IsBlockPass は指定座標に静的障害物があるかをO(1)で判定する。
// 未構築の場合はfalseを返す
func (si *SpatialIndex) IsBlockPass(x, y int) bool {
	if !si.Built {
		return false
	}
	return si.BlockPass[GridElement{X: consts.Tile(x), Y: consts.Tile(y)}]
}

// CharacterAt は指定座標のキャラクターを返す
func (si *SpatialIndex) CharacterAt(x, y int) (ecs.Entity, bool) {
	entity, ok := si.Characters[GridElement{X: consts.Tile(x), Y: consts.Tile(y)}]
	return entity, ok
}

// MoveCharacter はキャラクターの位置を増分更新する。
// 無効化→全再構築のチャーンを避け、移動のたびに O(1) でインデックスを最新に保つ。
// from タイルの登録が自分自身のときだけ削除し、位置入れ替えで別キャラが入った場合を壊さない
// （actor と隊員をどちらの順で更新しても最終状態が正しくなる）。
// 未構築の場合は何もしない（次回アクセス時に真から再構築される）。
func (si *SpatialIndex) MoveCharacter(fromX, fromY, toX, toY int, e ecs.Entity) {
	if !si.Built {
		return
	}
	fromKey := GridElement{X: consts.Tile(fromX), Y: consts.Tile(fromY)}
	if cur, ok := si.Characters[fromKey]; ok && cur == e {
		delete(si.Characters, fromKey)
	}
	si.Characters[GridElement{X: consts.Tile(toX), Y: consts.Tile(toY)}] = e
}

// Invalidate はインデックスを無効化する。次回アクセス時に再構築させる
func (si *SpatialIndex) Invalidate() {
	si.Built = false
	si.BlockPass = nil
	si.Characters = nil
	si.PlayerEntity = nil
}
