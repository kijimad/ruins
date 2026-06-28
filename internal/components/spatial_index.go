package components

import (
	"github.com/kijimaD/ruins/internal/consts"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// SpatialIndex はターン内で再利用可能な空間インデックスを保持する。
// 壁・キャラクター・プレイヤーの位置をキャッシュし、O(1)で判定できるようにする。
// すべてターン開始時に1回構築し、ターン終了時に無効化する
type SpatialIndex struct {
	MapWidth, MapHeight int
	// 壁位置のインデックス。BlockPassコンポーネントを持つエンティティの位置
	BlockPass map[GridElement]bool
	// キャラクター位置のインデックス。PlayerまたはAIMoveFSMを持つエンティティの位置
	Characters map[GridElement]ecs.Entity
	// プレイヤーエンティティのキャッシュ。プレイヤーが存在しない場合はnil
	PlayerEntity *ecs.Entity
	// 構築済みフラグ。falseの場合は初回アクセス時に構築する
	Built bool
}

// NewSpatialIndex は未構築の空インデックスを作成する
func NewSpatialIndex() *SpatialIndex {
	return &SpatialIndex{}
}

// IsBlockPass は指定座標に通行不可エンティティがあるかをO(1)で判定する。
// 未構築の場合はfalseを返す
func (si *SpatialIndex) IsBlockPass(x, y int) bool {
	if !si.Built {
		return false
	}
	return si.BlockPass[GridElement{X: consts.Tile(x), Y: consts.Tile(y)}]
}

// IsCharacterAt は指定座標に自分以外のキャラクターがいるかをO(1)で判定する
func (si *SpatialIndex) IsCharacterAt(x, y int, excludeEntity ecs.Entity) bool {
	if !si.Built {
		return false
	}
	entity, exists := si.Characters[GridElement{X: consts.Tile(x), Y: consts.Tile(y)}]
	return exists && entity != excludeEntity
}

// CharacterAt は指定座標のキャラクターエンティティを返す。存在しなければfalse
func (si *SpatialIndex) CharacterAt(x, y int) (ecs.Entity, bool) {
	if !si.Built {
		return ecs.Entity(0), false
	}
	entity, exists := si.Characters[GridElement{X: consts.Tile(x), Y: consts.Tile(y)}]
	return entity, exists
}

// Invalidate はインデックスを無効化する。次回アクセス時に再構築させる
func (si *SpatialIndex) Invalidate() {
	si.Built = false
	si.BlockPass = nil
	si.Characters = nil
	si.PlayerEntity = nil
}
