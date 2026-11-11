package raw

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindDisassembly(t *testing.T) {
	t.Parallel()

	propDisassembly := &rawschema.Disassembly{
		ToolCategory: new(rawschema.Prying),
		Yields:       []rawschema.DisassemblyYield{{Id: "鉄くず", Count: "1d1"}},
	}
	raws := rawschema.Raws{
		Props: &[]rawschema.Prop{
			{Id: "棚", Name: "棚", Disassembly: propDisassembly},
			{Id: "机", Name: "机"},
		},
	}

	t.Run("propの分解定義を返す", func(t *testing.T) {
		t.Parallel()
		got, ok := FindDisassembly(raws, "棚")
		require.True(t, ok)
		assert.Equal(t, propDisassembly, got)
	})

	t.Run("分解定義を持たないpropはfalseを返す", func(t *testing.T) {
		t.Parallel()
		got, ok := FindDisassembly(raws, "机")
		assert.False(t, ok)
		assert.Nil(t, got)
	})

	t.Run("存在しない名前はfalseを返す", func(t *testing.T) {
		t.Parallel()
		got, ok := FindDisassembly(raws, "存在しない名前")
		assert.False(t, ok)
		assert.Nil(t, got)
	})
}

func TestFindTool(t *testing.T) {
	t.Parallel()

	tool := &rawschema.Tool{
		Categories: []rawschema.ToolCategory{rawschema.Prying, rawschema.Cutting},
		Grade:      2,
	}
	raws := rawschema.Raws{
		Items: &[]rawschema.Item{
			{Id: "バール", Name: "バール", Tool: tool},
			{Id: "回復薬", Name: "回復薬"},
		},
	}

	t.Run("アイテムの道具の性質を返す", func(t *testing.T) {
		t.Parallel()
		got, ok := FindTool(raws, "バール")
		require.True(t, ok)
		assert.Equal(t, tool, got)
	})

	t.Run("道具の性質を持たないアイテムはfalseを返す", func(t *testing.T) {
		t.Parallel()
		got, ok := FindTool(raws, "回復薬")
		assert.False(t, ok)
		assert.Nil(t, got)
	})

	t.Run("存在しないアイテム名はfalseを返す", func(t *testing.T) {
		t.Parallel()
		got, ok := FindTool(raws, "存在しないアイテム")
		assert.False(t, ok)
		assert.Nil(t, got)
	})
}
