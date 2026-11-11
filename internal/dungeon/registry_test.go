package dungeon

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadRaws は実データの raw を読む
func loadRaws(t *testing.T) rawschema.Raws {
	t.Helper()
	raws, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	return raws
}

func TestGetAllDungeons_実データ(t *testing.T) {
	t.Parallel()

	type planner struct {
		Name   string
		Weight float64
	}
	type view struct {
		Name, EnemyTable, ItemTable string
		BaseTemp                    int
		Planners                    []planner
	}
	defs := GetAllDungeons(loadRaws(t))
	got := make([]view, 0, len(defs))
	for _, d := range defs {
		v := view{Name: d.Name(), EnemyTable: d.EnemyTableName(), ItemTable: d.ItemTableName(), BaseTemp: d.BaseTemperature()}
		for _, p := range d.PlannerPool() {
			v.Planners = append(v.Planners, planner{Name: p.PlannerType.Name, Weight: p.Weight})
		}
		got = append(got, v)
	}
	assert.Equal(t, []view{
		{Name: "dead_forest", EnemyTable: "forest", ItemTable: "forest", BaseTemp: 0, Planners: []planner{{"Forest", 5}, {"Small Room", 2}, {"Big Room", 1}}},
		{Name: "ash_cave", EnemyTable: "cave", ItemTable: "cave", BaseTemp: 5, Planners: []planner{{"Cave", 6}, {"Small Room", 1}, {"Big Room", 2}}},
		{Name: "forgotten_ruins", EnemyTable: "ruins_area", ItemTable: "ruins_area", BaseTemp: 15, Planners: []planner{{"Small Room", 4}, {"Ruins", 3}, {"Big Room", 2}}},
	}, got)
}

func TestGetStageDefinition(t *testing.T) {
	t.Parallel()
	raws := loadRaws(t)

	t.Run("raw の行のダンジョンを id で引ける", func(t *testing.T) {
		t.Parallel()
		def, found := GetStageDefinition(raws, "dead_forest")
		require.True(t, found)
		assert.Equal(t, "dead_forest", def.Name())
		_, ok := def.(*DungeonDefinition)
		require.True(t, ok, "通常ダンジョンは DungeonDefinition")
	})

	t.Run("オーバーワールドは OverworldDefinition として引ける", func(t *testing.T) {
		t.Parallel()
		def, found := GetStageDefinition(raws, "Overworld")
		require.True(t, found)
		_, ok := def.(*OverworldDefinition)
		assert.True(t, ok, "オーバーワールドは OverworldDefinition でフロアを生成しない別の型")
	})

	t.Run("存在しないダンジョンはfalseを返す", func(t *testing.T) {
		t.Parallel()
		_, found := GetStageDefinition(raws, "存在しないダンジョン")
		assert.False(t, found)
	})
}

func TestBaseTemperatureFor(t *testing.T) {
	t.Parallel()
	raws := loadRaws(t)

	t.Run("raw の行のダンジョンの気温を返す", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 5, BaseTemperatureFor(raws, "ash_cave"))
	})

	t.Run("internalDefinitions登録のダンジョンの気温を返す", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, DungeonDebug.BaseTemperature(), BaseTemperatureFor(raws, DungeonDebug.Name()))
	})

	t.Run("未登録の名前は0を返す", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0, BaseTemperatureFor(raws, "存在しないダンジョン"))
	})
}

func TestDefinitions(t *testing.T) {
	t.Parallel()

	t.Run("DungeonOverworldの設定が正しい", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "Overworld", DungeonOverworld.Name())
		assert.Equal(t, 0, DungeonOverworld.BaseTemperature())
		chunkW, chunkH, cols, rows := DungeonOverworld.BandShape()
		assert.Equal(t, consts.Tile(24), chunkW)
		assert.Equal(t, consts.Tile(24), chunkH)
		assert.Equal(t, consts.Chunk(7), cols)
		assert.Equal(t, consts.Chunk(9), rows)
	})
}
