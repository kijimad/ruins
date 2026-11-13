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

// makeCollectionRaws は出現確率 chance の行を1つ持つアイテムコレクションと、その行が指すアイテムの Raws を生成する
func makeCollectionRaws(chance rawschema.ItemCollectionChance) rawschema.Raws {
	raws := makeItemRaws(func(*rawschema.Item) {})
	raws.ItemCollections = &[]rawschema.ItemCollection{{
		Id: "c", Name: "c", Entries: []rawschema.ItemCollectionEntry{{Id: "テスト武器", Chance: chance, Pack: "1d1"}},
	}}
	return raws
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

func TestValidateDisassemblies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prop    rawschema.Prop
		wantErr error
	}{
		{
			name: "道具の分類と HP を持てば通る",
			prop: rawschema.Prop{Name: "棚", Hp: new(rawschema.HitPoints(10)), Disassembly: &rawschema.Disassembly{ToolCategory: new(rawschema.Prying)}},
		},
		{
			name:    "道具が必須なのに道具の分類が無いとエラー",
			prop:    rawschema.Prop{Name: "木", Hp: new(rawschema.HitPoints(10)), Disassembly: &rawschema.Disassembly{ToolRequired: new(true)}},
			wantErr: errDisassemblyToolRequiredNoTool,
		},
		{
			name:    "HPの無いpropに分解定義があるとエラー",
			prop:    rawschema.Prop{Name: "草", Disassembly: &rawschema.Disassembly{}},
			wantErr: errDisassemblyWithoutHP,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateDisassemblies(rawschema.Raws{Props: &[]rawschema.Prop{tt.prop}})
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidateZoneBaseFacilities(t *testing.T) {
	t.Parallel()

	baseZones := []rawschema.FacilityZone{
		{Zone: rawschema.Residential, Weight: 10, MinSpan: 2},
		{Zone: rawschema.Downtown, Weight: 10, MinSpan: 2},
		{Zone: rawschema.Industrial, Weight: 10, MinSpan: 2},
	}

	t.Run("全地区に基本施設があれば通る", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{Facilities: &[]rawschema.Facility{{Id: "house", Zones: baseZones}}}
		require.NoError(t, validateZoneBaseFacilities(raws))
	})

	t.Run("施設を持たない raw は通る", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateZoneBaseFacilities(rawschema.Raws{}))
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
			raws := rawschema.Raws{Facilities: &[]rawschema.Facility{{Id: "clinic", Zones: zones}}}
			require.ErrorIs(t, validateZoneBaseFacilities(raws), errZoneNoBaseFacility)
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

func TestValidateRules_各検証の失敗を伝播する(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		raws    rawschema.Raws
		wantErr error
	}{
		{
			name:    "地区の基本施設が無い",
			raws:    rawschema.Raws{Facilities: &[]rawschema.Facility{{Id: "f1", Zones: []rawschema.FacilityZone{{Zone: rawschema.Downtown, Weight: 1, MinSpan: 3}}}}},
			wantErr: errZoneNoBaseFacility,
		},
		{
			name:    "地図記号が1文字でない",
			raws:    rawschema.Raws{MapGlyphs: &[]rawschema.MapGlyph{{Id: "g1", Glyph: "ab"}}},
			wantErr: errMapGlyphNotSingleRune,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, validateRules(c.raws), c.wantErr)
		})
	}
}

func TestValidateRules_空のRawsは通る(t *testing.T) {
	t.Parallel()
	require.NoError(t, validateRules(rawschema.Raws{}))
}
