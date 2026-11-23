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
		return rawschema.SpriteRender{SpriteSheetName: "field", SpriteKey: key, Depth: rawschema.SpriteDepthFloor}
	}
	tile := func(id string, spawn *rawschema.TileSpawn) rawschema.Tile {
		return rawschema.Tile{Id: id, Name: id, Description: id, SpriteRender: render(id), Spawn: spawn}
	}
	raws := rawschema.Raws{
		ToolCategories: &[]rawschema.ToolCategory{{Id: "mining"}},
		AmmoKinds:      &[]rawschema.AmmoKind{{Id: "9mm"}},
		CrashSites:     &[]rawschema.CrashSite{{Id: "house", Weight: 1}},
		Zones:          &[]rawschema.Zone{{Id: "residential"}, {Id: "downtown"}},
		FoliageEffects: &[]rawschema.Foliage{{Id: "grass", TempModifier: -1}},
		Materials:      &[]rawschema.Material{{Id: "metal", Name: "Metal", Disassembly: &rawschema.MaterialDisassembly{Item: "iron", PerWeight: "1 kg"}}},
		Recipes:        &[]rawschema.Recipe{{Id: "iron", Name: "Iron", Inputs: []rawschema.RecipeInput{{Id: "iron", Amount: 1}}}},
		Items: &[]rawschema.Item{{
			Id: "iron", Name: "Iron", Description: "Iron", SpriteSheetName: "field", SpriteKey: "iron",
			Consumable: &rawschema.Consumable{TargetGroup: rawschema.TargetGroupALLY, TargetNum: rawschema.SINGLE},
			Melee:      &rawschema.Melee{Accuracy: 100, Damage: 1, AttackCount: 1, AttackCategory: rawschema.AttackCategoryFist, Element: rawschema.ElementNONE, TargetGroup: rawschema.TargetGroupENEMY, TargetNum: rawschema.SINGLE},
			Charge:     &rawschema.Charge{Max: 150},
			Remedy:     &rawschema.Remedy{Treats: []rawschema.ConditionTypeName{"fracture"}, Potency: 1},
			Book:       &rawschema.Book{TotalEffort: 1, Skill: &rawschema.SkillBook{TargetSkill: "sword", MaxLevel: 1}},
		}},
		Members: &[]rawschema.Member{{
			Id: "slime", Name: "Slime", SpriteSheetName: "field", SpriteKey: "slime",
			CommandTableId: new("fists"),
			DropTableId:    new("remains"),
			RemainsId:      new("splinters"),
			Dialog:         &rawschema.Dialog{MessageKey: "greeting"},
			StartingEquips: &[]rawschema.ProfessionEquip{{Name: "iron", Slot: rawschema.EquipSlotHAND1}},
			StartingItems:  &[]rawschema.ProfessionItem{{Name: "iron", Count: 1}},
			Stock:          &[]rawschema.ProfessionItem{{Name: "iron", Count: 1}},
		}},
		BodyAttacks:     &[]rawschema.BodyAttack{{Id: "punch", Name: "Punch", Melee: rawschema.Melee{Accuracy: 100, Damage: 1, AttackCount: 1, AttackCategory: rawschema.AttackCategoryFist, Element: rawschema.ElementNONE, TargetGroup: rawschema.TargetGroupENEMY, TargetNum: rawschema.SINGLE}}},
		CommandTables:   &[]rawschema.CommandTable{{Id: "fists", Name: "Fists", Entries: []rawschema.CommandTableEntry{{BodyAttack: new("punch"), Weight: 1}, {Item: new("iron"), Weight: 1}}}},
		DropTables:      &[]rawschema.DropTable{{Id: "remains", Name: "Remains", Entries: []rawschema.DropTableEntry{{Id: "iron", Weight: 1}}}},
		ItemGroups:      &[]rawschema.ItemGroup{{Id: "junk", Name: "Junk", Entries: []rawschema.ItemGroupEntry{{Id: "iron", Weight: 1, Pack: "1d1"}}}},
		ItemCollections: &[]rawschema.ItemCollection{{Id: "scraps", Name: "Scraps", Entries: []rawschema.ItemCollectionEntry{{Id: "iron", Chance: 50, Pack: "1d1"}}}},
		ItemTables:      &[]rawschema.ItemTable{{Id: "chest", Name: "Chest", Entries: []rawschema.ItemTableEntry{{Id: "junk"}, {Id: "scraps"}}}},
		EnemyTables:     &[]rawschema.EnemyTable{{Id: "wild", Name: "Wild", Entries: []rawschema.EnemyTableEntry{{Id: "slime", Weight: 1, Pack: "1d1"}}}},
		Facilities: &[]rawschema.Facility{{
			Id: "house", EnemyTable: "wild",
			Zones: []rawschema.FacilityZone{{Zone: "residential", Weight: 1}},
		}},
		Districts: &[]rawschema.District{{
			Id: "town", Name: "Town", Spacing: 7, Separation: 2, Pattern: rawschema.DistrictPatternRings, Layers: []rawschema.DistrictLayer{{Zones: []rawschema.ZoneId{"residential"}}, {Zones: []rawschema.ZoneId{"downtown"}}}, Landmark: new(rawschema.EntityID("house")),
			Street: "floor", Verge: new(rawschema.EntityID("floor")), StreetEnemies: "1d2",
			Sizes: []rawschema.DistrictSize{{W: 2, H: 2, Weight: 1}},
		}},
		Landmarks: &[]rawschema.Landmark{
			{Id: "house", Weight: 1},
		},
		Remains: &[]rawschema.Remains{{Id: "splinters", Name: "Splinters", SpriteSheetName: "field", SpriteKeys: []rawschema.SpriteKey{"debris"}}},
		Decors:  &[]rawschema.Decor{{Id: "moss", Name: "Moss", SpriteRender: render("moss")}},
		PointFeatures: &[]rawschema.PointFeature{
			{Id: rawschema.PointFeatureKindSettlement, Spacing: 3, Separation: 1, Salt: 1, Road: &rawschema.Road{Tile: "floor", Width: 2}},
		},
		ScatterZones: &[]rawschema.ScatterZone{{
			Id: "meadow", Ground: "floor", LootGroup: "junk",
			Entries: []rawschema.ScatterEntry{
				{Ref: "", Weight: 1},
				{Ref: "bed", Weight: 1, Satellites: &[]rawschema.PropSpot{{Name: "bed"}}},
			},
			Grass: []rawschema.GroundCoverEntry{{Ref: "moss", Weight: 1}},
			Soil:  []rawschema.GroundCoverEntry{{Ref: "moss", Weight: 1}},
		}},
		MapGlyphs: &[]rawschema.MapGlyph{{Id: "house", Glyph: "h", Name: "House"}},
		Props: &[]rawschema.Prop{{
			Id: "bed", Name: "Bed", Description: "Bed", SpriteRender: render("bed"),
			Hp:          new(rawschema.HitPoints(1)),
			BurnoutItem: new("iron"),
			RemainsId:   new("splinters"),
			Disassembly: &rawschema.Disassembly{
				Yields: []rawschema.DisassemblyYield{{Id: "iron", Count: "1d1"}},
				Bonus:  &[]rawschema.DisassemblyBonus{{Id: "iron", Count: "1d1"}},
			},
		}},
		Conditions: &[]rawschema.Condition{{Id: "fracture", Name: "Fracture", Description: "Fracture", Parts: &[]rawschema.BodyPartId{"arm"}}},
		BodyParts:  &[]rawschema.BodyPart{{Id: "arm", Name: "Arm", HitWeight: 1}},
		Skills: &[]rawschema.Skill{{
			Id: "sword", Name: "Sword", Summary: "Sword", GainedBy: "Sword", Effect: "Sword", Ability: rawschema.Str, Category: "combat",
			Proficiencies: &[]rawschema.SkillProficiency{{Id: rawschema.ProficiencyIdSwordDamage}},
		}},
		SkillCategories: &[]rawschema.SkillCategory{{Id: "combat", Name: "Combat"}},
		AttackTypes: &[]rawschema.AttackType{{
			Id: rawschema.AttackCategorySword, Name: "Sword", Range: rawschema.AttackRangeMelee, Injury: "fracture",
			Skill: "sword", Damage: rawschema.ProficiencyIdSwordDamage, Accuracy: rawschema.ProficiencyIdSwordAccuracy,
		}},
		Professions: &[]rawschema.Profession{{
			Id: "soldier", Name: "Soldier", Description: "Soldier",
			Skills: &[]rawschema.ProfessionSkill{{Id: "sword"}},
			Items:  []rawschema.ProfessionItem{}, Equips: []rawschema.ProfessionEquip{},
		}},
		Dungeons: &[]rawschema.Dungeon{{
			Id: "forest", Name: "Forest", Description: "Forest", ImageKey: "forest", EnemyTable: "wild", ItemTable: "chest",
			Planners:   []rawschema.DungeonPlannerWeight{{Id: "cave", Weight: 1}},
			EnemyCount: rawschema.RandInt{Base: 1, Spread: 1}, ItemCount: rawschema.RandInt{Base: 1, Spread: 1},
		}},
		Settlements: &[]rawschema.Settlement{{Id: "dwelling", Weight: 1, Facility: "house"}},
		Dialogues:   &[]rawschema.Dialogue{{Id: "greeting", Pages: []rawschema.DialoguePage{"Hi."}}},
		Tiles:       &[]rawschema.Tile{tile("floor", &rawschema.TileSpawn{Tile: new("wall")}), tile("wall", nil)},
		Layouts: &[]rawschema.Layout{{
			Id: "cave", Tiles: rawschema.LayoutTiles{Floor: "floor", Wall: "wall", Void: "wall"},
			Start: "noise", Steps: []rawschema.EntityID{"widen", "trail"},
		}},
		NoiseStarts:    &[]rawschema.NoiseStart{{Id: "noise", WallChance: 0.3}},
		TemplateStarts: &[]rawschema.TemplateStart{{Id: "town", Template: "town"}},
		WidenSteps:     &[]rawschema.WidenStep{{Id: "widen", Chance: 0.3}},
		TrailSteps:     &[]rawschema.TrailStep{{Id: "trail", Tile: "floor"}},
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
		{"コマンドテーブルの体の攻撃", func(r *rawschema.Raws) { (*r.CommandTables)[0].Entries[0].BodyAttack = new("none") }, "commandTables.0.entries.0.bodyAttack_ref"},
		{"コマンドテーブルの武器", func(r *rawschema.Raws) { (*r.CommandTables)[0].Entries[1].Item = new("none") }, "commandTables.0.entries.1.item_ref"},
		{"コマンドテーブルの近接攻撃を持たない武器", func(r *rawschema.Raws) { (*r.Items)[0].Melee = nil }, "commandTables.0.entries.1.item_ref"},
		{"コマンドテーブルの体の攻撃と武器の両方", func(r *rawschema.Raws) { (*r.CommandTables)[0].Entries[0].Item = new("iron") }, "commandTables.0.entries.0.itemAbsent"},
		{"コマンドテーブルの体の攻撃も武器も無い", func(r *rawschema.Raws) { (*r.CommandTables)[0].Entries[0].BodyAttack = nil }, "commandTables.0.entries.0.item"},
		{"ドロップテーブルの素材", func(r *rawschema.Raws) { (*r.DropTables)[0].Entries[0].Id = "none" }, "dropTables.0.entries.0.id_ref"},
		{"道具の分類", func(r *rawschema.Raws) {
			(*r.Items)[0].Tool = &rawschema.Tool{Categories: []rawschema.ToolCategoryId{"none"}, Grade: 1}
		}, "items.0.tool.categories_ref.0"},
		{"施設の地区", func(r *rawschema.Raws) { (*r.Facilities)[0].Zones[0].Zone = "none" }, "facilities.0.zones.0.zone_ref"},
		{"区域の層の地区", func(r *rawschema.Raws) { (*r.Districts)[0].Layers[1].Zones[0] = "none" }, "districts.0.layers.1.zones_ref.0"},
		{"アイテムの材質", func(r *rawschema.Raws) { (*r.Items)[0].Material = new("none") }, "itemMaterials_ref.iron"},
		{"タイルの植生", func(r *rawschema.Raws) { (*r.Tiles)[1].Foliage = new("none") }, "tiles.1.foliage_ref"},
		{"分解の道具の分類", func(r *rawschema.Raws) { (*r.Props)[0].Disassembly.ToolCategory = new("none") }, "props.0.disassembly.toolCategory_ref"},
		{"弾の口径", func(r *rawschema.Raws) {
			(*r.Items)[0].Ammo = &rawschema.Ammo{AmmoTag: new("none"), DamageBonus: 1, AccuracyBonus: 1}
		}, "items.0.ammo.ammoTag_ref"},
		{"施設の敵テーブル", func(r *rawschema.Raws) { (*r.Facilities)[0].EnemyTable = "none" }, "facilities.0.enemyTable_ref"},
		{"ランドマークの施設", func(r *rawschema.Raws) { (*r.Landmarks)[0].Id = "none" }, "landmarks.0.id_ref"},
		{"区域の塀の家具", func(r *rawschema.Raws) { (*r.Districts)[0].Fence = new(rawschema.EntityID("none")) }, "districts.0.fence_ref"},
		{"地物の道のタイル", func(r *rawschema.Raws) { (*r.PointFeatures)[0].Road.Tile = "none" }, "pointFeatures.0.road.tile_ref"},
		{"散布ゾーンの下地のタイル", func(r *rawschema.Raws) { (*r.ScatterZones)[0].Ground = "none" }, "scatterZones.0.ground_ref"},
		{"散布ゾーンの戦利品グループ", func(r *rawschema.Raws) { (*r.ScatterZones)[0].LootGroup = "none" }, "scatterZones.0.lootGroup_ref"},
		{"区域の目玉の施設", func(r *rawschema.Raws) { (*r.Districts)[0].Landmark = new(rawschema.EntityID("none")) }, "districts.0.landmark_ref"},
		{"区域の街路のタイル", func(r *rawschema.Raws) { (*r.Districts)[0].Street = "none" }, "districts.0.street_ref"},
		{"区域の植え込みのタイル", func(r *rawschema.Raws) { (*r.Districts)[0].Verge = new(rawschema.EntityID("none")) }, "districts.0.verge_ref"},
		{"散布の prop", func(r *rawschema.Raws) { (*r.ScatterZones)[0].Entries[1].Ref = "none" }, "scatterZones.0.entries.1.ref_ref"},
		{"散布の衛星の prop", func(r *rawschema.Raws) { (*(*r.ScatterZones)[0].Entries[1].Satellites)[0].Name = "none" }, "scatterZones.0.entries.1.satellites_ref.0"},
		{"草", func(r *rawschema.Raws) { (*r.ScatterZones)[0].Grass[0].Ref = "none" }, "scatterZones.0.grass_ref.0"},
		{"土の飾り", func(r *rawschema.Raws) { (*r.ScatterZones)[0].Soil[0].Ref = "none" }, "scatterZones.0.soil_ref.0"},
		{"member のコマンドテーブル", func(r *rawschema.Raws) { (*r.Members)[0].CommandTableId = new("none") }, "members.0.commandTableId_ref"},
		{"member のドロップテーブル", func(r *rawschema.Raws) { (*r.Members)[0].DropTableId = new("none") }, "members.0.dropTableId_ref"},
		{"member の会話", func(r *rawschema.Raws) { (*r.Members)[0].Dialog.MessageKey = "none" }, "members.0.dialog.messageKey_ref"},
		{"member の初期装備", func(r *rawschema.Raws) { (*(*r.Members)[0].StartingEquips)[0].Name = "none" }, "members.0.startingEquips_ref.0"},
		{"member の初期アイテム", func(r *rawschema.Raws) { (*(*r.Members)[0].StartingItems)[0].Name = "none" }, "members.0.startingItems_ref.0"},
		{"member の在庫", func(r *rawschema.Raws) { (*(*r.Members)[0].Stock)[0].Name = "none" }, "members.0.stock_ref.0"},
		{"分解の産出", func(r *rawschema.Raws) { (*r.Props)[0].Disassembly.Yields[0].Id = "none" }, "props.0.disassembly.yields.0.id_ref"},
		{"分解のボーナス", func(r *rawschema.Raws) { (*(*r.Props)[0].Disassembly.Bonus)[0].Id = "none" }, "props.0.disassembly.bonus.0.id_ref"},
		{"燃え残り", func(r *rawschema.Raws) { (*r.Props)[0].BurnoutItem = new("none") }, "props.0.burnoutItem_ref"},
		{"member の跡", func(r *rawschema.Raws) { (*r.Members)[0].RemainsId = new("none") }, "members.0.remainsId_ref"},
		{"置物の跡", func(r *rawschema.Raws) { (*r.Props)[0].RemainsId = new("none") }, "props.0.remainsId_ref"},
		{"不調の部位", func(r *rawschema.Raws) { (*(*r.Conditions)[0].Parts)[0] = "tail" }, "conditions.0.parts_ref.0"},
		{"治療の対象", func(r *rawschema.Raws) { (*r.Items)[0].Remedy.Treats[0] = "bruise" }, "items.0.remedy.treats_ref.0"},
		{"材質の分解の戻し先", func(r *rawschema.Raws) { (*r.Materials)[0].Disassembly.Item = "none" }, "materials.0.disassembly.item_ref"},
		{"レシピの産物", func(r *rawschema.Raws) { (*r.Recipes)[0].Id = "none" }, "recipes.0.id_ref"},
		{"レシピの素材", func(r *rawschema.Raws) { (*r.Recipes)[0].Inputs[0].Id = "none" }, "recipes.0.inputs.0.id_ref"},
		{"攻撃種別の怪我", func(r *rawschema.Raws) { (*r.AttackTypes)[0].Injury = "bruise" }, "attackTypes.0.injury_ref"},
		{"爆風に怪我を付ける", func(r *rawschema.Raws) {
			injury := (*r.AttackTypes)[0].Injury
			(*r.Items)[0].Throw = &rawschema.Throw{Damage: 1, Accuracy: 50, Blast: &rawschema.Blast{Radius: 1, Condition: &injury}}
		}, "items.0.throw.blast.condition_notInjury"},
		{"スキルのカテゴリ", func(r *rawschema.Raws) { (*r.Skills)[0].Category = "magic" }, "skills.0.category_ref"},
		{"本のスキル", func(r *rawschema.Raws) { (*r.Items)[0].Book.Skill.TargetSkill = "spear" }, "items.0.book.skill.targetSkill_ref"},
		{"攻撃の種類のスキル", func(r *rawschema.Raws) { (*r.AttackTypes)[0].Skill = "spear" }, "attackTypes.0.skill_ref"},
		{"道具の分類のスキル", func(r *rawschema.Raws) { (*r.ToolCategories)[0].Skill = new("spear") }, "toolCategories.0.skill_ref"},
		{"職業のスキル", func(r *rawschema.Raws) { (*(*r.Professions)[0].Skills)[0].Id = "spear" }, "professions.0.skills.0.id_ref"},
		{"ダンジョンの敵テーブル", func(r *rawschema.Raws) { (*r.Dungeons)[0].EnemyTable = "none" }, "dungeons.0.enemyTable_ref"},
		{"ダンジョンのアイテムテーブル", func(r *rawschema.Raws) { (*r.Dungeons)[0].ItemTable = "none" }, "dungeons.0.itemTable_ref"},
		{"ダンジョンの間取り", func(r *rawschema.Raws) { (*r.Dungeons)[0].Planners[0].Id = "none" }, "dungeons.0.planners.0.id_ref"},
		{"間取りのタイル", func(r *rawschema.Raws) { (*r.Layouts)[0].Tiles.Void = "none" }, "layouts.0.tiles.void_ref"},
		{"間取りの最初の段", func(r *rawschema.Raws) { (*r.Layouts)[0].Start = "none" }, "layouts.0.start_ref"},
		{"間取りの中の段", func(r *rawschema.Raws) { (*r.Layouts)[0].Steps[1] = "none" }, "layouts.0.steps_ref.1"},
		{"小道の段のタイル", func(r *rawschema.Raws) { (*r.TrailSteps)[0].Tile = "none" }, "trailSteps.0.tile_ref"},
		{"集落の施設", func(r *rawschema.Raws) { (*r.Settlements)[0].Facility = "none" }, "settlements.0.facility_ref"},
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
		{"体の攻撃", func(r *rawschema.Raws) { *r.BodyAttacks = append(*r.BodyAttacks, (*r.BodyAttacks)[0]) }, "bodyAttackIds_unique.punch"},
		{"アイテム", func(r *rawschema.Raws) { *r.Items = append(*r.Items, (*r.Items)[0]) }, "itemIds_unique.iron"},
		{"レシピ", func(r *rawschema.Raws) { *r.Recipes = append(*r.Recipes, (*r.Recipes)[0]) }, "recipeIds_unique.iron"},
		{"メンバー", func(r *rawschema.Raws) { *r.Members = append(*r.Members, (*r.Members)[0]) }, "memberIds_unique.slime"},
		{"コマンドテーブル", func(r *rawschema.Raws) { *r.CommandTables = append(*r.CommandTables, (*r.CommandTables)[0]) }, "commandTableIds_unique.fists"},
		{"ドロップテーブル", func(r *rawschema.Raws) { *r.DropTables = append(*r.DropTables, (*r.DropTables)[0]) }, "dropTableIds_unique.remains"},
		{"アイテムテーブル", func(r *rawschema.Raws) { *r.ItemTables = append(*r.ItemTables, (*r.ItemTables)[0]) }, "itemTableIds_unique.chest"},
		{"敵テーブル", func(r *rawschema.Raws) { *r.EnemyTables = append(*r.EnemyTables, (*r.EnemyTables)[0]) }, "enemyTableIds_unique.wild"},
		{"墜落現場", func(r *rawschema.Raws) { *r.CrashSites = append(*r.CrashSites, (*r.CrashSites)[0]) }, "crashSiteIds_unique.house"},
		{"職業", func(r *rawschema.Raws) { *r.Professions = append(*r.Professions, (*r.Professions)[0]) }, "professionIds_unique.soldier"},
		{"タイル", func(r *rawschema.Raws) { *r.Tiles = append(*r.Tiles, (*r.Tiles)[0]) }, "tileIds_unique.floor"},
		{"置物", func(r *rawschema.Raws) { *r.Props = append(*r.Props, (*r.Props)[0]) }, "propIds_unique.bed"},
		{"施設", func(r *rawschema.Raws) { *r.Facilities = append(*r.Facilities, (*r.Facilities)[0]) }, "facilityIds_unique.house"},
		{"ランドマーク", func(r *rawschema.Raws) { *r.Landmarks = append(*r.Landmarks, (*r.Landmarks)[0]) }, "landmarkIds_unique.house"},
		{"跡", func(r *rawschema.Raws) { *r.Remains = append(*r.Remains, (*r.Remains)[0]) }, "remainsIds_unique.splinters"},
		{"地面の飾り", func(r *rawschema.Raws) { *r.Decors = append(*r.Decors, (*r.Decors)[0]) }, "decorIds_unique.moss"},
		{"散布ゾーン", func(r *rawschema.Raws) { *r.ScatterZones = append(*r.ScatterZones, (*r.ScatterZones)[0]) }, "scatterZoneIds_unique.meadow"},
		{"地図記号", func(r *rawschema.Raws) { *r.MapGlyphs = append(*r.MapGlyphs, (*r.MapGlyphs)[0]) }, "mapGlyphIds_unique.house"},
		{"地区", func(r *rawschema.Raws) { *r.Zones = append(*r.Zones, (*r.Zones)[0]) }, "zoneIds_unique.residential"},
		{"材質", func(r *rawschema.Raws) { *r.Materials = append(*r.Materials, (*r.Materials)[0]) }, "materialIds_unique.metal"},
		{"植生", func(r *rawschema.Raws) { *r.FoliageEffects = append(*r.FoliageEffects, (*r.FoliageEffects)[0]) }, "foliageEffectIds_unique.grass"},
		{"道具の分類", func(r *rawschema.Raws) { *r.ToolCategories = append(*r.ToolCategories, (*r.ToolCategories)[0]) }, "toolCategoryIds_unique.mining"},
		{"弾の口径", func(r *rawschema.Raws) { *r.AmmoKinds = append(*r.AmmoKinds, (*r.AmmoKinds)[0]) }, `ammoKindIds_unique."9mm"`},
		{"集落", func(r *rawschema.Raws) { *r.Settlements = append(*r.Settlements, (*r.Settlements)[0]) }, "settlementIds_unique.dwelling"},
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
			s.Id = "spear"
			*r.Skills = append(*r.Skills, s)
		}, "proficiencyIds_unique.sword_damage"},
		{"会話", func(r *rawschema.Raws) { *r.Dialogues = append(*r.Dialogues, (*r.Dialogues)[0]) }, "dialogueIds_unique.greeting"},
		{"ダンジョン", func(r *rawschema.Raws) { *r.Dungeons = append(*r.Dungeons, (*r.Dungeons)[0]) }, "dungeonIds_unique.forest"},
		{"アイテムグループとアイテムコレクション", func(r *rawschema.Raws) { (*r.ItemCollections)[0].Id = "junk" }, "itemGroupIds_unique.junk"},
		{"間取り", func(r *rawschema.Raws) { *r.Layouts = append(*r.Layouts, (*r.Layouts)[0]) }, "layoutIds_unique.cave"},
		{"最初の段が表をまたぐ", func(r *rawschema.Raws) { (*r.TemplateStarts)[0].Id = "noise" }, "startIds_unique.noise"},
		{"中の段が表をまたぐ", func(r *rawschema.Raws) { (*r.TrailSteps)[0].Id = "widen" }, "stepIds_unique.widen"},
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
