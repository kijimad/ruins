package states

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyEquipChoice_空きスロットに候補を装着する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	sword, err := lifecycle.SpawnBackpackItem(world, "iron_sword", 1)
	require.NoError(t, err)

	// 武器2スロットは空き。初期装備の松明が居る武器1を避ける
	err = applyEquipChoice(world, equipChoice{entity: sword}, gc.SlotWeapon2, player, nil)
	require.NoError(t, err)

	weapons := query.GetWeapons(world, player)
	require.NotNil(t, weapons[1], "武器2スロットが埋まる")
	assert.Equal(t, sword, *weapons[1], "装着したのは鉄の剣")
	assert.False(t, world.Components.LocationInBackpack.Has(sword), "装着後は持ち物から外れる")
}

func TestApplyEquipChoice_装備済みを外す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	weapons := query.GetWeapons(world, player)
	require.NotNil(t, weapons[0], "初期装備の武器がある")
	equipped := *weapons[0]

	err = applyEquipChoice(world, equipChoice{unequip: true}, gc.SlotWeapon1, player, &equipped)
	require.NoError(t, err)

	assert.True(t, world.Components.LocationInBackpack.Has(equipped), "外すと装備が持ち物へ戻る")
	weapons = query.GetWeapons(world, player)
	assert.Nil(t, weapons[0], "外したので武器1スロットは空く")
}

func TestApplyEquipChoice_装備済みを候補で差し替える(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	weapons := query.GetWeapons(world, player)
	require.NotNil(t, weapons[0], "初期装備の武器がある")
	old := *weapons[0]
	fresh, err := lifecycle.SpawnBackpackItem(world, "iron_sword", 1)
	require.NoError(t, err)

	err = applyEquipChoice(world, equipChoice{entity: fresh}, gc.SlotWeapon1, player, &old)
	require.NoError(t, err)

	assert.True(t, world.Components.LocationInBackpack.Has(old), "旧装備が持ち物へ戻る")
	assert.False(t, world.Components.LocationInBackpack.Has(fresh), "新装備が持ち物から外れる")
	weapons = query.GetWeapons(world, player)
	require.NotNil(t, weapons[0], "武器1スロットが埋まる")
	assert.Equal(t, fresh, *weapons[0], "武器1スロットは差し替え後の武器")
}

func TestApplyEquipChoice_外す対象が無ければ何もしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// 外すを選んでも現装備が無ければ無害
	require.NoError(t, applyEquipChoice(world, equipChoice{unequip: true}, gc.SlotWeapon2, player, nil))
	weapons := query.GetWeapons(world, player)
	assert.Nil(t, weapons[1], "武器2スロットは空きのまま")
}

func TestEquipChoiceAt_装備済みは先頭が外すで候補がずれる(t *testing.T) {
	t.Parallel()
	sword := ecs.Entity{}
	prev := ecs.Entity{}

	// 空スロットは先頭から候補が並ぶ
	empty := EquipSelectProps{Items: []ecs.Entity{sword}}
	c, ok := equipChoiceAt(empty, 0)
	require.True(t, ok)
	assert.False(t, c.unequip, "空スロットの index0 は候補")

	// 装備済みは index0 が外す、index1 から候補
	occupied := EquipSelectProps{Items: []ecs.Entity{sword}, PreviousEquipment: &prev}
	c0, ok := equipChoiceAt(occupied, 0)
	require.True(t, ok)
	assert.True(t, c0.unequip, "装備済みの index0 は外す")
	c1, ok := equipChoiceAt(occupied, 1)
	require.True(t, ok)
	assert.False(t, c1.unequip, "装備済みの index1 は候補")

	_, ok = equipChoiceAt(occupied, 2)
	assert.False(t, ok, "候補数を超えた index は選べない")
}

// TestGolden_EquipSelect は装備選択ポップアップの見た目を固定する。候補ごとにアイテムの
// アイコンが名前の左に出ることを覆う。ポップアップは widget の見た目が主題で、背景の全画面に
// 依存させると環境差の合成ノイズで脆くなるため、内容コンテナを直接描いて決定的に撮る
func TestGolden_EquipSelect(t *testing.T) {
	t.Parallel()

	// InitReplayWorld は SpriteSheets 込みの完全な world を作る。アイコン解決にテクスチャが要る。
	// フェイスは InitReplayWorld が既にテストごと独立で持つので上書きしない
	world := vrt.InitReplayWorld(t)
	sword, err := lifecycle.SpawnBackpackItem(world, "iron_sword", 1)
	require.NoError(t, err)
	gun, err := lifecycle.SpawnBackpackItem(world, "ray_gun", 1)
	require.NoError(t, err)

	props := EquipSelectProps{Items: []ecs.Entity{sword, gun}, SlotNumber: gc.SlotWeapon1}
	tree := buildEquipSelectUI(world, props, 0, world.Resources.UIResources)
	screen := ebiten.NewImage(consts.GameWidth, consts.GameHeight)
	tree.Draw(uicore.NewEbitenCanvas(screen))
	vrt.AssertFrameGolden(t, "TestGolden_EquipSelect", screen)
}
