package overlay

import (
	"image"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDetail_初期状態は非表示(t *testing.T) {
	t.Parallel()
	d := NewDetail(func(_ w.World) (DetailContent, bool) { return DetailContent{}, false })

	assert.False(t, d.Active())
}

// TestDetail_Open_中身が無ければ開かない は、アイテムの無いメニューで詳細モーダルを開こうとしても
// 開かないことを固定する。開くと中身の無いモーダルが入力を奪い、ESC 以外で抜けられなくなる回帰を防ぐ。
func TestDetail_Open_中身が無ければ開かない(t *testing.T) {
	t.Parallel()

	hasContent := false
	d := NewDetail(func(_ w.World) (DetailContent, bool) {
		return DetailContent{Name: "item"}, hasContent
	})
	var world w.World

	d.Open(world)
	assert.False(t, d.Active(), "中身が無いときは開かない")

	hasContent = true
	d.Open(world)
	assert.True(t, d.Active(), "中身があれば開く")
}

func TestDetailOpen_開くと先頭ページに戻る(t *testing.T) {
	t.Parallel()
	d := NewDetail(func(_ w.World) (DetailContent, bool) {
		return DetailContent{Name: "回復薬"}, true
	})
	d.page = 3

	d.Open(w.World{})

	assert.True(t, d.Active())
	assert.Equal(t, 0, d.page, "開くと先頭ページに戻る")
}

func TestDetailHandleInput_非表示のときは何もしない(t *testing.T) {
	t.Parallel()
	called := false
	d := NewDetail(func(_ w.World) (DetailContent, bool) {
		called = true
		return DetailContent{}, false
	})

	err := d.HandleInput(w.World{})

	require.NoError(t, err)
	assert.False(t, called, "非表示のときprovideは呼ばれない")
	assert.False(t, d.Active())
}

// TestEntityDetailContent_死んだ実体は空を返しpanicしない は、生存していない実体を渡しても
// 性能行を引かずに空の内容を返すことを固定する。ゼロ実体への Get で落ちる回帰を防ぐ。
func TestEntityDetailContent_死んだ実体は空を返しpanicしない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	var content DetailContent
	assert.NotPanics(t, func() {
		content = EntityDetailContent(world, ecs.Entity{})
	})
	assert.Empty(t, content.Name, "対象が無いので名前は出さない")
	assert.Nil(t, content.Rows, "対象が無いので性能行は出さない")
}

// TestEntityDetailContent_生存している実体は名前と説明と性能行を返す は、Name・Description・
// Abilities を持つ実体から詳細内容を正しく組み立てることを固定する。
// EntityDetailContent は Name・Description・Abilities の3コンポーネントだけを参照し、spawn
// 関数が付与する他コンポーネントの有無は結果に影響しないため、手作り実体で検証する。
func TestEntityDetailContent_生存している実体は名前と説明と性能行を返す(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	e := world.ECS.NewEntity()
	world.Components.Name.Add(e, &gc.Name{Name: "テストアイテム"})
	world.Components.Description.Add(e, &gc.Description{Description: "テスト説明"})
	world.Components.Abilities.Add(e, &gc.Abilities{
		Vitality: gc.Ability{Base: 10},
		Strength: gc.Ability{Base: 5},
	})

	content := EntityDetailContent(world, e)

	assert.Equal(t, "テストアイテム", content.Name, "未訳のNameは原文のまま返る")
	assert.Equal(t, "テスト説明", content.Desc, "未訳のDescriptionは原文のまま返る")
	assert.Len(t, content.Rows, 6, "Abilitiesの性能行が6行分含まれる")
}

// TestEntityDetailContent_説明コンポーネントが無ければ空文字 は、Description を持たない実体でも
// panicせず空文字を返すことを固定する。EntityDetailContent が参照するのは Name・Description・
// Abilities のみなので、手作り実体で検証する。
func TestEntityDetailContent_説明コンポーネントが無ければ空文字(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	e := world.ECS.NewEntity()
	world.Components.Name.Add(e, &gc.Name{Name: "説明なしアイテム"})

	content := EntityDetailContent(world, e)

	assert.Equal(t, "説明なしアイテム", content.Name)
	assert.Empty(t, content.Desc, "Descriptionコンポーネントが無ければ空文字を返す")
}

func TestNewEntityDetail_対象が無ければ開かない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	d := NewEntityDetail(func() (ecs.Entity, bool) { return ecs.Entity{}, false })

	d.Open(world)

	assert.False(t, d.Active(), "provideがokを返さなければ開かない")
}

// TestNewEntityDetail_対象が死んでいれば開かない は、provide が返す実体がすでに削除されていても
// ゼロ実体への Get で落ちずに開かないことを固定する。
func TestNewEntityDetail_対象が死んでいれば開かない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	e := world.ECS.NewEntity()
	world.ECS.RemoveEntity(e)
	d := NewEntityDetail(func() (ecs.Entity, bool) { return e, true })

	assert.NotPanics(t, func() {
		d.Open(world)
	})
	assert.False(t, d.Active(), "死んだ実体では開かない")
}

func TestNewEntityDetail_対象があれば実体から詳細を組み立てて開く(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	e := world.ECS.NewEntity()
	world.Components.Name.Add(e, &gc.Name{Name: "回復薬"})
	d := NewEntityDetail(func() (ecs.Entity, bool) { return e, true })

	d.Open(world)

	assert.True(t, d.Active(), "生存している実体があれば開く")
}

func TestDetailRenderOverlay_対象が無ければnilを返す(t *testing.T) {
	t.Parallel()
	d := NewDetail(func(_ w.World) (DetailContent, bool) { return DetailContent{}, false })

	got := d.RenderOverlay(w.World{}, image.Rect(0, 0, 100, 100))

	assert.Nil(t, got)
}

func TestDetailRenderOverlay_対象があれば名前とページ位置を表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	d := NewDetail(func(_ w.World) (DetailContent, bool) {
		return DetailContent{Name: "回復薬", Rows: []entityspec.SpecRow{{Label: "効果", Value: "10"}}}, true
	})
	d.Open(world)

	// internal/uicore のツリーを組むだけ。独立フェイスなのでロックは要らない
	tree := d.RenderOverlay(world, image.Rect(0, 0, 400, 400))

	require.NotNil(t, tree)
	labels := uicore.CollectLabels(uicore.Placeable([]uicore.Drawable{tree})[0])
	assert.Equal(t, []string{"回復薬", "効果", "10", "1/1"}, labels)
}
