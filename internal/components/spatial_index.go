package components

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/mlange-42/ark/ecs"
)

// SpatialIndex はターン内で再利用可能な空間インデックスを保持する。
// 壁・キャラクターの位置をキャッシュし、O(1)で判定できるようにする。
// すべてターン開始時に1回構築し、ターン終了時に無効化する
type SpatialIndex struct {
	// マップの寸法。タイル数で表す。Level.TileWidth/TileHeight と同じ単位
	MapWidth, MapHeight consts.Tile
	// 通れないマス。Passage.Passable が偽の、壊れていない実体の位置
	Blocked map[GridElement]bool
	// 移動コストの座標の表。壊れていない実体の Passage.Cost を足した値で、0 のマスは持たない。
	// 床を全て回して作るので、通れないマスの集合と別に、引かれたときに初めて作る。nil は未構築
	PassCosts map[GridElement]int
	// キャラクター位置のインデックス。プレイヤー・敵・中立NPCの位置
	Characters map[GridElement]ecs.Entity
	// プレイヤーエンティティのキャッシュ。プレイヤーが存在しない場合はnil
	PlayerEntity *ecs.Entity
	// 構築済みフラグ。falseの場合は初回アクセス時に構築する
	Built bool
	// BuildCount は累積の再構築回数。移動ごとの無効化→再構築チャーンを回帰テストで検知するための観測用。
	// Invalidate ではリセットしない
	BuildCount int
}

// NewSpatialIndex は未構築の空インデックスを作成する
func NewSpatialIndex() *SpatialIndex {
	return &SpatialIndex{}
}

// IsBlocked は指定タイルが通れない地形か置物を持つかをO(1)で判定する。
// 未構築の場合はfalseを返す
func (si *SpatialIndex) IsBlocked(pos consts.Coord[consts.Tile]) bool {
	if !si.Built {
		return false
	}
	return si.Blocked[GridElement{Coord: pos}]
}

// CharacterAt は指定タイルのキャラクターを返す
func (si *SpatialIndex) CharacterAt(pos consts.Coord[consts.Tile]) (ecs.Entity, bool) {
	entity, ok := si.Characters[GridElement{Coord: pos}]
	return entity, ok
}

// MoveCharacter はキャラクターの位置を増分更新する。
// 無効化→全再構築のチャーンを避け、移動のたびに O(1) でインデックスを最新に保つ。
// from タイルの登録が自分自身のときだけ削除し、位置入れ替えで別キャラが入った場合を壊さない
// 2体をどちらの順で更新しても最終状態が正しくなる。
// 未構築の場合は何もしない。次回アクセス時に真から再構築される。
func (si *SpatialIndex) MoveCharacter(from, to consts.Coord[consts.Tile], e ecs.Entity) {
	if !si.Built {
		return
	}
	fromKey := GridElement{Coord: from}
	if cur, ok := si.Characters[fromKey]; ok && cur == e {
		delete(si.Characters, fromKey)
	}
	si.Characters[GridElement{Coord: to}] = e
}

// Invalidate はインデックスを無効化する。次回アクセス時に再構築させる
func (si *SpatialIndex) Invalidate() {
	si.Built = false
	si.Blocked = nil
	si.PassCosts = nil
	si.Characters = nil
	si.PlayerEntity = nil
}
