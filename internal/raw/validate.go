package raw

import (
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

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
	errInteriorContentRefUndefined    = errors.New("interior recipe references undefined content")
	errInteriorContentDuplicateID     = errors.New("interior content has duplicate id")
	errInteriorFlavorContentMissing   = errors.New("interior flavor content is missing")
)

// ValidateRaws はoapi.RawsをOpenAPIスキーマの VisitJSON で一括検証する
func ValidateRaws(raws oapi.Raws) error {
	spec, err := oapi.GetSpec()
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
	return validateCommandTableWeaponReferences(raws)
}

// validateItemTableReferences はアイテムテーブルの参照グループ id がアイテムグループ定義に存在することを検証する。
// 空文字は参照なしとして扱い、非空の参照だけを検証する
func validateItemTableReferences(raws oapi.Raws) error {
	groups := PtrSlice(raws.ItemGroups)
	groupNames := make(map[string]struct{}, len(groups))
	for i := range groups {
		groupNames[groups[i].Id] = struct{}{}
	}

	itemTables := PtrSlice(raws.ItemTables)
	for i := range itemTables {
		for _, entry := range itemTables[i].Entries {
			if entry.Id == "" {
				continue
			}
			if _, ok := groupNames[entry.Id]; !ok {
				return fmt.Errorf("item table %q references group %q: %w", itemTables[i].Name, entry.Id, errItemTableRefUndefinedGroup)
			}
		}
	}
	return nil
}

// validateItemGroupReferences はアイテムグループの参照アイテム id がアイテム定義に存在することを検証する
func validateItemGroupReferences(raws oapi.Raws) error {
	items := PtrSlice(raws.Items)
	itemNames := make(map[string]struct{}, len(items))
	for i := range items {
		itemNames[items[i].Id] = struct{}{}
	}

	groups := PtrSlice(raws.ItemGroups)
	for i := range groups {
		for _, entry := range groups[i].Entries {
			if entry.Id == "" {
				continue
			}
			if _, ok := itemNames[entry.Id]; !ok {
				return fmt.Errorf("item group %q references item %q: %w", groups[i].Name, entry.Id, errItemGroupRefUndefinedItem)
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
			if entry.Id == "" {
				continue
			}
			if _, ok := memberNames[entry.Id]; !ok {
				return fmt.Errorf("enemy table %q references enemy %q: %w", enemyTables[i].Name, entry.Id, errEnemyTableRefUndefinedEnemy)
			}
		}
	}
	return nil
}

// validateFeatureUniqueIDs は地物の各表で id が重複しないことを検証する。重複すると id で引く行と全行を
// 走査する導出とで別の行を見て食い違う。
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

// validateUniqueIDs は rows の id が重複しないことを検証する。kind はエラーに出す表の名前。
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

// validateFacilityReferences は施設行の整合をロード時に検証する。
//   - enemyTable が enemyTables に存在する。
//   - facilityContents/facilityRooms の facility キーが facilities の id に存在する。
//   - Zone enum の全地区に minSpan<=UrbanMinSpan の施設が最低1つある。重みの正はスキーマが課すので、無いと
//     規模 gate で候補が空になり抽選が panic する条件をこれで塞ぐ。
//
// planner キーと実装の一致は raw から interior への循環を避け、interior の TestPlanners で固定する。
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

	// 施設0件の部分的な Raws は市街地生成を駆動しないので素通しする
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

// validateLandmarkReferences はランドマークの prop の参照先が props に実在することを検証する。drawer キーと
// 実装の一致は overworld の TestDrawers で固定する。
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
			if entry.Weapon == "" {
				continue
			}
			if _, ok := itemNames[entry.Weapon]; !ok {
				return fmt.Errorf("command table %q references weapon %q: %w", commandTables[i].Name, entry.Weapon, errCommandTableRefUndefinedWeapon)
			}
		}
	}
	return nil
}

// validateDisassemblyReferences は分解定義の産出名がアイテム定義に存在することを検証する
func validateDisassemblyReferences(raws oapi.Raws) error {
	items := PtrSlice(raws.Items)
	itemNames := make(map[string]struct{}, len(items))
	for i := range items {
		itemNames[items[i].Id] = struct{}{}
	}

	check := func(ownerKind string, ownerName string, def *oapi.Disassembly) error {
		if def == nil {
			return nil
		}
		for _, y := range def.Yields {
			if _, ok := itemNames[y.Id]; !ok {
				return fmt.Errorf("%s %q disassembly yield %q: %w", ownerKind, ownerName, y.Id, errDisassemblyYieldUndefined)
			}
		}
		if def.Bonus == nil {
			return nil
		}
		for _, b := range *def.Bonus {
			if _, ok := itemNames[b.Id]; !ok {
				return fmt.Errorf("%s %q disassembly bonus %q: %w", ownerKind, ownerName, b.Id, errDisassemblyBonusUndefined)
			}
		}
		return nil
	}

	props := PtrSlice(raws.Props)
	for i := range props {
		if err := check("prop", props[i].Name, props[i].Disassembly); err != nil {
			return err
		}
	}
	for i := range items {
		if err := check("item", items[i].Name, items[i].Disassembly); err != nil {
			return err
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
			// 空文字はドロップなしを意味する正規の値
			if entry.Material == "" {
				continue
			}
			if _, ok := itemNames[entry.Material]; !ok {
				return fmt.Errorf("drop table %q material %q: %w", dropTables[i].Name, entry.Material, errDropTableMaterialUndefined)
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

// validateScatterZoneReferences は散布ゾーンの prop と屋外 loot group の参照先が実在することを検証する。
// ref の空文字は「置かない」なので検証しない。
func validateScatterZoneReferences(raws oapi.Raws) error {
	props := PtrSlice(raws.Props)
	propNames := make(map[string]struct{}, len(props))
	for i := range props {
		propNames[props[i].Id] = struct{}{}
	}
	groups := PtrSlice(raws.ItemGroups)
	groupIDs := make(map[string]struct{}, len(groups))
	for i := range groups {
		groupIDs[groups[i].Id] = struct{}{}
	}

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

// validateMapGlyphReferences は地図記号が1文字で互いに重複しないこと、施設とランドマークの全 id が記号を
// 持つことを検証する。Go 側の placeType の記号の網羅は overworld のテストで固定する。
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
