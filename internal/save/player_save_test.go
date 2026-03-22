package save

import (
	"os"
	"path/filepath"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestPlayerComponentSaveLoad(t *testing.T) {
	t.Parallel()
	// テスト用ディレクトリを準備
	testDir := "./test_player_save"
	defer func() {
		_ = os.RemoveAll(testDir)
	}()

	// テスト用のワールドを作成
	world := testutil.InitTestWorld(t)

	// プレイヤーエンティティを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.Name, &gc.Name{Name: "主人公"})
	player.AddComponent(world.Components.Abilities, &gc.Abilities{
		Vitality:  gc.Ability{Base: 10},
		Strength:  gc.Ability{Base: 15},
		Sensation: gc.Ability{Base: 12},
		Dexterity: gc.Ability{Base: 14},
		Agility:   gc.Ability{Base: 13},
		Defense:   gc.Ability{Base: 8},
	})
	player.AddComponent(world.Components.Pools, &gc.Pools{})
	player.AddComponent(world.Components.FactionAlly, &gc.FactionAllyData{})
	player.AddComponent(world.Components.Player, &gc.Player{})

	// セーブマネージャーを作成してセーブ
	saveManager := NewSerializationManager(testDir)
	err := saveManager.SaveWorld(world, "player_test")
	require.NoError(t, err)

	// セーブファイルの存在確認
	saveFile := filepath.Join(testDir, "player_test.json")
	_, err = os.Stat(saveFile)
	assert.NoError(t, err, "Save file should exist")

	// 新しいワールドを作成してロード
	newWorld := testutil.InitTestWorld(t)

	err = saveManager.LoadWorld(newWorld, "player_test")
	require.NoError(t, err)

	playerCount := 0
	playerEntity := ecs.Entity(0)

	// プレイヤーエンティティを探す
	newWorld.Manager.Join(
		newWorld.Components.Player,
		newWorld.Components.Name,
		newWorld.Components.FactionAlly,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		playerCount++
		playerEntity = entity
	}))

	// 検証
	assert.Equal(t, 1, playerCount, "Should have 1 player")

	// プレイヤーの詳細検証
	assert.True(t, playerEntity.HasComponent(newWorld.Components.Player), "Player should have Player component")
	assert.True(t, playerEntity.HasComponent(newWorld.Components.Name), "Player should have Name component")
	assert.True(t, playerEntity.HasComponent(newWorld.Components.Abilities), "Player should have Abilities component")
	assert.True(t, playerEntity.HasComponent(newWorld.Components.Pools), "Player should have Pools component")
	assert.True(t, playerEntity.HasComponent(newWorld.Components.FactionAlly), "Player should have FactionAlly component")
	assert.True(t, playerEntity.HasComponent(newWorld.Components.Player), "Player should have Player component")

	// プレイヤーのデータ検証
	playerName := newWorld.Components.Name.Get(playerEntity).(*gc.Name)
	assert.Equal(t, "主人公", playerName.Name)

	playerPools := newWorld.Components.Pools.Get(playerEntity).(*gc.Pools)
	assert.NotNil(t, playerPools)

	playerAbils := newWorld.Components.Abilities.Get(playerEntity).(*gc.Abilities)
	assert.Equal(t, 10, playerAbils.Vitality.Base)
	assert.Equal(t, 15, playerAbils.Strength.Base)

	t.Logf("Player entity: %v", playerEntity)
}
