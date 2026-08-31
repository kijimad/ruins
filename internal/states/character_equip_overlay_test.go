package states

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
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

// moveEquipCursor は装備選択のカーソルを steps 回だけ下へ動かす。UseTabMenu で reducer を
// 登録してから nav を流す。DispatchNav は reducer を即時適用するので Update は要らない
func moveEquipCursor(o *characterEquipOverlay, steps int) {
	props := o.mount.GetProps()
	hooks.UseTabMenu(o.mount.Store(), "char_equip", hooks.TabMenuConfig{TabCount: 1, ItemCounts: []int{equipChoiceCount(props)}})
	for range steps {
		o.mount.DispatchNav(inputmapper.ActionMenuDown)
	}
}

func TestCharacterEquipOverlay_装備済みは先頭が外すで実行すると外れる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// SpawnPlayer が武器1スロットに初期装備を着ける。これを外す対象にする
	weapons := query.GetWeapons(world, player)
	require.NotNil(t, weapons[0], "初期装備の武器がある")
	equipped := *weapons[0]

	o := newEquipOverlayForTest()
	o.Open(world, equipItemData{SlotNumber: gc.SlotWeapon1, Character: player, Entity: &equipped})

	// 装備済みで開くとカーソルは先頭の「外す」に乗る
	_, ok := o.selectedItem()
	assert.False(t, ok, "先頭は候補でなく外すなので候補としては選べない")
	choice, ok := o.selection()
	require.True(t, ok)
	assert.True(t, choice.unequip, "先頭の選択は外す")

	require.NoError(t, o.execute(world))
	assert.True(t, world.Components.LocationInBackpack.Has(equipped), "外すと装備が持ち物へ戻る")
	weapons = query.GetWeapons(world, player)
	assert.Nil(t, weapons[0], "外したので武器1スロットは空く")
}

func TestCharacterEquipOverlay_装備済みは候補を選ぶと差し替わる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// SpawnPlayer の初期装備を差し替えで持ち物へ戻る旧装備にする
	weapons := query.GetWeapons(world, player)
	require.NotNil(t, weapons[0], "初期装備の武器がある")
	old := *weapons[0]

	// 持ち物の武器が差し替え先の候補になる
	fresh, err := lifecycle.SpawnBackpackItem(world, "iron_sword", 1)
	require.NoError(t, err)
	o := newEquipOverlayForTest()
	o.Open(world, equipItemData{SlotNumber: gc.SlotWeapon1, Character: player, Entity: &old})

	// 先頭の「外す」を1つ飛ばして候補へカーソルを移す
	moveEquipCursor(&o, 1)
	got, ok := o.selectedItem()
	require.True(t, ok, "候補にカーソルが乗る")
	assert.Equal(t, fresh, got, "候補は持ち物に残る新しい武器だけ")

	require.NoError(t, o.execute(world))

	assert.True(t, world.Components.LocationInBackpack.Has(old), "旧装備が持ち物へ戻る")
	assert.False(t, world.Components.LocationInBackpack.Has(fresh), "新装備が持ち物から外れる")
	weapons = query.GetWeapons(world, player)
	require.NotNil(t, weapons[0], "武器1スロットが埋まる")
	assert.Equal(t, fresh, *weapons[0], "武器1スロットは差し替え後の武器")
}

func TestEquipChoiceAt_装備済みは先頭が外すで候補がずれる(t *testing.T) {
	t.Parallel()
	sword := ecs.Entity{}
	prev := ecs.Entity{}

	// 空スロットは先頭から候補が並ぶ
	empty := charEquipProps{Items: []ecs.Entity{sword}}
	c, ok := equipChoiceAt(empty, 0)
	require.True(t, ok)
	assert.False(t, c.unequip, "空スロットの index0 は候補")

	// 装備済みは index0 が外す、index1 から候補
	occupied := charEquipProps{Items: []ecs.Entity{sword}, PreviousEquipment: &prev}
	c0, ok := equipChoiceAt(occupied, 0)
	require.True(t, ok)
	assert.True(t, c0.unequip, "装備済みの index0 は外す")
	c1, ok := equipChoiceAt(occupied, 1)
	require.True(t, ok)
	assert.False(t, c1.unequip, "装備済みの index1 は候補")

	_, ok = equipChoiceAt(occupied, 2)
	assert.False(t, ok, "候補数を超えた index は選べない")
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

	// InitReplayWorld は SpriteSheets 込みの完全な world を作る。アイコン解決にテクスチャが要る。
	// フェイスは InitReplayWorld が既にテストごと独立で持つので上書きしない
	world := vrt.InitReplayWorld(t)
	sword, err := lifecycle.SpawnBackpackItem(world, "iron_sword", 1)
	require.NoError(t, err)
	gun, err := lifecycle.SpawnBackpackItem(world, "ray_gun", 1)
	require.NoError(t, err)

	props := charEquipProps{Items: []ecs.Entity{sword, gun}, SlotNumber: gc.SlotWeapon1}
	tree := buildEquipSelectUI(world, props, 0, world.Resources.UIResources)
	screen := ebiten.NewImage(consts.GameWidth, consts.GameHeight)
	tree.Draw(uicore.NewEbitenCanvas(screen))
	vrt.AssertFrameGolden(t, "TestGolden_EquipSelect", screen)
}

func TestEquipCompareRows_同項目に差分を色分けで併記する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// クレイモアは鉄の剣より攻撃力が高く、命中と攻撃コストは不利
	current, err := lifecycle.SpawnBackpackItem(world, "iron_sword", 1)
	require.NoError(t, err)
	candidate, err := lifecycle.SpawnBackpackItem(world, "claymore", 1)
	require.NoError(t, err)

	rows := equipCompareRows(world, candidate, &current)
	byLabel := map[string]entityspec.SpecRow{}
	for _, r := range rows {
		if !r.Header {
			byLabel[r.Label] = r
		}
	}

	atk := byLabel[query.T(world, "Attack power")]
	assert.Equal(t, "18 (+3)", atk.Value, "攻撃力は上がり差分を併記する")
	require.NotNil(t, atk.Color, "変化した項目は色付き")
	assert.Equal(t, theme.StatusSuccess, *atk.Color, "有利な変化は緑")

	acc := byLabel[query.T(world, "Accuracy")]
	assert.Equal(t, "75 (-15)", acc.Value, "命中は下がり差分を併記する")
	require.NotNil(t, acc.Color)
	assert.Equal(t, theme.StatusDanger, *acc.Color, "不利な変化は赤")

	cost := byLabel[query.T(world, "Attack cost")]
	assert.Equal(t, "3 (+1)", cost.Value, "攻撃コストは上がり差分を併記する")
	require.NotNil(t, cost.Color)
	assert.Equal(t, theme.StatusDanger, *cost.Color, "コストは小さいほど良いので増加は赤")

	// 変化しない項目には差分も色も付けない
	hits := byLabel[query.T(world, "Hits")]
	assert.Equal(t, "1", hits.Value, "同値の項目は差分を出さない")
	assert.Nil(t, hits.Color, "同値の項目は色を付けない")
}

func TestEquipCompareRows_現装備が無ければ差分を付けない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	candidate, err := lifecycle.SpawnBackpackItem(world, "claymore", 1)
	require.NoError(t, err)

	rows := equipCompareRows(world, candidate, nil)
	for _, r := range rows {
		assert.Nil(t, r.Color, "現装備が無ければどの行も色を付けない: %s", r.Label)
		assert.NotContains(t, r.Value, "(", "現装備が無ければ差分の括弧を出さない: %s", r.Label)
	}
}
