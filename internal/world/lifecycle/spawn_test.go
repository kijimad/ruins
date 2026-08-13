package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetMaxStats(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		vitality    int
		strength    int
		sensation   int
		dexterity   int
		agility     int
		expectedHP  int
		description string
	}{
		{
			name:        "基本ステータス",
			vitality:    10,
			strength:    8,
			sensation:   7,
			dexterity:   6,
			agility:     9,
			expectedHP:  30 + 10*8 + 8 + 7, // 30 + 95 = 125
			description: "基本的なHP計算",
		},
		{
			name:        "中ステータス",
			vitality:    15,
			strength:    12,
			sensation:   10,
			dexterity:   8,
			agility:     11,
			expectedHP:  30 + 15*8 + 12 + 10, // 30 + 142 = 172
			description: "中ステータスでのHP計算",
		},
		{
			name:        "高ステータス",
			vitality:    20,
			strength:    18,
			sensation:   15,
			dexterity:   14,
			agility:     16,
			expectedHP:  30 + 20*8 + 18 + 15, // 30 + 193 = 223
			description: "高ステータスでのHP計算",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)

			entity := world.ECS.NewEntity()
			world.Components.Abilities.Add(entity, &gc.Abilities{
				Vitality:  gc.Ability{Base: tt.vitality, Total: 0},
				Strength:  gc.Ability{Base: tt.strength, Total: 0},
				Sensation: gc.Ability{Base: tt.sensation, Total: 0},
				Dexterity: gc.Ability{Base: tt.dexterity, Total: 0},
				Agility:   gc.Ability{Base: tt.agility, Total: 0},
				Defense:   gc.Ability{Base: 5, Total: 0},
			})

			world.Components.HP.Add(entity, &gc.HP{Current: 0, Max: 0})
			world.Components.WeightCapacity.Add(entity, &gc.WeightCapacity{})

			err := setMaxStats(world, entity)
			require.NoError(t, err)

			hp := world.Components.HP.Get(entity)
			abils := world.Components.Abilities.Get(entity)

			assert.Equal(t, tt.vitality, abils.Vitality.Total, "体力のTotal値が正しく初期化されていない")
			assert.Equal(t, tt.strength, abils.Strength.Total, "力のTotal値が正しく初期化されていない")
			assert.Equal(t, tt.sensation, abils.Sensation.Total, "感覚のTotal値が正しく初期化されていない")
			assert.Equal(t, tt.dexterity, abils.Dexterity.Total, "器用さのTotal値が正しく初期化されていない")
			assert.Equal(t, tt.agility, abils.Agility.Total, "素早さのTotal値が正しく初期化されていない")

			assert.Equal(t, tt.expectedHP, hp.Max, "最大HPの計算が正しくない: %s", tt.description)
			assert.Equal(t, tt.expectedHP, hp.Current, "現在HPが最大HPと同じでない: %s", tt.description)

			world.ECS.RemoveEntity(entity)
		})
	}
}

func TestSetMaxStats_WithoutComponents(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	entity := world.ECS.NewEntity()

	err := setMaxStats(world, entity)
	require.Error(t, err, "必要なコンポーネントがない場合はエラーを返すべき")
	assert.Contains(t, err.Error(), "does not have required components", "エラーメッセージが適切であるべき")

	world.ECS.RemoveEntity(entity)
}

func TestFullRecover(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// テスト用エンティティを作成
	entity := world.ECS.NewEntity()
	world.Components.Abilities.Add(entity, &gc.Abilities{
		Vitality:  gc.Ability{Base: 10, Total: 0},
		Strength:  gc.Ability{Base: 8, Total: 0},
		Sensation: gc.Ability{Base: 7, Total: 0},
		Dexterity: gc.Ability{Base: 6, Total: 0},
		Agility:   gc.Ability{Base: 9, Total: 0},
		Defense:   gc.Ability{Base: 5, Total: 0},
	})
	world.Components.HP.Add(entity, &gc.HP{Current: 0, Max: 0})
	world.Components.WeightCapacity.Add(entity, &gc.WeightCapacity{})

	err := FullRecover(world, entity)
	require.NoError(t, err, "FullRecoverがエラーを返すべきではない")

	hp := world.Components.HP.Get(entity)
	abils := world.Components.Abilities.Get(entity)

	assert.Equal(t, 10, abils.Vitality.Total, "体力のTotal値が正しく設定されていない")
	assert.Equal(t, 8, abils.Strength.Total, "力のTotal値が正しく設定されていない")

	expectedHP := 30 + 10*8 + 8 + 7 // 30 + 95 = 125
	assert.Equal(t, expectedHP, hp.Max, "最大HPが正しく計算されていない")
	assert.Equal(t, expectedHP, hp.Current, "現在HPが最大HPと一致していない")

	world.ECS.RemoveEntity(entity)
}

func TestSpawnEnemyHasAI(t *testing.T) {
	t.Parallel()
	// 敵エンティティがAIコンポーネントを持つことを確認
	world := testutil.InitTestWorld(t)

	// SpriteSheetsを初期化
	spriteSheets := make(map[string]gc.SpriteSheet)
	spriteSheets["field"] = gc.SpriteSheet{
		Sprites: map[string]gc.Sprite{
			"void":         {Width: 32, Height: 32},
			"wall_generic": {Width: 32, Height: 32},
			"floor":        {Width: 32, Height: 32},
			"player":       {Width: 32, Height: 32},
			"warp_next":    {Width: 32, Height: 32},
			"red_ball":     {Width: 32, Height: 32}, // 敵のスプライト
		},
	}
	world.Resources.SpriteSheets = spriteSheets

	// NPCを生成（タイル座標で指定）
	_, err := SpawnEnemy(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "fireball")
	require.NoError(t, err)

	// AIコンポーネントを持つエンティティが存在することを確認
	enemyFound := false
	enemyQuery := ecs.NewFilter2[gc.GridElement, gc.SoloAI](world.ECS).Query()
	for enemyQuery.Next() {
		enemyFound = true
	}

	assert.True(t, enemyFound, "SpawnEnemyで生成されたエンティティはAIコンポーネントを持つべき")
}

func TestSpawnEnemy_WithBoss(t *testing.T) {
	t.Parallel()

	initSpriteSheets := func(world w.World) {
		spriteSheets := make(map[string]gc.SpriteSheet)
		spriteSheets["field"] = gc.SpriteSheet{
			Sprites: map[string]gc.Sprite{
				"red_ball": {Width: 32, Height: 32},
			},
		}
		world.Resources.SpriteSheets = spriteSheets
	}

	t.Run("WithBossオプションでBossコンポーネントが付与される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		initSpriteSheets(world)
		enemy, err := SpawnEnemy(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "fireball", WithBoss())
		require.NoError(t, err)
		assert.True(t, world.Components.Boss.Has(enemy), "Bossコンポーネントを持つべき")
	})

	t.Run("オプションなしではBossコンポーネントが付与されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		initSpriteSheets(world)
		enemy, err := SpawnEnemy(world, consts.Coord[consts.Tile]{X: 6, Y: 6}, "fireball")
		require.NoError(t, err)
		assert.False(t, world.Components.Boss.Has(enemy), "Bossコンポーネントを持つべきではない")
	})
}

func TestSpawnEnemy_WithDropTable(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// SpriteSheetsを初期化
	spriteSheets := make(map[string]gc.SpriteSheet)
	spriteSheets["field"] = gc.SpriteSheet{
		Sprites: map[string]gc.Sprite{
			"red_ball": {Width: 32, Height: 32},
		},
	}
	world.Resources.SpriteSheets = spriteSheets

	// 「火の玉」を生成（DropTableが定義されている敵）
	enemy, err := SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "fireball")
	require.NoError(t, err, "火の玉の生成に失敗")

	// DropTableコンポーネントが付与されていることを確認
	assert.True(t, world.Components.DropTable.Has(enemy), "火の玉はDropTableコンポーネントを持つべき")

	dropTable := world.Components.DropTable.Get(enemy)
	assert.Equal(t, "fireball", dropTable.Name, "DropTableコンポーネントは同定キー(id)を持つべき")
}

func TestSpawnEnemy_AI(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	spriteSheets := make(map[string]gc.SpriteSheet)
	spriteSheets["field"] = gc.SpriteSheet{
		Sprites: map[string]gc.Sprite{
			"red_ball": {Width: 32, Height: 32},
		},
	}
	world.Resources.SpriteSheets = spriteSheets

	enemy, err := SpawnEnemy(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "fireball")
	require.NoError(t, err)

	assert.True(t, world.Components.SoloAI.Has(enemy))
	solo := world.Components.SoloAI.Get(enemy)
	assert.Equal(t, gc.CombatAttack, solo.CombatDefault)
	assert.Equal(t, gc.CombatAttack, solo.CombatCurrent)
}

func TestSpawnItem(t *testing.T) {
	t.Parallel()

	t.Run("Stackableなアイテムに複数個指定できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		item, err := SpawnBackpackItem(world, "healing_potion", 5)
		require.NoError(t, err)

		stackableComp := world.Components.Stackable.Get(item)
		assert.Equal(t, 5, stackableComp.Count)
	})

	t.Run("Stackableでないアイテムにcount=1を指定できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		item, err := SpawnBackpackItem(world, "wooden_sword", 1)
		require.NoError(t, err)

		assert.Equal(t, 1, query.GetEntityCount(world, item))
	})

	t.Run("Stackableでないアイテムにcount>1を指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := SpawnBackpackItem(world, "wooden_sword", 2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not stackable")
		assert.Contains(t, err.Error(), "count must be 1")
	})

	t.Run("count=0を指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := SpawnBackpackItem(world, "wooden_sword", 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "count must be positive")
	})

	t.Run("負のcountを指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := SpawnBackpackItem(world, "wooden_sword", -1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "count must be positive")
	})

	t.Run("存在しないアイテム名を指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := SpawnBackpackItem(world, "存在しないアイテム", 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "item not found")
	})
}

func TestSpawnDoor(t *testing.T) {
	t.Parallel()

	t.Run("縦向き扉のスポーン", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door, err := SpawnDoor(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, gc.DoorOrientationVertical)
		require.NoError(t, err, "SpawnDoor should not return an error")

		// SpriteRenderを確認（entity=0は有効なエンティティIDなので、コンポーネントの存在でチェック）
		require.True(t, world.Components.SpriteRender.Has(door))
		sprite := world.Components.SpriteRender.Get(door)
		assert.Equal(t, "field", sprite.SpriteSheetName)
		assert.Equal(t, "door_vertical_closed", sprite.SpriteKey)
		assert.Equal(t, gc.DepthNumTaller, sprite.Depth)

		// Doorコンポーネントを確認
		require.True(t, world.Components.Door.Has(door))
		doorComp := world.Components.Door.Get(door)
		assert.False(t, doorComp.IsOpen)
		assert.Equal(t, gc.DoorOrientationVertical, doorComp.Orientation)

		// BlockPass/BlockViewを確認
		assert.True(t, world.Components.BlockPass.Has(door))
		assert.True(t, world.Components.BlockView.Has(door))
	})

	t.Run("横向き扉のスポーン", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door, err := SpawnDoor(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, gc.DoorOrientationHorizontal)
		require.NoError(t, err)

		// SpriteRenderを確認
		sprite := world.Components.SpriteRender.Get(door)
		assert.Equal(t, "door_horizontal_closed", sprite.SpriteKey)

		// Doorコンポーネントを確認
		doorComp := world.Components.Door.Get(door)
		assert.Equal(t, gc.DoorOrientationHorizontal, doorComp.Orientation)
	})
}

func TestDoor_開閉で高さが切り替わる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	door, err := SpawnDoor(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, gc.DoorOrientationHorizontal)
	require.NoError(t, err)

	// 閉じた扉は高さのある障壁で、ドロップシャドウを落とす高さ
	assert.Equal(t, gc.DepthNumTaller, world.Components.SpriteRender.Get(door).Depth)

	// 開くと平らな低い物になり、影を落とさない高さへ下がる
	require.NoError(t, OpenDoor(world, door))
	assert.Equal(t, gc.DepthNumRug, world.Components.SpriteRender.Get(door).Depth)

	// 閉じると高さが戻る
	require.NoError(t, CloseDoor(world, door))
	assert.Equal(t, gc.DepthNumTaller, world.Components.SpriteRender.Get(door).Depth)
}

func TestSpawnVisualEffect(t *testing.T) {
	t.Parallel()

	t.Run("GridElementを持つエンティティにエフェクトが生成される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.GridElement.Add(entity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})

		effect := gc.NewHealEffect(10)
		SpawnVisualEffect(entity, effect, world)

		// エフェクトエンティティが生成されたことを確認
		var foundEffect bool
		effectQuery := ecs.NewFilter1[gc.VisualEffects](world.ECS).Query()
		for effectQuery.Next() {
			foundEffect = true
		}
		assert.True(t, foundEffect)
	})

	t.Run("GridElementがないエンティティではエフェクトは生成されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()

		effect := gc.NewHealEffect(10)
		SpawnVisualEffect(entity, effect, world)

		var foundEffect bool
		effectQuery := ecs.NewFilter1[gc.VisualEffects](world.ECS).Query()
		for effectQuery.Next() {
			foundEffect = true
		}
		assert.False(t, foundEffect)
	})
}

func TestAllItemsBelongToInventoryCategory(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	items := raw.PtrSlice(world.Resources.RawMaster.Items)
	require.NotEmpty(t, items, "rawデータにアイテムが存在する")

	var uncategorized []string
	for _, item := range items {
		entity, err := SpawnBackpackItem(world, item.Id, 1)
		require.NoError(t, err, "アイテム '%s' のスポーンに失敗", item.Id)

		_, ok := world.Components.CategoryOf(gc.InventoryCategoryKey, entity)
		if !ok {
			uncategorized = append(uncategorized, item.Id)
		}
	}
	assert.Empty(t, uncategorized, "InventoryCategoryに属していないアイテム: %v", uncategorized)
}

func TestSpawnNeutralNPC_商人を生成すると在庫と会話データを持つ(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	npc, err := SpawnNeutralNPC(world, consts.Coord[consts.Tile]{X: 3, Y: 4}, "merchant")
	require.NoError(t, err)

	require.True(t, world.Components.FactionNeutral.Has(npc), "中立派閥を持つ")
	require.True(t, world.Components.Dialog.Has(npc), "会話データを持つ")
	assert.Equal(t, "merchant_greeting", world.Components.Dialog.Get(npc).MessageKey)

	require.True(t, world.Components.GridElement.Has(npc))
	assert.Equal(t, consts.Coord[consts.Tile]{X: 3, Y: 4}, world.Components.GridElement.Get(npc).Coord)

	require.True(t, world.Components.HP.Has(npc))
	hp := world.Components.HP.Get(npc)
	assert.Equal(t, hp.Max, hp.Current, "全回復した状態で生成される")

	require.True(t, world.Components.SoloAI.Has(npc), "移動パターンを持つのでAIを持つ")
	solo := world.Components.SoloAI.Get(npc)
	assert.Equal(t, gc.AIStateWaiting, solo.SubState)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 3, Y: 4}, solo.Origin, "生成位置がパトロール原点になる")
	assert.Equal(t, consts.AIVisionDistance, solo.ViewDistance)

	stock := query.GetStorageItems(world, npc)
	assert.Len(t, stock, len(merchantStockItems), "商人は定義数の在庫を積む")
}

func TestSpawnNeutralNPC_商人以外は在庫を積まない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	npc, err := SpawnNeutralNPC(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "suspicious_scientist")
	require.NoError(t, err)

	require.True(t, world.Components.FactionNeutral.Has(npc))
	stock := query.GetStorageItems(world, npc)
	assert.Empty(t, stock, "商人でないNPCは在庫を積まない")
}

func TestSpawnNeutralNPC_中立でないメンバーはエラーになる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	_, err := SpawnNeutralNPC(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")

	require.Error(t, err)
	assert.EqualError(t, err, "'ash' is not a neutral NPC")
}

func TestSpawnNeutralNPC_存在しない名前はエラーになる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	_, err := SpawnNeutralNPC(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "no_such_member")

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to generate neutral NPC")
}

func TestSpawnFieldItem_腐敗食は生成時刻と保存期間を持つ(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetGameTime(world).TotalTurns = 700

	bread, err := SpawnFieldItem(world, "bread", 5, 5, 1)
	require.NoError(t, err)
	require.True(t, world.Components.Perishable.Has(bread), "bread は腐敗する")

	p := world.Components.Perishable.Get(bread)
	assert.Equal(t, consts.Turn(700), p.SpawnedAtTurn, "生成時の総ターンを刻む")
	assert.Equal(t, consts.Turn(1500), p.ShelfLife, "raw の shelfLife を持つ")
}

func TestSpawnFieldItem_保存期間なしは腐敗しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	sword, err := SpawnFieldItem(world, "wooden_sword", 5, 5, 1)
	require.NoError(t, err)
	assert.False(t, world.Components.Perishable.Has(sword), "shelfLife 無しは Perishable を持たない")
}
