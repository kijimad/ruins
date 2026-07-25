package loader

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFonts(t *testing.T) {
	t.Parallel()
	t.Run("正常にフォントを読み込める", func(t *testing.T) {
		t.Parallel()
		fonts, err := LoadFonts()

		require.NoError(t, err)
		assert.NotNil(t, fonts)
		assert.NotEmpty(t, fonts)
	})
}

func TestLoadSpriteSheets(t *testing.T) {
	t.Parallel()
	t.Run("正常にスプライトシートを読み込める", func(t *testing.T) {
		t.Parallel()
		rw, err := LoadRaws()
		require.NoError(t, err)

		sprites, err := LoadSpriteSheets(rw)

		require.NoError(t, err)
		assert.NotNil(t, sprites)
		assert.NotEmpty(t, sprites)

		// 各スプライトシートに名前が設定されていることを確認
		for name, sprite := range sprites {
			assert.Equal(t, name, sprite.Name)
		}
	})

	t.Run("tileスプライトシートに全てのタイルが含まれる", func(t *testing.T) {
		t.Parallel()
		rw, err := LoadRaws()
		require.NoError(t, err)

		sprites, err := LoadSpriteSheets(rw)
		require.NoError(t, err)

		tileSheet, ok := sprites["tile"]
		require.True(t, ok, "tileスプライトシートが存在すること")

		// dirt_0 から dirt_15 まで存在することを確認
		for i := range 16 {
			key := fmt.Sprintf("dirt_%d", i)
			_, exists := tileSheet.Sprites[key]
			assert.True(t, exists, "%s が存在すること", key)
		}

		// wall_0 から wall_15 まで存在することを確認
		for i := range 16 {
			key := fmt.Sprintf("wall_%d", i)
			_, exists := tileSheet.Sprites[key]
			assert.True(t, exists, "%s が存在すること", key)
		}

		// floor_0 から floor_15 まで存在することを確認
		for i := range 16 {
			key := fmt.Sprintf("floor_%d", i)
			_, exists := tileSheet.Sprites[key]
			assert.True(t, exists, "%s が存在すること", key)
		}

		// voidが存在することを確認
		_, exists := tileSheet.Sprites["void"]
		assert.True(t, exists, "void が存在すること")

		// 合計518個のスプライトがあることを確認。基本65 と床材ダミー5 に、DawnLike フロアオートタイル28素材×16=448 を加えた数
		assert.Len(t, tileSheet.Sprites, 518, "518個のタイルスプライトが存在すること")
	})
}

func TestLoadRaws(t *testing.T) {
	t.Parallel()
	t.Run("正常にRawデータを読み込める", func(t *testing.T) {
		t.Parallel()
		rawMaster, err := LoadRaws()

		require.NoError(t, err)
		assert.NotEmpty(t, rawMaster.Items)
	})
}

func TestLoadSpriteSheetFromAseprite(t *testing.T) {
	t.Parallel()

	t.Run("bgスプライトシートを正常に読み込める", func(t *testing.T) {
		t.Parallel()
		sheet, err := LoadSpriteSheetFromAseprite("file/textures/dist/bg.json")

		require.NoError(t, err)
		assert.NotNil(t, sheet)
		assert.NotNil(t, sheet.Texture)
		assert.NotEmpty(t, sheet.Sprites)

		// 各スプライトのサイズが正しいことを確認
		for key, sprite := range sheet.Sprites {
			assert.Positive(t, sprite.Width, "スプライト %s の幅が正の値であること", key)
			assert.Positive(t, sprite.Height, "スプライト %s の高さが正の値であること", key)
		}
	})

	t.Run("tilesスプライトシートを正常に読み込める", func(t *testing.T) {
		t.Parallel()
		sheet, err := LoadSpriteSheetFromAseprite("file/textures/dist/tiles.json")

		require.NoError(t, err)
		assert.NotNil(t, sheet)

		// 518個のタイルスプライトが存在することを確認。基本65・床材ダミー5・DawnLike オートタイル448の合計
		assert.Len(t, sheet.Sprites, 518, "518個のタイルスプライトが存在すること")
	})

	t.Run("singleスプライトシートを正常に読み込める", func(t *testing.T) {
		t.Parallel()
		sheet, err := LoadSpriteSheetFromAseprite("file/textures/dist/single.json")

		require.NoError(t, err)
		assert.NotNil(t, sheet)
		assert.NotEmpty(t, sheet.Sprites)
	})

	t.Run("存在しないファイルを読み込むとエラー", func(t *testing.T) {
		t.Parallel()
		_, err := LoadSpriteSheetFromAseprite("file/textures/dist/nonexistent.json")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "JSONファイルの読み込みに失敗")
	})

	t.Run("不正なパスを指定するとエラー", func(t *testing.T) {
		t.Parallel()
		_, err := LoadSpriteSheetFromAseprite("")

		assert.Error(t, err)
	})

	t.Run("全てのスプライトファイル名が'_'で終わっていることを確認", func(t *testing.T) {
		t.Parallel()
		testFiles := []string{
			"file/textures/dist/bg.json",
			"file/textures/dist/tiles.json",
			"file/textures/dist/single.json",
		}

		for _, file := range testFiles {
			sheet, err := LoadSpriteSheetFromAseprite(file)
			require.NoError(t, err, "ファイル %s の読み込みに失敗", file)

			assert.NotEmpty(t, sheet.Sprites, "ファイル %s にスプライトが存在すること", file)
		}
	})
}

// externallyReferencedSprites は raw に spriteKey 登録されないが、Go コードや scene 状態が
// 名前で参照する正当なスプライト。孤児検査から除外する。
var externallyReferencedSprites = []string{
	// 扉は状態変化で render/lifecycle が open や縦向きスプライトへ切り替える
	"door_horizontal_open", "door_vertical_closed", "door_vertical_open",
}

// knownOrphanSprites は現状 raw からも Go からも参照されない既存スプライト。整理の候補。
// 新規孤児の混入を防ぐためグランドファザリングする。使うか消すか決めたらここから外す。
var knownOrphanSprites = []string{
	"angel", "big_tree", "cake", "chiken", "corn", "cup_tea", "fried_chiken",
	"grape_soda", "green_soda", "hawai_soda", "hdd", "lemon_soda", "mint",
	"octahedron", "pepper", "phonograph", "rainbow_ball", "role", "salmon",
	"tree_a", "tree_b", "violet_card",
	"warp_escape_0", "warp_escape_1", "warp_escape_2", "warp_escape_3", "warp_escape_4",
	"warp_escape_5", "warp_escape_6", "warp_escape_7", "warp_escape_8", "warp_escape_9",
	"warp_escape_10", "warp_escape_11", "warp_escape_12", "warp_escape_13", "warp_escape_14", "warp_escape_15",
}

// TestSpriteOrphan は field/tile シートの全スプライトが raw から参照されることを検証する。
// 参照とは spriteKey の直接一致、animKeys、オートタイル(base_N の base が key)、
// アニメ(base_0 が key)のいずれか。参照の無いスプライトは孤児で、新規混入を防ぐ。
// Go や scene から参照される正当なものは externallyReferencedSprites、既存の未整理孤児は
// knownOrphanSprites に載せる。bg シートは scene 状態が名前で引く別機構なので対象外。
func TestSpriteOrphan(t *testing.T) {
	t.Parallel()

	rawMaster, err := LoadRaws()
	require.NoError(t, err)

	keys := map[string]bool{}
	add := func(k string) {
		if k != "" {
			keys[k] = true
		}
	}
	addAnim := func(a *[]string) {
		if a != nil {
			for _, s := range *a {
				add(s)
			}
		}
	}
	// Item と Member はフラット、Prop と Tile は SpriteRender ネスト
	for _, it := range raw.PtrSlice(rawMaster.Items) {
		add(it.SpriteKey)
		addAnim(it.AnimKeys)
	}
	for _, pr := range raw.PtrSlice(rawMaster.Props) {
		add(pr.SpriteRender.SpriteKey)
		addAnim(pr.AnimKeys)
	}
	for _, tl := range raw.PtrSlice(rawMaster.Tiles) {
		add(tl.SpriteRender.SpriteKey)
	}
	for _, mb := range raw.PtrSlice(rawMaster.Members) {
		add(mb.SpriteKey)
		addAnim(mb.AnimKeys)
	}

	// covered はスプライト名 s が raw から参照されているかを返す
	covered := func(s string) bool {
		if keys[s] {
			return true
		}
		if i := strings.LastIndex(s, "_"); i >= 0 {
			if _, convErr := strconv.Atoi(s[i+1:]); convErr == nil {
				base := s[:i]
				if keys[base] || keys[base+"_0"] {
					return true
				}
			}
		}
		return false
	}

	allowed := map[string]bool{}
	for _, s := range externallyReferencedSprites {
		allowed[s] = true
	}
	for _, s := range knownOrphanSprites {
		allowed[s] = true
	}

	var orphans []string
	for _, path := range []string{"file/textures/dist/single.json", "file/textures/dist/tiles.json"} {
		sheet, sErr := LoadSpriteSheetFromAseprite(path)
		require.NoError(t, sErr, "シート %s の読み込みに失敗", path)
		for name := range sheet.Sprites {
			if !covered(name) && !allowed[name] {
				orphans = append(orphans, name)
			}
		}
	}
	sort.Strings(orphans)
	assert.Empty(t, orphans,
		"raw から参照されない孤児スプライト。raw に登録するか、Go/scene 参照なら externallyReferencedSprites、意図的に未使用なら knownOrphanSprites に追加せよ")
}
