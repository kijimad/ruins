package raw

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "リペア"
Description = "半分程度回復する"

[[Items]]
Name = "回復薬"
Description = "半分程度回復する"
`
	raw, err := Load(str)
	assert.NoError(t, err)

	expect := Master{
		Raws: Raws{
			Items: []Item{
				Item{Name: "リペア", Description: "半分程度回復する"},
				Item{Name: "回復薬", Description: "半分程度回復する"},
			},
		},
		ItemIndex: map[string]int{
			"リペア": 0,
			"回復薬": 1,
		},
		MemberIndex:       map[string]int{},
		RecipeIndex:       map[string]int{},
		CommandTableIndex: map[string]int{},
		DropTableIndex:    map[string]int{},
		ItemTableIndex:    map[string]int{},
		EnemyTableIndex:   map[string]int{},
		SpriteSheetIndex:  map[string]int{},
		TileIndex:         map[string]int{},
		PropIndex:         map[string]int{},
	}
	assert.Equal(t, expect, raw)
}

func TestGenerateItem(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "リペア"
SpriteSheetName = "field"
SpriteKey = "repair_item"
`
	raw, err := Load(str)
	assert.NoError(t, err)
	entitySpec, err := raw.NewItemSpec("リペア")
	assert.NoError(t, err)
	loc := gc.ItemLocationInPlayerBackpack
	entitySpec.ItemLocationType = &loc
	assert.NotNil(t, entitySpec.Name)
	assert.NotNil(t, entitySpec.Item)
	assert.NotNil(t, entitySpec.Description)
	assert.NotNil(t, entitySpec.SpriteRender)
}

func TestGenerateItemWithoutSprite(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "テストアイテム"
Description = "スプライトなしアイテム"
`
	raw, err := Load(str)
	assert.NoError(t, err)

	// 現在の実装ではスプライト情報なしでも生成される（デフォルト値が設定される）
	entitySpec, err := raw.NewItemSpec("テストアイテム")
	assert.NoError(t, err)
	loc := gc.ItemLocationInPlayerBackpack
	entitySpec.ItemLocationType = &loc
	assert.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Equal(t, "field", entitySpec.SpriteRender.SpriteSheetName)
	assert.Equal(t, "field_item", entitySpec.SpriteRender.SpriteKey)
}

func TestGenerateMemberWithSprite(t *testing.T) {
	t.Parallel()
	str := `
[[Members]]
Name = "テストプレイヤー"
Player = true
SpriteSheetName = "field"
SpriteKey = "player"
[Members.Attributes]
Vitality = 50
Strength = 50
Sensation = 5
Dexterity = 6
Agility = 5
Defense = 0
`
	raw, err := Load(str)
	assert.NoError(t, err)
	entitySpec, err := raw.NewPlayerSpec("テストプレイヤー")
	assert.NoError(t, err)

	// 基本コンポーネントの確認
	assert.NotNil(t, entitySpec.Name)
	assert.NotNil(t, entitySpec.Player)

	// SpriteRenderコンポーネントの確認
	assert.NotNil(t, entitySpec.SpriteRender, "SpriteRenderコンポーネントが設定されていない")
	assert.Equal(t, "field", entitySpec.SpriteRender.SpriteSheetName, "SpriteSheetNameが正しくない")
	assert.Equal(t, "player", entitySpec.SpriteRender.SpriteKey, "SpriteKeyが正しくない")
	assert.Equal(t, gc.DepthNumPlayer, entitySpec.SpriteRender.Depth, "Depthが正しくない")
}

func TestGenerateMemberWithoutSprite(t *testing.T) {
	t.Parallel()
	str := `
[[Members]]
Name = "スプライトなしキャラ"
Player = true
[Members.Attributes]
Vitality = 50
Strength = 50
Sensation = 5
Dexterity = 6
Agility = 5
Defense = 0
`
	raw, err := Load(str)
	assert.NoError(t, err)

	// 現在の実装ではスプライト情報なしでも生成される（空文字列が設定される）
	entitySpec, err := raw.NewPlayerSpec("スプライトなしキャラ")
	assert.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Equal(t, "", entitySpec.SpriteRender.SpriteSheetName)
	assert.Equal(t, "", entitySpec.SpriteRender.SpriteKey)
}

func TestGenerateMaterialWithSprite(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "テスト素材"
Description = "スプライト付き素材"
SpriteSheetName = "field"
SpriteKey = "field_item"
Stackable = true
`
	raw, err := Load(str)
	assert.NoError(t, err)
	entitySpec, err := raw.NewItemSpec("テスト素材")
	assert.NoError(t, err)
	loc := gc.ItemLocationInPlayerBackpack
	entitySpec.ItemLocationType = &loc

	// 基本コンポーネントの確認
	assert.NotNil(t, entitySpec.Name)
	assert.NotNil(t, entitySpec.Description)
	// Stackable=trueのアイテムにはStackableコンポーネントが付与される
	assert.NotNil(t, entitySpec.Stackable)

	// SpriteRenderコンポーネントの確認
	assert.NotNil(t, entitySpec.SpriteRender, "SpriteRenderコンポーネントが設定されていない")
	assert.Equal(t, "field", entitySpec.SpriteRender.SpriteSheetName, "SpriteSheetNameが正しくない")
	assert.Equal(t, "field_item", entitySpec.SpriteRender.SpriteKey, "SpriteKeyが正しくない")
	assert.Equal(t, gc.DepthNumRug, entitySpec.SpriteRender.Depth, "Depthが正しくない")
}

func TestGenerateMaterialWithoutSprite(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "スプライトなし素材"
Description = "スプライトなし素材"
`
	raw, err := Load(str)
	assert.NoError(t, err)

	// 現在の実装ではスプライト情報なしでも生成される（デフォルト値が設定される）
	entitySpec, err := raw.NewItemSpec("スプライトなし素材")
	assert.NoError(t, err)
	loc := gc.ItemLocationInPlayerBackpack
	entitySpec.ItemLocationType = &loc
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Equal(t, "field", entitySpec.SpriteRender.SpriteSheetName)
	assert.Equal(t, "field_item", entitySpec.SpriteRender.SpriteKey)
	// Stackable=true が設定されていないので Stackable コンポーネントは付かない
	assert.Nil(t, entitySpec.Stackable)
}

func TestLoadTilesFromRaw(t *testing.T) {
	t.Parallel()

	// テスト用のTOMLデータ（タイル定義を含む）
	tomlData := `
[[Tiles]]
Name = "TestFloor"
Description = "テスト用床タイル"

[[Tiles]]
Name = "TestWall"
Description = "テスト用壁タイル"
BlockPass = true

[[Items]]
Name = "テストアイテム"
Description = "テスト用"
`

	master, err := Load(tomlData)
	require.NoError(t, err, "raw.goからの読み込みに失敗")

	// 基本的なタイルが定義されていることを確認
	expectedTiles := []string{"TestFloor", "TestWall"}
	for _, tileName := range expectedTiles {
		_, ok := master.TileIndex[tileName]
		assert.True(t, ok, "タイル '%s' がインデックスに見つかりません", tileName)
	}
}

func TestGenerateTileFromRaw(t *testing.T) {
	t.Parallel()

	tomlData := `
[[Tiles]]
Name = "GenerateTestFloor"
Description = "生成テスト用床タイル"

[[Tiles]]
Name = "GenerateTestWall"
Description = "生成テスト用壁タイル"
BlockPass = true
`

	master, err := Load(tomlData)
	require.NoError(t, err, "テストTOMLの読み込みに失敗")

	// 床タイルの取得をテスト
	floorTile, err := master.GetTile("GenerateTestFloor")
	require.NoError(t, err, "床タイルの取得に失敗")
	assert.False(t, floorTile.BlockPass)

	// 壁タイルの取得をテスト
	wallTile, err := master.GetTile("GenerateTestWall")
	require.NoError(t, err, "壁タイルの取得に失敗")
	assert.True(t, wallTile.BlockPass)

	// 存在しないタイルのテスト（エラーが発生する）
	_, err = master.GetTile("NonExistent")
	assert.Error(t, err, "存在しないタイルでエラーが発生すべき")
}

// TestGenerateTileSpecFromRaw - TileSpecは削除されたためこのテストは不要

func TestTileHelperFunctionsFromRaw(t *testing.T) {
	t.Parallel()

	tomlData := `
[[Tiles]]
Name = "Helper1"
Description = "ヘルパー関数テスト1"

[[Tiles]]
Name = "Helper2"
Description = "ヘルパー関数テスト2"
BlockPass = true
`

	master, err := Load(tomlData)
	require.NoError(t, err, "テストTOMLの読み込みに失敗")

	// GetTile のテスト（存在するタイル）
	tileRaw, err := master.GetTile("Helper1")
	require.NoError(t, err, "タイル取得に失敗")
	assert.False(t, tileRaw.BlockPass)

	// GetTile のテスト（存在しないタイル）
	_, err = master.GetTile("NonExistent")
	assert.Error(t, err, "存在しないタイルでエラーが発生すべき")
}

func TestTilePropertiesFromRaw(t *testing.T) {
	t.Parallel()

	// Walkableフィールドのテスト
	tomlData := `
[[Tiles]]
Name = "EmptyLike"
Description = "空のような性質"
BlockPass = true

[[Tiles]]
Name = "FloorLike"
Description = "床のような性質"

[[Tiles]]
Name = "WallLike"
Description = "壁のような性質"
BlockPass = true
`

	master, err := Load(tomlData)
	require.NoError(t, err, "テストTOMLの読み込みに失敗")

	testCases := []struct {
		name         string
		expectedWalk bool
	}{
		{"EmptyLike", false},
		{"FloorLike", true},
		{"WallLike", false},
	}

	for _, tc := range testCases {
		tile, err := master.GetTile(tc.name)
		require.NoError(t, err, "タイル取得に失敗: %s", tc.name)
		actualWalk := !tile.BlockPass
		assert.Equal(t, tc.expectedWalk, actualWalk, "Walkableが期待値と一致しない: %s", tc.name)
	}
}

func TestLoadFromRealTileFile(t *testing.T) {
	t.Parallel()

	// 実際のraw.tomlファイルからタイル定義を読み込み
	master, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err, "実際のraw.tomlファイルの読み込みに失敗")

	// 基本的なタイルが定義されていることを確認
	expectedTiles := []string{"floor", "wall", "dirt"}
	for _, tileName := range expectedTiles {
		_, ok := master.TileIndex[tileName]
		assert.True(t, ok, "タイル '%s' が実際のファイルで見つかりません", tileName)
	}

	// 実際のタイル取得テスト
	floorTile, err := master.GetTile("floor")
	require.NoError(t, err, "床タイル取得に失敗")
	assert.False(t, floorTile.BlockPass)

	// 壁タイルテスト
	wallTile, err := master.GetTile("wall")
	require.NoError(t, err, "壁タイル取得に失敗")
	assert.True(t, wallTile.BlockPass)
}

func TestLoadWithUnknownFields(t *testing.T) {
	t.Parallel()

	// 未知のフィールドを含むTOMLデータ
	invalidToml := `
[[Items]]
Name = "テストアイテム"
Description = "正常なアイテム"
UnknownField = "これは未知のフィールド"

[[UnknownSection]]
SomeField = "これは未知のセクション"
`

	_, err := Load(invalidToml)
	assert.Error(t, err, "未知のフィールドがあるTOMLでエラーが発生すべき")
	assert.Contains(t, err.Error(), "unknown keys found in TOML", "エラーメッセージに未知のキーについての情報が含まれるべき")
}

func TestLoadWithValidFields(t *testing.T) {
	t.Parallel()

	// 正常なTOMLデータ（既知のフィールドのみ）
	validToml := `
[[Items]]
Name = "テストアイテム"
Description = "正常なアイテム"
SpriteSheetName = "test_sheet"
SpriteKey = "test_key"

[[Tiles]]
Name = "テストタイル"
Description = "正常なタイル"
`

	master, err := Load(validToml)
	assert.NoError(t, err, "正常なTOMLでエラーが発生してはいけない")
	assert.Equal(t, 1, len(master.Raws.Items), "アイテムが1つ読み込まれるべき")
	assert.Equal(t, 1, len(master.Raws.Tiles), "タイルが1つ読み込まれるべき")
}

func TestItemWithAnimKeys(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "アニメーションアイテム"
Description = "2フレームアニメーションするアイテム"
SpriteSheetName = "field"
SpriteKey = "item_0"
AnimKeys = ["item_0", "item_1"]
`
	raw, err := Load(str)
	assert.NoError(t, err)

	// AnimKeysが正しく読み込まれていることを確認
	assert.Equal(t, 1, len(raw.Raws.Items))
	item := raw.Raws.Items[0]
	assert.Equal(t, []string{"item_0", "item_1"}, item.AnimKeys)

	// NewItemSpecでAnimKeysがSpriteRenderに設定されることを確認
	entitySpec, err := raw.NewItemSpec("アニメーションアイテム")
	assert.NoError(t, err)
	loc := gc.ItemLocationInPlayerBackpack
	entitySpec.ItemLocationType = &loc
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Equal(t, []string{"item_0", "item_1"}, entitySpec.SpriteRender.AnimKeys)
}

func TestItemWithoutAnimKeys(t *testing.T) {
	t.Parallel()
	str := `
[[Items]]
Name = "静的アイテム"
Description = "アニメーションしないアイテム"
SpriteSheetName = "field"
SpriteKey = "static_item"
`
	raw, err := Load(str)
	assert.NoError(t, err)

	// AnimKeysが指定されていない場合はnil
	item := raw.Raws.Items[0]
	assert.Nil(t, item.AnimKeys)

	// NewItemSpecでもAnimKeysはnil
	entitySpec, err := raw.NewItemSpec("静的アイテム")
	assert.NoError(t, err)
	loc := gc.ItemLocationInPlayerBackpack
	entitySpec.ItemLocationType = &loc
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Nil(t, entitySpec.SpriteRender.AnimKeys)
}

func TestMemberWithAnimKeys(t *testing.T) {
	t.Parallel()
	str := `
[[Members]]
Name = "アニメーションキャラ"
Player = true
SpriteSheetName = "field"
SpriteKey = "player_0"
AnimKeys = ["player_0", "player_1"]
[Members.Attributes]
Vitality = 50
Strength = 50
Sensation = 5
Dexterity = 6
Agility = 5
Defense = 0
`
	raw, err := Load(str)
	assert.NoError(t, err)

	// AnimKeysが正しく読み込まれていることを確認
	assert.Equal(t, 1, len(raw.Raws.Members))
	member := raw.Raws.Members[0]
	assert.Equal(t, []string{"player_0", "player_1"}, member.AnimKeys)

	// NewPlayerSpecでAnimKeysがSpriteRenderに設定されることを確認
	entitySpec, err := raw.NewPlayerSpec("アニメーションキャラ")
	assert.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Equal(t, []string{"player_0", "player_1"}, entitySpec.SpriteRender.AnimKeys)
}

func TestMemberWithoutAnimKeys(t *testing.T) {
	t.Parallel()
	str := `
[[Members]]
Name = "静的キャラ"
Player = true
SpriteSheetName = "field"
SpriteKey = "static_player"
[Members.Attributes]
Vitality = 50
Strength = 50
Sensation = 5
Dexterity = 6
Agility = 5
Defense = 0
`
	raw, err := Load(str)
	assert.NoError(t, err)

	// AnimKeysが指定されていない場合はnil
	member := raw.Raws.Members[0]
	assert.Nil(t, member.AnimKeys)

	// NewPlayerSpecでもAnimKeysはnil
	entitySpec, err := raw.NewPlayerSpec("静的キャラ")
	assert.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Nil(t, entitySpec.SpriteRender.AnimKeys)
}

func TestPropWithAnimKeys(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "アニメーションProp"
Description = "2フレームアニメーションする置物"
BlockPass = false
BlockView = false
AnimKeys = ["fire_0_", "fire_1_"]

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "fire_0_"
Depth = 1
`
	raw, err := Load(str)
	assert.NoError(t, err)

	// AnimKeysが正しく読み込まれていることを確認
	assert.Equal(t, 1, len(raw.Raws.Props))
	prop := raw.Raws.Props[0]
	assert.Equal(t, []string{"fire_0_", "fire_1_"}, prop.AnimKeys)

	// NewPropSpecでAnimKeysがSpriteRenderに設定されることを確認
	entitySpec, err := raw.NewPropSpec("アニメーションProp")
	assert.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Equal(t, []string{"fire_0_", "fire_1_"}, entitySpec.SpriteRender.AnimKeys)
}

func TestPropWithoutAnimKeys(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "静的Prop"
Description = "アニメーションしない置物"
BlockPass = true
BlockView = false

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "prop_table"
Depth = 1
`
	raw, err := Load(str)
	assert.NoError(t, err)

	// AnimKeysが指定されていない場合はnil
	prop := raw.Raws.Props[0]
	assert.Nil(t, prop.AnimKeys)

	// NewPropSpecでもAnimKeysはnil
	entitySpec, err := raw.NewPropSpec("静的Prop")
	assert.NoError(t, err)
	assert.NotNil(t, entitySpec.SpriteRender)
	assert.Nil(t, entitySpec.SpriteRender.AnimKeys)
}
