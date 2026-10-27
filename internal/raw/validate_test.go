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

func TestRealData_HasLandmarks(t *testing.T) {
	t.Parallel()

	master, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	require.NotEmpty(t, PtrSlice(master.Landmarks), "overworld 生成が要求するので実 raw.toml は landmarks を持つべき")
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
			assert.Contains(t, err.Error(), "validation error")
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// makeItemRaws は正常なアイテムを1つ持つRawsを生成し、modifyで値を改変する
func makeItemRaws(modify func(*oapi.Item)) oapi.Raws {
	item := oapi.Item{
		Id:              "テスト武器",
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
		{Id: "鉄くず", Name: "鉄くず"},
		{Id: "分解対象", Name: "分解対象"},
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
					Yields:       []oapi.DisassemblyYield{{Id: "鉄くず", Count: "1d1"}},
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
					Yields:       []oapi.DisassemblyYield{{Id: "存在しない素材", Count: "1d1"}},
				},
			}},
		}
		err := validateDisassemblyReferences(raws)
		require.ErrorIs(t, err, errDisassemblyYieldUndefined)
	})

	t.Run("itemのボーナス名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		items := []oapi.Item{
			{Id: "鉄くず", Name: "鉄くず"},
			{Id: "分解対象", Name: "分解対象", Disassembly: &oapi.Disassembly{
				ToolCategory: oapi.Precision,
				BaseAP:       100,
				Yields:       []oapi.DisassemblyYield{{Id: "鉄くず", Count: "1d1"}},
				Bonus:        &[]oapi.DisassemblyBonus{{Id: "存在しないボーナス", Count: "1d1", MinSkill: new(oapi.SkillLevel(10))}},
			}},
		}
		raws := oapi.Raws{Items: &items}
		err := validateDisassemblyReferences(raws)
		require.ErrorIs(t, err, errDisassemblyBonusUndefined)
	})
}

func TestValidateDropTableReferences(t *testing.T) {
	t.Parallel()

	items := &[]oapi.Item{{Id: "鉄くず", Name: "鉄くず"}}

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
		require.ErrorIs(t, err, errDropTableMaterialUndefined)
	})

	t.Run("メンバーのテーブル名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Items:   items,
			Members: &[]oapi.Member{{Name: "スライム", DropTableId: new(oapi.EntityName("未定義テーブル"))}},
		}
		err := validateDropTableReferences(raws)
		require.ErrorIs(t, err, errMemberDropTableUndefined)
	})
}

func TestValidateCommandTableReferences(t *testing.T) {
	t.Parallel()

	t.Run("実在するテーブル名と未指定は通る", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			CommandTables: &[]oapi.CommandTable{{Id: "素手", Name: "素手"}},
			Members: &[]oapi.Member{
				{Name: "戦うNPC", CommandTableId: new(oapi.EntityName("素手"))},
				{Name: "未指定NPC"},
			},
		}
		require.NoError(t, validateCommandTableReferences(raws))
	})

	t.Run("空文字はテーブル名として不正でエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			CommandTables: &[]oapi.CommandTable{{Id: "素手", Name: "素手"}},
			Members:       &[]oapi.Member{{Name: "空文字NPC", CommandTableId: new(oapi.EntityName(""))}},
		}
		err := validateCommandTableReferences(raws)
		require.ErrorIs(t, err, errMemberCommandTableUndefined)
	})

	t.Run("テーブル名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Members: &[]oapi.Member{{Name: "スライム", CommandTableId: new(oapi.EntityName("未定義テーブル"))}},
		}
		err := validateCommandTableReferences(raws)
		require.ErrorIs(t, err, errMemberCommandTableUndefined)
	})
}

func TestValidateItemTableReferences(t *testing.T) {
	t.Parallel()

	groups := &[]oapi.ItemGroup{{Id: "雑貨", Name: "雑貨"}}

	t.Run("実在するグループと空文字は通る", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			ItemGroups: groups,
			ItemTables: &[]oapi.ItemTable{{Name: "宝箱", Entries: []oapi.ItemTableEntry{{Id: "雑貨"}, {Id: ""}}}},
		}
		require.NoError(t, validateItemTableReferences(raws))
	})

	t.Run("グループ名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			ItemGroups: groups,
			ItemTables: &[]oapi.ItemTable{{Name: "宝箱", Entries: []oapi.ItemTableEntry{{Id: "未定義グループ"}}}},
		}
		err := validateItemTableReferences(raws)
		require.ErrorIs(t, err, errItemTableRefUndefinedGroup)
	})
}

func TestValidateItemGroupReferences(t *testing.T) {
	t.Parallel()

	items := &[]oapi.Item{{Id: "鉄くず", Name: "鉄くず"}}

	t.Run("実在するアイテムは通る", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Items:      items,
			ItemGroups: &[]oapi.ItemGroup{{Name: "素材", Entries: []oapi.ItemGroupEntry{{Id: "鉄くず"}}}},
		}
		require.NoError(t, validateItemGroupReferences(raws))
	})

	t.Run("アイテム名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Items:      items,
			ItemGroups: &[]oapi.ItemGroup{{Name: "素材", Entries: []oapi.ItemGroupEntry{{Id: "未定義アイテム"}}}},
		}
		err := validateItemGroupReferences(raws)
		require.ErrorIs(t, err, errItemGroupRefUndefinedItem)
	})
}

func TestValidateEnemyTableReferences(t *testing.T) {
	t.Parallel()

	members := &[]oapi.Member{{Id: "スライム", Name: "スライム"}}

	t.Run("実在するメンバーは通る", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Members:     members,
			EnemyTables: &[]oapi.EnemyTable{{Name: "通常", Entries: []oapi.EnemyTableEntry{{Id: "スライム"}}}},
		}
		require.NoError(t, validateEnemyTableReferences(raws))
	})

	t.Run("敵名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Members:     members,
			EnemyTables: &[]oapi.EnemyTable{{Name: "通常", Entries: []oapi.EnemyTableEntry{{Id: "未定義敵"}}}},
		}
		err := validateEnemyTableReferences(raws)
		require.ErrorIs(t, err, errEnemyTableRefUndefinedEnemy)
	})
}

func TestValidateFacilityReferences(t *testing.T) {
	t.Parallel()

	enemyTables := &[]oapi.EnemyTable{{Id: "clinic_enemies", Name: "診療所"}}
	baseZones := []oapi.FacilityZone{
		{Zone: oapi.Residential, Weight: 10, MinSpan: 2},
		{Zone: oapi.Downtown, Weight: 10, MinSpan: 2},
		{Zone: oapi.Industrial, Weight: 10, MinSpan: 2},
	}

	t.Run("実在する敵テーブルと基本施設は通る", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			EnemyTables: enemyTables,
			Facilities:  &[]oapi.Facility{{Id: "clinic", EnemyTable: "clinic_enemies", Planner: oapi.Clinic, Zones: baseZones}},
		}
		require.NoError(t, validateFacilityReferences(raws))
	})

	t.Run("敵テーブルが存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			EnemyTables: enemyTables,
			Facilities:  &[]oapi.Facility{{Id: "clinic", EnemyTable: "no_such_table", Planner: oapi.Clinic, Zones: baseZones}},
		}
		require.ErrorIs(t, validateFacilityReferences(raws), errFacilityEnemyTableRefUndefined)
	})

	t.Run("地区に基本施設が無いとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			EnemyTables: enemyTables,
			Facilities:  &[]oapi.Facility{{Id: "clinic", EnemyTable: "clinic_enemies", Planner: oapi.Clinic, Zones: []oapi.FacilityZone{{Zone: oapi.Downtown, Weight: 10, MinSpan: 3}}}},
		}
		require.ErrorIs(t, validateFacilityReferences(raws), errZoneNoBaseFacility)
	})

	t.Run("未登録施設を参照するfacilityContentsはエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			EnemyTables:      enemyTables,
			Facilities:       &[]oapi.Facility{{Id: "clinic", EnemyTable: "clinic_enemies", Planner: oapi.Clinic, Zones: baseZones}},
			FacilityContents: &[]oapi.FacilityContent{{Facility: "no_such", Variants: []string{"x"}}},
		}
		require.ErrorIs(t, validateFacilityReferences(raws), errFacilityKeyUndefined)
	})

	for _, missing := range []oapi.Zone{oapi.Downtown, oapi.Residential, oapi.Industrial} {
		t.Run("地区 "+string(missing)+" だけ基本施設が無いとエラー", func(t *testing.T) {
			t.Parallel()
			zones := make([]oapi.FacilityZone, 0, len(baseZones))
			for _, z := range baseZones {
				if z.Zone == missing {
					z.MinSpan = 3
				}
				zones = append(zones, z)
			}
			raws := oapi.Raws{
				EnemyTables: enemyTables,
				Facilities:  &[]oapi.Facility{{Id: "clinic", EnemyTable: "clinic_enemies", Planner: oapi.Clinic, Zones: zones}},
			}
			err := validateFacilityReferences(raws)
			require.ErrorIs(t, err, errZoneNoBaseFacility)
			assert.ErrorContains(t, err, string(missing))
		})
	}

	t.Run("全地区に基本施設があれば通る", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			EnemyTables: enemyTables,
			Facilities: &[]oapi.Facility{{Id: "house", EnemyTable: "clinic_enemies", Planner: oapi.House, Zones: []oapi.FacilityZone{
				{Zone: oapi.Residential, Weight: 10, MinSpan: 2},
				{Zone: oapi.Downtown, Weight: 10, MinSpan: 2},
				{Zone: oapi.Industrial, Weight: 10, MinSpan: 2},
			}}},
		}
		require.NoError(t, validateFacilityReferences(raws))
	})
}

func TestValidateInteriorContentReferences(t *testing.T) {
	t.Parallel()

	contents := &[]oapi.InteriorContent{{Id: "house"}, {Id: "bedroom"}}

	t.Run("実在する content 参照は通る", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			InteriorContents: contents,
			FacilityContents: &[]oapi.FacilityContent{{Facility: "house", Variants: []oapi.EntityID{"house"}}},
			FacilityRooms: &[]oapi.FacilityRooms{{
				Facility: "house",
				Rooms:    &[]oapi.RoomContent{{Role: "bedroom", Content: "bedroom"}},
				Fallback: "bedroom",
			}},
			FlavorContent: &oapi.InteriorContent{Id: "flavor"},
		}
		require.NoError(t, validateInteriorContentReferences(raws))
	})

	t.Run("存在しない変種はエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			InteriorContents: contents,
			FacilityContents: &[]oapi.FacilityContent{{Facility: "house", Variants: []oapi.EntityID{"no_such"}}},
		}
		require.ErrorIs(t, validateInteriorContentReferences(raws), errInteriorContentRefUndefined)
	})

	t.Run("存在しない奥室 content はエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			InteriorContents: contents,
			FacilityRooms: &[]oapi.FacilityRooms{{
				Facility: "house",
				Rooms:    &[]oapi.RoomContent{{Role: "bedroom", Content: "no_such"}},
				Fallback: "bedroom",
			}},
		}
		require.ErrorIs(t, validateInteriorContentReferences(raws), errInteriorContentRefUndefined)
	})

	t.Run("存在しない fallback はエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			InteriorContents: contents,
			FacilityRooms:    &[]oapi.FacilityRooms{{Facility: "house", Fallback: "no_such"}},
		}
		require.ErrorIs(t, validateInteriorContentReferences(raws), errInteriorContentRefUndefined)
	})

	t.Run("id が重複するとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			InteriorContents: &[]oapi.InteriorContent{{Id: "house"}, {Id: "house"}},
		}
		require.ErrorIs(t, validateInteriorContentReferences(raws), errInteriorContentDuplicateID)
	})

	t.Run("interior を積むのに flavorContent が無いとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			InteriorContents: &[]oapi.InteriorContent{{Id: "house"}},
		}
		require.ErrorIs(t, validateInteriorContentReferences(raws), errInteriorFlavorContentMissing)
	})
}

func TestValidateCommandTableWeaponReferences(t *testing.T) {
	t.Parallel()

	items := &[]oapi.Item{{Id: "刀", Name: "刀"}}

	t.Run("実在する武器と空文字は通る", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Items:         items,
			CommandTables: &[]oapi.CommandTable{{Name: "剣術", Entries: []oapi.CommandTableEntry{{Weapon: "刀"}, {Weapon: ""}}}},
		}
		require.NoError(t, validateCommandTableWeaponReferences(raws))
	})

	t.Run("武器名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Items:         items,
			CommandTables: &[]oapi.CommandTable{{Name: "剣術", Entries: []oapi.CommandTableEntry{{Weapon: "未定義武器"}}}},
		}
		err := validateCommandTableWeaponReferences(raws)
		require.ErrorIs(t, err, errCommandTableRefUndefinedWeapon)
	})
}

func TestValidateLandmarkReferences(t *testing.T) {
	t.Parallel()

	props := &[]oapi.Prop{{Id: "candle"}}

	t.Run("prop が実在すれば通る", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Props:     props,
			Landmarks: &[]oapi.Landmark{{Id: "shrine", Weight: 10, Drawer: oapi.Open, Props: []oapi.PropSpot{{Name: "candle"}}}},
		}
		require.NoError(t, validateLandmarkReferences(raws))
	})

	t.Run("prop が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Props:     props,
			Landmarks: &[]oapi.Landmark{{Id: "shrine", Weight: 10, Drawer: oapi.Open, Props: []oapi.PropSpot{{Name: "no_such_prop"}}}},
		}
		require.ErrorIs(t, validateLandmarkReferences(raws), errLandmarkPropUndefined)
	})
}

func TestValidateScatterZoneReferences(t *testing.T) {
	t.Parallel()

	props := &[]oapi.Prop{{Id: "tree_a"}}
	groups := &[]oapi.ItemGroup{{Id: "junk"}}

	t.Run("prop と item group が実在すれば通る", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Props:      props,
			ItemGroups: groups,
			ScatterZones: &[]oapi.ScatterZone{{Id: "wild", LootGroup: "junk", Entries: []oapi.ScatterEntry{
				{Ref: "", Weight: 10},
				{Ref: "tree_a", Weight: 10, Satellites: &[]oapi.PropSpot{{Name: "tree_a"}}},
			}}},
		}
		require.NoError(t, validateScatterZoneReferences(raws))
	})

	t.Run("item group が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Props:        props,
			ItemGroups:   groups,
			ScatterZones: &[]oapi.ScatterZone{{Id: "wild", LootGroup: "no_such_group"}},
		}
		require.ErrorIs(t, validateScatterZoneReferences(raws), errScatterLootGroupUndefined)
	})

	t.Run("散布 prop が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Props:        props,
			ItemGroups:   groups,
			ScatterZones: &[]oapi.ScatterZone{{Id: "wild", LootGroup: "junk", Entries: []oapi.ScatterEntry{{Ref: "no_such_prop", Weight: 10}}}},
		}
		require.ErrorIs(t, validateScatterZoneReferences(raws), errScatterPropUndefined)
	})

	t.Run("satellite の prop が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Props:      props,
			ItemGroups: groups,
			ScatterZones: &[]oapi.ScatterZone{{Id: "wild", LootGroup: "junk", Entries: []oapi.ScatterEntry{
				{Ref: "tree_a", Weight: 10, Satellites: &[]oapi.PropSpot{{Name: "no_such_prop"}}},
			}}},
		}
		require.ErrorIs(t, validateScatterZoneReferences(raws), errScatterPropUndefined)
	})
}

func TestValidateMapGlyphReferences(t *testing.T) {
	t.Parallel()

	t.Run("1文字で重複せず全施設とランドマークに記号があれば通る", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Facilities: &[]oapi.Facility{{Id: "house"}},
			Landmarks:  &[]oapi.Landmark{{Id: "shrine"}},
			MapGlyphs:  &[]oapi.MapGlyph{{Id: "house", Glyph: "h"}, {Id: "shrine", Glyph: "s"}},
		}
		require.NoError(t, validateMapGlyphReferences(raws))
	})

	t.Run("記号が2文字だとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{MapGlyphs: &[]oapi.MapGlyph{{Id: "house", Glyph: "hh"}}}
		require.ErrorIs(t, validateMapGlyphReferences(raws), errMapGlyphNotSingleRune)
	})

	t.Run("記号が空だとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{MapGlyphs: &[]oapi.MapGlyph{{Id: "house", Glyph: ""}}}
		require.ErrorIs(t, validateMapGlyphReferences(raws), errMapGlyphNotSingleRune)
	})

	t.Run("記号が重複するとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{MapGlyphs: &[]oapi.MapGlyph{{Id: "house", Glyph: "h"}, {Id: "hamlet", Glyph: "h"}}}
		require.ErrorIs(t, validateMapGlyphReferences(raws), errMapGlyphDuplicate)
	})

	t.Run("施設に記号が無いとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Facilities: &[]oapi.Facility{{Id: "house"}},
			MapGlyphs:  &[]oapi.MapGlyph{{Id: "field", Glyph: "."}},
		}
		require.ErrorIs(t, validateMapGlyphReferences(raws), errMapGlyphMissing)
	})

	t.Run("ランドマークに記号が無いとエラー", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Landmarks: &[]oapi.Landmark{{Id: "shrine"}},
			MapGlyphs: &[]oapi.MapGlyph{{Id: "field", Glyph: "."}},
		}
		require.ErrorIs(t, validateMapGlyphReferences(raws), errMapGlyphMissing)
	})
}

func TestValidateFeatureUniqueIDs(t *testing.T) {
	t.Parallel()

	t.Run("id が一意なら通る", func(t *testing.T) {
		t.Parallel()
		raws := oapi.Raws{
			Facilities:   &[]oapi.Facility{{Id: "house"}, {Id: "store"}},
			Landmarks:    &[]oapi.Landmark{{Id: "shrine"}},
			ScatterZones: &[]oapi.ScatterZone{{Id: "wild"}},
			MapGlyphs:    &[]oapi.MapGlyph{{Id: "house"}},
		}
		require.NoError(t, validateFeatureUniqueIDs(raws))
	})

	cases := []struct {
		name string
		raws oapi.Raws
	}{
		{"施設の id が重複するとエラー", oapi.Raws{Facilities: &[]oapi.Facility{{Id: "house"}, {Id: "house"}}}},
		{"ランドマークの id が重複するとエラー", oapi.Raws{Landmarks: &[]oapi.Landmark{{Id: "shrine"}, {Id: "shrine"}}}},
		{"散布ゾーンの id が重複するとエラー", oapi.Raws{ScatterZones: &[]oapi.ScatterZone{{Id: "wild"}, {Id: "wild"}}}},
		{"地図記号の id が重複するとエラー", oapi.Raws{MapGlyphs: &[]oapi.MapGlyph{{Id: "house"}, {Id: "house"}}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateFeatureUniqueIDs(c.raws), errDuplicateID)
		})
	}
}

func TestValidateRaws_抽選重みは1以上(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*oapi.Raws)
	}{
		{"施設の地区重みが0", func(r *oapi.Raws) { (*r.Facilities)[0].Zones[0].Weight = 0 }},
		{"ランドマークの重みが0", func(r *oapi.Raws) { (*r.Landmarks)[0].Weight = 0 }},
		{"散布 prop の重みが0", func(r *oapi.Raws) { (*r.ScatterZones)[0].Entries[0].Weight = 0 }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
			require.NoError(t, err)
			c.mutate(&raws)
			assert.ErrorContains(t, ValidateRaws(raws), "number must be at least 1")
		})
	}
}

func TestSchemaEnum(t *testing.T) {
	t.Parallel()

	t.Run("enum 型は全値を返す", func(t *testing.T) {
		t.Parallel()
		got, err := SchemaEnum("Zone")
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{string(oapi.Downtown), string(oapi.Residential), string(oapi.Industrial)}, got)
	})

	t.Run("enum でない型は error", func(t *testing.T) {
		t.Parallel()
		_, err := SchemaEnum("Raws")
		assert.Error(t, err)
	})
}
