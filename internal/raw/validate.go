package raw

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"unicode/utf8"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
)

var (
	errZoneNoBaseFacility            = errors.New("zone has no base facility")
	errMapGlyphNotSingleRune         = errors.New("map glyph is not a single rune")
	errMapGlyphDuplicate             = errors.New("map glyph is duplicated")
	errMapGlyphMissing               = errors.New("map glyph is missing")
	errDuplicateID                   = errors.New("duplicate id")
	errPlayerMemberMissing           = errors.New("no member is marked as the player")
	errPlayerMemberDuplicate         = errors.New("more than one member is marked as the player")
	errDisassemblyToolRequiredNoTool = errors.New("disassembly requires a tool but has no tool category")
	errDisassemblyWithoutHP          = errors.New("prop with disassembly has no hp and cannot be broken")
	errInteriorFlavorContentMissing  = errors.New("interior flavor content is missing")
	errBodyPartRowMissing            = errors.New("body part has no raw row")
	errBodyPartUnknown               = errors.New("body part is not defined in code")
	errConditionPartWholeBody        = errors.New("condition parts include whole body")
	errConditionNoHitPart            = errors.New("condition has no body part that can be hit")
	errAttackRangeParams             = errors.New("only ranged attack types must have all range params")
	errResistNone                    = errors.New("no proficiency can resist the none element")
	errWeaponProficiencyOwner        = errors.New("weapon proficiency is owned by another skill")
	errSpellTurnsReversed            = errors.New("weather spell turns min exceeds max")
	errNeedStageNoZero               = errors.New("need stages have no stage from zero")
	errEnumRowMissing                = errors.New("enum value has no raw row")
	errTableMissing                  = errors.New("table required by code is missing")
	errSiteWeightZero                = errors.New("site types have no weight for homes or shops")
)

// validateRules は行の間の決まりのうち、CUE のスキーマで書かないものを検証する。
func validateRules(raws rawschema.Raws) error {
	validators := []func(rawschema.Raws) error{
		validateDisassemblies,
		validateZoneBaseFacilities,
		validateFlavorContent,
		validateMapGlyphReferences,
		validateConditions,
		validateInteriorProfile,
		validateWeathers,
		validateTileEffects,
		validateNeedStages,
	}
	for _, v := range validators {
		if err := v(raws); err != nil {
			return err
		}
	}
	return nil
}

// validateZoneBaseFacilities は施設を持つ raw で、全地区に基本施設があることを検証する。
func validateZoneBaseFacilities(raws rawschema.Raws) error {
	facilities := PtrSlice(raws.Facilities)
	if len(facilities) == 0 {
		return nil
	}
	baseSpan := int32(consts.UrbanMinSpan)
	zoneHasBase := make(map[string]bool)
	for i := range facilities {
		for _, z := range facilities[i].Zones {
			if z.MinSpan <= baseSpan {
				zoneHasBase[string(z.Zone)] = true
			}
		}
	}
	for _, zone := range enumStrings(rawschema.ZoneValues()) {
		if !zoneHasBase[zone] {
			return fmt.Errorf("zone %q has no facility with minSpan<=%d: %w", zone, baseSpan, errZoneNoBaseFacility)
		}
	}
	return nil
}

// validateFlavorContent は interior を積む raw が flavorContent を持つことを検証する。
// flavor は Go が全室へ引く必須レイヤで、未設定なら生成時に落ちる。interior を使わない部分的な Raws は素通しする
func validateFlavorContent(raws rawschema.Raws) error {
	if len(PtrSlice(raws.InteriorContents)) > 0 && raws.FlavorContent == nil {
		return fmt.Errorf("flavorContent: %w", errInteriorFlavorContentMissing)
	}
	return nil
}

// validateDisassemblies は prop の分解定義が、道具が必須なら分類を持ち、prop が HP を持つことを検証する
func validateDisassemblies(raws rawschema.Raws) error {
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
	}
	return nil
}

// validateMapGlyphReferences は地図記号が1文字で重複せず、全施設とランドマークと集落が記号を持つことを検証する。
func validateMapGlyphReferences(raws rawschema.Raws) error {
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

	owners := []struct {
		kind string
		ids  []string
	}{
		{"facility", rowIDs(PtrSlice(raws.Facilities), func(f rawschema.Facility) string { return f.Id })},
		{"landmark", rowIDs(PtrSlice(raws.Landmarks), func(l rawschema.Landmark) string { return l.Id })},
		{"settlement", rowIDs(PtrSlice(raws.Settlements), func(s rawschema.Settlement) string { return s.Id })},
	}
	for _, o := range owners {
		for _, id := range o.ids {
			if _, ok := ids[id]; !ok {
				return fmt.Errorf("%s %q: %w", o.kind, id, errMapGlyphMissing)
			}
		}
	}
	var chunks []string
	for _, g := range glyphs {
		if g.Chunk != nil {
			chunks = append(chunks, string(*g.Chunk))
		}
	}
	return validateEnumRows("GlyphChunk", enumStrings(rawschema.GlyphChunkValues()), chunks)
}

// validateConditions は部位の行が Go の部位にあり、不調の部位が全身を含まず、怪我の抽選の対象が空にならないことを検証する
func validateConditions(raws rawschema.Raws) error {
	bodyParts := PtrSlice(raws.BodyParts)
	hitWeights := make(map[string]float64, len(bodyParts))
	for _, b := range bodyParts {
		if _, ok := gc.BodyPartByID(b.Id); !ok {
			return fmt.Errorf("body part %q: %w", b.Id, errBodyPartUnknown)
		}
		hitWeights[b.Id] = b.HitWeight
	}
	for _, c := range PtrSlice(raws.Conditions) {
		if err := validateConditionParts(c, hitWeights); err != nil {
			return err
		}
	}
	return nil
}

// validateConditionParts は不調の起こりうる部位が全身を含まず、命中の重みが正の部位を1つ以上持つことを検証する。
// parts を省いた不調は全部位が対象になる
func validateConditionParts(c rawschema.Condition, hitWeights map[string]float64) error {
	parts := PtrSlice(c.Parts)
	if slices.Contains(parts, gc.BodyPartWholeBody.ID()) {
		return fmt.Errorf("condition %q: %w", c.Id, errConditionPartWholeBody)
	}
	candidates := parts
	if len(parts) == 0 {
		candidates = slices.Collect(maps.Keys(hitWeights))
	}
	for _, id := range candidates {
		if hitWeights[id] > 0 {
			return nil
		}
	}
	return fmt.Errorf("condition %q: %w", c.Id, errConditionNoHitPart)
}

// validateCodeRows は Go が名前で参照する行と、読み込みに要る表が raw にあることを検証する。
// 一部だけの raws を組むテストを妨げないよう、LoadFromFile だけが呼ぶ
func validateCodeRows(raws rawschema.Raws) error {
	players := 0
	members := PtrSlice(raws.Members)
	for i := range members {
		if members[i].Player != nil && *members[i].Player {
			players++
		}
	}
	switch {
	case players == 0:
		return errPlayerMemberMissing
	case players > 1:
		return errPlayerMemberDuplicate
	}

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
	if err := validateEnumRows("ConditionOnset", enumStrings(rawschema.ConditionOnsetValues()), onsets); err != nil {
		return err
	}

	attackTypes := PtrSlice(raws.AttackTypes)
	attackTypeIDs := make([]string, 0, len(attackTypes))
	for i := range attackTypes {
		a := &attackTypes[i]
		hasAll := a.OptimalRange != nil && a.MaxRange != nil && a.PenaltyPerTile != nil
		hasAny := a.OptimalRange != nil || a.MaxRange != nil || a.PenaltyPerTile != nil
		if a.Range == rawschema.AttackRangeRanged && !hasAll || a.Range != rawschema.AttackRangeRanged && hasAny {
			return fmt.Errorf("attack type %q: %w", a.Id, errAttackRangeParams)
		}
		attackTypeIDs = append(attackTypeIDs, string(a.Id))
	}
	if err := validateEnumRows("AttackCategory", enumStrings(rawschema.AttackCategoryValues()), attackTypeIDs); err != nil {
		return err
	}
	if err := validateSkillCodeRows(raws); err != nil {
		return err
	}
	enumRows := []struct {
		schema string
		values []string
		ids    []string
	}{
		{"FreshnessStageId", enumStrings(rawschema.FreshnessStageIdValues()), rowIDs(PtrSlice(raws.FreshnessStages), func(r rawschema.FreshnessStage) rawschema.FreshnessStageId { return r.Id })},
		{"Material", enumStrings(rawschema.MaterialValues()), rowIDs(PtrSlice(raws.Materials), func(r rawschema.MaterialDef) rawschema.Material { return r.Id })},
		{"ToolCategory", enumStrings(rawschema.ToolCategoryValues()), rowIDs(PtrSlice(raws.ToolCategories), func(r rawschema.ToolCategoryDef) rawschema.ToolCategory { return r.Id })},
		{"TimeOfDayId", enumStrings(rawschema.TimeOfDayIdValues()), rowIDs(PtrSlice(raws.TimesOfDay), func(r rawschema.TimeOfDay) rawschema.TimeOfDayId { return r.Id })},
		{"SeasonId", enumStrings(rawschema.SeasonIdValues()), rowIDs(PtrSlice(raws.Seasons), func(r rawschema.Season) rawschema.SeasonId { return r.Id })},
	}
	for _, e := range enumRows {
		if err := validateEnumRows(e.schema, e.values, e.ids); err != nil {
			return err
		}
	}

	required := []struct {
		name    string
		present bool
	}{
		{"interiorProfile", raws.InteriorProfile != nil},
		{"weathers", len(PtrSlice(raws.Weathers)) > 0},
		{"shelterEffects", len(PtrSlice(raws.ShelterEffects)) > 0},
		{"waterEffects", len(PtrSlice(raws.WaterEffects)) > 0},
		{"foliageEffects", len(PtrSlice(raws.FoliageEffects)) > 0},
		{"hungerStages", len(PtrSlice(raws.HungerStages)) > 0},
		{"fatigueStages", len(PtrSlice(raws.FatigueStages)) > 0},
		{"dungeons", len(PtrSlice(raws.Dungeons)) > 0},
		{"settlements", len(PtrSlice(raws.Settlements)) > 0},
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
func validateSkillCodeRows(raws rawschema.Raws) error {
	skills := PtrSlice(raws.Skills)
	skillIDs := make([]string, 0, len(skills))
	var proficiencyIDs []string
	owners := make(map[rawschema.ProficiencyId]rawschema.SkillId)
	for i := range skills {
		id := skills[i].Id
		skillIDs = append(skillIDs, string(id))
		for _, p := range PtrSlice(skills[i].Proficiencies) {
			proficiencyIDs = append(proficiencyIDs, string(p.Id))
			owners[p.Id] = id
		}
	}
	if err := validateEnumRows("SkillId", enumStrings(rawschema.SkillIdValues()), skillIDs); err != nil {
		return err
	}
	if err := validateEnumRows("ProficiencyId", enumStrings(rawschema.ProficiencyIdValues()), proficiencyIDs); err != nil {
		return err
	}
	if err := validateResists(skills); err != nil {
		return err
	}
	attackTypes := PtrSlice(raws.AttackTypes)
	for i := range attackTypes {
		at := &attackTypes[i]
		for _, key := range []rawschema.ProficiencyId{at.Damage, at.Accuracy} {
			if owners[key] != at.Skill {
				return fmt.Errorf("attack type %q proficiency %q owned by %q: %w", at.Id, key, owners[key], errWeaponProficiencyOwner)
			}
		}
	}
	return nil
}

// validateResists は無属性を除く攻撃属性それぞれを、ちょうど1つの熟練の効果倍率が軽減することを検証する
func validateResists(skills []rawschema.Skill) error {
	// 無属性は軽減の対象外なので、enum の網羅の検証では持っているものとして数える
	elements := []string{string(rawschema.ElementNONE)}
	for i := range skills {
		for _, p := range PtrSlice(skills[i].Proficiencies) {
			if p.Resists == nil {
				continue
			}
			if *p.Resists == rawschema.ElementNONE {
				return fmt.Errorf("proficiency %q: %w", p.Id, errResistNone)
			}
			elements = append(elements, string(*p.Resists))
		}
	}
	return validateEnumRows("Element", enumStrings(rawschema.ElementValues()), elements)
}

// validateInteriorProfile は生活感の抽選表と敷地の類型が enum の全値を1回ずつ持ち、
// 敷地の類型の重みが民家と店のどちらでも1以上あることを検証する
func validateInteriorProfile(raws rawschema.Raws) error {
	p := raws.InteriorProfile
	if p == nil {
		return nil
	}
	enumRows := []struct {
		schema string
		values []string
		ids    []string
	}{
		{"BuildingDamage", enumStrings(rawschema.BuildingDamageValues()), rowIDs(p.Damages, func(c rawschema.DamageChoice) rawschema.BuildingDamage { return c.Id })},
		{"BuildingClutter", enumStrings(rawschema.BuildingClutterValues()), rowIDs(p.Clutters, func(c rawschema.ClutterChoice) rawschema.BuildingClutter { return c.Id })},
		{"BuildingVariant", enumStrings(rawschema.BuildingVariantValues()), rowIDs(p.Variants, func(c rawschema.VariantChoice) rawschema.BuildingVariant { return c.Id })},
		{"SiteType", enumStrings(rawschema.SiteTypeValues()), rowIDs(p.Sites, func(c rawschema.SiteChoice) rawschema.SiteType { return c.Id })},
	}
	for _, e := range enumRows {
		if err := validateEnumRows(e.schema, e.values, e.ids); err != nil {
			return err
		}
	}
	var home, shop int32
	for _, s := range p.Sites {
		home += s.HomeWeight
		shop += s.ShopWeight
	}
	if home == 0 || shop == 0 {
		return errSiteWeightZero
	}
	return nil
}

// rowIDs は行それぞれの enum の id を文字列にして返す
func rowIDs[R any, ID ~string](rows []R, id func(R) ID) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, string(id(r)))
	}
	return out
}

// validateEnumRows は enum の値 values それぞれを ids がちょうど1回ずつ持つことを検証する
func validateEnumRows(name string, values, ids []string) error {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			return fmt.Errorf("%s %q: %w", name, id, errDuplicateID)
		}
		seen[id] = struct{}{}
	}
	for _, v := range values {
		if _, ok := seen[v]; !ok {
			return fmt.Errorf("%s %q: %w", name, v, errEnumRowMissing)
		}
	}
	return nil
}

// enumStrings は enum の値を文字列にして返す
func enumStrings[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

// validateWeathers は天候の行が enum の全値を1回ずつ持ち、各行が季節の全値の重みを持ち、スペルの下限が上限以下であることを検証する
func validateWeathers(raws rawschema.Raws) error {
	weathers := PtrSlice(raws.Weathers)
	if len(weathers) == 0 {
		return nil
	}
	for _, w := range weathers {
		if err := validateEnumRows("SeasonId", enumStrings(rawschema.SeasonIdValues()), rowIDs(w.Seasons, func(sw rawschema.WeatherSeasonWeight) rawschema.SeasonId { return sw.Id })); err != nil {
			return fmt.Errorf("weather %q: %w", w.Id, err)
		}
		if w.SpellTurnsMin > w.SpellTurnsMax {
			return fmt.Errorf("weather %q: %w", w.Id, errSpellTurnsReversed)
		}
	}
	return validateEnumRows("WeatherId", enumStrings(rawschema.WeatherIdValues()), rowIDs(weathers, func(w rawschema.Weather) rawschema.WeatherId { return w.Id }))
}

// validateTileEffects は囲われ、水、植生の行が、それぞれ enum の全値を1回ずつ持つことを検証する
func validateTileEffects(raws rawschema.Raws) error {
	if shelters := PtrSlice(raws.ShelterEffects); len(shelters) > 0 {
		if err := validateEnumRows("Shelter", enumStrings(rawschema.ShelterValues()), rowIDs(shelters, func(s rawschema.ShelterEffect) rawschema.Shelter { return s.Id })); err != nil {
			return err
		}
	}
	if water := PtrSlice(raws.WaterEffects); len(water) > 0 {
		if err := validateEnumRows("WaterExposure", enumStrings(rawschema.WaterExposureValues()), rowIDs(water, func(w rawschema.WaterEffect) rawschema.WaterExposure { return w.Id })); err != nil {
			return err
		}
	}
	if foliage := PtrSlice(raws.FoliageEffects); len(foliage) > 0 {
		if err := validateEnumRows("Foliage", enumStrings(rawschema.FoliageValues()), rowIDs(foliage, func(f rawschema.FoliageEffect) rawschema.Foliage { return f.Id })); err != nil {
			return err
		}
	}
	return nil
}

// validateNeedStages は空腹と疲労の段階が enum の全値を1回ずつ持ち、空腹と疲労と鮮度の段階が0から始まる段階を持つことを検証する。
// 0の段階が無いと、どの段階にも当たらない割合や劣化量が生まれる
func validateNeedStages(raws rawschema.Raws) error {
	if hunger := PtrSlice(raws.HungerStages); len(hunger) > 0 {
		if err := validateEnumRows("HungerStageId", enumStrings(rawschema.HungerStageIdValues()), rowIDs(hunger, func(h rawschema.HungerStage) rawschema.HungerStageId { return h.Id })); err != nil {
			return err
		}
		if !slices.ContainsFunc(hunger, func(h rawschema.HungerStage) bool { return h.From == 0 }) {
			return fmt.Errorf("hunger: %w", errNeedStageNoZero)
		}
	}
	if fatigue := PtrSlice(raws.FatigueStages); len(fatigue) > 0 {
		if err := validateEnumRows("FatigueStageId", enumStrings(rawschema.FatigueStageIdValues()), rowIDs(fatigue, func(f rawschema.FatigueStage) rawschema.FatigueStageId { return f.Id })); err != nil {
			return err
		}
		if !slices.ContainsFunc(fatigue, func(f rawschema.FatigueStage) bool { return f.From == 0 }) {
			return fmt.Errorf("fatigue: %w", errNeedStageNoZero)
		}
	}
	if freshness := PtrSlice(raws.FreshnessStages); len(freshness) > 0 {
		if !slices.ContainsFunc(freshness, func(f rawschema.FreshnessStage) bool { return f.From == 0 }) {
			return fmt.Errorf("freshness: %w", errNeedStageNoZero)
		}
	}
	return nil
}
