package states

import (
	"fmt"
	"strings"

	"github.com/ebitenui/ebitenui/widget"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/styled"
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

type statusTabData struct {
	ID    string
	Label string
	Items []statusItemData
}

type statusItemData struct {
	Label       string
	Value       string
	Modifier    string
	Description string
	IsHeader    bool // カテゴリヘッダー行かどうか
	BodyPart    gc.BodyPart
	Details     []statusDetailRow // 詳細に表示する内訳
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
	if !query.AliveHas(world, world.Components.CharModifiers, playerEntity) {
		return items
	}
	e := world.Components.CharModifiers.Get(playerEntity)

	items = append(items, statusItemData{Label: query.T(world, "Combat"), IsHeader: true, Description: query.T(world, "Combat effects")})
	for _, id := range gc.AllSkillIDs {
		if mult, ok := e.WeaponDamage[id]; ok {
			name := query.T(world, gc.SkillName(id))
			items = append(items, statusItemData{Label: query.T(world, "%s attack power", name), Value: fmt.Sprintf("%d%%", mult), Description: query.T(world, "%s weapon damage multiplier", name), Details: sourceToDetails(e.Sources, gc.WeaponDamageKey(id))})
		}
	}
	for _, id := range gc.AllSkillIDs {
		if mult, ok := e.WeaponAccuracy[id]; ok {
			name := query.T(world, gc.SkillName(id))
			items = append(items, statusItemData{Label: query.T(world, "%s accuracy", name), Value: fmt.Sprintf("%d%%", mult), Description: query.T(world, "%s weapon accuracy multiplier", name), Details: sourceToDetails(e.Sources, gc.WeaponAccuracyKey(id))})
		}
	}
	for _, elem := range []gc.ElementType{gc.ElementTypeFire, gc.ElementTypeThunder, gc.ElementTypeChill, gc.ElementTypePhoton} {
		if mult, ok := e.ElementResist[elem]; ok {
			items = append(items, statusItemData{Label: query.T(world, "%s resistance", elem.String()), Value: fmt.Sprintf("%d%%", mult), Description: query.T(world, "%s element damage multiplier. Lower reduces more", elem.String()), Details: sourceToDetails(e.Sources, gc.ElementResistKey(elem))})
		}
	}

	items = append(items, statusItemData{Label: query.T(world, "Survival"), IsHeader: true, Description: query.T(world, "Survival effects")})
	items = append(items,
		statusItemData{Label: query.T(world, "Hypothermia progress"), Value: fmt.Sprintf("%d%%", e.ColdProgress), Description: query.T(world, "Hypothermia progress rate. Lower is slower"), Details: sourceToDetails(e.Sources, gc.ModColdProgress)},
		statusItemData{Label: query.T(world, "Hyperthermia progress"), Value: fmt.Sprintf("%d%%", e.HeatProgress), Description: query.T(world, "Hyperthermia progress rate. Lower is slower"), Details: sourceToDetails(e.Sources, gc.ModHeatProgress)},
		statusItemData{Label: query.T(world, "Hunger progress"), Value: fmt.Sprintf("%d%%", e.HungerProgress), Description: query.T(world, "Hunger progress rate. Lower is slower"), Details: sourceToDetails(e.Sources, gc.ModHungerProgress)},
		statusItemData{Label: query.T(world, "Healing effect"), Value: fmt.Sprintf("%d%%", e.HealingEffect), Description: query.T(world, "Healing item effect multiplier. Higher heals more"), Details: sourceToDetails(e.Sources, gc.ModHealingEffect)},
	)

	items = append(items, statusItemData{Label: query.T(world, "Action"), IsHeader: true, Description: query.T(world, "Action effects")})
	items = append(items,
		statusItemData{Label: query.T(world, "Move speed"), Value: fmt.Sprintf("%d%%", e.MoveCost), Description: query.T(world, "AP cost multiplier when moving. Lower moves with less AP"), Details: sourceToDetails(e.Sources, gc.ModMoveCost)},
		statusItemData{Label: query.T(world, "Discovery"), Value: fmt.Sprintf("%d%%", e.Exploration), Description: query.T(world, "Item discovery rate multiplier. Higher finds more"), Details: sourceToDetails(e.Sources, gc.ModExploration)},
		statusItemData{Label: query.T(world, "Detection"), Value: fmt.Sprintf("%d%%", e.EnemyVision), Description: query.T(world, "Enemy detection distance multiplier. Lower is harder to find"), Details: sourceToDetails(e.Sources, gc.ModEnemyVision)},
		statusItemData{Label: query.T(world, "Night vision"), Value: fmt.Sprintf("%d%%", e.NightVision), Description: query.T(world, "Vision multiplier in dark. Higher sees more"), Details: sourceToDetails(e.Sources, gc.ModNightVision)},
	)

	items = append(items, statusItemData{Label: query.T(world, "Production"), IsHeader: true, Description: query.T(world, "Production and trade effects")})
	items = append(items,
		statusItemData{Label: query.T(world, "Material cost"), Value: fmt.Sprintf("%d%%", e.CraftCost), Description: query.T(world, "Material consumption multiplier when crafting. Lower saves materials"), Details: sourceToDetails(e.Sources, gc.ModCraftCost)},
		statusItemData{Label: query.T(world, "Craft quality"), Value: fmt.Sprintf("%d%%", e.SmithQuality), Description: query.T(world, "Quality multiplier when crafting. Higher makes better goods"), Details: sourceToDetails(e.Sources, gc.ModSmithQuality)},
		statusItemData{Label: query.T(world, "Buy price"), Value: fmt.Sprintf("%d%%", e.BuyPrice), Description: query.T(world, "Purchase price multiplier. Lower buys cheaper"), Details: sourceToDetails(e.Sources, gc.ModBuyPrice)},
		statusItemData{Label: query.T(world, "Sell price"), Value: fmt.Sprintf("%d%%", e.SellPrice), Description: query.T(world, "Sell price multiplier. Higher sells higher"), Details: sourceToDetails(e.Sources, gc.ModSellPrice)},
		statusItemData{Label: query.T(world, "Max weight"), Value: fmt.Sprintf("%d%%", e.MaxWeight), Description: query.T(world, "Max carry weight multiplier"), Details: sourceToDetails(e.Sources, gc.ModMaxWeight)},
		statusItemData{Label: query.T(world, "Max load"), Value: fmt.Sprintf("%d%%", e.HeavyArmor), Description: query.T(world, "Max load multiplier"), Details: sourceToDetails(e.Sources, gc.ModHeavyArmor)},
	)
	return items
}

func (st *CharacterState) createHealthItems(world w.World, playerEntity ecs.Entity) []statusItemData {
	items := make([]statusItemData, 0, int(gc.BodyPartCount))
	var hs *gc.HealthStatus
	if query.AliveHas(world, world.Components.HealthStatus, playerEntity) {
		hs = world.Components.HealthStatus.Get(playerEntity)
	}
	for i := range int(gc.BodyPartCount) {
		part := gc.BodyPart(i)
		var conditionStr strings.Builder
		if hs != nil {
			conditions := hs.Parts[i].Conditions
			for j, cond := range conditions {
				if j > 0 {
					conditionStr.WriteString(", ")
				}
				conditionStr.WriteString(query.T(world, cond.DisplayName()))
			}
		}
		value := conditionStr.String()
		if value == "" {
			value = query.T(world, "Normal")
		}
		items = append(items, statusItemData{Label: query.T(world, part.String()), Value: value, Description: query.T(world, getHealthPartDescription(part)), BodyPart: part})
	}
	return items
}

func getHealthPartDescription(part gc.BodyPart) string {
	switch part {
	case gc.BodyPartTorso:
		return "Torso"
	case gc.BodyPartHead:
		return "Head"
	case gc.BodyPartArms:
		return "Arms"
	case gc.BodyPartHands:
		return "Hands"
	case gc.BodyPartLegs:
		return "Legs"
	case gc.BodyPartFeet:
		return "Feet"
	case gc.BodyPartWholeBody:
		return "Whole body"
	default:
		return ""
	}
}

// sourceToDetails はModifierSourceのスライスから内訳表示用の行を生成する。変化量が0のソースは表示しない
func sourceToDetails(sources map[gc.ModifierKey][]gc.ModifierSource, key gc.ModifierKey) []statusDetailRow {
	srcs, ok := sources[key]
	if !ok {
		return nil
	}
	var rows []statusDetailRow
	for _, s := range srcs {
		if s.Value == 0 {
			continue
		}
		rows = append(rows, statusDetailRow{Label: s.Label, Value: fmt.Sprintf("%+d%%", s.Value)})
	}
	return rows
}

// buildInfoTable は情報タブの一覧を組み立てる。能力タブは補正列を加える
func buildInfoTable(world w.World, tab statusTabData, itemIndex int, res resources.UIResources) *widget.Container {
	hasModifier := tab.ID == tabAbilities
	var columnWidths []int
	var aligns []styled.TextAlign
	if hasModifier {
		columnWidths = []int{100, 60, 60}
		aligns = []styled.TextAlign{styled.AlignLeft, styled.AlignRight, styled.AlignRight}
	} else {
		columnWidths = []int{100, 60}
		aligns = []styled.TextAlign{styled.AlignLeft, styled.AlignRight}
	}

	rows := make([]menuRow, len(tab.Items))
	for i, it := range tab.Items {
		cells := make([]string, len(columnWidths))
		cells[0] = it.Label
		if !it.IsHeader {
			cells[1] = it.Value
			if hasModifier {
				cells[2] = it.Modifier
			}
		}
		rows[i] = menuRow{Cells: styled.TextCells(cells...), Header: it.IsHeader}
	}
	// この画面は見出しとタブ帯の両方が縦を食うので、その構成での実測容量を使う
	return renderMenuList(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: query.T(world, "No entries"), ItemsPerPage: menuframe.ListCapacity(res, true, true)}, res)
}

// buildInfoTableUI は buildInfoTable の internal/ui 版。情報タブの表を行ウィジェット列で返す。
func buildInfoTableUI(world w.World, tab statusTabData, itemIndex int, res resources.UIResources) []ui.Widget {
	hasModifier := tab.ID == tabAbilities
	var columnWidths []int
	var aligns []styled.TextAlign
	if hasModifier {
		columnWidths = []int{100, 60, 60}
		aligns = []styled.TextAlign{styled.AlignLeft, styled.AlignRight, styled.AlignRight}
	} else {
		columnWidths = []int{100, 60}
		aligns = []styled.TextAlign{styled.AlignLeft, styled.AlignRight}
	}
	rows := make([]menuRow, len(tab.Items))
	for i, it := range tab.Items {
		cells := make([]string, len(columnWidths))
		cells[0] = it.Label
		if !it.IsHeader {
			cells[1] = it.Value
			if hasModifier {
				cells[2] = it.Modifier
			}
		}
		rows[i] = menuRow{Cells: styled.TextCells(cells...), Header: it.IsHeader}
	}
	return renderMenuListUI(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: query.T(world, "No entries"), ItemsPerPage: menuframe.ListCapacity(res, true, true)}, res.Text.BodyFace)
}
