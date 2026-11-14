package raw

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"unicode/utf8"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/rawschema"
)

var (
	errZoneNoBaseFacility            = errors.New("zone has no base facility")
	errDistrictSizeOutOfRegion       = errors.New("district size does not fit in its region")
	errDistrictTooShort              = errors.New("district is shorter than its layers")
	errDistrictLayers                = errors.New("district layers do not fit its pattern")
	errMapGlyphNotSingleRune         = errors.New("map glyph is not a single rune")
	errMapGlyphDuplicate             = errors.New("map glyph is duplicated")
	errMapGlyphMissing               = errors.New("map glyph is missing")
	errDuplicateID                   = errors.New("duplicate id")
	errPlayerMemberMissing           = errors.New("no member is marked as the player")
	errPlayerMemberDuplicate         = errors.New("more than one member is marked as the player")
	errDisassemblyToolRequiredNoTool = errors.New("disassembly requires a tool but has no tool category")
	errDisassemblyWithoutHP          = errors.New("prop with disassembly has no hp and cannot be broken")
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
)

// validateRules は行の間の決まりのうち、CUE のスキーマで書かないものを検証する。
func validateRules(raws rawschema.Raws) error {
	validators := []func(rawschema.Raws) error{
		validateDisassemblies,
		validateDistricts,
		validateZoneBaseFacilities,
		validateMapGlyphReferences,
		validateConditions,
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

// validateDistricts は区域の層が並べ方に合い、全ての大きさの候補が自分のリージョンの縦の範囲 spacing - separation に収まることを検証する。
// 区域がリージョンからはみ出すと、チャンクを含む区域を1つのリージョンだけ見て決められなくなる。
// rows の区域は、行の数を書いた層を並べたうえで残りの行を占める層に1行以上を残す
func validateDistricts(raws rawschema.Raws) error {
	for _, d := range PtrSlice(raws.Districts) {
		if err := validateDistrictLayers(d); err != nil {
			return err
		}
		fixed := 0
		for _, l := range d.Layers {
			fixed += deref(l.Rows)
		}
		for _, size := range d.Sizes {
			if size.H > d.Spacing-d.Separation {
				return fmt.Errorf("district %q size %dx%d, spacing %d, separation %d: %w", d.Id, size.W, size.H, d.Spacing, d.Separation, errDistrictSizeOutOfRegion)
			}
			if d.Pattern == rawschema.DistrictPatternRows && size.H < fixed+1 {
				return fmt.Errorf("district %q size %dx%d, layer rows %d: %w", d.Id, size.W, size.H, fixed, errDistrictTooShort)
			}
		}
	}
	return nil
}

// validateDistrictLayers は区域 d の層の数と行の数の書き方が、並べ方に合うことを検証する。
// rings は外側と中心の2層で行の数を書かず、rows は2層以上で行の数を省いた層がちょうど1つある
func validateDistrictLayers(d rawschema.District) error {
	rest := 0
	for _, l := range d.Layers {
		if l.Rows == nil {
			rest++
		}
	}
	switch d.Pattern {
	case rawschema.DistrictPatternRings:
		if len(d.Layers) != 2 || rest != len(d.Layers) {
			return fmt.Errorf("district %q rings has %d layers, %d with rows: %w", d.Id, len(d.Layers), len(d.Layers)-rest, errDistrictLayers)
		}
	case rawschema.DistrictPatternRows:
		if len(d.Layers) < 2 || rest != 1 {
			return fmt.Errorf("district %q rows has %d layers, %d without rows: %w", d.Id, len(d.Layers), rest, errDistrictLayers)
		}
	}
	return nil
}

// DistrictZones は区域 d が割り振る地区を、層の順に重複なく返す
func DistrictZones(d rawschema.District) []rawschema.Zone {
	var out []rawschema.Zone
	for _, l := range d.Layers {
		for _, z := range l.Zones {
			if !slices.Contains(out, z) {
				out = append(out, z)
			}
		}
	}
	return out
}

// validateZoneBaseFacilities は、どの区域でも一番小さい大きさのときに、割り振る全ての地区に建てられる施設があることを検証する。
// 施設は区域の長い方の辺が minSpan 以上のときだけ建つので、一番小さい大きさの辺で足切りを越える施設が要る
func validateZoneBaseFacilities(raws rawschema.Raws) error {
	facilities := PtrSlice(raws.Facilities)
	for _, d := range PtrSlice(raws.Districts) {
		baseSpan := 0
		for _, size := range d.Sizes {
			if span := max(size.W, size.H); baseSpan == 0 || span < baseSpan {
				baseSpan = span
			}
		}
		for _, zone := range DistrictZones(d) {
			hasBase := slices.ContainsFunc(facilities, func(f rawschema.Facility) bool {
				// 目玉の施設は区域に1つだけで、残りのチャンクの抽選からは外す
				if d.Landmark != nil && f.Id == *d.Landmark {
					return false
				}
				return slices.ContainsFunc(f.Zones, func(z rawschema.FacilityZone) bool {
					return z.Zone == zone && int(z.MinSpan) <= baseSpan
				})
			})
			if !hasBase {
				return fmt.Errorf("district %q zone %q has no facility with minSpan<=%d: %w", d.Id, zone, baseSpan, errZoneNoBaseFacility)
			}
		}
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
