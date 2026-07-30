package raw

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRaws_RealData(t *testing.T) {
	t.Parallel()

	master, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)

	err = ValidateRaws(master)
	assert.NoError(t, err)
}

func TestValidateRaws_ValidItem(t *testing.T) {
	t.Parallel()

	raws := makeItemRaws(func(*oapi.Item) {})
	err := ValidateRaws(raws)
	assert.NoError(t, err)
}

func TestValidateRaws_EmptyRaws(t *testing.T) {
	t.Parallel()

	err := ValidateRaws(oapi.Raws{})
	assert.NoError(t, err)
}

func TestValidateRaws_InvalidCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raws    oapi.Raws
		wantErr string
	}{
		{
			name: "名前が長すぎる",
			raws: makeItemRaws(func(i *oapi.Item) {
				i.Name = "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをん12345678901234567890"
			}),
			wantErr: "maximum string length is 50",
		},
		{
			name: "命中率が範囲外",
			raws: makeItemRaws(func(i *oapi.Item) {
				i.Melee.Accuracy = 999
			}),
			wantErr: "number must be at most 100",
		},
		{
			name: "スプライトキーのパターン不正",
			raws: makeItemRaws(func(i *oapi.Item) {
				i.SpriteKey = "INVALID-KEY!"
			}),
			wantErr: `string doesn't match the regular expression`,
		},
		{
			name: "不正な攻撃種別",
			raws: makeItemRaws(func(i *oapi.Item) {
				i.Melee.AttackCategory = "INVALID"
			}),
			wantErr: "value is not one of the allowed values",
		},
		{
			name: "不正な属性",
			raws: makeItemRaws(func(i *oapi.Item) {
				i.Melee.Element = "INVALID_ELEMENT"
			}),
			wantErr: "value is not one of the allowed values",
		},
		{
			name: "ダメージが負の値",
			raws: makeItemRaws(func(i *oapi.Item) {
				i.Melee.Damage = -1
			}),
			wantErr: "number must be at least 0",
		},
		{
			name: "攻撃回数がゼロ",
			raws: makeItemRaws(func(i *oapi.Item) {
				i.Melee.AttackCount = 0
			}),
			wantErr: "number must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateRaws(tt.raws)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "バリデーションエラー")
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// makeItemRaws は正常なアイテムを1つ持つRawsを生成し、modifyで値を改変する
func makeItemRaws(modify func(*oapi.Item)) oapi.Raws {
	item := oapi.Item{
		Name:            "テスト武器",
		Description:     "テスト用の武器",
		SpriteSheetName: "test_sheet",
		SpriteKey:       "test_key",
		Value:           100,
		Melee: &oapi.Melee{
			Accuracy:       80,
			Damage:         10,
			AttackCount:    1,
			Element:        "NONE",
			AttackCategory: "SWORD",
			Cost:           5,
			TargetGroup:    "ENEMY",
			TargetNum:      "SINGLE",
		},
	}
	modify(&item)
	return oapi.Raws{Items: &[]oapi.Item{item}}
}

func TestValidateDisassemblyReferences(t *testing.T) {
	t.Parallel()

	validItems := &[]oapi.Item{
		{Name: "鉄くず"},
		{Name: "分解対象"},
	}

	t.Run("実在する産出名なら通る", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Items: validItems,
			Props: &[]oapi.Prop{{
				Name: "棚",
				Disassembly: &oapi.Disassembly{
					ToolCategory: oapi.Prying,
					BaseAP:       100,
					Yields:       []oapi.DisassemblyYield{{Name: "鉄くず", Count: "1"}},
				},
			}},
		}
		require.NoError(t, validateDisassemblyReferences(raws))
	})

	t.Run("propの産出名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Items: validItems,
			Props: &[]oapi.Prop{{
				Name: "棚",
				Disassembly: &oapi.Disassembly{
					ToolCategory: oapi.Prying,
					BaseAP:       100,
					Yields:       []oapi.DisassemblyYield{{Name: "存在しない素材", Count: "1"}},
				},
			}},
		}
		err := validateDisassemblyReferences(raws)
		require.Error(t, err)
		require.ErrorContains(t, err, "存在しない素材")
		require.ErrorContains(t, err, "棚")
	})

	t.Run("itemのボーナス名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		items := []oapi.Item{
			{Name: "鉄くず"},
			{Name: "分解対象", Disassembly: &oapi.Disassembly{
				ToolCategory: oapi.Precision,
				BaseAP:       100,
				Yields:       []oapi.DisassemblyYield{{Name: "鉄くず", Count: "1"}},
				Bonus:        &[]oapi.DisassemblyBonus{{Name: "存在しないボーナス", Count: "1", MinSkill: new(oapi.SkillLevel(10))}},
			}},
		}
		raws := oapi.Raws{Items: &items}
		err := validateDisassemblyReferences(raws)
		require.Error(t, err)
		require.ErrorContains(t, err, "存在しないボーナス")
	})
}

func TestValidateDropTableReferences(t *testing.T) {
	t.Parallel()

	items := &[]oapi.Item{{Name: "鉄くず"}}

	t.Run("実在する素材と空文字は通る", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Items: items,
			DropTables: &[]oapi.DropTable{{
				Name: "廃墟",
				Entries: []oapi.DropTableEntry{
					{Material: "鉄くず", Weight: 1},
					{Material: "", Weight: 3},
				},
			}},
		}
		require.NoError(t, validateDropTableReferences(raws))
	})

	t.Run("素材名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Items: items,
			DropTables: &[]oapi.DropTable{{
				Name:    "廃墟",
				Entries: []oapi.DropTableEntry{{Material: "存在しない素材", Weight: 1}},
			}},
		}
		err := validateDropTableReferences(raws)
		require.Error(t, err)
		require.ErrorContains(t, err, "存在しない素材")
		require.ErrorContains(t, err, "廃墟")
	})

	t.Run("メンバーのテーブル名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Items:   items,
			Members: &[]oapi.Member{{Name: "スライム", DropTableName: new(oapi.EntityName("未定義テーブル"))}},
		}
		err := validateDropTableReferences(raws)
		require.Error(t, err)
		require.ErrorContains(t, err, "未定義テーブル")
		require.ErrorContains(t, err, "スライム")
	})
}

func TestValidateSpawnDice(t *testing.T) {
	t.Parallel()

	t.Run("正しいダイス表記は通る", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			EnemyTables: &[]oapi.EnemyTable{{Name: "通常", Entries: []oapi.EnemyTableEntry{{EnemyName: "スライム", Pack: "1d3"}}}},
			ItemGroups:  &[]oapi.ItemGroup{{Name: "回復", Entries: []oapi.ItemGroupEntry{{ItemName: "回復薬", Pack: "2"}}}},
			Props:       &[]oapi.Prop{{Name: "木箱", Storage: &oapi.StorageRaw{LootCount: new(oapi.Dice("1d2"))}}},
		}
		require.NoError(t, validateSpawnDice(raws))
	})

	t.Run("敵テーブルの不正なパック表記はエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			EnemyTables: &[]oapi.EnemyTable{{Name: "通常", Entries: []oapi.EnemyTableEntry{{EnemyName: "スライム", Pack: "0d6"}}}},
		}
		err := validateSpawnDice(raws)
		require.Error(t, err)
		assert.ErrorContains(t, err, "スライム")
		assert.ErrorContains(t, err, "個数は1以上")
	})

	t.Run("収納の不正なlootCountはエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Props: &[]oapi.Prop{{Name: "木箱", Storage: &oapi.StorageRaw{LootCount: new(oapi.Dice("abc"))}}},
		}
		err := validateSpawnDice(raws)
		require.Error(t, err)
		assert.ErrorContains(t, err, "木箱")
	})
}

func TestValidateCommandTableReferences(t *testing.T) {
	t.Parallel()

	t.Run("実在するテーブル名と未指定と空文字は通る", func(t *testing.T) {
		t.Parallel()
		empty := oapi.EntityName("")
		raws := oapi.Raws{
			CommandTables: &[]oapi.CommandTable{{Name: "素手"}},
			Members: &[]oapi.Member{
				{Name: "戦うNPC", CommandTableName: new(oapi.EntityName("素手"))},
				{Name: "未指定NPC"},
				{Name: "空文字NPC", CommandTableName: &empty},
			},
		}
		require.NoError(t, validateCommandTableReferences(raws))
	})

	t.Run("テーブル名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Members: &[]oapi.Member{{Name: "スライム", CommandTableName: new(oapi.EntityName("未定義テーブル"))}},
		}
		err := validateCommandTableReferences(raws)
		require.Error(t, err)
		require.ErrorContains(t, err, "未定義テーブル")
		require.ErrorContains(t, err, "スライム")
	})
}
