package rawcheck_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/rawschema/rawcheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeRefRaws は参照する側とされる側の行を表ごとに持ち、検証を通る raws を作り、mutate で崩す
func makeRefRaws(mutate func(*rawschema.Raws)) rawschema.Raws {
	render := func(key string) rawschema.SpriteRender {
		return rawschema.SpriteRender{SpriteSheetName: "field", SpriteKey: key, Depth: rawschema.N0}
	}
	tile := func(id string, spawn *rawschema.TileSpawn) rawschema.Tile {
		return rawschema.Tile{Id: id, Name: id, Description: id, SpriteRender: render(id), Spawn: spawn}
	}
	raws := rawschema.Raws{
		Materials: &[]rawschema.MaterialDef{{Id: rawschema.METAL, Name: "Metal", Disassembly: &rawschema.MaterialDisassembly{Item: "iron", PerWeight: "1 kg"}}},
		Recipes:   &[]rawschema.Recipe{{Id: "iron", Name: "Iron", Inputs: []rawschema.RecipeInput{{Id: "iron", Amount: 1}}}},
		Items: &[]rawschema.Item{{
			Id: "iron", Name: "Iron", Description: "Iron", SpriteSheetName: "field", SpriteKey: "iron",
			Remedy: &rawschema.Remedy{Treats: []rawschema.ConditionTypeName{"fracture"}, Potency: 1},
			Book:   &rawschema.Book{TotalEffort: 1, Skill: &rawschema.SkillBook{TargetSkill: rawschema.SkillIdSword, MaxLevel: 1}},
		}},
		Members: &[]rawschema.Member{{
			Id: "slime", Name: "Slime", SpriteSheetName: "field", SpriteKey: "slime",
			CommandTableId: new("fists"),
			DropTableId:    new("remains"),
			Dialog:         &rawschema.Dialog{MessageKey: "greeting"},
			StartingEquips: &[]rawschema.ProfessionEquip{{Name: "iron", Slot: rawschema.EquipSlotWEAPON1}},
			StartingItems:  &[]rawschema.ProfessionItem{{Name: "iron", Count: 1}},
			Stock:          &[]rawschema.ProfessionItem{{Name: "iron", Count: 1}},
		}},
		CommandTables:   &[]rawschema.CommandTable{{Id: "fists", Name: "Fists", Entries: []rawschema.CommandTableEntry{{Id: "iron", Weight: 1}}}},
		DropTables:      &[]rawschema.DropTable{{Id: "remains", Name: "Remains", Entries: []rawschema.DropTableEntry{{Id: "iron", Weight: 1}}}},
		ItemGroups:      &[]rawschema.ItemGroup{{Id: "junk", Name: "Junk", Entries: []rawschema.ItemGroupEntry{{Id: "iron", Weight: 1, Pack: "1d1"}}}},
		ItemCollections: &[]rawschema.ItemCollection{{Id: "scraps", Name: "Scraps", Entries: []rawschema.ItemCollectionEntry{{Id: "iron", Chance: 50, Pack: "1d1"}}}},
		ItemTables:      &[]rawschema.ItemTable{{Id: "chest", Name: "Chest", Entries: []rawschema.ItemTableEntry{{Id: "junk"}, {Id: "scraps"}}}},
		EnemyTables:     &[]rawschema.EnemyTable{{Id: "wild", Name: "Wild", Entries: []rawschema.EnemyTableEntry{{Id: "slime", Weight: 1, Pack: "1d1"}}}},
		Facilities: &[]rawschema.Facility{{
			Id: "house", Planner: rawschema.House, EnemyTable: "wild",
			Zones: []rawschema.FacilityZone{{Zone: rawschema.Residential, Weight: 1}},
		}},
		FacilityContents: &[]rawschema.FacilityContent{{Facility: "house", Variants: []rawschema.EntityID{"house_main"}}},
		FacilityRooms: &[]rawschema.FacilityRooms{{
			Facility: "house", Fallback: "house_main",
			Rooms: &[]rawschema.RoomContent{{Role: rawschema.Bedroom, Content: "house_main"}},
		}},
		InteriorContents: &[]rawschema.InteriorContent{{Id: "house_main", Groups: &[]rawschema.ContentGroup{{
			Style: rawschema.PickEach,
			Items: []rawschema.ContentStuff{{
				Kind: rawschema.Furniture, Ref: "bed", Amount: "1d1", LootTable: new("chest"),
				Satellites: &[]rawschema.ContentSatellite{{Kind: rawschema.Furniture, Ref: "bed", Offsets: []rawschema.ContentVec{}, LootTable: new("chest")}},
			}},
		}}}},
		Landmarks: &[]rawschema.Landmark{{Id: "shrine", Weight: 1, Drawer: rawschema.Open, Props: []rawschema.PropSpot{{Name: "bed"}}}},
		ScatterZones: &[]rawschema.ScatterZone{{
			Id: "meadow", LootGroup: "junk",
			Entries: []rawschema.ScatterEntry{
				{Ref: "", Weight: 1},
				{Ref: "bed", Weight: 1, Satellites: &[]rawschema.PropSpot{{Name: "bed"}}},
			},
			Grass: []rawschema.GroundCoverEntry{{Ref: "bed", Weight: 1}},
			Soil:  []rawschema.GroundCoverEntry{{Ref: "bed", Weight: 1}},
		}},
		MapGlyphs: &[]rawschema.MapGlyph{{Id: "house", Glyph: "h", Name: "House"}},
		Props: &[]rawschema.Prop{{
			Id: "bed", Name: "Bed", Description: "Bed", SpriteRender: render("bed"),
			Hp:          new(rawschema.HitPoints(1)),
			BurnoutItem: new("iron"),
			Disassembly: &rawschema.Disassembly{
				Yields: []rawschema.DisassemblyYield{{Id: "iron", Count: "1d1"}},
				Bonus:  &[]rawschema.DisassemblyBonus{{Id: "iron", Count: "1d1"}},
			},
		}},
		Conditions: &[]rawschema.Condition{{Id: "fracture", Name: "Fracture", Description: "Fracture", Parts: &[]rawschema.BodyPartId{"arm"}}},
		BodyParts:  &[]rawschema.BodyPart{{Id: "arm", Name: "Arm", HitWeight: 1}},
		Skills: &[]rawschema.Skill{{
			Id: rawschema.SkillIdSword, Name: "Sword", Summary: "Sword", GainedBy: "Sword", Effect: "Sword", Ability: rawschema.Str, Category: "combat",
			Proficiencies: &[]rawschema.SkillProficiency{{Id: rawschema.ProficiencyIdSwordDamage}},
		}},
		SkillCategories: &[]rawschema.SkillCategory{{Id: "combat", Name: "Combat"}},
		AttackTypes: &[]rawschema.AttackType{{
			Id: rawschema.AttackCategorySword, Name: "Sword", Range: rawschema.AttackRangeMelee, Injury: "fracture",
			Skill: rawschema.SkillIdSword, Damage: rawschema.ProficiencyIdSwordDamage, Accuracy: rawschema.ProficiencyIdSwordAccuracy,
		}},
		Professions: &[]rawschema.Profession{{
			Id: "soldier", Name: "Soldier", Description: "Soldier",
			Skills: &[]rawschema.ProfessionSkill{{Id: rawschema.SkillIdSword}},
			Items:  []rawschema.ProfessionItem{}, Equips: []rawschema.ProfessionEquip{},
		}},
		InteriorRefs: &[]rawschema.InteriorRef{{Id: "bed_ref", Prop: new("bed")}, {Id: "snacks", ItemGroup: new("junk")}},
		InteriorProfile: &rawschema.InteriorProfile{
			Damages: []rawschema.DamageChoice{}, Clutters: []rawschema.ClutterChoice{}, Variants: []rawschema.VariantChoice{},
			Densities: []rawschema.DensityChoice{}, Sites: []rawschema.SiteChoice{},
			Heroes:    []rawschema.HeroChoice{{Id: "reward", Weight: 1, Refs: []rawschema.EntityID{"bed_ref"}}},
			Facade:    rawschema.FacadeRefs{Window: "bed_ref", Shutter: "bed_ref", Sign: "bed_ref"},
			Lot:       rawschema.LotRefs{Fence: "bed_ref", Plant: "bed_ref", Vending: "bed_ref"},
			HeroOneIn: 1,
		},
		Dungeons: &[]rawschema.Dungeon{{
			Id: "forest", Name: "Forest", Description: "Forest", ImageKey: "forest", EnemyTable: "wild", ItemTable: "chest",
			Planners: []rawschema.DungeonPlannerWeight{{Id: rawschema.DungeonPlannerCave, Weight: 1}},
		}},
		Settlements: &[]rawschema.Settlement{{Id: "village", Weight: 1, Members: []rawschema.PropSpot{{Name: "slime"}}, Props: []rawschema.PropSpot{{Name: "bed"}}}},
		Dialogues:   &[]rawschema.Dialogue{{Id: "greeting", Pages: []rawschema.DialoguePage{"Hi."}}},
		Tiles:       &[]rawschema.Tile{tile("floor", &rawschema.TileSpawn{Tile: new("wall")}), tile("wall", nil)},
	}
	mutate(&raws)
	return raws
}

func TestValidate_参照先がそろえば通る(t *testing.T) {
	t.Parallel()
	require.NoError(t, rawcheck.Validate(makeRefRaws(func(*rawschema.Raws) {})))
}

func TestValidate_参照先の表を省いた参照の無い行は通る(t *testing.T) {
	t.Parallel()
	raws := rawschema.Raws{Members: &[]rawschema.Member{{Id: "slime", Name: "Slime", SpriteSheetName: "field", SpriteKey: "slime"}}}
	require.NoError(t, rawcheck.Validate(raws))
}

func TestValidate_参照先が無いと参照の位置で返す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mutate   func(*rawschema.Raws)
		wantPath string
	}{
		{"アイテムテーブルのグループ", func(r *rawschema.Raws) { (*r.ItemTables)[0].Entries[0].Id = "none" }, "itemTables.0.entries.0.id_ref"},
		{"アイテムテーブルのコレクション", func(r *rawschema.Raws) { (*r.ItemTables)[0].Entries[1].Id = "none" }, "itemTables.0.entries.1.id_ref"},
		{"アイテムグループのアイテム", func(r *rawschema.Raws) { (*r.ItemGroups)[0].Entries[0].Id = "none" }, "itemGroups.0.entries.0.id_ref"},
		{"アイテムコレクションのアイテム", func(r *rawschema.Raws) { (*r.ItemCollections)[0].Entries[0].Id = "none" }, "itemCollections.0.entries.0.id_ref"},
		{"敵テーブルの敵", func(r *rawschema.Raws) { (*r.EnemyTables)[0].Entries[0].Id = "none" }, "enemyTables.0.entries.0.id_ref"},
		{"コマンドテーブルの武器", func(r *rawschema.Raws) { (*r.CommandTables)[0].Entries[0].Id = "none" }, "commandTables.0.entries.0.id_ref"},
		{"ドロップテーブルの素材", func(r *rawschema.Raws) { (*r.DropTables)[0].Entries[0].Id = "none" }, "dropTables.0.entries.0.id_ref"},
		{"施設の敵テーブル", func(r *rawschema.Raws) { (*r.Facilities)[0].EnemyTable = "none" }, "facilities.0.enemyTable_ref"},
		{"主室の内装の施設", func(r *rawschema.Raws) { (*r.FacilityContents)[0].Facility = "none" }, "facilityContents.0.facility_ref"},
		{"主室の内装の変種", func(r *rawschema.Raws) { (*r.FacilityContents)[0].Variants[0] = "none" }, "facilityContents.0.variants_ref.0"},
		{"奥室の施設", func(r *rawschema.Raws) { (*r.FacilityRooms)[0].Facility = "none" }, "facilityRooms.0.facility_ref"},
		{"奥室の役割の内装", func(r *rawschema.Raws) { (*(*r.FacilityRooms)[0].Rooms)[0].Content = "none" }, "facilityRooms.0.rooms.0.content_ref"},
		{"奥室のフォールバック", func(r *rawschema.Raws) { (*r.FacilityRooms)[0].Fallback = "none" }, "facilityRooms.0.fallback_ref"},
		{"家具の戦利品表", func(r *rawschema.Raws) { contentStuff(r).LootTable = new("none") }, "interiorContents.0.groups.0.items.0.lootTable_ref"},
		{"衛星の戦利品表", func(r *rawschema.Raws) { (*contentStuff(r).Satellites)[0].LootTable = new("none") }, "interiorContents.0.groups.0.items.0.satellites.0.lootTable_ref"},
		{"ランドマークの prop", func(r *rawschema.Raws) { (*r.Landmarks)[0].Props[0].Name = "none" }, "landmarks.0.props_ref.0"},
		{"散布ゾーンの戦利品グループ", func(r *rawschema.Raws) { (*r.ScatterZones)[0].LootGroup = "none" }, "scatterZones.0.lootGroup_ref"},
		{"散布の prop", func(r *rawschema.Raws) { (*r.ScatterZones)[0].Entries[1].Ref = "none" }, "scatterZones.0.entries.1.ref_ref"},
		{"散布の衛星の prop", func(r *rawschema.Raws) { (*(*r.ScatterZones)[0].Entries[1].Satellites)[0].Name = "none" }, "scatterZones.0.entries.1.satellites_ref.0"},
		{"草の prop", func(r *rawschema.Raws) { (*r.ScatterZones)[0].Grass[0].Ref = "none" }, "scatterZones.0.grass_ref.0"},
		{"土の飾りの prop", func(r *rawschema.Raws) { (*r.ScatterZones)[0].Soil[0].Ref = "none" }, "scatterZones.0.soil_ref.0"},
		{"member のコマンドテーブル", func(r *rawschema.Raws) { (*r.Members)[0].CommandTableId = new("none") }, "members.0.commandTableId_ref"},
		{"member のドロップテーブル", func(r *rawschema.Raws) { (*r.Members)[0].DropTableId = new("none") }, "members.0.dropTableId_ref"},
		{"member の会話", func(r *rawschema.Raws) { (*r.Members)[0].Dialog.MessageKey = "none" }, "members.0.dialog.messageKey_ref"},
		{"member の初期装備", func(r *rawschema.Raws) { (*(*r.Members)[0].StartingEquips)[0].Name = "none" }, "members.0.startingEquips_ref.0"},
		{"member の初期アイテム", func(r *rawschema.Raws) { (*(*r.Members)[0].StartingItems)[0].Name = "none" }, "members.0.startingItems_ref.0"},
		{"member の在庫", func(r *rawschema.Raws) { (*(*r.Members)[0].Stock)[0].Name = "none" }, "members.0.stock_ref.0"},
		{"分解の産出", func(r *rawschema.Raws) { (*r.Props)[0].Disassembly.Yields[0].Id = "none" }, "props.0.disassembly.yields.0.id_ref"},
		{"分解のボーナス", func(r *rawschema.Raws) { (*(*r.Props)[0].Disassembly.Bonus)[0].Id = "none" }, "props.0.disassembly.bonus.0.id_ref"},
		{"燃え残り", func(r *rawschema.Raws) { (*r.Props)[0].BurnoutItem = new("none") }, "props.0.burnoutItem_ref"},
		{"不調の部位", func(r *rawschema.Raws) { (*(*r.Conditions)[0].Parts)[0] = "tail" }, "conditions.0.parts_ref.0"},
		{"治療の対象", func(r *rawschema.Raws) { (*r.Items)[0].Remedy.Treats[0] = "bruise" }, "items.0.remedy.treats_ref.0"},
		{"材質の分解の戻し先", func(r *rawschema.Raws) { (*r.Materials)[0].Disassembly.Item = "none" }, "materials.0.disassembly.item_ref"},
		{"レシピの産物", func(r *rawschema.Raws) { (*r.Recipes)[0].Id = "none" }, "recipes.0.id_ref"},
		{"レシピの素材", func(r *rawschema.Raws) { (*r.Recipes)[0].Inputs[0].Id = "none" }, "recipes.0.inputs.0.id_ref"},
		{"攻撃種別の怪我", func(r *rawschema.Raws) { (*r.AttackTypes)[0].Injury = "bruise" }, "attackTypes.0.injury_ref"},
		{"スキルのカテゴリ", func(r *rawschema.Raws) { (*r.Skills)[0].Category = "magic" }, "skills.0.category_ref"},
		{"本のスキル", func(r *rawschema.Raws) { (*r.Items)[0].Book.Skill.TargetSkill = rawschema.SkillIdSpear }, "items.0.book.skill.targetSkill_ref"},
		{"職業のスキル", func(r *rawschema.Raws) { (*(*r.Professions)[0].Skills)[0].Id = rawschema.SkillIdSpear }, "professions.0.skills.0.id_ref"},
		{"内装の Ref の prop", func(r *rawschema.Raws) { (*r.InteriorRefs)[0].Prop = new("none") }, "interiorRefs.0.prop_ref"},
		{"内装の Ref のアイテムグループ", func(r *rawschema.Raws) { (*r.InteriorRefs)[1].ItemGroup = new("none") }, "interiorRefs.1.itemGroup_ref"},
		{"hero の Ref が prop を持たない", func(r *rawschema.Raws) { r.InteriorProfile.Heroes[0].Refs[0] = "snacks" }, "interiorProfile.heroes.0.refs_ref.0"},
		{"外皮の Ref が実在しない", func(r *rawschema.Raws) { r.InteriorProfile.Facade.Sign = "none" }, "interiorProfile.facade.sign_ref"},
		{"外構の Ref が prop を持たない", func(r *rawschema.Raws) { r.InteriorProfile.Lot.Vending = "snacks" }, "interiorProfile.lot.vending_ref"},
		{"ダンジョンの敵テーブル", func(r *rawschema.Raws) { (*r.Dungeons)[0].EnemyTable = "none" }, "dungeons.0.enemyTable_ref"},
		{"ダンジョンのアイテムテーブル", func(r *rawschema.Raws) { (*r.Dungeons)[0].ItemTable = "none" }, "dungeons.0.itemTable_ref"},
		{"集落の住人", func(r *rawschema.Raws) { (*r.Settlements)[0].Members[0].Name = "none" }, "settlements.0.members_ref.0"},
		{"集落の prop", func(r *rawschema.Raws) { (*r.Settlements)[0].Props[0].Name = "none" }, "settlements.0.props_ref.0"},
		{"タイルの生成の仕様", func(r *rawschema.Raws) { (*r.Tiles)[0].Spawn.Tile = new("none") }, "tileSpawns_ref.floor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var ce *rawcheck.Error
			require.ErrorAs(t, rawcheck.Validate(makeRefRaws(tt.mutate)), &ce)
			assert.Equal(t, []string{tt.wantPath}, ce.Paths())
		})
	}
}

func TestValidate_idが重複すると重複のidの位置で返す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mutate   func(*rawschema.Raws)
		wantPath string
	}{
		{"施設", func(r *rawschema.Raws) { *r.Facilities = append(*r.Facilities, (*r.Facilities)[0]) }, "facilityIds_unique.house"},
		{"ランドマーク", func(r *rawschema.Raws) { *r.Landmarks = append(*r.Landmarks, (*r.Landmarks)[0]) }, "landmarkIds_unique.shrine"},
		{"散布ゾーン", func(r *rawschema.Raws) { *r.ScatterZones = append(*r.ScatterZones, (*r.ScatterZones)[0]) }, "scatterZoneIds_unique.meadow"},
		{"地図記号", func(r *rawschema.Raws) { *r.MapGlyphs = append(*r.MapGlyphs, (*r.MapGlyphs)[0]) }, "mapGlyphIds_unique.house"},
		{"集落", func(r *rawschema.Raws) { *r.Settlements = append(*r.Settlements, (*r.Settlements)[0]) }, "settlementIds_unique.village"},
		{"部位", func(r *rawschema.Raws) { *r.BodyParts = append(*r.BodyParts, (*r.BodyParts)[0]) }, "bodyPartIds_unique.arm"},
		{"不調", func(r *rawschema.Raws) { *r.Conditions = append(*r.Conditions, (*r.Conditions)[0]) }, "conditionIds_unique.fracture"},
		{"攻撃種別", func(r *rawschema.Raws) { *r.AttackTypes = append(*r.AttackTypes, (*r.AttackTypes)[0]) }, "attackTypeIds_unique.sword"},
		{"スキル", func(r *rawschema.Raws) {
			s := (*r.Skills)[0]
			s.Proficiencies = nil
			*r.Skills = append(*r.Skills, s)
		}, "skillIds_unique.sword"},
		{"スキルのカテゴリ", func(r *rawschema.Raws) { *r.SkillCategories = append(*r.SkillCategories, (*r.SkillCategories)[0]) }, "skillCategoryIds_unique.combat"},
		{"熟練の効果倍率がスキルをまたぐ", func(r *rawschema.Raws) {
			s := (*r.Skills)[0]
			s.Id = rawschema.SkillIdSpear
			*r.Skills = append(*r.Skills, s)
		}, "proficiencyIds_unique.sword_damage"},
		{"内装の Ref", func(r *rawschema.Raws) { *r.InteriorRefs = append(*r.InteriorRefs, (*r.InteriorRefs)[0]) }, "interiorRefIds_unique.bed_ref"},
		{"hero の種類", func(r *rawschema.Raws) {
			r.InteriorProfile.Heroes = append(r.InteriorProfile.Heroes, r.InteriorProfile.Heroes[0])
		}, "interiorProfile.heroIds_unique.reward"},
		{"会話", func(r *rawschema.Raws) { *r.Dialogues = append(*r.Dialogues, (*r.Dialogues)[0]) }, "dialogueIds_unique.greeting"},
		{"ダンジョン", func(r *rawschema.Raws) { *r.Dungeons = append(*r.Dungeons, (*r.Dungeons)[0]) }, "dungeonIds_unique.forest"},
		{"内装レシピ", func(r *rawschema.Raws) {
			*r.InteriorContents = append(*r.InteriorContents, (*r.InteriorContents)[0])
		}, "interiorContentIds_unique.house_main"},
		{"アイテムグループとアイテムコレクション", func(r *rawschema.Raws) { (*r.ItemCollections)[0].Id = "junk" }, "itemGroupIds_unique.junk"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var ce *rawcheck.Error
			require.ErrorAs(t, rawcheck.Validate(makeRefRaws(tt.mutate)), &ce)
			assert.Equal(t, []string{tt.wantPath}, ce.Paths())
		})
	}
}

// contentStuff は makeRefRaws の内装レシピの最初の家具を返す
func contentStuff(r *rawschema.Raws) *rawschema.ContentStuff {
	return &(*(*r.InteriorContents)[0].Groups)[0].Items[0]
}
