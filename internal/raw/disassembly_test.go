package raw

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindDisassembly(t *testing.T) {
	t.Parallel()

	propDisassembly := &oapi.Disassembly{
		ToolCategory: oapi.Prying,
		BaseAP:       100,
		Yields:       []oapi.DisassemblyYield{{Name: "鉄くず", Count: "1d1"}},
	}
	itemDisassembly := &oapi.Disassembly{
		ToolCategory: oapi.Precision,
		BaseAP:       200,
		Yields:       []oapi.DisassemblyYield{{Name: "ネジ", Count: "1d2"}},
	}
	raws := oapi.Raws{
		Props: &[]oapi.Prop{
			{Id: "棚", Name: "棚", Disassembly: propDisassembly},
			{Id: "机", Name: "机"},
		},
		Items: &[]oapi.Item{
			{Id: "廃品", Name: "廃品", Disassembly: itemDisassembly},
			{Id: "回復薬", Name: "回復薬"},
		},
	}

	t.Run("propの分解定義を返す", func(t *testing.T) {
		t.Parallel()
		got, ok := FindDisassembly(raws, "棚")
		require.True(t, ok)
		assert.Equal(t, propDisassembly, got)
	})

	t.Run("itemの分解定義を返す", func(t *testing.T) {
		t.Parallel()
		got, ok := FindDisassembly(raws, "廃品")
		require.True(t, ok)
		assert.Equal(t, itemDisassembly, got)
	})

	t.Run("分解定義を持たないpropはfalseを返す", func(t *testing.T) {
		t.Parallel()
		got, ok := FindDisassembly(raws, "机")
		assert.False(t, ok)
		assert.Nil(t, got)
	})

	t.Run("分解定義を持たないitemはfalseを返す", func(t *testing.T) {
		t.Parallel()
		got, ok := FindDisassembly(raws, "回復薬")
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

func TestFindDisassemblyTool(t *testing.T) {
	t.Parallel()

	tool := &oapi.DisassemblyTool{
		Categories: []oapi.ToolCategory{oapi.Prying, oapi.Cutting},
		Grade:      2,
	}
	raws := oapi.Raws{
		Items: &[]oapi.Item{
			{Id: "バール", Name: "バール", DisassemblyTool: tool},
			{Id: "回復薬", Name: "回復薬"},
		},
	}

	t.Run("アイテムの分解工具定義を返す", func(t *testing.T) {
		t.Parallel()
		got, ok := FindDisassemblyTool(raws, "バール")
		require.True(t, ok)
		assert.Equal(t, tool, got)
	})

	t.Run("分解工具定義を持たないアイテムはfalseを返す", func(t *testing.T) {
		t.Parallel()
		got, ok := FindDisassemblyTool(raws, "回復薬")
		assert.False(t, ok)
		assert.Nil(t, got)
	})

	t.Run("存在しないアイテム名はfalseを返す", func(t *testing.T) {
		t.Parallel()
		got, ok := FindDisassemblyTool(raws, "存在しないアイテム")
		assert.False(t, ok)
		assert.Nil(t, got)
	})
}

func TestValidateReferences(t *testing.T) {
	t.Parallel()

	t.Run("空のRawsは成功する", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, ValidateReferences(oapi.Raws{}))
	})

	t.Run("全チェックを満たすRawsは成功する", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Items: &[]oapi.Item{
				{Id: "鉄くず", Name: "鉄くず"},
				{Id: "刀", Name: "刀"},
			},
			Props: &[]oapi.Prop{{
				Name: "棚",
				Disassembly: &oapi.Disassembly{
					ToolCategory: oapi.Prying,
					BaseAP:       100,
					Yields:       []oapi.DisassemblyYield{{Name: "鉄くず", Count: "1d1"}},
				},
			}},
			DropTables: &[]oapi.DropTable{{
				Id:      "廃墟",
				Name:    "廃墟",
				Entries: []oapi.DropTableEntry{{Material: "鉄くず", Weight: 1}},
			}},
			EnemyTables: &[]oapi.EnemyTable{{
				Id:      "通常",
				Name:    "通常",
				Entries: []oapi.EnemyTableEntry{{Id: "スライム", Pack: "1d3"}},
			}},
			CommandTables: &[]oapi.CommandTable{{
				Id:      "剣術",
				Name:    "剣術",
				Entries: []oapi.CommandTableEntry{{Weapon: "刀"}},
			}},
			ItemGroups: &[]oapi.ItemGroup{{
				Id:      "素材",
				Name:    "素材",
				Entries: []oapi.ItemGroupEntry{{Id: "鉄くず", Pack: "1d1"}},
			}},
			ItemTables: &[]oapi.ItemTable{{
				Id:      "宝箱",
				Name:    "宝箱",
				Entries: []oapi.ItemTableEntry{{Id: "素材"}},
			}},
			Members: &[]oapi.Member{{
				Id:             "スライム",
				Name:           "スライム",
				DropTableId:    new(oapi.EntityName("廃墟")),
				CommandTableId: new(oapi.EntityName("剣術")),
			}},
		}
		assert.NoError(t, ValidateReferences(raws))
	})

	t.Run("分解参照エラーが最初に検出される", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Items: &[]oapi.Item{{Name: "鉄くず"}},
			Props: &[]oapi.Prop{{
				Name: "棚",
				Disassembly: &oapi.Disassembly{
					ToolCategory: oapi.Prying,
					BaseAP:       100,
					Yields:       []oapi.DisassemblyYield{{Name: "存在しない素材", Count: "1d1"}},
				},
			}},
			DropTables: &[]oapi.DropTable{{
				Name:    "廃墟",
				Entries: []oapi.DropTableEntry{{Material: "別の存在しない素材", Weight: 1}},
			}},
		}
		err := ValidateReferences(raws)
		require.Error(t, err)
		require.ErrorContains(t, err, "disassembly yield")
		require.ErrorContains(t, err, "存在しない素材")
	})

	t.Run("武器参照エラーが最後のチェックで検出される", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Items: &[]oapi.Item{{Name: "鉄くず"}},
			CommandTables: &[]oapi.CommandTable{{
				Name:    "剣術",
				Entries: []oapi.CommandTableEntry{{Weapon: "未定義武器"}},
			}},
		}
		err := ValidateReferences(raws)
		require.Error(t, err)
		require.ErrorContains(t, err, "未定義武器")
		require.ErrorContains(t, err, "剣術")
	})
}
