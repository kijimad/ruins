package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeSleepableWarm はテスト世界を入眠可能な適温にする。Validate の温度ゲートを通すため、
// 現ステージの基本気温を適温帯に収め、時刻を春の夜明けに固定する
func makeSleepableWarm(world w.World) {
	query.GetGameTime(world).TotalTurns = 0
	query.EnsureStageField(world, query.GetDungeon(world).CurrentStage).BaseTemp = 10
}

func TestSleepBehavior_疲れていないと眠れない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 0, Max: 2000})

	result, err := Execute(NewSleepActivity(), actor, world)
	require.NoError(t, err, "入眠拒否はユーザエラーで err にはならない")
	assert.False(t, result.Success, "快調では入眠できない")
	assert.False(t, world.Components.Sleeping.Has(actor), "Sleeping は付かない")
}

func TestSleepBehavior_入眠するとSleepingが付く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
	// 疲労段階を Tired にして眠れるようにする
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 1200, Max: 2000})
	makeSleepableWarm(world)

	result, err := Execute(NewSleepActivity(), actor, world)
	require.NoError(t, err)
	require.True(t, result.Success, "入眠できる")
	assert.True(t, world.Components.Sleeping.Has(actor), "入眠で Sleeping が付く")
}

func TestSleepBehavior_普通の疲労でも眠れる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
	// Normal 段階。30〜50% の比率。Rested だけが弾かれるので Normal は眠れる
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 800, Max: 2000})
	require.Equal(t, gc.FatigueNormal, world.Components.Fatigue.Get(actor).GetLevel(), "前提: Normal 段階")
	makeSleepableWarm(world)

	result, err := Execute(NewSleepActivity(), actor, world)
	require.NoError(t, err)
	assert.True(t, result.Success, "普通の疲労でも眠れる")
	assert.True(t, world.Components.Sleeping.Has(actor), "Sleeping が付く")
}

func TestSleepBehavior_足元の寝具品質を写す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 1200, Max: 2000})

	bed := world.ECS.NewEntity()
	world.Components.GridElement.Add(bed, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
	world.Components.Bedding.Add(bed, &gc.Bedding{Quality: 150})
	makeSleepableWarm(world)

	_, err := Execute(NewSleepActivity(), actor, world)
	require.NoError(t, err)
	assert.Equal(t, consts.Percent(150), world.Components.Sleeping.Get(actor).Quality,
		"足元の寝具の Quality を写す")
}

func TestSleepBehavior_中断すると起床処理が走り理由が記録される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 1200, Max: 2000})
	// 入眠時は満腹にして眠れるようにする。飢餓は Validate では見ず DoTurn で中断する
	world.Components.Hunger.Add(actor, &gc.Hunger{Current: 400, Max: 500})
	makeSleepableWarm(world)

	result, err := Execute(NewSleepActivity(), actor, world)
	require.NoError(t, err)
	require.True(t, result.Success, "入眠できる")
	require.True(t, world.Components.Sleeping.Has(actor), "入眠で Sleeping が付く")

	// 眠っている間に飢餓へ落とし、次の継続ステップで DoTurn が自ら中断する状況を作る
	world.Components.Hunger.Get(actor).Current = 0
	stepActivity(actor, world)

	assert.False(t, world.Components.Sleeping.Has(actor), "中断で Sleeping が外れる。起床処理が走った証跡")
	last := GetLastResult(actor, world)
	require.NotNil(t, last)
	assert.Equal(t, gc.ActivityStateCanceled, last.State, "結果は中断")
	assert.Equal(t, "woke up from hunger", last.Message, "中断理由が記録される")
}
