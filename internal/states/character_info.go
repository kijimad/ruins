package states

import (
	"fmt"
	"strings"

	"github.com/ebitenui/ebitenui/widget"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/systems"
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
	envTemp := 0
	if query.AliveHas(world, world.Components.GridElement, player) {
		gridElement := world.Components.GridElement.Get(player)
		if temp, err := systems.CalculateEnvTemperature(world, gridElement.X, gridElement.Y); err == nil {
			envTemp = temp
		}
	}
	professionName := ""
	if query.AliveHas(world, world.Components.Profession, player) {
		profComp := world.Components.Profession.Get(player)
		if prof, err := raw.GetProfession(world.Resources.RawMaster, profComp.ID); err == nil {
			professionName = prof.Name
		}
	}

	return []statusTabData{
		{ID: tabAbilities, Label: "能力", Items: st.createAbilityItems(world, player)},
		{ID: tabSkills, Label: "スキル", Items: st.createSkillItems(world, player)},
		{ID: tabEffects, Label: "効果", Items: st.createEffectItems(world, player)},
		{ID: tabHealth, Label: "健康", Items: st.createHealthItems(world, player)},
		{ID: tabBasic, Label: "基本", Items: st.createBasicItems(world, player, envTemp, professionName)},
	}
}

func (st *CharacterState) createBasicItems(world w.World, playerEntity ecs.Entity, envTemp int, professionName string) []statusItemData {
	items := []statusItemData{}

	if professionName != "" {
		items = append(items, statusItemData{Label: "職業", Value: professionName, Description: "職業"})
	}
	if query.AliveHas(world, world.Components.HP, playerEntity) {
		hp := world.Components.HP.Get(playerEntity)
		items = append(items, statusItemData{Label: "HP", Value: fmt.Sprintf("%d", hp.Max), Description: "体力。0になると死亡する"})
	}
	if query.AliveHas(world, world.Components.WeightCapacity, playerEntity) {
		cw := world.Components.WeightCapacity.Get(playerEntity)
		items = append(items, statusItemData{Label: "最大重量", Value: cw.Max.KgString(), Description: "所持可能な最大重量"})
	}
	if query.AliveHas(world, world.Components.Hunger, playerEntity) {
		hunger := world.Components.Hunger.Get(playerEntity)
		items = append(items, statusItemData{Label: "空腹度", Value: hunger.GetLevel().String(), Description: "空腹度。高いと行動に支障が出る"})
	}
	items = append(items,
		statusItemData{Label: "環境気温", Value: fmt.Sprintf("%d%s", envTemp, consts.IconDegree), Description: "現在地の気温"},
		statusItemData{Label: "時間帯", Value: query.GetGameTime(world).GetTimeOfDay().String(), Description: "現在の時間帯。屋外では気温に影響する"},
	)
	return items
}

func (st *CharacterState) createAbilityItems(world w.World, playerEntity ecs.Entity) []statusItemData {
	items := []statusItemData{}
	if query.AliveHas(world, world.Components.Abilities, playerEntity) {
		abils := world.Components.Abilities.Get(playerEntity)
		items = append(items,
			statusItemData{Label: consts.VitalityLabel, Value: fmt.Sprintf("%d", abils.Vitality.Total), Modifier: fmt.Sprintf("(%+d)", abils.Vitality.Modifier), Description: "体力。HPとSPの最大値に影響する"},
			statusItemData{Label: consts.StrengthLabel, Value: fmt.Sprintf("%d", abils.Strength.Total), Modifier: fmt.Sprintf("(%+d)", abils.Strength.Modifier), Description: "筋力。近接攻撃のダメージに影響する"},
			statusItemData{Label: consts.SensationLabel, Value: fmt.Sprintf("%d", abils.Sensation.Total), Modifier: fmt.Sprintf("(%+d)", abils.Sensation.Modifier), Description: "感覚。射撃攻撃のダメージに影響する"},
			statusItemData{Label: consts.DexterityLabel, Value: fmt.Sprintf("%d", abils.Dexterity.Total), Modifier: fmt.Sprintf("(%+d)", abils.Dexterity.Modifier), Description: "器用さ。命中率に影響する"},
			statusItemData{Label: consts.AgilityLabel, Value: fmt.Sprintf("%d", abils.Agility.Total), Modifier: fmt.Sprintf("(%+d)", abils.Agility.Modifier), Description: "敏捷。回避率と行動速度に影響する"},
			statusItemData{Label: consts.DefenseLabel, Value: fmt.Sprintf("%d", abils.Defense.Total), Modifier: fmt.Sprintf("(%+d)", abils.Defense.Modifier), Description: "防御。被ダメージを軽減する"},
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
		items = append(items, statusItemData{Label: cat.Name, IsHeader: true, Description: fmt.Sprintf("%sカテゴリのスキル", cat.Name)})
		for _, id := range cat.IDs {
			s := skills.Get(id)
			expFrac := 0
			if s.Exp.Max > 0 {
				expFrac = s.Exp.Current * 1000 / s.Exp.Max
			}
			info := gc.SkillDescription(id)
			items = append(items, statusItemData{
				Label:       gc.SkillName(id),
				Value:       fmt.Sprintf("%d.%03d", s.Value, expFrac),
				Description: info.Summary,
				Details: []statusDetailRow{
					{Label: "獲得条件", Value: info.GainedBy},
					{Label: "効果", Value: info.Effect},
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

	items = append(items, statusItemData{Label: "戦闘", IsHeader: true, Description: "戦闘に関する効果"})
	for _, id := range gc.AllSkillIDs {
		if mult, ok := e.WeaponDamage[id]; ok {
			name := gc.SkillName(id)
			items = append(items, statusItemData{Label: name + "攻撃力", Value: fmt.Sprintf("%d%%", mult), Description: fmt.Sprintf("%s武器のダメージ倍率", name), Details: sourceToDetails(e.Sources, gc.WeaponDamageKey(id))})
		}
	}
	for _, id := range gc.AllSkillIDs {
		if mult, ok := e.WeaponAccuracy[id]; ok {
			name := gc.SkillName(id)
			items = append(items, statusItemData{Label: name + "命中", Value: fmt.Sprintf("%d%%", mult), Description: fmt.Sprintf("%s武器の命中倍率", name), Details: sourceToDetails(e.Sources, gc.WeaponAccuracyKey(id))})
		}
	}
	for _, elem := range []gc.ElementType{gc.ElementTypeFire, gc.ElementTypeThunder, gc.ElementTypeChill, gc.ElementTypePhoton} {
		if mult, ok := e.ElementResist[elem]; ok {
			items = append(items, statusItemData{Label: elem.String() + "耐性", Value: fmt.Sprintf("%d%%", mult), Description: fmt.Sprintf("%s属性ダメージの倍率。低いほど軽減される", elem.String()), Details: sourceToDetails(e.Sources, gc.ElementResistKey(elem))})
		}
	}

	items = append(items, statusItemData{Label: "生存", IsHeader: true, Description: "生存に関する効果"})
	items = append(items,
		statusItemData{Label: "低体温進行", Value: fmt.Sprintf("%d%%", e.ColdProgress), Description: "低体温の進行速度。低いほど遅くなる", Details: sourceToDetails(e.Sources, gc.ModColdProgress)},
		statusItemData{Label: "高体温進行", Value: fmt.Sprintf("%d%%", e.HeatProgress), Description: "高体温の進行速度。低いほど遅くなる", Details: sourceToDetails(e.Sources, gc.ModHeatProgress)},
		statusItemData{Label: "空腹進行", Value: fmt.Sprintf("%d%%", e.HungerProgress), Description: "空腹の進行速度。低いほど遅くなる", Details: sourceToDetails(e.Sources, gc.ModHungerProgress)},
		statusItemData{Label: "回復効果", Value: fmt.Sprintf("%d%%", e.HealingEffect), Description: "回復アイテムの効果倍率。高いほど多く回復する", Details: sourceToDetails(e.Sources, gc.ModHealingEffect)},
	)

	items = append(items, statusItemData{Label: "行動", IsHeader: true, Description: "行動に関する効果"})
	items = append(items,
		statusItemData{Label: "移動速度", Value: fmt.Sprintf("%d%%", e.MoveCost), Description: "移動時のAPコスト倍率。低いほど少ないAPで移動できる", Details: sourceToDetails(e.Sources, gc.ModMoveCost)},
		statusItemData{Label: "発見", Value: fmt.Sprintf("%d%%", e.Exploration), Description: "アイテム発見率の倍率。高いほど見つけやすい", Details: sourceToDetails(e.Sources, gc.ModExploration)},
		statusItemData{Label: "被発見", Value: fmt.Sprintf("%d%%", e.EnemyVision), Description: "敵に発見される距離の倍率。低いほど見つかりにくい", Details: sourceToDetails(e.Sources, gc.ModEnemyVision)},
		statusItemData{Label: "暗所視界", Value: fmt.Sprintf("%d%%", e.NightVision), Description: "暗所での視界の倍率。高いほど見える", Details: sourceToDetails(e.Sources, gc.ModNightVision)},
	)

	items = append(items, statusItemData{Label: "生産", IsHeader: true, Description: "生産・取引に関する効果"})
	items = append(items,
		statusItemData{Label: "素材消費", Value: fmt.Sprintf("%d%%", e.CraftCost), Description: "合成時の素材消費量倍率。低いほど素材が節約できる", Details: sourceToDetails(e.Sources, gc.ModCraftCost)},
		statusItemData{Label: "合成品質", Value: fmt.Sprintf("%d%%", e.SmithQuality), Description: "調合時の品質倍率。高いほど良い品ができる", Details: sourceToDetails(e.Sources, gc.ModSmithQuality)},
		statusItemData{Label: "買値", Value: fmt.Sprintf("%d%%", e.BuyPrice), Description: "買い物の価格倍率。低いほど安く買える", Details: sourceToDetails(e.Sources, gc.ModBuyPrice)},
		statusItemData{Label: "売値", Value: fmt.Sprintf("%d%%", e.SellPrice), Description: "売却の価格倍率。高いほど高く売れる", Details: sourceToDetails(e.Sources, gc.ModSellPrice)},
		statusItemData{Label: "最大重量", Value: fmt.Sprintf("%d%%", e.MaxWeight), Description: "所持可能な最大重量の倍率", Details: sourceToDetails(e.Sources, gc.ModMaxWeight)},
		statusItemData{Label: "最大荷重", Value: fmt.Sprintf("%d%%", e.HeavyArmor), Description: "最大荷重倍率", Details: sourceToDetails(e.Sources, gc.ModHeavyArmor)},
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
				conditionStr.WriteString(cond.DisplayName())
			}
		}
		value := conditionStr.String()
		if value == "" {
			value = "正常"
		}
		items = append(items, statusItemData{Label: part.String(), Value: value, Description: getHealthPartDescription(part), BodyPart: part})
	}
	return items
}

func getHealthPartDescription(part gc.BodyPart) string {
	switch part {
	case gc.BodyPartTorso:
		return "胴体"
	case gc.BodyPartHead:
		return "頭部"
	case gc.BodyPartArms:
		return "腕"
	case gc.BodyPartHands:
		return "手"
	case gc.BodyPartLegs:
		return "脚"
	case gc.BodyPartFeet:
		return "足"
	case gc.BodyPartWholeBody:
		return "全身"
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

// buildInfoTable は読み取り専用タブのアイテム一覧をテーブルで組み立てる
// buildInfoTable は情報タブの一覧を組み立てる。能力タブは補正列を加える
func buildInfoTable(tab statusTabData, itemIndex int, res resources.UIResources) *widget.Container {
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
		rows[i] = menuRow{Cells: cells, Header: it.IsHeader}
	}
	list := renderMenuList(Selection{ItemIndex: itemIndex}, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true}, res)

	if len(tab.Items) == 0 {
		list.AddChild(styled.NewDescriptionText("(項目なし)", res))
	}
	return list
}
