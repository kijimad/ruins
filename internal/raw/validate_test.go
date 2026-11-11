package raw

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/rawschema/rawcheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealData_HasLandmarks(t *testing.T) {
	t.Parallel()

	master, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	require.NotEmpty(t, PtrSlice(master.Landmarks), "overworld 生成が要求するので実 raw.toml は landmarks を持つべき")
}

func TestSchema_正常なアイテムは通る(t *testing.T) {
	t.Parallel()

	require.NoError(t, rawcheck.Validate(makeItemRaws(func(*rawschema.Item) {})))
}

func TestSchema_空のRawsは通る(t *testing.T) {
	t.Parallel()

	require.NoError(t, rawcheck.Validate(rawschema.Raws{}))
}

func TestSchema_制約の違反を位置で返す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raws     rawschema.Raws
		wantPath string
	}{
		{
			name: "名前が長すぎる",
			raws: makeItemRaws(func(i *rawschema.Item) {
				i.Name = "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをん12345678901234567890"
			}),
			wantPath: "items.0.name",
		},
		{
			name: "命中率が範囲外",
			raws: makeItemRaws(func(i *rawschema.Item) {
				i.Melee.Accuracy = 999
			}),
			wantPath: "items.0.melee.accuracy",
		},
		{
			name: "スプライトキーのパターン不正",
			raws: makeItemRaws(func(i *rawschema.Item) {
				i.SpriteKey = "INVALID-KEY!"
			}),
			wantPath: "items.0.spriteKey",
		},
		{
			name: "不正な攻撃種別",
			raws: makeItemRaws(func(i *rawschema.Item) {
				i.Melee.AttackCategory = "INVALID"
			}),
			wantPath: "items.0.melee.attackCategory",
		},
		{
			name: "不正な属性",
			raws: makeItemRaws(func(i *rawschema.Item) {
				i.Melee.Element = "INVALID_ELEMENT"
			}),
			wantPath: "items.0.melee.element",
		},
		{
			name: "ダメージが負の値",
			raws: makeItemRaws(func(i *rawschema.Item) {
				i.Melee.Damage = -1
			}),
			wantPath: "items.0.melee.damage",
		},
		{
			name: "攻撃回数がゼロ",
			raws: makeItemRaws(func(i *rawschema.Item) {
				i.Melee.AttackCount = 0
			}),
			wantPath: "items.0.melee.attackCount",
		},
		{
			name:     "コレクションの出現確率が0",
			raws:     makeCollectionRaws(0),
			wantPath: "itemCollections.0.entries.0.chance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var ce *rawcheck.Error
			require.ErrorAs(t, rawcheck.Validate(tt.raws), &ce)
			assert.Equal(t, []string{tt.wantPath}, ce.Paths())
		})
	}
}

// makeCollectionRaws は出現確率 chance の行を1つ持つアイテムコレクションだけの Raws を生成する
func makeCollectionRaws(chance rawschema.ItemCollectionChance) rawschema.Raws {
	return rawschema.Raws{ItemCollections: &[]rawschema.ItemCollection{{
		Id: "c", Name: "c", Entries: []rawschema.ItemCollectionEntry{{Id: "鉄くず", Chance: chance, Pack: "1d1"}},
	}}}
}

func TestSchema_コレクションの出現確率が正なら通る(t *testing.T) {
	t.Parallel()
	require.NoError(t, rawcheck.Validate(makeCollectionRaws(0.5)))
}

// makeItemRaws は正常なアイテムを1つ持つRawsを生成し、modifyで値を改変する
func makeItemRaws(modify func(*rawschema.Item)) rawschema.Raws {
	item := rawschema.Item{
		Id:              "テスト武器",
		Name:            "テスト武器",
		Description:     "テスト用の武器",
		SpriteSheetName: "test_sheet",
		SpriteKey:       "test_key",
		Value:           100,
		Melee: &rawschema.Melee{
			Accuracy:       80,
			Damage:         10,
			AttackCount:    1,
			Element:        "NONE",
			AttackCategory: "sword",
			Cost:           5,
			TargetGroup:    "ENEMY",
			TargetNum:      "SINGLE",
		},
	}
	modify(&item)
	return rawschema.Raws{Items: &[]rawschema.Item{item}}
}

func TestValidateDisassemblyReferences(t *testing.T) {
	t.Parallel()

	validItems := &[]rawschema.Item{
		{Id: "鉄くず", Name: "鉄くず"},
	}

	t.Run("実在する産出名なら通る", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Items: validItems,
			Props: &[]rawschema.Prop{{
				Name: "棚",
				Hp:   new(rawschema.HitPoints(10)),
				Disassembly: &rawschema.Disassembly{
					ToolCategory: new(rawschema.Prying),
					Yields:       []rawschema.DisassemblyYield{{Id: "鉄くず", Count: "1d1"}},
				},
			}},
		}
		require.NoError(t, validateDisassemblyReferences(raws))
	})

	t.Run("道具が必須なのに道具の分類が無いとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Items: validItems,
			Props: &[]rawschema.Prop{{
				Name: "木",
				Hp:   new(rawschema.HitPoints(10)),
				Disassembly: &rawschema.Disassembly{
					ToolRequired: new(true),
					Yields:       []rawschema.DisassemblyYield{{Id: "鉄くず", Count: "1d1"}},
				},
			}},
		}
		require.ErrorIs(t, validateDisassemblyReferences(raws), errDisassemblyToolRequiredNoTool)
	})

	t.Run("HPの無いpropに分解定義があるとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Items: validItems,
			Props: &[]rawschema.Prop{{
				Name: "草",
				Disassembly: &rawschema.Disassembly{
					Yields: []rawschema.DisassemblyYield{{Id: "鉄くず", Count: "1d1"}},
				},
			}},
		}
		require.ErrorIs(t, validateDisassemblyReferences(raws), errDisassemblyWithoutHP)
	})

	t.Run("propの産出名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Items: validItems,
			Props: &[]rawschema.Prop{{
				Name: "棚",
				Hp:   new(rawschema.HitPoints(10)),
				Disassembly: &rawschema.Disassembly{
					ToolCategory: new(rawschema.Prying),
					Yields:       []rawschema.DisassemblyYield{{Id: "存在しない素材", Count: "1d1"}},
				},
			}},
		}
		err := validateDisassemblyReferences(raws)
		require.ErrorIs(t, err, errDisassemblyYieldUndefined)
	})

	t.Run("propのボーナス名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Items: &[]rawschema.Item{{Id: "鉄くず", Name: "鉄くず"}},
			Props: &[]rawschema.Prop{{Id: "分解対象", Name: "分解対象", Hp: new(rawschema.HitPoints(10)), Disassembly: &rawschema.Disassembly{
				ToolCategory: new(rawschema.Precision),
				Yields:       []rawschema.DisassemblyYield{{Id: "鉄くず", Count: "1d1"}},
				Bonus:        &[]rawschema.DisassemblyBonus{{Id: "存在しないボーナス", Count: "1d1", MinSkill: new(rawschema.SkillLevel(10))}},
			}}},
		}
		err := validateDisassemblyReferences(raws)
		require.ErrorIs(t, err, errDisassemblyBonusUndefined)
	})
}

func TestValidateDropTableReferences(t *testing.T) {
	t.Parallel()

	items := &[]rawschema.Item{{Id: "鉄くず", Name: "鉄くず"}}

	t.Run("実在する素材は通る", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Items: items,
			DropTables: &[]rawschema.DropTable{{
				Name:    "廃墟",
				Nothing: new(rawschema.EntryWeight(3)),
				Entries: []rawschema.DropTableEntry{{Id: "鉄くず", Weight: 1}},
			}},
		}
		require.NoError(t, validateDropTableReferences(raws))
	})

	t.Run("素材名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Items: items,
			DropTables: &[]rawschema.DropTable{{
				Name:    "廃墟",
				Entries: []rawschema.DropTableEntry{{Id: "存在しない素材", Weight: 1}},
			}},
		}
		err := validateDropTableReferences(raws)
		require.ErrorIs(t, err, errDropTableMaterialUndefined)
	})

	t.Run("メンバーのテーブル名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Items:   items,
			Members: &[]rawschema.Member{{Name: "スライム", DropTableId: new(rawschema.EntityName("未定義テーブル"))}},
		}
		err := validateDropTableReferences(raws)
		require.ErrorIs(t, err, errMemberDropTableUndefined)
	})
}

func TestValidateCommandTableReferences(t *testing.T) {
	t.Parallel()

	t.Run("実在するテーブル名と未指定は通る", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			CommandTables: &[]rawschema.CommandTable{{Id: "素手", Name: "素手"}},
			Members: &[]rawschema.Member{
				{Name: "戦うNPC", CommandTableId: new(rawschema.EntityName("素手"))},
				{Name: "未指定NPC"},
			},
		}
		require.NoError(t, validateCommandTableReferences(raws))
	})

	t.Run("空文字はテーブル名として不正でエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			CommandTables: &[]rawschema.CommandTable{{Id: "素手", Name: "素手"}},
			Members:       &[]rawschema.Member{{Name: "空文字NPC", CommandTableId: new(rawschema.EntityName(""))}},
		}
		err := validateCommandTableReferences(raws)
		require.ErrorIs(t, err, errMemberCommandTableUndefined)
	})

	t.Run("テーブル名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Members: &[]rawschema.Member{{Name: "スライム", CommandTableId: new(rawschema.EntityName("未定義テーブル"))}},
		}
		err := validateCommandTableReferences(raws)
		require.ErrorIs(t, err, errMemberCommandTableUndefined)
	})
}

func TestValidateItemTableReferences(t *testing.T) {
	t.Parallel()

	groups := &[]rawschema.ItemGroup{{Id: "雑貨", Name: "雑貨"}}

	t.Run("実在するグループとコレクションは通る", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			ItemGroups:      groups,
			ItemCollections: &[]rawschema.ItemCollection{{Id: "がらくた", Name: "がらくた"}},
			ItemTables:      &[]rawschema.ItemTable{{Name: "宝箱", Entries: []rawschema.ItemTableEntry{{Id: "雑貨"}, {Id: "がらくた"}}}},
		}
		require.NoError(t, validateItemTableReferences(raws))
	})

	t.Run("グループ名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			ItemGroups: groups,
			ItemTables: &[]rawschema.ItemTable{{Name: "宝箱", Entries: []rawschema.ItemTableEntry{{Id: "未定義グループ"}}}},
		}
		err := validateItemTableReferences(raws)
		require.ErrorIs(t, err, errItemTableRefUndefinedGroup)
	})
}

func TestValidateItemGroupReferences(t *testing.T) {
	t.Parallel()

	items := &[]rawschema.Item{{Id: "鉄くず", Name: "鉄くず"}}

	t.Run("実在するアイテムは通る", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Items:      items,
			ItemGroups: &[]rawschema.ItemGroup{{Name: "素材", Entries: []rawschema.ItemGroupEntry{{Id: "鉄くず"}}}},
		}
		require.NoError(t, validateItemGroupReferences(raws))
	})

	t.Run("アイテム名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Items:      items,
			ItemGroups: &[]rawschema.ItemGroup{{Name: "素材", Entries: []rawschema.ItemGroupEntry{{Id: "未定義アイテム"}}}},
		}
		err := validateItemGroupReferences(raws)
		require.ErrorIs(t, err, errItemGroupRefUndefinedItem)
	})

	t.Run("コレクションのアイテム名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Items:           items,
			ItemCollections: &[]rawschema.ItemCollection{{Name: "素材", Entries: []rawschema.ItemCollectionEntry{{Id: "未定義アイテム"}}}},
		}
		err := validateItemGroupReferences(raws)
		require.ErrorIs(t, err, errItemGroupRefUndefinedItem)
	})

	t.Run("グループとコレクションで id が重なるとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Items:           items,
			ItemGroups:      &[]rawschema.ItemGroup{{Id: "素材", Entries: []rawschema.ItemGroupEntry{{Id: "鉄くず"}}}},
			ItemCollections: &[]rawschema.ItemCollection{{Id: "素材", Entries: []rawschema.ItemCollectionEntry{{Id: "鉄くず"}}}},
		}
		err := validateItemGroupReferences(raws)
		require.ErrorIs(t, err, errItemGroupIDDuplicated)
	})
}

func TestValidateEnemyTableReferences(t *testing.T) {
	t.Parallel()

	members := &[]rawschema.Member{{Id: "スライム", Name: "スライム"}}

	t.Run("実在するメンバーは通る", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Members:     members,
			EnemyTables: &[]rawschema.EnemyTable{{Name: "通常", Entries: []rawschema.EnemyTableEntry{{Id: "スライム"}}}},
		}
		require.NoError(t, validateEnemyTableReferences(raws))
	})

	t.Run("敵名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Members:     members,
			EnemyTables: &[]rawschema.EnemyTable{{Name: "通常", Entries: []rawschema.EnemyTableEntry{{Id: "未定義敵"}}}},
		}
		err := validateEnemyTableReferences(raws)
		require.ErrorIs(t, err, errEnemyTableRefUndefinedEnemy)
	})
}

func TestValidateContentLootTables(t *testing.T) {
	t.Parallel()

	itemTables := &[]rawschema.ItemTable{{Id: "house_closet"}}
	known, unknown := "house_closet", "no_such"
	tests := []struct {
		name    string
		item    *rawschema.EntityID
		sat     *rawschema.EntityID
		wantErr error
	}{
		{"実在する表は通る", &known, &known, nil},
		{"表を書かない家具は通る", nil, nil, nil},
		{"家具の表が無いとエラー", &unknown, nil, errContentLootTableUndefined},
		{"衛星の表が無いとエラー", nil, &unknown, errContentLootTableUndefined},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raws := rawschema.Raws{
				ItemTables: itemTables,
				InteriorContents: &[]rawschema.InteriorContent{{Id: "bedroom", Groups: &[]rawschema.ContentGroup{{
					Style: rawschema.PickEach,
					Items: []rawschema.ContentStuff{{
						Kind: rawschema.Furniture, Ref: "closet", Amount: "1d1", LootTable: tt.item,
						Satellites: &[]rawschema.ContentSatellite{{Kind: rawschema.Furniture, Ref: "closet", LootTable: tt.sat}},
					}},
				}}}},
			}
			require.ErrorIs(t, validateContentLootTables(raws), tt.wantErr)
		})
	}
}

func TestValidateFacilityReferences(t *testing.T) {
	t.Parallel()

	enemyTables := &[]rawschema.EnemyTable{{Id: "clinic_enemies", Name: "診療所"}}
	baseZones := []rawschema.FacilityZone{
		{Zone: rawschema.Residential, Weight: 10, MinSpan: 2},
		{Zone: rawschema.Downtown, Weight: 10, MinSpan: 2},
		{Zone: rawschema.Industrial, Weight: 10, MinSpan: 2},
	}

	t.Run("実在する敵テーブルと基本施設は通る", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			EnemyTables: enemyTables,
			Facilities:  &[]rawschema.Facility{{Id: "clinic", EnemyTable: "clinic_enemies", Planner: rawschema.Clinic, Zones: baseZones}},
		}
		require.NoError(t, validateFacilityReferences(raws))
	})

	t.Run("敵テーブルが存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			EnemyTables: enemyTables,
			Facilities:  &[]rawschema.Facility{{Id: "clinic", EnemyTable: "no_such_table", Planner: rawschema.Clinic, Zones: baseZones}},
		}
		require.ErrorIs(t, validateFacilityReferences(raws), errFacilityEnemyTableRefUndefined)
	})

	t.Run("地区に基本施設が無いとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			EnemyTables: enemyTables,
			Facilities:  &[]rawschema.Facility{{Id: "clinic", EnemyTable: "clinic_enemies", Planner: rawschema.Clinic, Zones: []rawschema.FacilityZone{{Zone: rawschema.Downtown, Weight: 10, MinSpan: 3}}}},
		}
		require.ErrorIs(t, validateFacilityReferences(raws), errZoneNoBaseFacility)
	})

	t.Run("未登録施設を参照するfacilityContentsはエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			EnemyTables:      enemyTables,
			Facilities:       &[]rawschema.Facility{{Id: "clinic", EnemyTable: "clinic_enemies", Planner: rawschema.Clinic, Zones: baseZones}},
			FacilityContents: &[]rawschema.FacilityContent{{Facility: "no_such", Variants: []string{"x"}}},
		}
		require.ErrorIs(t, validateFacilityReferences(raws), errFacilityKeyUndefined)
	})

	for _, missing := range []rawschema.Zone{rawschema.Downtown, rawschema.Residential, rawschema.Industrial} {
		t.Run("地区 "+string(missing)+" だけ基本施設が無いとエラー", func(t *testing.T) {
			t.Parallel()
			zones := make([]rawschema.FacilityZone, 0, len(baseZones))
			for _, z := range baseZones {
				if z.Zone == missing {
					z.MinSpan = 3
				}
				zones = append(zones, z)
			}
			raws := rawschema.Raws{
				EnemyTables: enemyTables,
				Facilities:  &[]rawschema.Facility{{Id: "clinic", EnemyTable: "clinic_enemies", Planner: rawschema.Clinic, Zones: zones}},
			}
			err := validateFacilityReferences(raws)
			require.ErrorIs(t, err, errZoneNoBaseFacility)
		})
	}

	t.Run("全地区に基本施設があれば通る", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			EnemyTables: enemyTables,
			Facilities: &[]rawschema.Facility{{Id: "house", EnemyTable: "clinic_enemies", Planner: rawschema.House, Zones: []rawschema.FacilityZone{
				{Zone: rawschema.Residential, Weight: 10, MinSpan: 2},
				{Zone: rawschema.Downtown, Weight: 10, MinSpan: 2},
				{Zone: rawschema.Industrial, Weight: 10, MinSpan: 2},
			}}},
		}
		require.NoError(t, validateFacilityReferences(raws))
	})
}

func TestValidateInteriorContentReferences(t *testing.T) {
	t.Parallel()

	contents := &[]rawschema.InteriorContent{{Id: "house"}, {Id: "bedroom"}}

	t.Run("実在する content 参照は通る", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			InteriorContents: contents,
			FacilityContents: &[]rawschema.FacilityContent{{Facility: "house", Variants: []rawschema.EntityID{"house"}}},
			FacilityRooms: &[]rawschema.FacilityRooms{{
				Facility: "house",
				Rooms:    &[]rawschema.RoomContent{{Role: "bedroom", Content: "bedroom"}},
				Fallback: "bedroom",
			}},
			FlavorContent: &rawschema.InteriorContent{Id: "flavor"},
		}
		require.NoError(t, validateInteriorContentReferences(raws))
	})

	t.Run("存在しない変種はエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			InteriorContents: contents,
			FacilityContents: &[]rawschema.FacilityContent{{Facility: "house", Variants: []rawschema.EntityID{"no_such"}}},
		}
		require.ErrorIs(t, validateInteriorContentReferences(raws), errInteriorContentRefUndefined)
	})

	t.Run("存在しない奥室 content はエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			InteriorContents: contents,
			FacilityRooms: &[]rawschema.FacilityRooms{{
				Facility: "house",
				Rooms:    &[]rawschema.RoomContent{{Role: "bedroom", Content: "no_such"}},
				Fallback: "bedroom",
			}},
		}
		require.ErrorIs(t, validateInteriorContentReferences(raws), errInteriorContentRefUndefined)
	})

	t.Run("存在しない fallback はエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			InteriorContents: contents,
			FacilityRooms:    &[]rawschema.FacilityRooms{{Facility: "house", Fallback: "no_such"}},
		}
		require.ErrorIs(t, validateInteriorContentReferences(raws), errInteriorContentRefUndefined)
	})

	t.Run("id が重複するとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			InteriorContents: &[]rawschema.InteriorContent{{Id: "house"}, {Id: "house"}},
		}
		require.ErrorIs(t, validateInteriorContentReferences(raws), errInteriorContentDuplicateID)
	})

	t.Run("interior を積むのに flavorContent が無いとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			InteriorContents: &[]rawschema.InteriorContent{{Id: "house"}},
		}
		require.ErrorIs(t, validateInteriorContentReferences(raws), errInteriorFlavorContentMissing)
	})
}

func TestValidateCommandTableWeaponReferences(t *testing.T) {
	t.Parallel()

	items := &[]rawschema.Item{{Id: "刀", Name: "刀"}}

	t.Run("実在する武器は通る", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Items:         items,
			CommandTables: &[]rawschema.CommandTable{{Name: "剣術", Entries: []rawschema.CommandTableEntry{{Id: "刀"}}}},
		}
		require.NoError(t, validateCommandTableWeaponReferences(raws))
	})

	t.Run("武器名が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Items:         items,
			CommandTables: &[]rawschema.CommandTable{{Name: "剣術", Entries: []rawschema.CommandTableEntry{{Id: "未定義武器"}}}},
		}
		err := validateCommandTableWeaponReferences(raws)
		require.ErrorIs(t, err, errCommandTableRefUndefinedWeapon)
	})
}

func TestValidateLandmarkReferences(t *testing.T) {
	t.Parallel()

	props := &[]rawschema.Prop{{Id: "candle"}}

	t.Run("prop が実在すれば通る", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Props:     props,
			Landmarks: &[]rawschema.Landmark{{Id: "shrine", Weight: 10, Drawer: rawschema.Open, Props: []rawschema.PropSpot{{Name: "candle"}}}},
		}
		require.NoError(t, validateLandmarkReferences(raws))
	})

	t.Run("prop が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Props:     props,
			Landmarks: &[]rawschema.Landmark{{Id: "shrine", Weight: 10, Drawer: rawschema.Open, Props: []rawschema.PropSpot{{Name: "no_such_prop"}}}},
		}
		require.ErrorIs(t, validateLandmarkReferences(raws), errLandmarkPropUndefined)
	})
}

func TestValidateScatterZoneReferences(t *testing.T) {
	t.Parallel()

	props := &[]rawschema.Prop{{Id: "tree_a"}}
	groups := &[]rawschema.ItemGroup{{Id: "junk"}}

	t.Run("prop と item group が実在すれば通る", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Props:      props,
			ItemGroups: groups,
			ScatterZones: &[]rawschema.ScatterZone{{Id: "wild", LootGroup: "junk", Entries: []rawschema.ScatterEntry{
				{Ref: "", Weight: 10},
				{Ref: "tree_a", Weight: 10, Satellites: &[]rawschema.PropSpot{{Name: "tree_a"}}},
			}, Grass: []rawschema.GroundCoverEntry{{Ref: "tree_a", Weight: 1}}, Soil: []rawschema.GroundCoverEntry{{Ref: "tree_a", Weight: 1}}}},
		}
		require.NoError(t, validateScatterZoneReferences(raws))
	})

	t.Run("土の飾りの prop が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Props:        props,
			ItemGroups:   groups,
			ScatterZones: &[]rawschema.ScatterZone{{Id: "wild", LootGroup: "junk", Soil: []rawschema.GroundCoverEntry{{Ref: "no_such_prop", Weight: 1}}}},
		}
		require.ErrorIs(t, validateScatterZoneReferences(raws), errScatterPropUndefined)
	})

	t.Run("item group が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Props:        props,
			ItemGroups:   groups,
			ScatterZones: &[]rawschema.ScatterZone{{Id: "wild", LootGroup: "no_such_group"}},
		}
		require.ErrorIs(t, validateScatterZoneReferences(raws), errScatterLootGroupUndefined)
	})

	t.Run("散布 prop が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Props:        props,
			ItemGroups:   groups,
			ScatterZones: &[]rawschema.ScatterZone{{Id: "wild", LootGroup: "junk", Entries: []rawschema.ScatterEntry{{Ref: "no_such_prop", Weight: 10}}}},
		}
		require.ErrorIs(t, validateScatterZoneReferences(raws), errScatterPropUndefined)
	})

	t.Run("satellite の prop が存在しないとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Props:      props,
			ItemGroups: groups,
			ScatterZones: &[]rawschema.ScatterZone{{Id: "wild", LootGroup: "junk", Entries: []rawschema.ScatterEntry{
				{Ref: "tree_a", Weight: 10, Satellites: &[]rawschema.PropSpot{{Name: "no_such_prop"}}},
			}}},
		}
		require.ErrorIs(t, validateScatterZoneReferences(raws), errScatterPropUndefined)
	})
}

func TestValidateMapGlyphReferences(t *testing.T) {
	t.Parallel()

	wasteland, entrance := rawschema.Wasteland, rawschema.DungeonEntrance
	chunkGlyphs := func() []rawschema.MapGlyph {
		return []rawschema.MapGlyph{{Id: "field", Glyph: ".", Chunk: &wasteland}, {Id: "dungeon_entrance", Glyph: ">", Chunk: &entrance}}
	}

	t.Run("1文字で重複せず全施設とランドマークとチャンクの種類に記号があれば通る", func(t *testing.T) {
		t.Parallel()
		glyphs := append(chunkGlyphs(), rawschema.MapGlyph{Id: "house", Glyph: "h"}, rawschema.MapGlyph{Id: "shrine", Glyph: "s"})
		raws := rawschema.Raws{
			Facilities: &[]rawschema.Facility{{Id: "house"}},
			Landmarks:  &[]rawschema.Landmark{{Id: "shrine"}},
			MapGlyphs:  &glyphs,
		}
		require.NoError(t, validateMapGlyphReferences(raws))
	})

	t.Run("チャンクの種類に記号が無いとエラー", func(t *testing.T) {
		t.Parallel()
		glyphs := chunkGlyphs()[:1]
		require.ErrorIs(t, validateMapGlyphReferences(rawschema.Raws{MapGlyphs: &glyphs}), errEnumRowMissing)
	})

	t.Run("チャンクの種類を2つの記号が指すとエラー", func(t *testing.T) {
		t.Parallel()
		glyphs := append(chunkGlyphs(), rawschema.MapGlyph{Id: "field2", Glyph: ",", Chunk: &wasteland})
		require.ErrorIs(t, validateMapGlyphReferences(rawschema.Raws{MapGlyphs: &glyphs}), errDuplicateID)
	})

	t.Run("記号が2文字だとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{MapGlyphs: &[]rawschema.MapGlyph{{Id: "house", Glyph: "hh"}}}
		require.ErrorIs(t, validateMapGlyphReferences(raws), errMapGlyphNotSingleRune)
	})

	t.Run("記号が空だとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{MapGlyphs: &[]rawschema.MapGlyph{{Id: "house", Glyph: ""}}}
		require.ErrorIs(t, validateMapGlyphReferences(raws), errMapGlyphNotSingleRune)
	})

	t.Run("記号が重複するとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{MapGlyphs: &[]rawschema.MapGlyph{{Id: "house", Glyph: "h"}, {Id: "hamlet", Glyph: "h"}}}
		require.ErrorIs(t, validateMapGlyphReferences(raws), errMapGlyphDuplicate)
	})

	t.Run("施設に記号が無いとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Facilities: &[]rawschema.Facility{{Id: "house"}},
			MapGlyphs:  &[]rawschema.MapGlyph{{Id: "field", Glyph: "."}},
		}
		require.ErrorIs(t, validateMapGlyphReferences(raws), errMapGlyphMissing)
	})

	t.Run("ランドマークに記号が無いとエラー", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Landmarks: &[]rawschema.Landmark{{Id: "shrine"}},
			MapGlyphs: &[]rawschema.MapGlyph{{Id: "field", Glyph: "."}},
		}
		require.ErrorIs(t, validateMapGlyphReferences(raws), errMapGlyphMissing)
	})
}

func TestValidateFeatureUniqueIDs(t *testing.T) {
	t.Parallel()

	t.Run("id が一意なら通る", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{
			Facilities:   &[]rawschema.Facility{{Id: "house"}, {Id: "store"}},
			Landmarks:    &[]rawschema.Landmark{{Id: "shrine"}},
			ScatterZones: &[]rawschema.ScatterZone{{Id: "wild"}},
			MapGlyphs:    &[]rawschema.MapGlyph{{Id: "house"}},
		}
		require.NoError(t, validateFeatureUniqueIDs(raws))
	})

	cases := []struct {
		name string
		raws rawschema.Raws
	}{
		{"施設の id が重複するとエラー", rawschema.Raws{Facilities: &[]rawschema.Facility{{Id: "house"}, {Id: "house"}}}},
		{"ランドマークの id が重複するとエラー", rawschema.Raws{Landmarks: &[]rawschema.Landmark{{Id: "shrine"}, {Id: "shrine"}}}},
		{"散布ゾーンの id が重複するとエラー", rawschema.Raws{ScatterZones: &[]rawschema.ScatterZone{{Id: "wild"}, {Id: "wild"}}}},
		{"地図記号の id が重複するとエラー", rawschema.Raws{MapGlyphs: &[]rawschema.MapGlyph{{Id: "house"}, {Id: "house"}}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateFeatureUniqueIDs(c.raws), errDuplicateID)
		})
	}
}

func TestSchema_抽選重みは1以上(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		mutate   func(*rawschema.Raws)
		wantPath string
	}{
		{"施設の地区重みが0", func(r *rawschema.Raws) { (*r.Facilities)[0].Zones[0].Weight = 0 }, "facilities.0.zones.0.weight"},
		{"ランドマークの重みが0", func(r *rawschema.Raws) { (*r.Landmarks)[0].Weight = 0 }, "landmarks.0.weight"},
		{"散布 prop の重みが0", func(r *rawschema.Raws) { (*r.ScatterZones)[0].Entries[0].Weight = 0 }, "scatterZones.0.entries.0.weight"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
			require.NoError(t, err)
			c.mutate(&raws)
			var ce *rawcheck.Error
			require.ErrorAs(t, rawcheck.Validate(raws), &ce)
			assert.Equal(t, []string{c.wantPath}, ce.Paths())
		})
	}
}

func TestValidateReferences_各検証の失敗を伝播する(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		raws    rawschema.Raws
		wantErr error
	}{
		{
			name:    "id重複はerrDuplicateID",
			raws:    rawschema.Raws{Facilities: &[]rawschema.Facility{{Id: "a"}, {Id: "a"}}},
			wantErr: errDuplicateID,
		},
		{
			name: "施設の敵テーブル未存在はerrFacilityEnemyTableRefUndefined",
			raws: rawschema.Raws{
				Facilities: &[]rawschema.Facility{{Id: "f1", EnemyTable: "存在しないテーブル"}},
			},
			wantErr: errFacilityEnemyTableRefUndefined,
		},
		{
			name: "内装コンテンツID重複はerrInteriorContentDuplicateID",
			raws: rawschema.Raws{
				InteriorContents: &[]rawschema.InteriorContent{{Id: "c"}, {Id: "c"}},
			},
			wantErr: errInteriorContentDuplicateID,
		},
		{
			name: "ランドマークのprop未存在はerrLandmarkPropUndefined",
			raws: rawschema.Raws{
				Landmarks: &[]rawschema.Landmark{{Id: "lm1", Props: []rawschema.PropSpot{{Name: "存在しないprop"}}}},
			},
			wantErr: errLandmarkPropUndefined,
		},
		{
			name: "散布ゾーンのlootGroup未存在はerrScatterLootGroupUndefined",
			raws: rawschema.Raws{
				ScatterZones: &[]rawschema.ScatterZone{{Id: "z1", LootGroup: "存在しないグループ"}},
			},
			wantErr: errScatterLootGroupUndefined,
		},
		{
			name: "地図記号が1文字でないとerrMapGlyphNotSingleRune",
			raws: rawschema.Raws{
				MapGlyphs: &[]rawschema.MapGlyph{{Id: "g1", Glyph: "ab"}},
			},
			wantErr: errMapGlyphNotSingleRune,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateReferences(c.raws)
			require.ErrorIs(t, err, c.wantErr)
		})
	}
}

func TestValidateBurnoutItems(t *testing.T) {
	t.Parallel()

	ashes := "ashes"
	missing := "missing"
	tests := []struct {
		name    string
		burnout *string
		wantErr error
	}{
		{name: "燃え残りが実在するアイテムなら通る", burnout: &ashes},
		{name: "燃え残りを持たない prop は通る"},
		{name: "燃え残りが実在しないアイテムを指す", burnout: &missing, wantErr: errBurnoutItemUndefined},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raws := rawschema.Raws{
				Items: &[]rawschema.Item{{Id: "ashes"}},
				Props: &[]rawschema.Prop{{Id: "fire", BurnoutItem: tt.burnout}},
			}
			err := validateBurnoutItems(raws)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidateMemberItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		member  rawschema.Member
		wantErr error
	}{
		{name: "実在するアイテムを指せば通る", member: rawschema.Member{Id: "ash", StartingEquips: &[]rawschema.ProfessionEquip{{Name: "torch", Slot: rawschema.EquipSlotWEAPON1}}, StartingItems: &[]rawschema.ProfessionItem{{Name: "torch", Count: 1}}, Stock: &[]rawschema.ProfessionItem{{Name: "torch", Count: 1}}}},
		{name: "初期装備が実在しないアイテムを指す", member: rawschema.Member{Id: "ash", StartingEquips: &[]rawschema.ProfessionEquip{{Name: "missing", Slot: rawschema.EquipSlotWEAPON1}}}, wantErr: errMemberItemUndefined},
		{name: "初期アイテムが実在しないアイテムを指す", member: rawschema.Member{Id: "ash", StartingItems: &[]rawschema.ProfessionItem{{Name: "missing", Count: 1}}}, wantErr: errMemberItemUndefined},
		{name: "在庫が実在しないアイテムを指す", member: rawschema.Member{Id: "merchant", Stock: &[]rawschema.ProfessionItem{{Name: "missing", Count: 1}}}, wantErr: errMemberItemUndefined},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raws := rawschema.Raws{Items: &[]rawschema.Item{{Id: "torch"}}, Members: &[]rawschema.Member{tt.member}}
			err := validateMemberItems(raws)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidateDungeons(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*rawschema.Dungeon)
		extra   bool
		wantErr error
	}{
		{name: "表が実在すれば通る", mutate: func(*rawschema.Dungeon) {}},
		{name: "敵の表が実在しない", mutate: func(d *rawschema.Dungeon) { d.EnemyTable = "missing" }, wantErr: errDungeonEnemyTableUndefined},
		{name: "アイテムの表が実在しない", mutate: func(d *rawschema.Dungeon) { d.ItemTable = "missing" }, wantErr: errDungeonItemTableUndefined},
		{name: "id が重複する", mutate: func(*rawschema.Dungeon) {}, extra: true, wantErr: errDuplicateID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := rawschema.Dungeon{Id: "dead_forest", EnemyTable: "forest", ItemTable: "forest"}
			tt.mutate(&d)
			dungeons := []rawschema.Dungeon{d}
			if tt.extra {
				dungeons = append(dungeons, d)
			}
			raws := rawschema.Raws{
				EnemyTables: &[]rawschema.EnemyTable{{Id: "forest"}},
				ItemTables:  &[]rawschema.ItemTable{{Id: "forest"}},
				Dungeons:    &dungeons,
			}
			err := validateDungeons(raws)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidateSettlements(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*rawschema.Settlement)
		wantErr error
	}{
		{name: "住人と prop と記号が実在すれば通る", mutate: func(*rawschema.Settlement) {}},
		{name: "住人が実在しない", mutate: func(s *rawschema.Settlement) { s.Members[0].Name = "missing" }, wantErr: errSettlementMemberUndefined},
		{name: "prop が実在しない", mutate: func(s *rawschema.Settlement) { s.Props[0].Name = "missing" }, wantErr: errSettlementPropUndefined},
		{name: "地図の記号が無い", mutate: func(s *rawschema.Settlement) { s.Id = "town" }, wantErr: errMapGlyphMissing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := rawschema.Settlement{Id: "village", Weight: 1, Members: []rawschema.PropSpot{{Name: "merchant"}}, Props: []rawschema.PropSpot{{Name: "hearth"}}}
			tt.mutate(&s)
			raws := rawschema.Raws{
				Members:     &[]rawschema.Member{{Id: "merchant"}},
				Props:       &[]rawschema.Prop{{Id: "hearth"}},
				MapGlyphs:   &[]rawschema.MapGlyph{{Id: "village"}},
				Settlements: &[]rawschema.Settlement{s},
			}
			err := validateSettlements(raws)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidateSettlements_id_が重複する(t *testing.T) {
	t.Parallel()
	raws := rawschema.Raws{
		MapGlyphs:   &[]rawschema.MapGlyph{{Id: "village"}},
		Settlements: &[]rawschema.Settlement{{Id: "village", Weight: 1}, {Id: "village", Weight: 1}},
	}
	require.ErrorIs(t, validateSettlements(raws), errDuplicateID)
}

func TestValidateDialogues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		key       string
		dialogues []rawschema.Dialogue
		wantErr   error
	}{
		{name: "member の会話が実在すれば通る", key: "greeting", dialogues: []rawschema.Dialogue{{Id: "greeting", Pages: []string{"Hi."}}}},
		{name: "member の会話が実在しない", key: "farewell", dialogues: []rawschema.Dialogue{{Id: "greeting", Pages: []string{"Hi."}}}, wantErr: errMemberDialogueUndefined},
		{name: "会話の id が重複する", key: "greeting", dialogues: []rawschema.Dialogue{{Id: "greeting", Pages: []string{"Hi."}}, {Id: "greeting", Pages: []string{"Yo."}}}, wantErr: errDuplicateID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raws := rawschema.Raws{
				Members:   &[]rawschema.Member{{Id: "villager", Dialog: &rawschema.Dialog{MessageKey: tt.key}}, {Id: "rat"}},
				Dialogues: &tt.dialogues,
			}
			err := validateDialogues(raws)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidateTileSpawns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		spawn   *rawschema.TileSpawn
		wantErr error
	}{
		{name: "生成するタイルが実在すれば通る", spawn: &rawschema.TileSpawn{Tile: new("dwall"), Autotile: true}},
		{name: "生成するタイルを省けば自身で通る", spawn: &rawschema.TileSpawn{Autotile: true}},
		{name: "生成の仕様を持たないタイルは通る"},
		{name: "生成するタイルが実在しない", spawn: &rawschema.TileSpawn{Tile: new("missing")}, wantErr: errTileSpawnUndefined},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raws := rawschema.Raws{Tiles: &[]rawschema.Tile{{Id: "dwall"}, {Id: "wall", Spawn: tt.spawn}}}
			err := validateTileSpawns(raws)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}
