package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestSetMaxHPSP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		vitality    int
		strength    int
		sensation   int
		dexterity   int
		agility     int
		expectedHP  int
		expectedSP  int
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
			expectedSP:  10*2 + 6 + 9,      // 35
			description: "基本的なHP/SP計算",
		},
		{
			name:        "中ステータス",
			vitality:    15,
			strength:    12,
			sensation:   10,
			dexterity:   8,
			agility:     11,
			expectedHP:  30 + 15*8 + 12 + 10, // 30 + 142 = 172
			expectedSP:  15*2 + 8 + 11,       // 49
			description: "中ステータスでのHP/SP計算",
		},
		{
			name:        "高ステータス",
			vitality:    20,
			strength:    18,
			sensation:   15,
			dexterity:   14,
			agility:     16,
			expectedHP:  30 + 20*8 + 18 + 15, // 30 + 193 = 223
			expectedSP:  20*2 + 14 + 16,      // 70
			description: "高ステータスでのHP/SP計算",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// 独立したworldを作成
			world := testutil.InitTestWorld(t)

			// エンティティを作成
			entity := world.Manager.NewEntity()

			// Attributesコンポーネントを追加（BaseとTotalを0に設定してsetMaxHPSPの初期化をテスト）
			entity.AddComponent(world.Components.Attributes, &gc.Attributes{
				Vitality:  gc.Attribute{Base: tt.vitality, Total: 0},
				Strength:  gc.Attribute{Base: tt.strength, Total: 0},
				Sensation: gc.Attribute{Base: tt.sensation, Total: 0},
				Dexterity: gc.Attribute{Base: tt.dexterity, Total: 0},
				Agility:   gc.Attribute{Base: tt.agility, Total: 0},
				Defense:   gc.Attribute{Base: 5, Total: 0},
			})

			// Poolsコンポーネントを追加
			entity.AddComponent(world.Components.Pools, &gc.Pools{
				HP: gc.Pool{Current: 0, Max: 0},
				SP: gc.Pool{Current: 0, Max: 0},
			})

			// 関数を実行
			err := setMaxHPSP(world, entity)
			require.NoError(t, err)

			// 結果を検証
			pools := world.Components.Pools.Get(entity).(*gc.Pools)
			attrs := world.Components.Attributes.Get(entity).(*gc.Attributes)

			// Totalが正しく初期化されたことを確認
			assert.Equal(t, tt.vitality, attrs.Vitality.Total, "体力のTotal値が正しく初期化されていない")
			assert.Equal(t, tt.strength, attrs.Strength.Total, "力のTotal値が正しく初期化されていない")
			assert.Equal(t, tt.sensation, attrs.Sensation.Total, "感覚のTotal値が正しく初期化されていない")
			assert.Equal(t, tt.dexterity, attrs.Dexterity.Total, "器用さのTotal値が正しく初期化されていない")
			assert.Equal(t, tt.agility, attrs.Agility.Total, "素早さのTotal値が正しく初期化されていない")

			// HP/SPが正しく計算されたことを確認
			assert.Equal(t, tt.expectedHP, pools.HP.Max, "最大HPの計算が正しくない: %s", tt.description)
			assert.Equal(t, tt.expectedHP, pools.HP.Current, "現在HPが最大HPと同じでない: %s", tt.description)
			assert.Equal(t, tt.expectedSP, pools.SP.Max, "最大SPの計算が正しくない: %s", tt.description)
			assert.Equal(t, tt.expectedSP, pools.SP.Current, "現在SPが最大SPと同じでない: %s", tt.description)

			// クリーンアップ
			world.Manager.DeleteEntity(entity)
		})
	}
}

func TestSetMaxHPSP_WithoutComponents(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 必要なコンポーネントがないエンティティ
	entity := world.Manager.NewEntity()

	// 関数を実行してエラーが発生することを確認
	err := setMaxHPSP(world, entity)
	require.Error(t, err, "必要なコンポーネントがない場合はエラーを返すべき")
	assert.Contains(t, err.Error(), "does not have required components", "エラーメッセージが適切であるべき")

	// クリーンアップ
	world.Manager.DeleteEntity(entity)
}

func TestFullRecover(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// テスト用エンティティを作成
	entity := world.Manager.NewEntity()
	entity.AddComponent(world.Components.Attributes, &gc.Attributes{
		Vitality:  gc.Attribute{Base: 10, Total: 0},
		Strength:  gc.Attribute{Base: 8, Total: 0},
		Sensation: gc.Attribute{Base: 7, Total: 0},
		Dexterity: gc.Attribute{Base: 6, Total: 0},
		Agility:   gc.Attribute{Base: 9, Total: 0},
		Defense:   gc.Attribute{Base: 5, Total: 0},
	})
	entity.AddComponent(world.Components.Pools, &gc.Pools{
		HP: gc.Pool{Current: 0, Max: 0},
		SP: gc.Pool{Current: 0, Max: 0},
	})

	// FullRecoverを実行
	err := fullRecover(world, entity)
	require.NoError(t, err, "FullRecoverがエラーを返すべきではない")

	// 結果を検証
	pools := world.Components.Pools.Get(entity).(*gc.Pools)
	attrs := world.Components.Attributes.Get(entity).(*gc.Attributes)

	// 属性のTotalが正しく設定されたことを確認
	assert.Equal(t, 10, attrs.Vitality.Total, "体力のTotal値が正しく設定されていない")
	assert.Equal(t, 8, attrs.Strength.Total, "力のTotal値が正しく設定されていない")

	// HP/SPが正しく計算されたことを確認
	expectedHP := int(30 + float64(10*8+8+7)*1.0) // 30 + 95 = 125
	expectedSP := int(float64(10*2+6+9) * 1.0)    // 35
	assert.Equal(t, expectedHP, pools.HP.Max, "最大HPが正しく計算されていない")
	assert.Equal(t, expectedHP, pools.HP.Current, "現在HPが最大HPと一致していない")
	assert.Equal(t, expectedSP, pools.SP.Max, "最大SPが正しく計算されていない")
	assert.Equal(t, expectedSP, pools.SP.Current, "現在SPが最大SPと一致していない")

	// クリーンアップ
	world.Manager.DeleteEntity(entity)
}

func TestSpawnNPCHasAIMoveFSM(t *testing.T) {
	t.Parallel()
	// NPCが敵として認識されるAIMoveFSMコンポーネントを持つことを確認
	world := testutil.InitTestWorld(t)

	// SpriteSheetsを初期化
	spriteSheets := make(map[string]gc.SpriteSheet)
	spriteSheets["field"] = gc.SpriteSheet{
		Sprites: map[string]gc.Sprite{
			"exterior":     {Width: 32, Height: 32},
			"wall_generic": {Width: 32, Height: 32},
			"floor":        {Width: 32, Height: 32},
			"player":       {Width: 32, Height: 32},
			"warp_next":    {Width: 32, Height: 32},
			"warp_escape":  {Width: 32, Height: 32},
			"red_ball":     {Width: 32, Height: 32}, // 敵のスプライト
		},
	}
	world.Resources.SpriteSheets = &spriteSheets

	// NPCを生成（タイル座標で指定）
	_, err := SpawnEnemy(world, 5, 5, "火の玉")
	require.NoError(t, err)

	// AIMoveFSMコンポーネントを持つエンティティが存在することを確認
	enemyFound := false
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.AIMoveFSM,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		enemyFound = true
	}))

	assert.True(t, enemyFound, "SpawnEnemyで生成されたエンティティはAIMoveFSMコンポーネントを持つべき")
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
	world.Resources.SpriteSheets = &spriteSheets

	// 「火の玉」を生成（DropTableが定義されている敵）
	enemy, err := SpawnEnemy(world, 10, 10, "火の玉")
	require.NoError(t, err, "火の玉の生成に失敗")

	// DropTableコンポーネントが付与されていることを確認
	assert.True(t, enemy.HasComponent(world.Components.DropTable), "火の玉はDropTableコンポーネントを持つべき")

	dropTable := world.Components.DropTable.Get(enemy).(*gc.DropTable)
	assert.Equal(t, "火の玉", dropTable.Name, "DropTableの名前が正しくない")
}

func TestSpawnItem(t *testing.T) {
	t.Parallel()

	t.Run("Stackableなアイテムに複数個指定できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		item, err := SpawnItem(world, "回復薬", 5, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)

		itemComp := world.Components.Item.Get(item).(*gc.Item)
		assert.Equal(t, 5, itemComp.Count)
	})

	t.Run("Stackableでないアイテムにcount=1を指定できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		item, err := SpawnItem(world, "木刀", 1, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)

		itemComp := world.Components.Item.Get(item).(*gc.Item)
		assert.Equal(t, 1, itemComp.Count)
	})

	t.Run("Stackableでないアイテムにcount>1を指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := SpawnItem(world, "木刀", 2, gc.ItemLocationInPlayerBackpack)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not stackable")
		assert.Contains(t, err.Error(), "count must be 1")
	})

	t.Run("count=0を指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := SpawnItem(world, "木刀", 0, gc.ItemLocationInPlayerBackpack)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "count must be positive")
	})

	t.Run("負のcountを指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := SpawnItem(world, "木刀", -1, gc.ItemLocationInPlayerBackpack)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "count must be positive")
	})

	t.Run("存在しないアイテム名を指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := SpawnItem(world, "存在しないアイテム", 1, gc.ItemLocationInPlayerBackpack)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "item not found")
	})
}

func TestSpawnDoor(t *testing.T) {
	t.Parallel()

	t.Run("縦向きドアのスポーン", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door, err := SpawnDoor(world, 10, 10, gc.DoorOrientationVertical)
		require.NoError(t, err, "SpawnDoor should not return an error")

		// SpriteRenderを確認（entity=0は有効なエンティティIDなので、コンポーネントの存在でチェック）
		require.True(t, door.HasComponent(world.Components.SpriteRender))
		sprite := world.Components.SpriteRender.Get(door).(*gc.SpriteRender)
		assert.Equal(t, "field", sprite.SpriteSheetName)
		assert.Equal(t, "door_vertical_closed", sprite.SpriteKey)
		assert.Equal(t, gc.DepthNumTaller, sprite.Depth)

		// Doorコンポーネントを確認
		require.True(t, door.HasComponent(world.Components.Door))
		doorComp := world.Components.Door.Get(door).(*gc.Door)
		assert.False(t, doorComp.IsOpen)
		assert.Equal(t, gc.DoorOrientationVertical, doorComp.Orientation)

		// BlockPass/BlockViewを確認
		assert.True(t, door.HasComponent(world.Components.BlockPass))
		assert.True(t, door.HasComponent(world.Components.BlockView))
	})

	t.Run("横向きドアのスポーン", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		door, err := SpawnDoor(world, 10, 10, gc.DoorOrientationHorizontal)
		require.NoError(t, err)

		// SpriteRenderを確認
		sprite := world.Components.SpriteRender.Get(door).(*gc.SpriteRender)
		assert.Equal(t, "door_horizontal_closed", sprite.SpriteKey)

		// Doorコンポーネントを確認
		doorComp := world.Components.Door.Get(door).(*gc.Door)
		assert.Equal(t, gc.DoorOrientationHorizontal, doorComp.Orientation)
	})
}

func TestMovePlayerToPosition(t *testing.T) {
	t.Parallel()

	t.Run("正常にプレイヤーの位置を更新できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
		player.AddComponent(world.Components.SpriteRender, &gc.SpriteRender{})
		player.AddComponent(world.Components.Camera, &gc.Camera{})

		// プレイヤーを移動
		err := MovePlayerToPosition(world, 10, 15)
		require.NoError(t, err)

		// 位置が更新されていることを確認
		gridElement := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, gc.Tile(10), gridElement.X)
		assert.Equal(t, gc.Tile(15), gridElement.Y)
	})

	t.Run("プレイヤーが存在しない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーなしで実行
		err := MovePlayerToPosition(world, 10, 15)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "必須コンポーネントを持つプレイヤーエンティティが見つかりません")
	})

	t.Run("必須コンポーネントが欠けている場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// GridElementなしのプレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.SpriteRender, &gc.SpriteRender{})
		player.AddComponent(world.Components.Camera, &gc.Camera{})

		err := MovePlayerToPosition(world, 10, 15)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "必須コンポーネントを持つプレイヤーエンティティが見つかりません")
	})
}
