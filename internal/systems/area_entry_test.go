package systems

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAreaEntrySystem_observe_別の区域に入ったときだけ入ったとみなす(t *testing.T) {
	t.Parallel()

	townA := overworld.Area{Key: "town:0,0", Name: "Town"}
	townB := overworld.Area{Key: "town:9,9", Name: "Town"}
	wasteland := overworld.Area{Name: "Wasteland"}
	shrine := overworld.Area{Name: "Shrine"}

	sys := &AreaEntrySystem{}
	seq := []overworld.Area{townA, townA, wasteland, townA, wasteland, townA, townB, wasteland, shrine, townB, wasteland, townA}
	got := make([]bool, 0, len(seq))
	for i, a := range seq {
		got = append(got, sys.observe(a, consts.Turn(100+i)))
	}
	assert.Equal(t, []bool{
		false, // 最初に見たエリアは記憶するだけ
		false, // 同じ区域の中
		false, // 荒れ地に出た
		false, // 荒れ地から同じ町に戻った。境目を行き来しても出さない
		false, // また荒れ地に出た
		false, // また同じ町に戻った
		true,  // 同じ種類の別の区域に移った
		false, // 荒れ地に出た
		false, // 小さな場所に入った
		false, // 小さな場所から直前と同じ区域に戻った
		false, // 荒れ地に出た
		true,  // 荒れ地を通って別の区域に入った
	}, got)
}

func TestAreaEntrySystem_observe_ターンが続いていなければ別の区域でも入ったとみなさない(t *testing.T) {
	t.Parallel()

	townA := overworld.Area{Key: "town:0,0", Name: "Town"}
	townB := overworld.Area{Key: "town:9,9", Name: "Town"}

	sys := &AreaEntrySystem{}
	got := []bool{
		sys.observe(townA, 100),
		sys.observe(townB, 500), // 別のセーブを読み込んだ、あるいはダンジョンから戻った
		sys.observe(townA, 501),
	}
	assert.Equal(t, []bool{false, false, true}, got)
}

func TestAreaLabel_地上でなければ空(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	assert.Empty(t, areaLabel(world))
}

func TestAreaLabel_地上では現在地の名前を出す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, 24, 24, 3, 1), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world))
	assert.Equal(t, "Wasteland", areaLabel(world), "開始地点は荒れ地")
}

func TestFormatAreaLabel(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	assert.Equal(t, []string{"Town / Clinic", "Shrine"}, []string{
		formatAreaLabel(world, overworld.Area{Key: "town:0,0", Name: "Town", Building: "Clinic"}),
		formatAreaLabel(world, overworld.Area{Name: "Shrine"}),
	})
}

func TestRunTurnEndSystems_登録したAreaEntrySystemが地上で回る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, 24, 24, 3, 1), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world))
	updaters, _ := InitializeSystems(world)
	name := AreaEntrySystem{}.String()
	sys, ok := updaters[name].(*AreaEntrySystem)
	require.True(t, ok, "InitializeSystems に登録する")
	// ほかのターン終わりの system はこの world では回らないので、登録したものだけを置く
	world.Updaters = map[string]w.Updater{name: sys}

	require.NoError(t, runTurnEndSystems(world))

	assert.True(t, sys.observed, "ターンの終わりにエリアを観測する")
}
