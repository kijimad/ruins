package maptemplate

import (
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaletteLoader_Load_不正なTOMLはパースエラー(t *testing.T) {
	t.Parallel()
	loader := NewPaletteLoader()
	_, err := loader.Load(strings.NewReader("this is = not [valid toml"))

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to parse palette TOML")
}

func TestPaletteLoader_Load_Propsのキーが2文字以上はエラー(t *testing.T) {
	t.Parallel()
	content := `[palette]
id = "invalid_props"
description = "無効なPropsキー"

[palette.props]
"TT" = { id = "table", tile = "floor" }
`
	loader := NewPaletteLoader()
	_, err := loader.Load(strings.NewReader(content))

	require.Error(t, err)
	assert.ErrorContains(t, err, "props key must be a single character")
}

func TestPaletteLoader_Load_NPCsのキーが2文字以上はエラー(t *testing.T) {
	t.Parallel()
	content := `[palette]
id = "invalid_npcs"
description = "無効なNPCsキー"

[palette.npcs]
"GG" = { id = "guard", tile = "floor" }
`
	loader := NewPaletteLoader()
	_, err := loader.Load(strings.NewReader(content))

	require.Error(t, err)
	assert.ErrorContains(t, err, "npcs key must be a single character")
}

func TestPalette_GetNPC(t *testing.T) {
	t.Parallel()
	palette := &Palette{
		ID: "test",
		NPCs: map[string]PaletteEntry{
			"G": {ID: "guard", Tile: "floor"},
		},
	}

	t.Run("存在するNPCのIDを取得できる", func(t *testing.T) {
		t.Parallel()
		npc, ok := palette.GetNPC("G")
		assert.True(t, ok)
		assert.Equal(t, "guard", npc)
	})

	t.Run("存在しないNPCはfalseを返す", func(t *testing.T) {
		t.Parallel()
		npc, ok := palette.GetNPC("X")
		assert.False(t, ok)
		assert.Empty(t, npc)
	})
}

func TestResolveMapCells_NPC文字をNPCとして解決する(t *testing.T) {
	t.Parallel()
	palette := &Palette{
		ID:      "npc_pal",
		Terrain: map[string]string{".": "floor"},
		NPCs: map[string]PaletteEntry{
			"G": {ID: "guard", Tile: "floor"},
		},
	}

	cells := ResolveMapCells(".G\n..", palette)

	require.Len(t, cells, 2)
	require.Len(t, cells[0], 2)
	assert.Equal(t, "floor", cells[0][1].Terrain)
	assert.Equal(t, "guard", cells[0][1].NPC)
}

func TestFormatResolvedMap_NPC付きセルを整形する(t *testing.T) {
	t.Parallel()
	cells := [][]MapCell{
		{
			{Terrain: "floor"},
			{Terrain: "floor", NPC: "guard"},
			{Terrain: "floor", Prop: "table", NPC: "merchant"},
		},
	}

	got := FormatResolvedMap(cells)

	assert.Equal(t, "floor floor::guard floor:table:merchant", got)
}

func TestTemplateLoader_Load_不正なTOMLはパースエラー(t *testing.T) {
	t.Parallel()
	loader := NewTemplateLoader()
	_, err := loader.Load(strings.NewReader("this is = not [valid toml"))

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to parse template TOML")
}

func TestTemplateLoader_Load_幅が数値でない名前はエラー(t *testing.T) {
	t.Parallel()
	content := `[[chunk]]
name = "axb_room"
weight = 1
map = "x"
`
	loader := NewTemplateLoader()
	_, err := loader.Load(strings.NewReader(content))

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to parse width")
}

func TestTemplateLoader_Load_高さが数値でない名前はエラー(t *testing.T) {
	t.Parallel()
	content := `[[chunk]]
name = "5xb_room"
weight = 1
map = "x"
`
	loader := NewTemplateLoader()
	_, err := loader.Load(strings.NewReader(content))

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to parse height")
}

func TestTemplateLoader_Load_空白のみのマップ行はエラー(t *testing.T) {
	t.Parallel()
	content := `[[chunk]]
name = "1x1_room"
weight = 1
map = " "
`
	loader := NewTemplateLoader()
	_, err := loader.Load(strings.NewReader(content))

	require.Error(t, err)
	assert.ErrorContains(t, err, "map row is empty")
}

func TestTemplateLoader_LoadChunk_存在しないファイルはエラー(t *testing.T) {
	t.Parallel()
	loader := NewTemplateLoader()
	err := loader.LoadChunk("levels/chunks/nonexistent.toml")

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to read chunk")
}

func TestTemplateLoader_RegisterAllChunks_存在しないディレクトリはエラー(t *testing.T) {
	t.Parallel()
	loader := NewTemplateLoader()
	err := loader.RegisterAllChunks([]string{"levels/nonexistent_dir"})

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to read directory")
}

func TestTemplateLoader_RegisterAllPalettes_存在しないディレクトリはエラー(t *testing.T) {
	t.Parallel()
	loader := NewTemplateLoader()
	err := loader.RegisterAllPalettes([]string{"levels/nonexistent_dir"})

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to read directory")
}

func TestTemplateLoader_LoadTemplateByName_未登録パレット参照はエラー(t *testing.T) {
	t.Parallel()
	loader := NewTemplateLoader()
	loader.RegisterChunk(&ChunkTemplate{
		Name:     "3x1_room",
		Weight:   100,
		Size:     Size{W: 3, H: 1},
		Map:      "...",
		Palettes: []string{"unregistered"},
	})

	_, _, _, err := loader.LoadTemplateByName("3x1_room", 0)

	require.Error(t, err)
	assert.ErrorContains(t, err, "palette 'unregistered' not found")
}

func TestTemplateLoader_selectChunkByWeight_候補名が空はエラー(t *testing.T) {
	t.Parallel()
	loader := NewTemplateLoader()
	rng := rand.New(rand.NewPCG(1, 1))

	_, err := loader.selectChunkByWeight(nil, rng)

	require.Error(t, err)
	assert.ErrorContains(t, err, "chunk candidates are empty")
}

func TestTemplateLoader_selectChunkByWeightFromList_候補が空はエラー(t *testing.T) {
	t.Parallel()
	loader := NewTemplateLoader()
	rng := rand.New(rand.NewPCG(1, 1))

	_, err := loader.selectChunkByWeightFromList(nil, rng)

	require.Error(t, err)
	assert.ErrorContains(t, err, "chunk candidates are empty")
}

func TestTemplateLoader_selectChunkByWeightFromList_重み合計が0はエラー(t *testing.T) {
	t.Parallel()
	loader := NewTemplateLoader()
	rng := rand.New(rand.NewPCG(1, 1))
	candidates := []*ChunkTemplate{
		{Name: "3x1_zero", Size: Size{W: 3, H: 1}, Map: "...", Weight: 0},
	}

	_, err := loader.selectChunkByWeightFromList(candidates, rng)

	require.Error(t, err)
	assert.ErrorContains(t, err, "total weight is 0")
}

func TestChunkTemplate_ExpandWithPlacements_placementのIDが空はエラー(t *testing.T) {
	t.Parallel()
	loader := NewTemplateLoader()
	loader.RegisterChunk(&ChunkTemplate{
		Name:   "2x2_child",
		Weight: 100,
		Size:   Size{W: 2, H: 2},
		Map:    "..\n..",
	})

	parent := &ChunkTemplate{
		Name:   "2x2_parent",
		Weight: 100,
		Size:   Size{W: 2, H: 2},
		Map:    "@@\n@@",
		Placements: []ChunkPlacement{
			{Chunks: []string{"2x2_child"}},
		},
	}

	_, err := parent.ExpandWithPlacements(loader, 0)

	require.Error(t, err)
	assert.ErrorContains(t, err, "ID is not specified")
}

func TestChunkTemplate_ExpandWithPlacements_未展開のプレースホルダが残るとエラー(t *testing.T) {
	t.Parallel()
	loader := NewTemplateLoader()
	loader.RegisterChunk(&ChunkTemplate{
		Name:   "2x2_child",
		Weight: 100,
		Size:   Size{W: 2, H: 2},
		Map:    "..\n..",
	})

	// 左上の '@' はどのplacement領域にも属さないため展開後も残る
	parent := &ChunkTemplate{
		Name:   "4x2_parent",
		Weight: 100,
		Size:   Size{W: 4, H: 2},
		Map:    "@.AA\n..AA",
		Placements: []ChunkPlacement{
			{ID: "A", Chunks: []string{"2x2_child"}},
		},
	}

	_, err := parent.ExpandWithPlacements(loader, 0)

	require.Error(t, err)
	assert.ErrorContains(t, err, "unexpanded placeholder '@' remains")
}

func TestFindAllPlaceholderRegionsByID_IDが1文字でないとエラー(t *testing.T) {
	t.Parallel()
	lines := []string{"AA", "AA"}

	_, err := findAllPlaceholderRegionsByID(lines, "AB")

	require.Error(t, err)
	assert.ErrorContains(t, err, "identifier must be a single character")
}

func TestValidateRectangle_矩形領域が範囲外はエラー(t *testing.T) {
	t.Parallel()

	t.Run("高さ方向が範囲外", func(t *testing.T) {
		t.Parallel()
		lines := []string{"AA"}
		err := validateRectangle(lines, "A", 0, 0, 2, 2, placeholderChar, 'A')
		require.Error(t, err)
		assert.ErrorContains(t, err, "y=1 out of range")
	})

	t.Run("幅方向が範囲外", func(t *testing.T) {
		t.Parallel()
		lines := []string{"A", "A"}
		err := validateRectangle(lines, "A", 0, 0, 2, 2, placeholderChar, 'A')
		require.Error(t, err)
		assert.ErrorContains(t, err, "x=1, y=0 out of range")
	})
}
