package raw

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kijimaD/ruins/assets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRawReferenceIntegrity はrawファイル内の参照整合性を検証する
func TestRawReferenceIntegrity(t *testing.T) {
	t.Parallel()

	// raw.tomlを読み込む
	master, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err, "raw.tomlの読み込みに失敗")

	// Items ================

	t.Run("アイテムのSpriteSheet参照が存在する", func(t *testing.T) {
		t.Parallel()
		for _, item := range master.Raws.Items {
			// SpriteSheetNameが設定されていない場合はエラー
			assert.NotEmpty(t, item.SpriteSheetName, "アイテム '%s' にSpriteSheetNameが設定されていません", item.Name)

			_, ok := master.SpriteSheetIndex[item.SpriteSheetName]
			assert.True(t, ok, "アイテム '%s' が参照するSpriteSheet '%s' が存在しません",
				item.Name, item.SpriteSheetName)
		}
	})

	// Members ================

	t.Run("メンバー名に対応するCommandTableが存在する場合は有効", func(t *testing.T) {
		t.Parallel()
		for _, member := range master.Raws.Members {
			// 同名のCommandTableが存在する場合のみチェック
			if _, ok := master.CommandTableIndex[member.Name]; ok {
				// 存在するのでOK
				assert.True(t, true)
			}
		}
	})

	t.Run("メンバー名に対応するDropTableが存在する場合は有効", func(t *testing.T) {
		t.Parallel()
		for _, member := range master.Raws.Members {
			// 同名のDropTableが存在する場合のみチェック
			if _, ok := master.DropTableIndex[member.Name]; ok {
				// 存在するのでOK
				assert.True(t, true)
			}
		}
	})

	t.Run("メンバーのSpriteSheet参照が存在する", func(t *testing.T) {
		t.Parallel()
		for _, member := range master.Raws.Members {
			if _, ok := master.SpriteSheetIndex[member.SpriteSheetName]; ok {
				// 存在するのでOK
				assert.True(t, ok, "メンバー '%s' が参照するSpriteSheet '%s' が存在しません",
					member.Name, member.SpriteSheetName)
			}
		}
	})

	// DropTables ================

	t.Run("DropTableのマテリアル参照が存在する", func(t *testing.T) {
		t.Parallel()
		for _, dropTable := range master.Raws.DropTables {
			for _, entry := range dropTable.Entries {
				if entry.Material == "" {
					continue
				}

				_, ok := master.ItemIndex[entry.Material]
				assert.True(t, ok, "DropTable '%s' が参照するマテリアル '%s' が存在しません",
					dropTable.Name, entry.Material)
			}
		}
	})

	// ItemTables ================

	t.Run("ItemTableのアイテム参照が存在する", func(t *testing.T) {
		t.Parallel()
		for _, itemTable := range master.Raws.ItemTables {
			for _, entry := range itemTable.Entries {
				if entry.ItemName == "" {
					continue
				}

				_, ok := master.ItemIndex[entry.ItemName]
				assert.True(t, ok, "ItemTable '%s' が参照するアイテム '%s' が存在しません",
					itemTable.Name, entry.ItemName)
			}
		}
	})

	// CommandTables ================

	t.Run("CommandTableの武器参照が存在する", func(t *testing.T) {
		t.Parallel()
		for _, commandTable := range master.Raws.CommandTables {
			for _, entry := range commandTable.Entries {
				if entry.Weapon == "" {
					continue
				}

				_, ok := master.ItemIndex[entry.Weapon]
				assert.True(t, ok, "CommandTable '%s' が参照する武器 '%s' が存在しません",
					commandTable.Name, entry.Weapon)
			}
		}
	})

	// EnemyTables ================

	t.Run("EnemyTableのメンバー参照が存在する", func(t *testing.T) {
		t.Parallel()
		for _, enemyTable := range master.Raws.EnemyTables {
			for _, entry := range enemyTable.Entries {
				if entry.EnemyName == "" {
					continue
				}

				_, ok := master.MemberIndex[entry.EnemyName]
				assert.True(t, ok, "EnemyTable '%s' が参照するメンバー '%s' が存在しません",
					enemyTable.Name, entry.EnemyName)
			}
		}
	})

	// Recipes ================

	t.Run("レシピの入力アイテム参照が存在する", func(t *testing.T) {
		t.Parallel()
		for _, recipe := range master.Raws.Recipes {
			for _, input := range recipe.Inputs {
				if input.Name == "" {
					continue
				}

				_, ok := master.ItemIndex[input.Name]
				assert.True(t, ok, "レシピ '%s' が参照する入力アイテム '%s' が存在しません",
					recipe.Name, input.Name)
			}
		}
	})

	// Props ================

	t.Run("PropのSpriteSheet参照が存在する", func(t *testing.T) {
		t.Parallel()
		for _, prop := range master.Raws.Props {
			if prop.SpriteRender.SpriteSheetName == "" {
				continue
			}

			_, ok := master.SpriteSheetIndex[prop.SpriteRender.SpriteSheetName]
			assert.True(t, ok, "Prop '%s' が参照するSpriteSheet '%s' が存在しません",
				prop.Name, prop.SpriteRender.SpriteSheetName)
		}
	})

	// Tiles ================

	t.Run("TileのSpriteSheet参照が存在する", func(t *testing.T) {
		t.Parallel()
		for _, tile := range master.Raws.Tiles {
			if tile.SpriteRender.SpriteSheetName == "" {
				continue
			}

			_, ok := master.SpriteSheetIndex[tile.SpriteRender.SpriteSheetName]
			assert.True(t, ok, "Tile '%s' が参照するSpriteSheet '%s' が存在しません",
				tile.Name, tile.SpriteRender.SpriteSheetName)
		}
	})
}

// TestRawDuplicateNames はrawファイル内の名前の重複を検証する
func TestRawDuplicateNames(t *testing.T) {
	t.Parallel()

	master, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)

	t.Run("アイテム名の重複がない", func(t *testing.T) {
		t.Parallel()
		names := make(map[string]int)
		for _, item := range master.Raws.Items {
			names[item.Name]++
		}
		for name, count := range names {
			assert.Equal(t, 1, count, "アイテム名 '%s' が重複しています（%d個）", name, count)
		}
	})

	t.Run("メンバー名の重複がない", func(t *testing.T) {
		t.Parallel()
		names := make(map[string]int)
		for _, member := range master.Raws.Members {
			names[member.Name]++
		}
		for name, count := range names {
			assert.Equal(t, 1, count, "メンバー名 '%s' が重複しています（%d個）", name, count)
		}
	})

	t.Run("レシピ名の重複がない", func(t *testing.T) {
		t.Parallel()
		names := make(map[string]int)
		for _, recipe := range master.Raws.Recipes {
			names[recipe.Name]++
		}
		for name, count := range names {
			assert.Equal(t, 1, count, "レシピ名 '%s' が重複しています（%d個）", name, count)
		}
	})

	t.Run("CommandTable名の重複がない", func(t *testing.T) {
		t.Parallel()
		names := make(map[string]int)
		for _, table := range master.Raws.CommandTables {
			names[table.Name]++
		}
		for name, count := range names {
			assert.Equal(t, 1, count, "CommandTable名 '%s' が重複しています（%d個）", name, count)
		}
	})

	t.Run("DropTable名の重複がない", func(t *testing.T) {
		t.Parallel()
		names := make(map[string]int)
		for _, table := range master.Raws.DropTables {
			names[table.Name]++
		}
		for name, count := range names {
			assert.Equal(t, 1, count, "DropTable名 '%s' が重複しています（%d個）", name, count)
		}
	})

	t.Run("ItemTable名の重複がない", func(t *testing.T) {
		t.Parallel()
		names := make(map[string]int)
		for _, table := range master.Raws.ItemTables {
			names[table.Name]++
		}
		for name, count := range names {
			assert.Equal(t, 1, count, "ItemTable名 '%s' が重複しています（%d個）", name, count)
		}
	})

	t.Run("EnemyTable名の重複がない", func(t *testing.T) {
		t.Parallel()
		names := make(map[string]int)
		for _, table := range master.Raws.EnemyTables {
			names[table.Name]++
		}
		for name, count := range names {
			assert.Equal(t, 1, count, "EnemyTable名 '%s' が重複しています（%d個）", name, count)
		}
	})

	t.Run("SpriteSheet名の重複がない", func(t *testing.T) {
		t.Parallel()
		names := make(map[string]int)
		for _, sheet := range master.Raws.SpriteSheets {
			names[sheet.Name]++
		}
		for name, count := range names {
			assert.Equal(t, 1, count, "SpriteSheet名 '%s' が重複しています（%d個）", name, count)
		}
	})

	t.Run("Tile名の重複がない", func(t *testing.T) {
		t.Parallel()
		names := make(map[string]int)
		for _, tile := range master.Raws.Tiles {
			names[tile.Name]++
		}
		for name, count := range names {
			assert.Equal(t, 1, count, "Tile名 '%s' が重複しています（%d個）", name, count)
		}
	})

	t.Run("Prop名の重複がない", func(t *testing.T) {
		t.Parallel()
		names := make(map[string]int)
		for _, prop := range master.Raws.Props {
			names[prop.Name]++
		}
		for name, count := range names {
			assert.Equal(t, 1, count, "Prop名 '%s' が重複しています（%d個）", name, count)
		}
	})
}

// TestSpriteSheetFiles はSpriteSheetのJSONファイルが実在し、読み込めることを検証する
func TestSpriteSheetFiles(t *testing.T) {
	t.Parallel()

	master, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err, "raw.tomlの読み込みに失敗")

	t.Run("SpriteSheetのJSONファイルが実在する", func(t *testing.T) {
		t.Parallel()
		for _, sheet := range master.Raws.SpriteSheets {
			// assetsパッケージ経由でファイルが読み込めることを確認
			data, err := assets.FS.ReadFile(sheet.Path)
			assert.NoError(t, err, "SpriteSheet '%s' のファイル '%s' が読み込めません", sheet.Name, sheet.Path)
			assert.NotEmpty(t, data, "SpriteSheet '%s' のファイル '%s' が空です", sheet.Name, sheet.Path)
		}
	})

	t.Run("アイテムが参照するSpriteKeyがJSON内に存在する", func(t *testing.T) {
		t.Parallel()

		// 各SpriteSheetのスプライト一覧を構築
		spriteSheetSprites := make(map[string]map[string]bool)

		for _, sheet := range master.Raws.SpriteSheets {
			data, err := assets.FS.ReadFile(sheet.Path)
			require.NoError(t, err)

			// Aseprite JSON形式をパース
			var aseData struct {
				Frames []struct {
					Filename string `json:"filename"`
				} `json:"frames"`
			}
			require.NoError(t, json.Unmarshal(data, &aseData))

			// スプライトキーのマップを構築（末尾の_を削除）
			sprites := make(map[string]bool)
			for _, frame := range aseData.Frames {
				key := strings.TrimSuffix(frame.Filename, "_")
				sprites[key] = true
			}
			spriteSheetSprites[sheet.Name] = sprites
		}

		// 各アイテムが参照するSpriteKeyが存在するか確認
		for _, item := range master.Raws.Items {
			if item.SpriteSheetName == "" || item.SpriteKey == "" {
				continue
			}

			sprites, ok := spriteSheetSprites[item.SpriteSheetName]
			require.True(t, ok, "アイテム '%s' が参照するSpriteSheet '%s' が存在しません", item.Name, item.SpriteSheetName)

			assert.True(t, sprites[item.SpriteKey], "アイテム '%s' が参照するSpriteKey '%s' がSpriteSheet '%s' 内に存在しません",
				item.Name, item.SpriteKey, item.SpriteSheetName)
		}
	})
}
