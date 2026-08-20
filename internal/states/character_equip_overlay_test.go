package states

import (
	"image"
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newEquipOverlayForTest は detail を使わない装備選択 overlay を用意する。execute は detail に触れない
func newEquipOverlayForTest() characterEquipOverlay {
	detail := overlay.NewDetail(func(w.World) (overlay.DetailContent, bool) {
		return overlay.DetailContent{}, false
	})
	return newCharacterEquipOverlay(&detail)
}

func TestCharacterEquipOverlay_Openで候補を読み込み選択中になる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	sword, err := lifecycle.SpawnBackpackItem(world, "iron_sword", 1)
	require.NoError(t, err)

	o := newEquipOverlayForTest()
	require.False(t, o.Active(), "開く前は非アクティブ")

	o.Open(world, equipItemData{SlotNumber: gc.SlotWeapon1, Character: player})
	require.True(t, o.Active(), "Open で装備選択中になる")

	got, ok := o.selectedItem()
	require.True(t, ok, "候補が選択できる")
	assert.Equal(t, sword, got, "先頭候補は所持する鉄の剣。カーソル未操作なので index 0")
}

func TestCharacterEquipOverlay_executeで空きスロットに候補を装着する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	sword, err := lifecycle.SpawnBackpackItem(world, "iron_sword", 1)
	require.NoError(t, err)

	o := newEquipOverlayForTest()
	o.Open(world, equipItemData{SlotNumber: gc.SlotWeapon1, Character: player})
	require.NoError(t, o.execute(world))

	// 鉄の剣が武器1スロットへ装備され、持ち物から外れる
	weapons := query.GetWeapons(world, player)
	require.NotNil(t, weapons[0], "武器1スロットが埋まる")
	assert.Equal(t, sword, *weapons[0], "装着したのは鉄の剣")
	assert.False(t, world.Components.LocationInBackpack.Has(sword), "装着後は持ち物から外れる")
}

func TestCharacterEquipOverlay_executeで既存装備を持ち物へ戻す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// 先に1本装備しておく。これが差し替えで持ち物へ戻る旧装備になる
	old, err := lifecycle.SpawnBackpackItem(world, "iron_sword", 1)
	require.NoError(t, err)
	lifecycle.MoveToEquip(world, old, player, gc.SlotWeapon1)
	require.False(t, world.Components.LocationInBackpack.Has(old), "旧装備は一旦持ち物から外れている")

	// 別の武器を持ち、旧装備を PreviousEquipment に持つスロットで開く
	fresh, err := lifecycle.SpawnBackpackItem(world, "iron_sword", 1)
	require.NoError(t, err)
	o := newEquipOverlayForTest()
	o.Open(world, equipItemData{SlotNumber: gc.SlotWeapon1, Character: player, Entity: &old})

	got, ok := o.selectedItem()
	require.True(t, ok)
	assert.Equal(t, fresh, got, "候補は持ち物に残る新しい武器だけ")

	require.NoError(t, o.execute(world))

	assert.True(t, world.Components.LocationInBackpack.Has(old), "旧装備が持ち物へ戻る")
	assert.False(t, world.Components.LocationInBackpack.Has(fresh), "新装備が持ち物から外れる")
	weapons := query.GetWeapons(world, player)
	require.NotNil(t, weapons[0], "武器1スロットが埋まる")
	assert.Equal(t, fresh, *weapons[0], "武器1スロットは差し替え後の武器")
}

// TestCharacterEquipOverlay_候補が無ければ何もしない は空スロットでも候補ゼロなら execute が無害なことを検証する
func TestCharacterEquipOverlay_候補が無ければ何もしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	o := newEquipOverlayForTest()
	o.Open(world, equipItemData{SlotNumber: gc.SlotWeapon1, Character: player})

	_, ok := o.selectedItem()
	assert.False(t, ok, "装備候補が無ければ選択できない")
	require.NoError(t, o.execute(world), "候補が無くても execute はエラーにせず何もしない")

	// SpawnPlayer が武器スロットに松明を初期装備する。候補が無いので装備は変わらない
	weapons := query.GetWeapons(world, player)
	assert.NotNil(t, weapons[0], "候補が無いので武器1スロットは初期装備のまま")
}

// TestGolden_EquipSelect は装備選択ポップアップの見た目を固定する。候補ごとにアイテムの
// アイコンが名前の左に出ることを覆う。ポップアップは widget の見た目が主題で、背景の全画面に
// 依存させると環境差の合成ノイズで脆くなるため、内容コンテナを直接描いて決定的に撮る
func TestGolden_EquipSelect(t *testing.T) {
	t.Parallel()

	// InitVRTWorld は SpriteSheets 込みの完全な world を作る。アイコン解決にテクスチャが要る
	world := vrt.InitVRTWorld(t)
	world.Resources.UIResources = vrt.SharedUIResources(t)
	sword, err := lifecycle.SpawnBackpackItem(world, "iron_sword", 1)
	require.NoError(t, err)
	gun, err := lifecycle.SpawnBackpackItem(world, "ray_gun", 1)
	require.NoError(t, err)

	props := charEquipProps{Items: []ecs.Entity{sword, gun}, SlotNumber: gc.SlotWeapon1}
	vrt.AssertContainerGolden(t, func() *widget.Container {
		win := buildEquipSelectWindow(world, props, 0, image.Rect(0, 0, 400, 300), world.Resources.UIResources)
		content, _ := win.Contents.(*widget.Container)
		return content
	}, 400, 300)
}
