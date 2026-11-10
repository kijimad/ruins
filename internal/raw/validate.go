package raw

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"unicode/utf8"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
)

// 参照整合の検証エラー。呼び出し側とテストが errors.Is で種類を同定できるよう sentinel にする。
var (
	errItemTableRefUndefinedGroup     = errors.New("item table references undefined item group")
	errItemGroupRefUndefinedItem      = errors.New("item group references undefined item")
	errEnemyTableRefUndefinedEnemy    = errors.New("enemy table references undefined enemy")
	errFacilityEnemyTableRefUndefined = errors.New("facility enemy table references undefined enemy table")
	errFacilityKeyUndefined           = errors.New("references undefined facility")
	errZoneNoBaseFacility             = errors.New("zone has no base facility")
	errContentLootTableUndefined      = errors.New("interior content references undefined loot table")
	errLandmarkPropUndefined          = errors.New("landmark references undefined prop")
	errScatterPropUndefined           = errors.New("scatter zone references undefined prop")
	errScatterLootGroupUndefined      = errors.New("scatter zone references undefined item group")
	errMapGlyphNotSingleRune          = errors.New("map glyph is not a single rune")
	errMapGlyphDuplicate              = errors.New("map glyph is duplicated")
	errMapGlyphMissing                = errors.New("map glyph is missing")
	errDuplicateID                    = errors.New("duplicate id")
	errCommandTableRefUndefinedWeapon = errors.New("command table references undefined weapon")
	errDropTableMaterialUndefined     = errors.New("drop table references undefined material")
	errMemberDropTableUndefined       = errors.New("member references undefined drop table")
	errMemberCommandTableUndefined    = errors.New("member references undefined command table")
	errDisassemblyYieldUndefined      = errors.New("disassembly yield references undefined item")
	errDisassemblyBonusUndefined      = errors.New("disassembly bonus references undefined item")
	errDisassemblyToolRequiredNoTool  = errors.New("disassembly requires a tool but has no tool category")
	errDisassemblyWithoutHP           = errors.New("prop with disassembly has no hp and cannot be broken")
	errInteriorContentRefUndefined    = errors.New("interior recipe references undefined content")
	errInteriorContentDuplicateID     = errors.New("interior content has duplicate id")
	errInteriorFlavorContentMissing   = errors.New("interior flavor content is missing")
	errItemGroupIDDuplicated          = errors.New("item group and item collection share an id")
	errBodyPartRowMissing             = errors.New("body part has no raw row")
	errBodyPartUnknown                = errors.New("body part is not defined in code")
	errConditionPartUndefined         = errors.New("condition references undefined body part")
	errConditionPartWholeBody         = errors.New("condition parts include whole body")
	errConditionNoHitPart             = errors.New("condition has no body part that can be hit")
	errRemedyConditionUndefined       = errors.New("remedy treats undefined condition")
	errAttackTypeInjuryUndefined      = errors.New("attack type references undefined condition")
	errAttackTypeRowMissing           = errors.New("attack type has no raw row")
	errSkillCategoryUndefined         = errors.New("skill references undefined category")
	errBookSkillUndefined             = errors.New("book references undefined skill")
	errProfessionSkillUndefined       = errors.New("profession references undefined skill")
	errWeaponProficiencyOwner         = errors.New("weapon proficiency is owned by another skill")
	errInteriorRefPropUndefined       = errors.New("interior ref references undefined prop")
	errInteriorRefGroupUndefined      = errors.New("interior ref references undefined item group")
	errHeroRefUndefined               = errors.New("hero ref is not an interior ref with a prop")
	errSpellTurnsReversed             = errors.New("weather spell turns min exceeds max")
	errNeedStageNoZero                = errors.New("need stages have no stage from zero")
	errEnumRowMissing                 = errors.New("enum value has no raw row")
	errTableMissing                   = errors.New("table required by code is missing")
)

// loadSpec は埋め込みの OpenAPI スキーマを1度だけパースして共有する。
var loadSpec = sync.OnceValues(oapi.GetSpec)

// ValidateRaws はoapi.RawsをOpenAPIスキーマの VisitJSON で一括検証する
func ValidateRaws(raws oapi.Raws) error {
	spec, err := loadSpec()
	if err != nil {
		return fmt.Errorf("failed to load OpenAPI schema: %w", err)
	}

	schemaRef, ok := spec.Components.Schemas["Raws"]
	if !ok {
		return fmt.Errorf("cannot find Raws component in OpenAPI schema")
	}
	schema := schemaRef.Value
	if schema == nil {
		return fmt.Errorf("got nil Raws schema value")
	}

	jsonBytes, err := json.Marshal(raws)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	var jsonData any
	if err := json.Unmarshal(jsonBytes, &jsonData); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if err := schema.VisitJSON(jsonData); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	return nil
}

// ValidateReferences は定義間の名前参照の整合を検証する。
// スキーマ検証は名前の参照整合を見ないため、ここで補う。実行時の解決失敗を
// ロード時エラーに前倒しする
func ValidateReferences(raws oapi.Raws) error {
	if err := validateDisassemblyReferences(raws); err != nil {
		return err
	}
	if err := validateDropTableReferences(raws); err != nil {
		return err
	}
	if err := validateCommandTableReferences(raws); err != nil {
		return err
	}
	if err := validateItemTableReferences(raws); err != nil {
		return err
	}
	if err := validateItemGroupReferences(raws); err != nil {
		return err
	}
	if err := validateEnemyTableReferences(raws); err != nil {
		return err
	}
	if err := validateFeatureUniqueIDs(raws); err != nil {
		return err
	}
	if err := validateFacilityReferences(raws); err != nil {
		return err
	}
	if err := validateContentLootTables(raws); err != nil {
		return err
	}
	if err := validateInteriorContentReferences(raws); err != nil {
		return err
	}
	if err := validateLandmarkReferences(raws); err != nil {
		return err
	}
	if err := validateScatterZoneReferences(raws); err != nil {
		return err
	}
	if err := validateMapGlyphReferences(raws); err != nil {
		return err
	}
	if err := validateConditions(raws); err != nil {
		return err
	}
	if err := validateSkills(raws); err != nil {
		return err
	}
	if err := validateInteriorRefs(raws); err != nil {
		return err
	}
	if err := validateWeathers(raws); err != nil {
		return err
	}
	if err := validateNeedStages(raws); err != nil {
		return err
	}
	return validateCommandTableWeaponReferences(raws)
}

// itemGroupIDs はアイテムグループとアイテムコレクションの id の集合を返す。どちらも同じ id 空間で参照される。
func itemGroupIDs(raws oapi.Raws) map[string]struct{} {
	ids := make(map[string]struct{})
	for _, g := range PtrSlice(raws.ItemGroups) {
		ids[g.Id] = struct{}{}
	}
	for _, c := range PtrSlice(raws.ItemCollections) {
		ids[c.Id] = struct{}{}
	}
	return ids
}

// validateItemTableReferences はアイテムテーブルの参照 id がアイテムグループかアイテムコレクションに存在することを検証する。
func validateItemTableReferences(raws oapi.Raws) error {
	groupNames := itemGroupIDs(raws)

	itemTables := PtrSlice(raws.ItemTables)
	for i := range itemTables {
		for _, entry := range itemTables[i].Entries {
			if _, ok := groupNames[entry.Id]; !ok {
				return fmt.Errorf("item table %q references group %q: %w", itemTables[i].Name, entry.Id, errItemTableRefUndefinedGroup)
			}
		}
	}
	return nil
}

// validateItemGroupReferences はアイテムグループとアイテムコレクションの参照アイテム id がアイテム定義に存在すること、
// 両者の id が重ならないことを検証する
func validateItemGroupReferences(raws oapi.Raws) error {
	items := PtrSlice(raws.Items)
	itemNames := make(map[string]struct{}, len(items))
	for i := range items {
		itemNames[items[i].Id] = struct{}{}
	}

	groups := PtrSlice(raws.ItemGroups)
	groupIDs := make(map[string]struct{}, len(groups))
	for i := range groups {
		groupIDs[groups[i].Id] = struct{}{}
		for _, entry := range groups[i].Entries {
			if _, ok := itemNames[entry.Id]; !ok {
				return fmt.Errorf("item group %q references item %q: %w", groups[i].Name, entry.Id, errItemGroupRefUndefinedItem)
			}
		}
	}
	collections := PtrSlice(raws.ItemCollections)
	for i := range collections {
		if _, ok := groupIDs[collections[i].Id]; ok {
			return fmt.Errorf("%q: %w", collections[i].Id, errItemGroupIDDuplicated)
		}
		for _, entry := range collections[i].Entries {
			if _, ok := itemNames[entry.Id]; !ok {
				return fmt.Errorf("item collection %q references item %q: %w", collections[i].Name, entry.Id, errItemGroupRefUndefinedItem)
			}
		}
	}
	return nil
}

// validateEnemyTableReferences は敵テーブルの参照敵 id がメンバー定義に存在することを検証する
func validateEnemyTableReferences(raws oapi.Raws) error {
	members := PtrSlice(raws.Members)
	memberNames := make(map[string]struct{}, len(members))
	for i := range members {
		memberNames[members[i].Id] = struct{}{}
	}

	enemyTables := PtrSlice(raws.EnemyTables)
	for i := range enemyTables {
		for _, entry := range enemyTables[i].Entries {
			if _, ok := memberNames[entry.Id]; !ok {
				return fmt.Errorf("enemy table %q references enemy %q: %w", enemyTables[i].Name, entry.Id, errEnemyTableRefUndefinedEnemy)
			}
		}
	}
	return nil
}

// validateFeatureUniqueIDs は地物の各表で id が重複しないことを検証する。
func validateFeatureUniqueIDs(raws oapi.Raws) error {
	if err := validateUniqueIDs("facility", PtrSlice(raws.Facilities), func(f oapi.Facility) string { return f.Id }); err != nil {
		return err
	}
	if err := validateUniqueIDs("landmark", PtrSlice(raws.Landmarks), func(l oapi.Landmark) string { return l.Id }); err != nil {
		return err
	}
	if err := validateUniqueIDs("scatter zone", PtrSlice(raws.ScatterZones), func(z oapi.ScatterZone) string { return z.Id }); err != nil {
		return err
	}
	return validateUniqueIDs("map glyph", PtrSlice(raws.MapGlyphs), func(g oapi.MapGlyph) string { return g.Id })
}

// validateUniqueIDs は rows の id が重複しないことを検証する。
func validateUniqueIDs[T any](kind string, rows []T, id func(T) string) error {
	seen := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		k := id(r)
		if _, ok := seen[k]; ok {
			return fmt.Errorf("%s %q: %w", kind, k, errDuplicateID)
		}
		seen[k] = struct{}{}
	}
	return nil
}

// validateFacilityReferences は施設行の参照先の実在と、全地区に基本施設があることを検証する。
func validateFacilityReferences(raws oapi.Raws) error {
	facilities := PtrSlice(raws.Facilities)
	facilityIDs := make(map[string]struct{}, len(facilities))
	for i := range facilities {
		facilityIDs[facilities[i].Id] = struct{}{}
	}

	enemyTables := PtrSlice(raws.EnemyTables)
	tableIDs := make(map[string]struct{}, len(enemyTables))
	for i := range enemyTables {
		tableIDs[enemyTables[i].Id] = struct{}{}
	}

	zones, err := SchemaEnum("Zone")
	if err != nil {
		return err
	}
	baseSpan := int32(consts.UrbanMinSpan)
	zoneHasBase := make(map[string]bool)
	for i := range facilities {
		if _, ok := tableIDs[facilities[i].EnemyTable]; !ok {
			return fmt.Errorf("facility %q references enemy table %q: %w", facilities[i].Id, facilities[i].EnemyTable, errFacilityEnemyTableRefUndefined)
		}
		for _, z := range facilities[i].Zones {
			if z.MinSpan <= baseSpan {
				zoneHasBase[string(z.Zone)] = true
			}
		}
	}

	if len(facilities) > 0 {
		for _, zone := range zones {
			if !zoneHasBase[zone] {
				return fmt.Errorf("zone %q has no facility with minSpan<=%d: %w", zone, baseSpan, errZoneNoBaseFacility)
			}
		}
	}

	for _, fc := range PtrSlice(raws.FacilityContents) {
		if _, ok := facilityIDs[fc.Facility]; !ok {
			return fmt.Errorf("facilityContents references unknown facility %q: %w", fc.Facility, errFacilityKeyUndefined)
		}
	}
	for _, fr := range PtrSlice(raws.FacilityRooms) {
		if _, ok := facilityIDs[fr.Facility]; !ok {
			return fmt.Errorf("facilityRooms references unknown facility %q: %w", fr.Facility, errFacilityKeyUndefined)
		}
	}
	return nil
}

// validateContentLootTables は内装レシピの家具と衛星が書く戦利品表が実在する itemTables であることを検証する。
func validateContentLootTables(raws oapi.Raws) error {
	tables := PtrSlice(raws.ItemTables)
	tableIDs := make(map[string]struct{}, len(tables))
	for i := range tables {
		tableIDs[tables[i].Id] = struct{}{}
	}
	check := func(content, ref string, table *oapi.EntityID) error {
		if table == nil {
			return nil
		}
		if _, ok := tableIDs[*table]; !ok {
			return fmt.Errorf("interior content %q ref %q references loot table %q: %w", content, ref, *table, errContentLootTableUndefined)
		}
		return nil
	}
	for _, ic := range PtrSlice(raws.InteriorContents) {
		for _, g := range PtrSlice(ic.Groups) {
			for _, it := range g.Items {
				if err := check(ic.Id, it.Ref, it.LootTable); err != nil {
					return err
				}
				for _, sat := range PtrSlice(it.Satellites) {
					if err := check(ic.Id, sat.Ref, sat.LootTable); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// validateLandmarkReferences はランドマークの prop の実在を検証する。
func validateLandmarkReferences(raws oapi.Raws) error {
	landmarks := PtrSlice(raws.Landmarks)
	if len(landmarks) == 0 {
		return nil
	}

	props := PtrSlice(raws.Props)
	propNames := make(map[string]struct{}, len(props))
	for i := range props {
		propNames[props[i].Id] = struct{}{}
	}

	for _, l := range landmarks {
		for _, p := range l.Props {
			if _, ok := propNames[p.Name]; !ok {
				return fmt.Errorf("landmark %q references prop %q: %w", l.Id, p.Name, errLandmarkPropUndefined)
			}
		}
	}
	return nil
}

// validateInteriorContentReferences は内装レシピの content 参照が interiorContents に存在することを検証する。
// facilityContents の変種、facilityRooms の役割別 content と fallback が指す id を、typo による空部屋の silent
// 生成を避けてロード時に前倒しで弾く。あわせて interiorContents の id 重複と、interior を積む raw での
// flavorContent 未設定を検出する。id は生成側の一意キーで重複すると取り違わる。
func validateInteriorContentReferences(raws oapi.Raws) error {
	contents := PtrSlice(raws.InteriorContents)
	contentIDs := make(map[string]struct{}, len(contents))
	for i := range contents {
		if _, dup := contentIDs[contents[i].Id]; dup {
			return fmt.Errorf("interior content %q: %w", contents[i].Id, errInteriorContentDuplicateID)
		}
		contentIDs[contents[i].Id] = struct{}{}
	}

	for _, fc := range PtrSlice(raws.FacilityContents) {
		for _, id := range fc.Variants {
			if id == "" {
				continue
			}
			if _, ok := contentIDs[id]; !ok {
				return fmt.Errorf("facility %q variant %q: %w", fc.Facility, id, errInteriorContentRefUndefined)
			}
		}
	}

	for _, fr := range PtrSlice(raws.FacilityRooms) {
		for _, r := range PtrSlice(fr.Rooms) {
			if r.Content == "" {
				continue
			}
			if _, ok := contentIDs[r.Content]; !ok {
				return fmt.Errorf("facility %q room %q content %q: %w", fr.Facility, r.Role, r.Content, errInteriorContentRefUndefined)
			}
		}
		if fr.Fallback == "" {
			continue
		}
		if _, ok := contentIDs[fr.Fallback]; !ok {
			return fmt.Errorf("facility %q fallback %q: %w", fr.Facility, fr.Fallback, errInteriorContentRefUndefined)
		}
	}

	// flavor は Go が全室へ引く必須レイヤ。interior を積む raw で未設定なら生成時に落ちるのでロード時に弾く。
	// flavorContent は id 参照でなくインラインの値を直接持つので、interiorContents との id 照合は要らず存在だけ見る。
	// interior を使わない部分的な Raws は素通しする
	if len(contents) > 0 && raws.FlavorContent == nil {
		return fmt.Errorf("flavorContent: %w", errInteriorFlavorContentMissing)
	}
	return nil
}

// validateCommandTableWeaponReferences はコマンドテーブルの参照武器名がアイテム定義に存在することを検証する。
// タイプミスは attack.go の getAttackParams が素手攻撃へ握り潰し無音で劣化するため、ロード時に前倒しで弾く。
func validateCommandTableWeaponReferences(raws oapi.Raws) error {
	items := PtrSlice(raws.Items)
	itemNames := make(map[string]struct{}, len(items))
	for i := range items {
		itemNames[items[i].Id] = struct{}{}
	}

	commandTables := PtrSlice(raws.CommandTables)
	for i := range commandTables {
		for _, entry := range commandTables[i].Entries {
			if _, ok := itemNames[entry.Id]; !ok {
				return fmt.Errorf("command table %q references weapon %q: %w", commandTables[i].Name, entry.Id, errCommandTableRefUndefinedWeapon)
			}
		}
	}
	return nil
}

// validateDisassemblyReferences は prop の分解定義の産出名、道具が必須なときの分類、prop の HP を検証する
func validateDisassemblyReferences(raws oapi.Raws) error {
	items := PtrSlice(raws.Items)
	itemNames := make(map[string]struct{}, len(items))
	for i := range items {
		itemNames[items[i].Id] = struct{}{}
	}

	props := PtrSlice(raws.Props)
	for i := range props {
		def := props[i].Disassembly
		if def == nil {
			continue
		}
		if def.ToolRequired != nil && *def.ToolRequired && def.ToolCategory == nil {
			return fmt.Errorf("prop %q: %w", props[i].Name, errDisassemblyToolRequiredNoTool)
		}
		if props[i].Hp == nil {
			return fmt.Errorf("prop %q: %w", props[i].Name, errDisassemblyWithoutHP)
		}
		for _, y := range def.Yields {
			if _, ok := itemNames[y.Id]; !ok {
				return fmt.Errorf("prop %q disassembly yield %q: %w", props[i].Name, y.Id, errDisassemblyYieldUndefined)
			}
		}
		for _, b := range PtrSlice(def.Bonus) {
			if _, ok := itemNames[b.Id]; !ok {
				return fmt.Errorf("prop %q disassembly bonus %q: %w", props[i].Name, b.Id, errDisassemblyBonusUndefined)
			}
		}
	}
	return nil
}

// validateDropTableReferences はドロップテーブルの素材 id がアイテム定義に存在すること、
// メンバーの dropTableId がテーブル定義に存在することを検証する
func validateDropTableReferences(raws oapi.Raws) error {
	items := PtrSlice(raws.Items)
	itemNames := make(map[string]struct{}, len(items))
	for i := range items {
		itemNames[items[i].Id] = struct{}{}
	}

	dropTables := PtrSlice(raws.DropTables)
	tableNames := make(map[string]struct{}, len(dropTables))
	for i := range dropTables {
		tableNames[dropTables[i].Id] = struct{}{}
		for _, entry := range dropTables[i].Entries {
			if _, ok := itemNames[entry.Id]; !ok {
				return fmt.Errorf("drop table %q material %q: %w", dropTables[i].Name, entry.Id, errDropTableMaterialUndefined)
			}
		}
	}

	members := PtrSlice(raws.Members)
	for i := range members {
		// 省略時は参照を持たない。空文字はテーブル名として不正なので存在チェックで弾く
		if members[i].DropTableId == nil {
			continue
		}
		if _, ok := tableNames[*members[i].DropTableId]; !ok {
			return fmt.Errorf("member %q drop table %q: %w", members[i].Name, *members[i].DropTableId, errMemberDropTableUndefined)
		}
	}
	return nil
}

// validateCommandTableReferences はメンバーの commandTableId がテーブル定義に存在することを検証する
func validateCommandTableReferences(raws oapi.Raws) error {
	commandTables := PtrSlice(raws.CommandTables)
	tableNames := make(map[string]struct{}, len(commandTables))
	for i := range commandTables {
		tableNames[commandTables[i].Id] = struct{}{}
	}

	members := PtrSlice(raws.Members)
	for i := range members {
		// 省略時は参照を持たない。空文字はテーブル名として不正なので存在チェックで弾く
		if members[i].CommandTableId == nil {
			continue
		}
		if _, ok := tableNames[*members[i].CommandTableId]; !ok {
			return fmt.Errorf("member %q command table %q: %w", members[i].Name, *members[i].CommandTableId, errMemberCommandTableUndefined)
		}
	}
	return nil
}

// validateScatterZoneReferences は散布ゾーンの prop と loot group の実在を検証する。
func validateScatterZoneReferences(raws oapi.Raws) error {
	props := PtrSlice(raws.Props)
	propNames := make(map[string]struct{}, len(props))
	for i := range props {
		propNames[props[i].Id] = struct{}{}
	}
	groupIDs := itemGroupIDs(raws)

	for _, z := range PtrSlice(raws.ScatterZones) {
		if _, ok := groupIDs[z.LootGroup]; !ok {
			return fmt.Errorf("scatter zone %q references item group %q: %w", z.Id, z.LootGroup, errScatterLootGroupUndefined)
		}
		for _, e := range z.Entries {
			if _, ok := propNames[e.Ref]; e.Ref != "" && !ok {
				return fmt.Errorf("scatter zone %q references prop %q: %w", z.Id, e.Ref, errScatterPropUndefined)
			}
			for _, s := range PtrSlice(e.Satellites) {
				if _, ok := propNames[s.Name]; !ok {
					return fmt.Errorf("scatter zone %q references prop %q: %w", z.Id, s.Name, errScatterPropUndefined)
				}
			}
		}
	}
	return nil
}

// validateMapGlyphReferences は地図記号が1文字で重複せず、全施設とランドマークが記号を持つことを検証する。
func validateMapGlyphReferences(raws oapi.Raws) error {
	glyphs := PtrSlice(raws.MapGlyphs)
	if len(glyphs) == 0 {
		return nil
	}

	ids := make(map[string]struct{}, len(glyphs))
	seen := make(map[string]string, len(glyphs))
	for _, g := range glyphs {
		if utf8.RuneCountInString(g.Glyph) != 1 {
			return fmt.Errorf("map glyph %q has glyph %q: %w", g.Id, g.Glyph, errMapGlyphNotSingleRune)
		}
		if other, ok := seen[g.Glyph]; ok {
			return fmt.Errorf("map glyph %q shares glyph %q with %q: %w", g.Id, g.Glyph, other, errMapGlyphDuplicate)
		}
		seen[g.Glyph] = g.Id
		ids[g.Id] = struct{}{}
	}

	for _, f := range PtrSlice(raws.Facilities) {
		if _, ok := ids[f.Id]; !ok {
			return fmt.Errorf("facility %q: %w", f.Id, errMapGlyphMissing)
		}
	}
	for _, l := range PtrSlice(raws.Landmarks) {
		if _, ok := ids[l.Id]; !ok {
			return fmt.Errorf("landmark %q: %w", l.Id, errMapGlyphMissing)
		}
	}
	return nil
}

// validateConditions は不調と部位の行と、それらへの参照を検証する。怪我の抽選の対象が空にならないことも確かめる
func validateConditions(raws oapi.Raws) error {
	bodyParts := PtrSlice(raws.BodyParts)
	conditions := PtrSlice(raws.Conditions)
	if err := validateUniqueIDs("body part", bodyParts, func(b oapi.BodyPart) string { return b.Id }); err != nil {
		return err
	}
	if err := validateUniqueIDs("condition", conditions, func(c oapi.Condition) string { return c.Id }); err != nil {
		return err
	}

	hitWeights := make(map[string]float64, len(bodyParts))
	for _, b := range bodyParts {
		if _, ok := gc.BodyPartByID(b.Id); !ok {
			return fmt.Errorf("body part %q: %w", b.Id, errBodyPartUnknown)
		}
		hitWeights[b.Id] = b.HitWeight
	}
	conditionIDs := make(map[string]struct{}, len(conditions))
	for _, c := range conditions {
		conditionIDs[c.Id] = struct{}{}
		if err := validateConditionParts(c, hitWeights); err != nil {
			return err
		}
	}

	attackTypes := PtrSlice(raws.AttackTypes)
	if err := validateUniqueIDs("attack type", attackTypes, func(a oapi.AttackType) string { return string(a.Id) }); err != nil {
		return err
	}
	for _, a := range attackTypes {
		if _, ok := conditionIDs[a.Injury]; !ok {
			return fmt.Errorf("attack type %q injury %q: %w", a.Id, a.Injury, errAttackTypeInjuryUndefined)
		}
	}

	items := PtrSlice(raws.Items)
	for i := range items {
		if items[i].Remedy == nil {
			continue
		}
		for _, t := range items[i].Remedy.Treats {
			if _, ok := conditionIDs[t]; !ok {
				return fmt.Errorf("item %q treats %q: %w", items[i].Id, t, errRemedyConditionUndefined)
			}
		}
	}
	return nil
}

// validateConditionParts は不調の起こりうる部位が実在し、全身を含まず、命中の重みが正の部位を1つ以上持つことを検証する。
// parts を省いた不調は全部位が対象になる
func validateConditionParts(c oapi.Condition, hitWeights map[string]float64) error {
	parts := PtrSlice(c.Parts)
	if len(parts) == 0 {
		for id := range hitWeights {
			parts = append(parts, id)
		}
	} else {
		for _, id := range parts {
			if _, ok := hitWeights[id]; !ok {
				return fmt.Errorf("condition %q part %q: %w", c.Id, id, errConditionPartUndefined)
			}
			if id == gc.BodyPartWholeBody.ID() {
				return fmt.Errorf("condition %q: %w", c.Id, errConditionPartWholeBody)
			}
		}
	}
	for _, id := range parts {
		if hitWeights[id] > 0 {
			return nil
		}
	}
	return fmt.Errorf("condition %q: %w", c.Id, errConditionNoHitPart)
}

// validateCodeRows は Go が名前で参照する部位、攻撃種別、スキル、熟練の効果倍率すべてに raw の行があり、
// 不調を起こす仕組みそれぞれをちょうど1つの不調が持ち、
// 生活感、天候、空腹と疲労の表があることを検証する。表の中の enum の網羅は ValidateReferences が見る
func validateCodeRows(raws oapi.Raws) error {
	bodyPartIDs := make(map[string]struct{})
	for _, b := range PtrSlice(raws.BodyParts) {
		bodyPartIDs[b.Id] = struct{}{}
	}
	for bp := range gc.BodyPartCount {
		if _, ok := bodyPartIDs[bp.ID()]; !ok {
			return fmt.Errorf("body part %q: %w", bp.ID(), errBodyPartRowMissing)
		}
	}

	var onsets []string
	for _, c := range PtrSlice(raws.Conditions) {
		if c.Onset != nil {
			onsets = append(onsets, string(*c.Onset))
		}
	}
	if err := validateEnumRows("ConditionOnset", onsets); err != nil {
		return err
	}

	attackTypeIDs := make(map[string]struct{})
	for _, a := range PtrSlice(raws.AttackTypes) {
		attackTypeIDs[string(a.Id)] = struct{}{}
	}
	for _, at := range gc.AllAttackTypes {
		if _, ok := attackTypeIDs[at.Type]; !ok {
			return fmt.Errorf("attack type %q: %w", at.Type, errAttackTypeRowMissing)
		}
	}
	if err := validateSkillCodeRows(raws); err != nil {
		return err
	}

	required := []struct {
		name    string
		present bool
	}{
		{"interiorProfile", raws.InteriorProfile != nil},
		{"weathers", len(PtrSlice(raws.Weathers)) > 0},
		{"hungerStages", len(PtrSlice(raws.HungerStages)) > 0},
		{"fatigueStages", len(PtrSlice(raws.FatigueStages)) > 0},
	}
	for _, r := range required {
		if !r.present {
			return fmt.Errorf("%s: %w", r.name, errTableMissing)
		}
	}
	return nil
}

// validateSkillCodeRows はスキーマの enum のスキルと熟練の効果倍率それぞれに raw の行がちょうど1つあり、
// 攻撃の種類のダメージと命中の効果倍率を、その攻撃の種類のスキルの行が持つことを検証する
func validateSkillCodeRows(raws oapi.Raws) error {
	skills := PtrSlice(raws.Skills)
	skillIDs := make([]string, 0, len(skills))
	var proficiencyIDs []string
	owners := make(map[oapi.ProficiencyId]oapi.SkillId)
	for i := range skills {
		id := skills[i].Id
		skillIDs = append(skillIDs, string(id))
		for _, p := range PtrSlice(skills[i].Proficiencies) {
			proficiencyIDs = append(proficiencyIDs, string(p.Id))
			owners[p.Id] = id
		}
	}
	if err := validateEnumRows("SkillId", skillIDs); err != nil {
		return err
	}
	if err := validateEnumRows("ProficiencyId", proficiencyIDs); err != nil {
		return err
	}
	for _, at := range PtrSlice(raws.AttackTypes) {
		for _, key := range []oapi.ProficiencyId{at.Damage, at.Accuracy} {
			if owners[key] != at.Skill {
				return fmt.Errorf("attack type %q proficiency %q owned by %q: %w", at.Id, key, owners[key], errWeaponProficiencyOwner)
			}
		}
	}
	return nil
}

// validateSkills はスキルとカテゴリの id が一意で、スキルのカテゴリ、本と職業のスキルが実在することを検証する。
// 熟練の効果倍率の id もスキルをまたいで一意にする
func validateSkills(raws oapi.Raws) error {
	skills := PtrSlice(raws.Skills)
	categories := PtrSlice(raws.SkillCategories)
	if err := validateUniqueIDs("skill", skills, func(s oapi.Skill) string { return string(s.Id) }); err != nil {
		return err
	}
	if err := validateUniqueIDs("skill category", categories, func(c oapi.SkillCategory) string { return c.Id }); err != nil {
		return err
	}
	var proficiencies []oapi.SkillProficiency
	for i := range skills {
		proficiencies = append(proficiencies, PtrSlice(skills[i].Proficiencies)...)
	}
	if err := validateUniqueIDs("proficiency", proficiencies, func(p oapi.SkillProficiency) string { return string(p.Id) }); err != nil {
		return err
	}

	categoryIDs := make(map[string]struct{}, len(categories))
	for _, c := range categories {
		categoryIDs[c.Id] = struct{}{}
	}
	skillIDs := make(map[string]struct{}, len(skills))
	for i := range skills {
		skillIDs[string(skills[i].Id)] = struct{}{}
		if _, ok := categoryIDs[skills[i].Category]; !ok {
			return fmt.Errorf("skill %q category %q: %w", skills[i].Id, skills[i].Category, errSkillCategoryUndefined)
		}
	}

	items := PtrSlice(raws.Items)
	for i := range items {
		if items[i].Book == nil || items[i].Book.Skill == nil {
			continue
		}
		if _, ok := skillIDs[string(items[i].Book.Skill.TargetSkill)]; !ok {
			return fmt.Errorf("item %q skill %q: %w", items[i].Id, items[i].Book.Skill.TargetSkill, errBookSkillUndefined)
		}
	}
	professions := PtrSlice(raws.Professions)
	for i := range professions {
		for _, ps := range PtrSlice(professions[i].Skills) {
			if _, ok := skillIDs[string(ps.Id)]; !ok {
				return fmt.Errorf("profession %q skill %q: %w", professions[i].Id, ps.Id, errProfessionSkillUndefined)
			}
		}
	}
	return nil
}

// validateInteriorRefs は内装の Ref の id が一意で、prop と item group が実在し、
// 生活感の抽選表が enum の全値を1回ずつ持ち、hero の Ref が prop を持つ Ref であることを検証する
func validateInteriorRefs(raws oapi.Raws) error {
	refs := PtrSlice(raws.InteriorRefs)
	if err := validateUniqueIDs("interior ref", refs, func(r oapi.InteriorRef) string { return r.Id }); err != nil {
		return err
	}
	props := PtrSlice(raws.Props)
	propIDs := make(map[string]struct{}, len(props))
	for i := range props {
		propIDs[props[i].Id] = struct{}{}
	}
	groupIDs := itemGroupIDs(raws)
	propRefs := make(map[string]struct{}, len(refs))
	for _, r := range refs {
		if r.Prop != nil {
			if _, ok := propIDs[*r.Prop]; !ok {
				return fmt.Errorf("interior ref %q prop %q: %w", r.Id, *r.Prop, errInteriorRefPropUndefined)
			}
			propRefs[r.Id] = struct{}{}
		}
		if r.ItemGroup != nil {
			if _, ok := groupIDs[*r.ItemGroup]; !ok {
				return fmt.Errorf("interior ref %q item group %q: %w", r.Id, *r.ItemGroup, errInteriorRefGroupUndefined)
			}
		}
	}

	p := raws.InteriorProfile
	if p == nil {
		return nil
	}
	ids := func(n int, id func(int) string) []string {
		out := make([]string, n)
		for i := range n {
			out[i] = id(i)
		}
		return out
	}
	enumRows := []struct {
		schema string
		ids    []string
	}{
		{"BuildingDamage", ids(len(p.Damages), func(i int) string { return string(p.Damages[i].Id) })},
		{"BuildingClutter", ids(len(p.Clutters), func(i int) string { return string(p.Clutters[i].Id) })},
		{"BuildingVariant", ids(len(p.Variants), func(i int) string { return string(p.Variants[i].Id) })},
	}
	for _, e := range enumRows {
		if err := validateEnumRows(e.schema, e.ids); err != nil {
			return err
		}
	}
	if err := validateUniqueIDs("hero", p.Heroes, func(h oapi.HeroChoice) string { return h.Id }); err != nil {
		return err
	}
	for _, h := range p.Heroes {
		for _, ref := range h.Refs {
			if _, ok := propRefs[ref]; !ok {
				return fmt.Errorf("hero %q ref %q: %w", h.Id, ref, errHeroRefUndefined)
			}
		}
	}
	return nil
}

// validateEnumRows は OpenAPI スキーマの enum の値それぞれを ids がちょうど1回ずつ持つことを検証する
func validateEnumRows(schemaName string, ids []string) error {
	if err := validateUniqueIDs(schemaName, ids, func(id string) string { return id }); err != nil {
		return err
	}
	spec, err := loadSpec()
	if err != nil {
		return fmt.Errorf("failed to load OpenAPI schema: %w", err)
	}
	ref, ok := spec.Components.Schemas[schemaName]
	if !ok || ref.Value == nil {
		return fmt.Errorf("cannot find %s component in OpenAPI schema", schemaName)
	}
	for _, v := range ref.Value.Enum {
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%s enum value %v is %T, not string", schemaName, v, v)
		}
		if !slices.Contains(ids, s) {
			return fmt.Errorf("%s %q: %w", schemaName, v, errEnumRowMissing)
		}
	}
	return nil
}

// validateWeathers は天候の行が enum の全値を1回ずつ持ち、各行が季節の全値の重みを持ち、スペルの下限が上限以下であることを検証する
func validateWeathers(raws oapi.Raws) error {
	weathers := PtrSlice(raws.Weathers)
	if len(weathers) == 0 {
		return nil
	}
	ids := make([]string, 0, len(weathers))
	for _, w := range weathers {
		ids = append(ids, string(w.Id))
		seasonIDs := make([]string, 0, len(w.Seasons))
		for _, sw := range w.Seasons {
			seasonIDs = append(seasonIDs, string(sw.Id))
		}
		if err := validateEnumRows("SeasonId", seasonIDs); err != nil {
			return fmt.Errorf("weather %q: %w", w.Id, err)
		}
		if w.SpellTurnsMin > w.SpellTurnsMax {
			return fmt.Errorf("weather %q: %w", w.Id, errSpellTurnsReversed)
		}
	}
	return validateEnumRows("WeatherId", ids)
}

// validateNeedStages は空腹と疲労の段階が enum の全値を1回ずつ持ち、割合0から始まる段階を持つことを検証する。
// 割合0の段階が無いと、どの段階にも当たらない割合が生まれる
func validateNeedStages(raws oapi.Raws) error {
	hunger := PtrSlice(raws.HungerStages)
	if len(hunger) > 0 {
		ids := make([]string, 0, len(hunger))
		froms := make([]float64, 0, len(hunger))
		for _, h := range hunger {
			ids = append(ids, string(h.Id))
			froms = append(froms, h.From)
		}
		if err := validateEnumRows("HungerStageId", ids); err != nil {
			return err
		}
		if !slices.Contains(froms, 0) {
			return fmt.Errorf("hunger: %w", errNeedStageNoZero)
		}
	}
	fatigue := PtrSlice(raws.FatigueStages)
	if len(fatigue) > 0 {
		ids := make([]string, 0, len(fatigue))
		froms := make([]float64, 0, len(fatigue))
		for _, f := range fatigue {
			ids = append(ids, string(f.Id))
			froms = append(froms, f.From)
		}
		if err := validateEnumRows("FatigueStageId", ids); err != nil {
			return err
		}
		if !slices.Contains(froms, 0) {
			return fmt.Errorf("fatigue: %w", errNeedStageNoZero)
		}
	}
	return nil
}
