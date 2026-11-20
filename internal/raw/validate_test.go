package raw

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/rawschema/rawcheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ringsLayers と rowsLayers は、今の町と研究所の敷地と同じ層
var (
	ringsLayers = []rawschema.DistrictLayer{
		{Zones: []rawschema.Zone{rawschema.Residential, rawschema.Industrial}},
		{Zones: []rawschema.Zone{rawschema.Downtown}},
	}
	rowsLayers = []rawschema.DistrictLayer{
		{Zones: []rawschema.Zone{rawschema.CampusGate}, Rows: new(1)},
		{Zones: []rawschema.Zone{rawschema.CampusInner}},
		{Zones: []rawschema.Zone{rawschema.CampusQuarantine}, Rows: new(1)},
	}
)

func TestRealData_HasLandmarks(t *testing.T) {
	t.Parallel()

	master, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	require.NotEmpty(t, PtrSlice(master.Landmarks), "overworld 生成が要求するので実 raw.toml は landmarks を持つべき")
}

func TestRealData_HasCrashSites(t *testing.T) {
	t.Parallel()

	master, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	require.NotEmpty(t, PtrSlice(master.CrashSites), "開始チャンクに墜落現場を必ず置くので実 raw.toml は crashSites を持つべき")
}

func TestRealData_プレイヤー以外の全てのmemberの行が跡を持つ(t *testing.T) {
	t.Parallel()

	master, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	missing := []string{}
	for _, m := range PtrSlice(master.Members) {
		if isPlayer := m.Player != nil && *m.Player; !isPlayer && m.RemainsId == nil {
			missing = append(missing, m.Id)
		}
	}
	assert.Equal(t, []string{}, missing, "倒したときに跡が出るよう、敵と NPC の行は remainsId を持つ")
}

func TestRealData_壊せる置物の行は構造物を除いて跡を持つ(t *testing.T) {
	t.Parallel()

	master, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	missing := []string{}
	for _, p := range PtrSlice(master.Props) {
		if p.Hp != nil && p.RemainsId == nil {
			missing = append(missing, p.Id)
		}
	}
	assert.Equal(t, []string{"warp_next", "warp_prev", "warp_escape", "dungeon_entrance"}, missing,
		"壊れない階段や入口のほかは、壊したときに跡が出るよう remainsId を持つ")
}

func TestValidateFlatShapes(t *testing.T) {
	t.Parallel()

	prop := func(depth rawschema.SpriteDepth, shape *rawschema.SpriteShape) rawschema.Raws {
		return rawschema.Raws{Props: &[]rawschema.Prop{{Id: "p", SpriteRender: rawschema.SpriteRender{Depth: depth, Shape: shape}}}}
	}
	tests := []struct {
		name    string
		raws    rawschema.Raws
		wantErr error
	}{
		{"床の模様の段を寝かせる", prop(rawschema.SpriteDepthDecal, new(rawschema.Flat)), nil},
		{"床の段を立てる", prop(rawschema.SpriteDepthFloor, nil), nil},
		{"床の模様の段を立てるとエラー", prop(rawschema.SpriteDepthDecal, nil), errFlatDepthMismatch},
		{"高さのある段を寝かせるとエラー", prop(rawschema.SpriteDepthTaller, new(rawschema.Flat)), errFlatDepthMismatch},
		{"床の模様の段の飾りを立てるとエラー", rawschema.Raws{Decors: &[]rawschema.Decor{{Id: "d", SpriteRender: rawschema.SpriteRender{Depth: rawschema.SpriteDepthDecal}}}}, errFlatDepthMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validateFlatShapes(tt.raws), tt.wantErr)
		})
	}
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

func TestValidateGrounds(t *testing.T) {
	t.Parallel()

	tiles := &[]rawschema.Tile{{Id: "dirt", Earth: new(true)}, {Id: "floor"}}
	zone := func(g rawschema.EntityID) *[]rawschema.ScatterZone {
		return &[]rawschema.ScatterZone{{Id: "wild", Ground: g}}
	}
	tests := []struct {
		name    string
		raws    rawschema.Raws
		wantErr error
	}{
		{name: "土の地面の下地は通る", raws: rawschema.Raws{Tiles: tiles, ScatterZones: zone("dirt"), Dungeons: &[]rawschema.Dungeon{{Id: "forest", Ground: new(rawschema.EntityID("dirt"))}, {Id: "cave"}}}},
		{name: "区分の下地が土でなければエラー", raws: rawschema.Raws{Tiles: tiles, ScatterZones: zone("floor")}, wantErr: errGroundNotEarth},
		{name: "ダンジョンの下地が土でなければエラー", raws: rawschema.Raws{Tiles: tiles, ScatterZones: zone("dirt"), Dungeons: &[]rawschema.Dungeon{{Id: "forest", Ground: new(rawschema.EntityID("floor"))}}}, wantErr: errGroundNotEarth},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateGrounds(tt.raws)
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

	towns := &[]rawschema.District{{Id: "town", Spacing: 7, Separation: 2, Pattern: rawschema.DistrictPatternRings, Layers: ringsLayers, Sizes: []rawschema.DistrictSize{{W: 3, H: 3, Weight: 1}, {W: 2, H: 2, Weight: 1}}}}

	t.Run("全地区に基本施設があれば通る", func(t *testing.T) {
		t.Parallel()
		raws := rawschema.Raws{Districts: towns, Facilities: &[]rawschema.Facility{{Id: "house", Zones: baseZones}}}
		require.NoError(t, validateZoneBaseFacilities(raws))
	})

	t.Run("区域を持たない raw は通る", func(t *testing.T) {
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
			raws := rawschema.Raws{Districts: towns, Facilities: &[]rawschema.Facility{{Id: "clinic", Zones: zones}}}
			require.ErrorIs(t, validateZoneBaseFacilities(raws), errZoneNoBaseFacility)
		})
	}

	t.Run("深い所にしか建たない施設は基本施設に数えない", func(t *testing.T) {
		t.Parallel()
		zones := make([]rawschema.FacilityZone, 0, len(baseZones))
		for _, z := range baseZones {
			if z.Zone == rawschema.Downtown {
				z.MinDanger = new(2)
			}
			zones = append(zones, z)
		}
		raws := rawschema.Raws{Districts: towns, Facilities: &[]rawschema.Facility{{Id: "comms", Zones: zones}}}
		require.ErrorIs(t, validateZoneBaseFacilities(raws), errZoneNoBaseFacility)
	})
}

func TestValidateZoneBaseFacilities_研究所の敷地は門と中ほどと奥の地区に施設が要る(t *testing.T) {
	t.Parallel()

	campus := &[]rawschema.District{{Id: "lab_campus", Spacing: 30, Separation: 4, Pattern: rawschema.DistrictPatternRows, Layers: rowsLayers, Fence: new(rawschema.EntityID("fence")), Sizes: []rawschema.DistrictSize{{W: 7, H: 6, Weight: 1}}}}
	zones := []rawschema.FacilityZone{
		{Zone: rawschema.CampusGate, Weight: 1, MinSpan: 1},
		{Zone: rawschema.CampusInner, Weight: 1, MinSpan: 1},
	}
	raws := rawschema.Raws{Districts: campus, Facilities: &[]rawschema.Facility{{Id: "lab", Zones: zones}}}
	require.ErrorIs(t, validateZoneBaseFacilities(raws), errZoneNoBaseFacility, "閉鎖区画の施設が無い")

	raws.Facilities = &[]rawschema.Facility{{Id: "lab", Zones: zones}, {Id: "lab_quarantine", Zones: []rawschema.FacilityZone{{Zone: rawschema.CampusQuarantine, Weight: 1, MinSpan: 1}}}}
	require.NoError(t, validateZoneBaseFacilities(raws))
	landmark := rawschema.EntityID("lab_quarantine")
	(*raws.Districts)[0].Landmark = &landmark
	require.ErrorIs(t, validateZoneBaseFacilities(raws), errZoneNoBaseFacility, "目玉の施設は地区の基本施設に数えない")
}

func TestValidateDistricts(t *testing.T) {
	t.Parallel()

	district := func(h int) rawschema.Raws {
		return rawschema.Raws{Districts: &[]rawschema.District{{Id: "town", Spacing: 7, Separation: 2, Pattern: rawschema.DistrictPatternRings, Layers: ringsLayers, Sizes: []rawschema.DistrictSize{{W: 2, H: 2, Weight: 1}, {W: 5, H: h, Weight: 1}}}}}
	}
	require.NoError(t, validateDistricts(district(5)), "高さが spacing - separation ちょうどなら収まる")
	require.ErrorIs(t, validateDistricts(district(6)), errDistrictSizeOutOfRegion)

	campus := func(h int) rawschema.Raws {
		return rawschema.Raws{Districts: &[]rawschema.District{{Id: "lab_campus", Spacing: 30, Separation: 4, Pattern: rawschema.DistrictPatternRows, Layers: rowsLayers, Fence: new(rawschema.EntityID("fence")), Sizes: []rawschema.DistrictSize{{W: 7, H: h, Weight: 1}}}}}
	}
	require.NoError(t, validateDistricts(campus(3)), "門、中ほど、奥が1行ずつ並ぶ")
	require.ErrorIs(t, validateDistricts(campus(2)), errDistrictTooShort, "残りの行を占める層に1行も残らない")
}

func TestValidateDistrictLayers(t *testing.T) {
	t.Parallel()

	zone := []rawschema.Zone{rawschema.Downtown}
	tests := []struct {
		name    string
		pattern rawschema.DistrictPattern
		layers  []rawschema.DistrictLayer
		wantErr error
	}{
		{"ringsは2層", rawschema.DistrictPatternRings, ringsLayers, nil},
		{"ringsの層が3つ", rawschema.DistrictPatternRings, []rawschema.DistrictLayer{{Zones: zone}, {Zones: zone}, {Zones: zone}}, errDistrictLayers},
		{"ringsが行の数を書く", rawschema.DistrictPatternRings, []rawschema.DistrictLayer{{Zones: zone, Rows: new(1)}, {Zones: zone}}, errDistrictLayers},
		{"rowsは残りの層が1つ", rawschema.DistrictPatternRows, rowsLayers, nil},
		{"rowsが1層", rawschema.DistrictPatternRows, []rawschema.DistrictLayer{{Zones: zone}}, errDistrictLayers},
		{"rowsに残りの層が無い", rawschema.DistrictPatternRows, []rawschema.DistrictLayer{{Zones: zone, Rows: new(1)}, {Zones: zone, Rows: new(1)}}, errDistrictLayers},
		{"rowsに残りの層が2つ", rawschema.DistrictPatternRows, []rawschema.DistrictLayer{{Zones: zone}, {Zones: zone}}, errDistrictLayers},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateDistrictLayers(&rawschema.District{Id: "d", Pattern: tt.pattern, Layers: tt.layers})
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
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

func TestSchema_地物の空きは間隔より小さい(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	f := &(*raws.PointFeatures)[0]
	f.Separation = f.Spacing
	var ce *rawcheck.Error
	require.ErrorAs(t, rawcheck.Validate(raws), &ce)
	assert.Equal(t, []string{"pointFeatures.0.separation"}, ce.Paths())
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
			raws:    rawschema.Raws{Districts: &[]rawschema.District{{Id: "town", Spacing: 7, Separation: 2, Pattern: rawschema.DistrictPatternRings, Layers: ringsLayers, Sizes: []rawschema.DistrictSize{{W: 2, H: 2, Weight: 1}}}}, Facilities: &[]rawschema.Facility{{Id: "f1", Zones: []rawschema.FacilityZone{{Zone: rawschema.Downtown, Weight: 1, MinSpan: 3}}}}},
			wantErr: errZoneNoBaseFacility,
		},
		{
			name:    "区域の大きさがリージョンに収まらない",
			raws:    rawschema.Raws{Districts: &[]rawschema.District{{Id: "town", Spacing: 3, Separation: 2, Sizes: []rawschema.DistrictSize{{W: 2, H: 2, Weight: 1}}}}},
			wantErr: errDistrictSizeOutOfRegion,
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

func TestValidateLandmarkMinDanger_目玉の施設に深さの下限を書くとエラー(t *testing.T) {
	t.Parallel()

	landmark := rawschema.EntityID("armory")
	layers := []rawschema.DistrictLayer{{Zones: []rawschema.Zone{rawschema.MilitaryArmory}}}
	raws := func(zone rawschema.Zone, minDanger *int) rawschema.Raws {
		return rawschema.Raws{
			Districts:  &[]rawschema.District{{Id: "base", Landmark: &landmark, Layers: layers}},
			Facilities: &[]rawschema.Facility{{Id: "armory", Zones: []rawschema.FacilityZone{{Zone: zone, Weight: 1, MinSpan: 1, MinDanger: minDanger}}}},
		}
	}
	require.NoError(t, validateLandmarkMinDanger(raws(rawschema.MilitaryArmory, nil)))
	require.ErrorIs(t, validateLandmarkMinDanger(raws(rawschema.MilitaryArmory, new(3))), errLandmarkMinDanger)
	require.NoError(t, validateLandmarkMinDanger(raws(rawschema.Downtown, new(3))), "区域が割り振らない地区の行は抽選を通るので深さの下限を書ける")
}

func TestValidateCommandTables(t *testing.T) {
	t.Parallel()

	items := &[]rawschema.Item{{Id: "cleaver", Name: "Cleaver", Melee: &rawschema.Melee{}}, {Id: "rock", Name: "Rock"}}
	table := func(e rawschema.CommandTableEntry) rawschema.Raws {
		return rawschema.Raws{Items: items, CommandTables: &[]rawschema.CommandTable{{Id: "bat", Name: "Bat", Entries: []rawschema.CommandTableEntry{e}}}}
	}
	tests := []struct {
		name    string
		raws    rawschema.Raws
		wantErr error
	}{
		{name: "体の攻撃だけを引く行は通る", raws: table(rawschema.CommandTableEntry{BodyAttack: new("bite"), Weight: 1})},
		{name: "アイテムだけを引く行は通る", raws: table(rawschema.CommandTableEntry{Item: new("cleaver"), Weight: 1})},
		{name: "両方を引く行はエラー", raws: table(rawschema.CommandTableEntry{BodyAttack: new("bite"), Item: new("cleaver"), Weight: 1}), wantErr: errCommandEntryTarget},
		{name: "どちらも引かない行はエラー", raws: table(rawschema.CommandTableEntry{Weight: 1}), wantErr: errCommandEntryTarget},
		{name: "近接攻撃を持たないアイテムを引く行はエラー", raws: table(rawschema.CommandTableEntry{Item: new("rock"), Weight: 1}), wantErr: errItemNoMelee},
		{name: "無いアイテムを引く行はエラー", raws: table(rawschema.CommandTableEntry{Item: new("typo"), Weight: 1}), wantErr: KeyNotFoundError{Key: "typo", Collection: "Items"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateCommandTables(tt.raws)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}
