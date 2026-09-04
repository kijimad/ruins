package states

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// キャラクター情報の読み取り専用タブ、能力・スキル・効果・健康・基本、の構築を担う。
// 装備タブと同じ CharacterState が持ち、装備の編集フローと1画面に統合する。

// 読み取り専用タブの ID
const (
	tabAbilities = "abilities"
	tabSkills    = "skills"
	tabEffects   = "effects"
	tabHealth    = "health"
	tabBasic     = "basic"
)

// healthEntryIndent は健康タブで症状エントリを部位カテゴリ見出しと見分けるための字下げ
const healthEntryIndent = "  "

type statusTabData struct {
	ID    string
	Label string
	Items []statusItemData
}

type statusItemData struct {
	// Label は行の表示ラベル
	Label string
	// Value は行の右に出す値。数値や倍率の文字列
	Value string
	// Modifier は能力値への補正表示。装備やバフによる増減を +N 形式で持つ
	Modifier string
	// Description は詳細モーダルやヘッダーの説明文
	Description string
	// IsHeader はカテゴリヘッダー行かどうか。真なら選択不可の見出し
	IsHeader bool
	// BodyPart は健康タブの症状エントリが属する部位
	BodyPart gc.BodyPart
	// ConditionType は健康タブの症状エントリが指す不調の種類。空なら症状でない行
	ConditionType gc.ConditionType
	// Details は詳細モーダルに表示する内訳
	Details []statusDetailRow
}

type statusDetailRow struct {
	Label string
	Value string
}

// fetchInfoTabs は能力・スキル・効果・健康・基本の読み取り専用タブを構築する
func (st *CharacterState) fetchInfoTabs(world w.World, player ecs.Entity) []statusTabData {
	professionName := ""
	if query.AliveHas(world, world.Components.Profession, player) {
		profComp := world.Components.Profession.Get(player)
		if prof, err := raw.GetProfession(world.Resources.RawMaster, profComp.ID); err == nil {
			professionName = query.T(world, prof.Name)
		}
	}

	return []statusTabData{
		{ID: tabAbilities, Label: query.T(world, "Abilities"), Items: st.createAbilityItems(world, player)},
		{ID: tabSkills, Label: query.T(world, "Skills"), Items: st.createSkillItems(world, player)},
		{ID: tabEffects, Label: query.T(world, "Effects"), Items: st.createEffectItems(world, player)},
		{ID: tabHealth, Label: query.T(world, "Health"), Items: st.createHealthItems(world, player)},
		{ID: tabBasic, Label: query.T(world, "Basic"), Items: st.createBasicItems(world, player, professionName)},
	}
}

func (st *CharacterState) createBasicItems(world w.World, playerEntity ecs.Entity, professionName string) []statusItemData {
	items := []statusItemData{}

	if professionName != "" {
		items = append(items, statusItemData{Label: query.T(world, "Profession"), Value: professionName, Description: query.T(world, "Profession")})
	}
	if query.AliveHas(world, world.Components.HP, playerEntity) {
		hp := world.Components.HP.Get(playerEntity)
		items = append(items, statusItemData{Label: "HP", Value: fmt.Sprintf("%d", hp.Max), Description: query.T(world, "HP. You die at 0")})
	}
	if query.AliveHas(world, world.Components.WeightCapacity, playerEntity) {
		cw := world.Components.WeightCapacity.Get(playerEntity)
		items = append(items, statusItemData{Label: query.T(world, "Max weight"), Value: cw.Max.KgString(), Description: query.T(world, "Maximum weight you can carry")})
	}
	if query.AliveHas(world, world.Components.Hunger, playerEntity) {
		hunger := world.Components.Hunger.Get(playerEntity)
		items = append(items, statusItemData{Label: query.T(world, "Hunger"), Value: query.T(world, hunger.GetLevel().String()), Description: query.T(world, "Hunger. High hunger hinders actions")})
	}
	return items
}

func (st *CharacterState) createAbilityItems(world w.World, playerEntity ecs.Entity) []statusItemData {
	items := []statusItemData{}
	if query.AliveHas(world, world.Components.Abilities, playerEntity) {
		abils := world.Components.Abilities.Get(playerEntity)
		items = append(items,
			statusItemData{Label: query.T(world, "Vitality"), Value: fmt.Sprintf("%d", abils.Vitality.Total), Modifier: fmt.Sprintf("(%+d)", abils.Vitality.Modifier), Description: query.T(world, "Vitality. Affects max HP and SP")},
			statusItemData{Label: query.T(world, "Strength"), Value: fmt.Sprintf("%d", abils.Strength.Total), Modifier: fmt.Sprintf("(%+d)", abils.Strength.Modifier), Description: query.T(world, "Strength. Affects melee attack damage")},
			statusItemData{Label: query.T(world, "Sensation"), Value: fmt.Sprintf("%d", abils.Sensation.Total), Modifier: fmt.Sprintf("(%+d)", abils.Sensation.Modifier), Description: query.T(world, "Sensation. Affects ranged attack damage")},
			statusItemData{Label: query.T(world, "Dexterity"), Value: fmt.Sprintf("%d", abils.Dexterity.Total), Modifier: fmt.Sprintf("(%+d)", abils.Dexterity.Modifier), Description: query.T(world, "Dexterity. Affects accuracy")},
			statusItemData{Label: query.T(world, "Agility"), Value: fmt.Sprintf("%d", abils.Agility.Total), Modifier: fmt.Sprintf("(%+d)", abils.Agility.Modifier), Description: query.T(world, "Agility. Affects evasion and action speed")},
			statusItemData{Label: query.T(world, "Defense"), Value: fmt.Sprintf("%d", abils.Defense.Total), Modifier: fmt.Sprintf("(%+d)", abils.Defense.Modifier), Description: query.T(world, "Defense. Reduces damage taken")},
		)
	}
	return items
}

func (st *CharacterState) createSkillItems(world w.World, playerEntity ecs.Entity) []statusItemData {
	items := []statusItemData{}
	if !query.AliveHas(world, world.Components.Skills, playerEntity) {
		return items
	}
	skills := world.Components.Skills.Get(playerEntity)
	for _, cat := range gc.SkillCategories {
		items = append(items, statusItemData{Label: cat.Name, IsHeader: true, Description: query.T(world, "%s category skills", cat.Name)})
		for _, id := range cat.IDs {
			s := skills.Get(id)
			expFrac := 0
			if s.Exp.Max > 0 {
				expFrac = s.Exp.Current * 1000 / s.Exp.Max
			}
			info := gc.SkillDescription(id)
			items = append(items, statusItemData{
				Label:       query.T(world, gc.SkillName(id)),
				Value:       fmt.Sprintf("%d.%03d", s.Value, expFrac),
				Description: query.T(world, info.Summary),
				Details: []statusDetailRow{
					{Label: query.T(world, "Gained by"), Value: query.T(world, info.GainedBy)},
					{Label: query.T(world, "Effect"), Value: query.T(world, info.Effect)},
				},
			})
		}
	}
	return items
}

func (st *CharacterState) createEffectItems(world w.World, playerEntity ecs.Entity) []statusItemData {
	items := []statusItemData{}
	if !world.ECS.Alive(playerEntity) {
		return items
	}
	val := func(key gc.ModifierKey) string {
		return fmt.Sprintf("%d%%", query.ModifierValue(world, playerEntity, key))
	}
	details := func(key gc.ModifierKey) []statusDetailRow {
		return sourceToDetails(world, query.ModifierSources(world, playerEntity, key))
	}
	caps := gc.HealthyCapacities()
	if world.Components.HealthStatus.Has(playerEntity) {
		caps = world.Components.HealthStatus.Get(playerEntity).Capacities()
	}

	items = append(items, statusItemData{Label: query.T(world, "Combat"), IsHeader: true, Description: query.T(world, "Combat effects")})
	for _, id := range gc.WeaponSkillIDs {
		name := query.T(world, gc.SkillName(id))
		items = append(items, statusItemData{Label: query.T(world, "%s attack power", name), Value: val(gc.WeaponDamageKey(id)), Description: query.T(world, "%s weapon damage multiplier", name), Details: details(gc.WeaponDamageKey(id))})
	}
	for _, id := range gc.WeaponSkillIDs {
		name := query.T(world, gc.SkillName(id))
		items = append(items, statusItemData{Label: query.T(world, "%s accuracy", name), Value: val(gc.WeaponAccuracyKey(id)), Description: query.T(world, "%s weapon accuracy multiplier", name), Details: details(gc.WeaponAccuracyKey(id))})
	}
	for _, elem := range []gc.ElementType{gc.ElementTypeFire, gc.ElementTypeThunder, gc.ElementTypeChill, gc.ElementTypePhoton} {
		items = append(items, statusItemData{Label: query.T(world, "%s resistance", elem.String()), Value: val(gc.ElementResistKey(elem)), Description: query.T(world, "%s element damage multiplier. Lower reduces more", elem.String()), Details: details(gc.ElementResistKey(elem))})
	}

	// 血液量が危険域まで落ちると失血で HP が減る。一覧は % だけにし、減少量は詳細モーダルの内訳に出す
	bloodDesc := query.T(world, "Blood volume. Bleeding and severe conditions lower it")
	var bloodDetails []statusDetailRow
	if world.Components.HealthStatus.Has(playerEntity) {
		if drain, _ := world.Components.HealthStatus.Get(playerEntity).BloodLossHPDrain(); drain > 0 {
			bloodDesc = query.T(world, "Critically low. Bleeding out and losing HP")
			bloodDetails = []statusDetailRow{{Label: query.T(world, "HP loss"), Value: fmt.Sprintf("-%d", drain)}}
		}
	}

	items = append(items, statusItemData{Label: query.T(world, "Body function"), IsHeader: true, Description: query.T(world, "Body capacities lowered by injuries and illness")})
	items = append(items,
		statusItemData{Label: query.T(world, "Pain"), Value: fmt.Sprintf("%d%%", caps.Pain), Description: query.T(world, "Pain from conditions. Lowers consciousness")},
		statusItemData{Label: query.T(world, "Blood"), Value: fmt.Sprintf("%d%%", caps.Blood), Description: bloodDesc, Details: bloodDetails},
		statusItemData{Label: query.T(world, "Consciousness"), Value: fmt.Sprintf("%d%%", caps.Consciousness), Description: query.T(world, "Master capacity. Multiplies all others")},
		statusItemData{Label: query.T(world, "Manipulation"), Value: fmt.Sprintf("%d%%", caps.Manipulation), Description: query.T(world, "Affects melee accuracy and crafting")},
		statusItemData{Label: query.T(world, "Moving"), Value: fmt.Sprintf("%d%%", caps.Moving), Description: query.T(world, "Affects move speed")},
		statusItemData{Label: query.T(world, "Sight"), Value: fmt.Sprintf("%d%%", caps.Sight), Description: query.T(world, "Affects ranged accuracy and vision")},
	)

	items = append(items, statusItemData{Label: query.T(world, "Survival"), IsHeader: true, Description: query.T(world, "Survival effects")})
	items = append(items,
		statusItemData{Label: query.T(world, "Hypothermia progress"), Value: val(gc.ModColdProgress), Description: query.T(world, "Hypothermia progress rate. Lower is slower"), Details: details(gc.ModColdProgress)},
		statusItemData{Label: query.T(world, "Hunger progress"), Value: val(gc.ModHungerProgress), Description: query.T(world, "Hunger progress rate. Lower is slower"), Details: details(gc.ModHungerProgress)},
		statusItemData{Label: query.T(world, "Healing effect"), Value: val(gc.ModHealingEffect), Description: query.T(world, "Healing item effect multiplier. Higher heals more"), Details: details(gc.ModHealingEffect)},
	)

	items = append(items, statusItemData{Label: query.T(world, "Action"), IsHeader: true, Description: query.T(world, "Action effects")})
	items = append(items,
		statusItemData{Label: query.T(world, "Move speed"), Value: val(gc.ModMoveCost), Description: query.T(world, "AP cost multiplier when moving. Lower moves with less AP"), Details: details(gc.ModMoveCost)},
		statusItemData{Label: query.T(world, "Discovery"), Value: val(gc.ModExploration), Description: query.T(world, "Item discovery rate multiplier. Higher finds more"), Details: details(gc.ModExploration)},
		statusItemData{Label: query.T(world, "Detection"), Value: val(gc.ModEnemyVision), Description: query.T(world, "Enemy detection distance multiplier. Lower is harder to find"), Details: details(gc.ModEnemyVision)},
		statusItemData{Label: query.T(world, "Night vision"), Value: val(gc.ModNightVision), Description: query.T(world, "Vision multiplier in dark. Higher sees more"), Details: details(gc.ModNightVision)},
	)

	items = append(items, statusItemData{Label: query.T(world, "Production"), IsHeader: true, Description: query.T(world, "Production and trade effects")})
	items = append(items,
		statusItemData{Label: query.T(world, "Material cost"), Value: val(gc.ModCraftCost), Description: query.T(world, "Material consumption multiplier when crafting. Lower saves materials"), Details: details(gc.ModCraftCost)},
		statusItemData{Label: query.T(world, "Craft quality"), Value: val(gc.ModSmithQuality), Description: query.T(world, "Quality multiplier when crafting. Higher makes better goods"), Details: details(gc.ModSmithQuality)},
		statusItemData{Label: query.T(world, "Buy price"), Value: val(gc.ModBuyPrice), Description: query.T(world, "Purchase price multiplier. Lower buys cheaper"), Details: details(gc.ModBuyPrice)},
		statusItemData{Label: query.T(world, "Sell price"), Value: val(gc.ModSellPrice), Description: query.T(world, "Sell price multiplier. Higher sells higher"), Details: details(gc.ModSellPrice)},
		statusItemData{Label: query.T(world, "Max weight"), Value: val(gc.ModMaxWeight), Description: query.T(world, "Max carry weight multiplier"), Details: details(gc.ModMaxWeight)},
		statusItemData{Label: query.T(world, "Max load"), Value: val(gc.ModHeavyArmor), Description: query.T(world, "Max load multiplier"), Details: details(gc.ModHeavyArmor)},
	)
	return items
}

// translatedConditionName は不調の種類名を訳して返す。重症度は進行度%で表すため名前に付けない
func translatedConditionName(world w.World, ct gc.ConditionType) string {
	return query.T(world, gc.ConditionTypeDisplayName(ct))
}

// treatmentStatus は不調の治療状態を表す。未治療か、治療済みならその質を出す
func treatmentStatus(world w.World, cond gc.HealthCondition) string {
	if cond.TendQuality <= 0 {
		return query.T(world, "Untreated")
	}
	return fmt.Sprintf("%s %d%%", query.T(world, "Tended"), int(cond.TendQuality))
}

func (st *CharacterState) createHealthItems(world w.World, playerEntity ecs.Entity) []statusItemData {
	var items []statusItemData
	var hs *gc.HealthStatus
	if query.AliveHas(world, world.Components.HealthStatus, playerEntity) {
		hs = world.Components.HealthStatus.Get(playerEntity)
	}
	for i := range int(gc.BodyPartCount) {
		part := gc.BodyPart(i)
		// 部位はカテゴリ見出し。カーソルは見出しを飛ばす
		items = append(items, statusItemData{Label: query.T(world, part.String()), IsHeader: true, BodyPart: part})

		var conds []gc.HealthCondition
		if hs != nil {
			conds = hs.Parts[i].Conditions
		}
		if len(conds) == 0 {
			// 症状の無い部位は健康の1エントリを置く。見出しと区別するため字下げする
			items = append(items, statusItemData{Label: healthEntryIndent + query.T(world, "Normal"), BodyPart: part})
			continue
		}
		// 症状ごとに1エントリ。見出しと区別するため字下げし、名前の右に治療状態、値に進行度を出す
		for _, cond := range conds {
			name := translatedConditionName(world, cond.Type) + "  " + treatmentStatus(world, cond)
			items = append(items, statusItemData{
				Label:         healthEntryIndent + name,
				Value:         fmt.Sprintf("%d%%", int(cond.Timer)),
				BodyPart:      part,
				ConditionType: cond.Type,
			})
		}
	}
	return items
}

// sourceToDetails はModifierSourceのスライスから内訳表示用の行を生成する。変化量が0のソースは表示しない。
// 計算側は事実だけを持つので、ラベルの整形と翻訳はここで行う
func sourceToDetails(world w.World, srcs []gc.ModifierSource) []statusDetailRow {
	var rows []statusDetailRow
	for _, s := range srcs {
		if s.Value == 0 {
			continue
		}
		rows = append(rows, statusDetailRow{Label: sourceLabel(world, s), Value: fmt.Sprintf("%+d%%", s.Value)})
	}
	return rows
}

// sourceLabel は内訳1件の表示ラベルを現在言語で整形する
func sourceLabel(world w.World, s gc.ModifierSource) string {
	switch s.Kind {
	case gc.SourceSkill:
		return fmt.Sprintf("%s Lv%d", query.T(world, gc.SkillName(s.Skill)), s.Amount)
	case gc.SourceAbility:
		return fmt.Sprintf("%s %d", query.T(world, gc.AbilityName(s.Ability)), s.Amount)
	case gc.SourceCapacity:
		return fmt.Sprintf("%s %d%%", query.T(world, string(s.Capacity)), s.Amount)
	}
	panic("unknown ModifierSourceKind: " + string(s.Kind))
}

// buildInfoTableUI は情報タブの表とフッタ右端のページ表示を返す。
func buildInfoTableUI(world w.World, tab statusTabData, itemIndex int, res resources.UIResources) ([]uicore.Drawable, string) {
	hasModifier := tab.ID == tabAbilities
	// 能力タブは 名前・値・修正値 の3列、他は 名前・値 の2列。値と修正値は右寄せ
	cols := styled.Cols(styled.Name(), styled.Num())
	if hasModifier {
		cols = styled.Cols(styled.Name(), styled.Num(), styled.Num())
	}
	rows := make([]menuframe.Row, len(tab.Items))
	for i, it := range tab.Items {
		cells := make([]string, len(cols))
		cells[0] = it.Label
		if !it.IsHeader {
			cells[1] = it.Value
			if hasModifier {
				cells[2] = it.Modifier
			}
		}
		rows[i] = menuframe.Row{Cells: styled.TextCells(cells...), Header: it.IsHeader}
	}
	return menuframe.RenderList(itemIndex, rows, cols, menuframe.ListOpts{EmptyText: query.T(world, "No entries"), ItemsPerPage: menuframe.ListCapacity(world, true, true)}, res)
}
