package raw

import (
	"errors"
	"fmt"
	"image/color"
	"math/rand/v2"

	"github.com/BurntSushi/toml"
	"github.com/kijimaD/ruins/assets"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
)

// PtrSlice はoapi.Rawsの*[]Tフィールドを安全にデリファレンスする
// nilポインタの場合はnilスライスを返す
func PtrSlice[T any](p *[]T) []T {
	if p == nil {
		return nil
	}
	return *p
}

// findByKey はスライスからキー関数でマッチする要素を線形検索する
func findByKey[T any](slice *[]T, keyFn func(T) string, target string) (T, bool) {
	for _, item := range PtrSlice(slice) {
		if keyFn(item) == target {
			return item, true
		}
	}
	var zero T
	return zero, false
}

// LoadFromFile はファイルからローデータを読み込み、OpenAPIスキーマで検証する
func LoadFromFile(path string) (oapi.Raws, error) {
	bs, err := assets.FS.ReadFile(path)
	if err != nil {
		return oapi.Raws{}, err
	}
	raws, err := DecodeRaws(string(bs))
	if err != nil {
		return oapi.Raws{}, err
	}
	if err := ValidateRaws(raws); err != nil {
		return oapi.Raws{}, fmt.Errorf("failed to validate raw data for %s: %w", path, err)
	}
	if err := ValidateReferences(raws); err != nil {
		return oapi.Raws{}, fmt.Errorf("failed to validate raw data for %s: %w", path, err)
	}
	if err := validateCodeRows(raws); err != nil {
		return oapi.Raws{}, fmt.Errorf("failed to validate raw data for %s: %w", path, err)
	}
	return raws, nil
}

// DecodeRaws はTOML文字列をoapi.Raws構造体にデコードする
// 未知のキーが含まれる場合はエラーを返す
func DecodeRaws(content string) (oapi.Raws, error) {
	var raws oapi.Raws
	metaData, err := toml.Decode(content, &raws)
	if err != nil {
		return oapi.Raws{}, fmt.Errorf("TOML decode error: %w", err)
	}
	if undecoded := metaData.Undecoded(); len(undecoded) > 0 {
		return oapi.Raws{}, fmt.Errorf("unknown keys found in TOML: %v", undecoded)
	}
	return raws, nil
}

// FindItem は指定された名前のアイテム定義を検索する
func FindItem(raws oapi.Raws, name string) (oapi.Item, error) {
	item, ok := findByKey(raws.Items, func(i oapi.Item) string { return i.Id }, name)
	if !ok {
		return oapi.Item{}, NewKeyNotFoundError(name, "Items")
	}
	return item, nil
}

// ItemName は id からアイテムの表示名を返す。定義が見つからなければ id をそのまま返す
func ItemName(raws oapi.Raws, id string) string {
	item, err := FindItem(raws, id)
	if err != nil {
		return id
	}
	return item.Name
}

// FindMember は指定された名前のメンバー定義を検索する
func FindMember(raws oapi.Raws, name string) (oapi.Member, error) {
	member, ok := findByKey(raws.Members, func(m oapi.Member) string { return m.Id }, name)
	if !ok {
		return oapi.Member{}, NewKeyNotFoundError(name, "Members")
	}
	return member, nil
}

// FindSpriteSheet は指定された名前のスプライトシートを検索する
func FindSpriteSheet(raws oapi.Raws, name string) (oapi.SpriteSheet, error) {
	sheet, ok := findByKey(raws.SpriteSheets, func(s oapi.SpriteSheet) string { return s.Name }, name)
	if !ok {
		return oapi.SpriteSheet{}, NewKeyNotFoundError(name, "SpriteSheets")
	}
	return sheet, nil
}

// toGCSpriteRender はoapi.SpriteRenderからgc.SpriteRenderに変換する
func toGCSpriteRender(s oapi.SpriteRender) gc.SpriteRender {
	return gc.SpriteRender{
		SpriteSheetName: s.SpriteSheetName,
		SpriteKey:       s.SpriteKey,
		Depth:           gc.DepthNum(int(s.Depth)),
	}
}

// toGCLightSource はoapi.LightSourceからgc.LightSourceに変換する
func toGCLightSource(ls *oapi.LightSource) *gc.LightSource {
	if ls == nil {
		return nil
	}
	return &gc.LightSource{
		Radius:  consts.Tile(ls.Radius),
		Enabled: ls.Enabled,
		Color: color.RGBA{
			R: ls.Color.R,
			G: ls.Color.G,
			B: ls.Color.B,
			A: ls.Color.A,
		},
	}
}

// toGCHeatSource はoapi.HeatSourceからgc.HeatSourceに変換する
func toGCHeatSource(hs *oapi.HeatSource) *gc.HeatSource {
	if hs == nil {
		return nil
	}
	return &gc.HeatSource{
		Radius: consts.Tile(hs.Radius),
		Warmth: hs.Warmth,
	}
}

// toGCBedding はoapi.Beddingからgc.Beddingに変換する
func toGCBedding(b *oapi.Bedding) *gc.Bedding {
	if b == nil {
		return nil
	}
	return &gc.Bedding{
		Quality: consts.Percent(b.Quality),
	}
}

// parseTargetType はTargetGroup/TargetNumの文字列ペアをパースする
// enum値の妥当性はOpenAPIスキーマで検証済み
func parseTargetType(targetGroup oapi.TargetGroup, targetNum oapi.TargetNum) gc.TargetType {
	if targetGroup == "" {
		return gc.TargetType{}
	}
	return gc.TargetType{
		TargetGroup: targetGroup,
		TargetNum:   targetNum,
	}
}

// parseMelee はoapi.Meleeからgc.Meleeを生成する
func parseMelee(m *oapi.Melee) (*gc.Melee, error) {
	attackType, err := gc.ParseAttackType(string(m.AttackCategory))
	if err != nil {
		return nil, err
	}
	return &gc.Melee{
		Accuracy:       m.Accuracy,
		Damage:         m.Damage,
		AttackCount:    m.AttackCount,
		Element:        m.Element,
		AttackCategory: attackType,
		Cost:           m.Cost,
		TargetType:     parseTargetType(m.TargetGroup, m.TargetNum),
	}, nil
}

// parseFire はoapi.Fireからgc.Fireを生成する
func parseFire(f *oapi.Fire) (*gc.Fire, error) {
	attackType, err := gc.ParseAttackType(string(f.AttackCategory))
	if err != nil {
		return nil, err
	}
	var ammoTag oapi.AmmoTag
	if f.AmmoTag != nil {
		ammoTag = *f.AmmoTag
	}
	return &gc.Fire{
		Accuracy:       f.Accuracy,
		Damage:         f.Damage,
		AttackCount:    f.AttackCount,
		Element:        f.Element,
		AttackCategory: attackType,
		Cost:           f.Cost,
		TargetType:     parseTargetType(f.TargetGroup, f.TargetNum),
		Magazine:       f.MagazineSize,
		MagazineSize:   f.MagazineSize,
		ReloadEffort:   f.ReloadEffort,
		AmmoTag:        ammoTag,
	}, nil
}

// newProvidesHealingFromAPI はoapi.ProvidesHealingからProvidesHealingコンポーネントを生成する
// enum値の妥当性はOpenAPIスキーマで検証済み
func newProvidesHealingFromAPI(h *oapi.ProvidesHealing) *gc.ProvidesHealing {
	switch h.ValueType {
	case oapi.PERCENTAGE:
		return &gc.ProvidesHealing{Kind: gc.HealRatio, Amount: h.Ratio}
	default:
		return &gc.ProvidesHealing{Kind: gc.HealNumeral, Amount: float64(h.Amount)}
	}
}

// newBookFromAPI はoapi.Bookからgc.Bookを生成する
func newBookFromAPI(b *oapi.Book) (*gc.Book, error) {
	if b.Skill == nil {
		return nil, fmt.Errorf("book requires a Skill")
	}

	skillID := b.Skill.TargetSkill

	return &gc.Book{
		Effort: gc.IntPool{Max: b.TotalEffort},
		Skill: &gc.SkillBookEffect{
			TargetSkill:   skillID,
			MaxLevel:      b.Skill.MaxLevel,
			RequiredLevel: b.Skill.RequiredLevel,
		},
	}, nil
}

// NewItemSpec は指定された名前のアイテムのEntitySpecを生成する
func NewItemSpec(raws oapi.Raws, name string) (gc.EntitySpec, error) {
	item, err := FindItem(raws, name)
	if err != nil {
		return gc.EntitySpec{}, err
	}

	entitySpec := gc.EntitySpec{}
	entitySpec.Item = &gc.Item{}
	entitySpec.Name = &gc.Name{Name: item.Name}
	entitySpec.RawID = &gc.RawID{ID: item.Id}
	entitySpec.Description = &gc.Description{Description: item.Description}

	// デフォルト値設定
	spriteSheetName := item.SpriteSheetName
	spriteKey := item.SpriteKey
	if spriteSheetName == "" {
		spriteSheetName = "field"
	}
	if spriteKey == "" {
		spriteKey = "field_item"
	}

	entitySpec.SpriteRender = &gc.SpriteRender{
		SpriteSheetName: spriteSheetName,
		SpriteKey:       spriteKey,
		AnimKeys:        PtrSlice(item.AnimKeys),
		Depth:           gc.DepthNumRug,
	}

	if item.Consumable != nil {
		entitySpec.Consumable = &gc.Consumable{
			UsableScene: item.Consumable.UsableScene,
			TargetType: gc.TargetType{
				TargetGroup: item.Consumable.TargetGroup,
				TargetNum:   item.Consumable.TargetNum,
			},
		}
	}

	if item.ProvidesHealing != nil {
		entitySpec.ProvidesHealing = newProvidesHealingFromAPI(item.ProvidesHealing)
	}
	if item.ProvidesNutrition != nil {
		entitySpec.ProvidesNutrition = &gc.ProvidesNutrition{Amount: *item.ProvidesNutrition}
	}
	// stageLength を持つ食料は腐敗する。RotUpdatedTurn は spawn 時に刻印するためここでは 0 のまま
	if item.StageLength != nil {
		entitySpec.Perishable = &gc.Perishable{StageLength: consts.Turn(*item.StageLength)}
	}
	if item.InflictsDamage != nil {
		entitySpec.InflictsDamage = &gc.InflictsDamage{Amount: *item.InflictsDamage}
	}
	if item.Remedy != nil {
		treats := make([]gc.ConditionType, 0, len(item.Remedy.Treats))
		for _, t := range item.Remedy.Treats {
			treats = append(treats, gc.ConditionType(t))
		}
		entitySpec.Remedy = &gc.Remedy{
			Treats:  treats,
			Potency: consts.Percent(item.Remedy.Potency),
		}
	}

	if item.Ammo != nil {
		var ammoAmmoTag oapi.AmmoTag
		if item.Ammo.AmmoTag != nil {
			ammoAmmoTag = *item.Ammo.AmmoTag
		}
		entitySpec.Ammo = &gc.Ammo{
			AmmoTag:       ammoAmmoTag,
			DamageBonus:   item.Ammo.DamageBonus,
			AccuracyBonus: item.Ammo.AccuracyBonus,
		}
	}

	if item.Melee != nil {
		melee, err := parseMelee(item.Melee)
		if err != nil {
			return gc.EntitySpec{}, err
		}
		entitySpec.Melee = melee
	}
	if item.Fire != nil {
		fire, err := parseFire(item.Fire)
		if err != nil {
			return gc.EntitySpec{}, err
		}
		entitySpec.Fire = fire
	}

	var bonus gc.EquipBonus
	if item.EquipBonus != nil {
		bonus = gc.EquipBonus{
			Vitality:  item.EquipBonus.Vitality,
			Strength:  item.EquipBonus.Strength,
			Sensation: item.EquipBonus.Sensation,
			Dexterity: item.EquipBonus.Dexterity,
			Agility:   item.EquipBonus.Agility,
		}
	}

	if item.Wearable != nil {
		entitySpec.Wearable = &gc.Wearable{
			Defense:           item.Wearable.Defense,
			EquipmentCategory: gc.EquipmentType(item.Wearable.EquipmentCategory),
			EquipBonus:        bonus,
			InsulationCold:    item.Wearable.InsulationCold,
			InsulationHeat:    item.Wearable.InsulationHeat,
		}
	}

	if item.CubeModule != nil {
		entitySpec.CubeModule = &gc.CubeModule{
			RangeBonus:        consts.Tile(item.CubeModule.RangeBonus),
			FuelCapacityBonus: consts.Heat(item.CubeModule.FuelCapacityBonus),
		}
	}

	entitySpec.Value = &gc.Value{Value: item.Value}

	if item.Weight != nil {
		mg, err := consts.ParseWeight(*item.Weight)
		if err != nil {
			return gc.EntitySpec{}, fmt.Errorf("item '%s' weight: %w", name, err)
		}
		entitySpec.Weight = &gc.Weight{Milligram: mg}
	}

	if item.Book != nil {
		book, err := newBookFromAPI(item.Book)
		if err != nil {
			return gc.EntitySpec{}, err
		}
		entitySpec.Book = book
	}

	// 携行光源。装備すると StatsChangedSystem が owner の LightSource へ転写する
	entitySpec.LightSource = toGCLightSource(item.LightSource)

	// 材質。可燃性と燃焼熱量の算出根拠。観察メニューで見せるためエンティティに保持する
	if item.Material != nil {
		entitySpec.Material = &gc.Material{Kind: *item.Material}
	}

	// 火種。所持していると隣接の燃焼物に着火できる道具
	if item.FireStarter != nil {
		entitySpec.FireStarter = &gc.FireStarter{}
	}

	// フィールドの相互作用。既定は拾えるアイテム。装着アイテムは applyDeployable が設備に応じて上書きする
	entitySpec.Interactable = &gc.Interactable{Interactions: []gc.InteractionKind{gc.InteractionItem}}
	if item.Deployable != nil {
		if err := applyDeployable(&entitySpec, item.Deployable, name); err != nil {
			return gc.EntitySpec{}, err
		}
	}

	return entitySpec, nil
}

// applyDeployable は装着アイテムの設備設定を1箇所に集める。マーカーと能力ごとのコンポーネントを付ける。
// 装着は設備画面で扱うので拾える相互作用は与えない。新設備は能力の枝を1つ足す。
// 能力枝を持たない装着、raw の空 [items.deployable] は Deployable マーカーだけを付ける。工作台が例。
func applyDeployable(entitySpec *gc.EntitySpec, dep *oapi.Deployable, name string) error {
	entitySpec.Deployable = &gc.Deployable{}
	entitySpec.Interactable = &gc.Interactable{Interactions: []gc.InteractionKind{}}
	if dep.Storage != nil {
		mg, err := consts.ParseWeight(dep.Storage.MaxWeight)
		if err != nil {
			return fmt.Errorf("item '%s' deployable storage: %w", name, err)
		}
		entitySpec.WeightCapacity = &gc.WeightCapacity{Max: mg}
		entitySpec.Interactable = &gc.Interactable{Interactions: []gc.InteractionKind{gc.InteractionStorage}}
	}
	// 携行光源と装着照明はどちらも実体の LightSource を使うが共存できず後勝ちになる。
	// 片方が痕跡なく消える矛盾記述なので、後勝ちを許さず error で弾く
	if dep.LightSource != nil {
		if entitySpec.LightSource != nil {
			return fmt.Errorf("item '%s': carried lightSource and deployable lightSource conflict", name)
		}
		entitySpec.LightSource = toGCLightSource(dep.LightSource)
	}
	if dep.Bedding != nil {
		entitySpec.Bedding = toGCBedding(dep.Bedding)
	}
	return nil
}

// NewRecipeSpec は指定された名前のレシピのEntitySpecを生成する
func NewRecipeSpec(raws oapi.Raws, name string) (gc.EntitySpec, error) {
	recipe, ok := findByKey(raws.Recipes, func(r oapi.Recipe) string { return r.Id }, name)
	if !ok {
		return gc.EntitySpec{}, NewKeyNotFoundError(name, "Recipes")
	}

	entitySpec := gc.EntitySpec{}
	entitySpec.Name = &gc.Name{Name: recipe.Name}
	entitySpec.Recipe = &gc.Recipe{}
	for _, input := range recipe.Inputs {
		entitySpec.Recipe.Inputs = append(entitySpec.Recipe.Inputs, gc.RecipeInput{ID: input.Id, Amount: input.Amount})
	}

	// 説明文や分類のため、マッチしたitemの定義から持ってくる
	itemSpec, err := NewItemSpec(raws, recipe.Id)
	if err != nil {
		return gc.EntitySpec{}, fmt.Errorf("%s: %w", "failed to generate item for recipe", err)
	}
	entitySpec.Description = &gc.Description{Description: itemSpec.Description.Description}
	if itemSpec.Melee != nil {
		entitySpec.Melee = itemSpec.Melee
	}
	if itemSpec.Fire != nil {
		entitySpec.Fire = itemSpec.Fire
	}
	if itemSpec.Wearable != nil {
		entitySpec.Wearable = itemSpec.Wearable
	}
	if itemSpec.Consumable != nil {
		entitySpec.Consumable = itemSpec.Consumable
	}
	if itemSpec.Value != nil {
		entitySpec.Value = itemSpec.Value
	}
	if itemSpec.Weight != nil {
		entitySpec.Weight = itemSpec.Weight
	}

	return entitySpec, nil
}

// NewWeaponSpec は指定された名前の武器のEntitySpecを生成する
func NewWeaponSpec(raws oapi.Raws, name string) (gc.EntitySpec, error) {
	// 武器はアイテムの一種なので、Itemsから検索して存在確認
	if _, err := FindItem(raws, name); err != nil {
		return gc.EntitySpec{}, err
	}

	itemSpec, err := NewItemSpec(raws, name)
	if err != nil {
		return gc.EntitySpec{}, fmt.Errorf("failed to generate weapon spec: %w", err)
	}

	// Melee/Fire のいずれも持たない場合は武器ではない
	if itemSpec.Melee == nil && itemSpec.Fire == nil {
		return gc.EntitySpec{}, fmt.Errorf("%s is not a weapon (Melee/Fire component missing)", name)
	}

	return itemSpec, nil
}

// NewMemberSpec は指定された名前のメンバーのEntitySpecを生成する
func NewMemberSpec(raws oapi.Raws, name string) (gc.EntitySpec, error) {
	member, ok := findByKey(raws.Members, func(m oapi.Member) string { return m.Id }, name)
	if !ok {
		return gc.EntitySpec{}, NewKeyNotFoundError(name, "Members")
	}

	entitySpec := gc.EntitySpec{}
	entitySpec.Name = &gc.Name{Name: member.Name}
	entitySpec.RawID = &gc.RawID{ID: member.Id}
	entitySpec.TurnBased = &gc.TurnBased{AP: gc.IntPool{Current: 100, Max: 100}} // TODO: Abilitiesから計算する
	entitySpec.SpriteRender = &gc.SpriteRender{
		SpriteSheetName: member.SpriteSheetName,
		SpriteKey:       member.SpriteKey,
		AnimKeys:        PtrSlice(member.AnimKeys),
		Depth:           gc.DepthNumPlayer,
	}
	entitySpec.Abilities = &gc.Abilities{
		Vitality:  gc.Ability{Base: member.Abilities.Vitality},
		Strength:  gc.Ability{Base: member.Abilities.Strength},
		Sensation: gc.Ability{Base: member.Abilities.Sensation},
		Dexterity: gc.Ability{Base: member.Abilities.Dexterity},
		Agility:   gc.Ability{Base: member.Abilities.Agility},
		Defense:   gc.Ability{Base: member.Abilities.Defense},
	}
	entitySpec.HP = &gc.HP{}
	entitySpec.WeightCapacity = &gc.WeightCapacity{}
	// 体重は能力値の体格から算出する。売買や運搬で member 自身の重量として扱う
	entitySpec.Weight = &gc.Weight{Milligram: entitySpec.Abilities.BodyWeight()}

	// 戦闘者共通の健康機構。不調が身体機能を下げ、命中や速度へ効く
	entitySpec.Skills = gc.NewSkills()
	entitySpec.HealthStatus = &gc.HealthStatus{}
	if member.Player != nil && *member.Player {
		entitySpec.Player = &gc.Player{}
	}

	// テーブル名はロード時に参照検証済みで、ここで失敗するのは整合性バグ
	if member.CommandTableId != nil {
		ct, err := GetCommandTable(raws, *member.CommandTableId)
		if err != nil {
			return gc.EntitySpec{}, fmt.Errorf("failed to get command table for member '%s': %w", name, err)
		}
		entitySpec.CommandTable = &gc.CommandTable{Name: ct.Id}
	}

	if member.DropTableId != nil {
		dt, err := GetDropTable(raws, *member.DropTableId)
		if err != nil {
			return gc.EntitySpec{}, fmt.Errorf("failed to get drop table for member '%s': %w", name, err)
		}
		entitySpec.DropTable = &gc.DropTable{Name: dt.Id}
	}

	entitySpec.LightSource = toGCLightSource(member.LightSource)

	// 味方と敵の派閥は NewPlayerSpec と NewEnemySpec が付ける。raw が指せるのは中立だけ
	if member.FactionType != nil && *member.FactionType == oapi.FactionNeutral {
		entitySpec.FactionNeutral = &gc.FactionNeutral{}
	}

	// AI の組み立て。SoloAIとして構築し、スポーン関数がそのまま使う
	{
		solo := gc.SoloAI{
			CombatDefault: oapi.Attack,
			CombatCurrent: oapi.Attack,
		}
		if member.CombatPolicy != nil && *member.CombatPolicy != "" {
			solo.CombatDefault = *member.CombatPolicy
			solo.CombatCurrent = *member.CombatPolicy
		}
		if member.MovementPattern != nil && *member.MovementPattern != "" {
			solo.Movement = *member.MovementPattern
		}
		entitySpec.SoloAI = &solo
	}

	if member.Dialog != nil {
		entitySpec.Dialog = &gc.Dialog{
			MessageKey: member.Dialog.MessageKey,
		}
		entitySpec.Interactable = &gc.Interactable{Interactions: []gc.InteractionKind{gc.InteractionTalk}}
	}

	return entitySpec, nil
}

// NewPlayerSpec は指定された名前のプレイヤーのEntitySpecを生成する
func NewPlayerSpec(raws oapi.Raws, name string) (gc.EntitySpec, error) {
	entitySpec, err := NewMemberSpec(raws, name)
	if err != nil {
		return gc.EntitySpec{}, err
	}
	entitySpec.FactionAlly = &gc.FactionAlly{}
	entitySpec.Player = &gc.Player{}
	entitySpec.Hunger = gc.NewHunger()
	entitySpec.Fatigue = gc.NewFatigue()
	entitySpec.SoloAI = nil
	return entitySpec, nil
}

// NewEnemySpec は指定された名前の敵のEntitySpecを生成する
func NewEnemySpec(raws oapi.Raws, name string) (gc.EntitySpec, error) {
	entitySpec, err := NewMemberSpec(raws, name)
	if err != nil {
		return gc.EntitySpec{}, err
	}
	entitySpec.FactionEnemy = &gc.FactionEnemy{}

	return entitySpec, nil
}

// GetCommandTable は指定された名前のコマンドテーブルを取得する
func GetCommandTable(raws oapi.Raws, name string) (oapi.CommandTable, error) {
	ct, ok := findByKey(raws.CommandTables, func(c oapi.CommandTable) string { return c.Id }, name)
	if !ok {
		return oapi.CommandTable{}, NewKeyNotFoundError(name, "CommandTables")
	}
	return ct, nil
}

// GetDropTable は指定された名前のドロップテーブルを取得する
func GetDropTable(raws oapi.Raws, name string) (oapi.DropTable, error) {
	dt, ok := findByKey(raws.DropTables, func(d oapi.DropTable) string { return d.Id }, name)
	if !ok {
		return oapi.DropTable{}, NewKeyNotFoundError(name, "DropTables")
	}
	return dt, nil
}

// errItemGroupNotExist は指定した ID のアイテムグループもアイテムコレクションも存在しないときに返す。
var errItemGroupNotExist = errors.New("item group does not exist")

// GetItemGroup は指定された名前のアイテムグループを取得する
func GetItemGroup(raws oapi.Raws, name string) (oapi.ItemGroup, error) {
	ig, ok := findByKey(raws.ItemGroups, func(g oapi.ItemGroup) string { return g.Id }, name)
	if !ok {
		return oapi.ItemGroup{}, fmt.Errorf("%w: %s", errItemGroupNotExist, name)
	}
	return ig, nil
}

// GetItemCollection は指定された id のアイテムコレクションを取得する
func GetItemCollection(raws oapi.Raws, id string) (oapi.ItemCollection, error) {
	c, ok := findByKey(raws.ItemCollections, func(c oapi.ItemCollection) string { return c.Id }, id)
	if !ok {
		return oapi.ItemCollection{}, NewKeyNotFoundError(id, "ItemCollections")
	}
	return c, nil
}

// GetItemTable は指定された名前のアイテムテーブルを取得する
func GetItemTable(raws oapi.Raws, name string) (oapi.ItemTable, error) {
	it, ok := findByKey(raws.ItemTables, func(t oapi.ItemTable) string { return t.Id }, name)
	if !ok {
		return oapi.ItemTable{}, NewKeyNotFoundError(name, "ItemTables")
	}
	return it, nil
}

// GetEnemyTable は指定された名前の敵テーブルを取得する
func GetEnemyTable(raws oapi.Raws, name string) (oapi.EnemyTable, error) {
	et, ok := findByKey(raws.EnemyTables, func(t oapi.EnemyTable) string { return t.Id }, name)
	if !ok {
		return oapi.EnemyTable{}, NewKeyNotFoundError(name, "EnemyTables")
	}
	return et, nil
}

// SchemaEnum は OpenAPI スキーマの enum 型 name の値を返す。
func SchemaEnum(name string) ([]string, error) {
	spec, err := loadSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI schema: %w", err)
	}
	ref, ok := spec.Components.Schemas[name]
	if !ok || ref.Value == nil || len(ref.Value.Enum) == 0 {
		return nil, fmt.Errorf("schema %q is not an enum", name)
	}
	out := make([]string, len(ref.Value.Enum))
	for i, v := range ref.Value.Enum {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("schema %q enum value %v is not a string", name, v)
		}
		out[i] = s
	}
	return out, nil
}

// GetFacility は id の施設宣言を返す。未登録は false。
func GetFacility(raws oapi.Raws, id string) (oapi.Facility, bool) {
	return findByKey(raws.Facilities, func(f oapi.Facility) string { return f.Id }, id)
}

// FacilityLootedChance は施設 fac の収納家具が荒らされて空になっている確率を百分率で返す。省略は0。
func FacilityLootedChance(raws oapi.Raws, fac string) (int, error) {
	f, ok := GetFacility(raws, fac)
	if !ok {
		return 0, NewKeyNotFoundError(fac, "Facilities")
	}
	if f.LootedChance == nil {
		return 0, nil
	}
	return int(*f.LootedChance), nil
}

// GetLandmark は id のランドマーク宣言を返す。未登録は false。
func GetLandmark(raws oapi.Raws, id string) (oapi.Landmark, bool) {
	return findByKey(raws.Landmarks, func(l oapi.Landmark) string { return l.Id }, id)
}

// GetScatterZone は id の散布ゾーン宣言を返す。未登録は false。
func GetScatterZone(raws oapi.Raws, id string) (oapi.ScatterZone, bool) {
	return findByKey(raws.ScatterZones, func(z oapi.ScatterZone) string { return z.Id }, id)
}

// GetMapGlyph は id の地図記号宣言を返す。未登録は false。
func GetMapGlyph(raws oapi.Raws, id string) (oapi.MapGlyph, bool) {
	return findByKey(raws.MapGlyphs, func(m oapi.MapGlyph) string { return m.Id }, id)
}

// GetTile は指定された名前のタイルを取得する
// 計画段階でタイルの性質（Walkableなど）を参照する場合に使用する
func GetTile(raws oapi.Raws, id string) (oapi.Tile, error) {
	tile, ok := findByKey(raws.Tiles, func(t oapi.Tile) string { return t.Id }, id)
	if !ok {
		return oapi.Tile{}, NewKeyNotFoundError(id, "Tiles")
	}
	return tile, nil
}

// NewTileSpec は指定された名前のタイルのEntitySpecを生成する
// 実際にエンティティを生成する際に使用する
func NewTileSpec(raws oapi.Raws, name string, x, y consts.Tile, autoTileIndex *int) (gc.EntitySpec, error) {
	tileRaw, err := GetTile(raws, name)
	if err != nil {
		return gc.EntitySpec{}, err
	}

	entitySpec := gc.EntitySpec{}
	entitySpec.Name = &gc.Name{Name: tileRaw.Name}
	entitySpec.RawID = &gc.RawID{ID: tileRaw.Id}
	entitySpec.Description = &gc.Description{Description: tileRaw.Description}
	entitySpec.GridElement = &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}}

	// SpriteRenderを設定
	sprite := toGCSpriteRender(tileRaw.SpriteRender)
	// オートタイルインデックスが指定されている場合はspriteKeyを動的に生成
	if autoTileIndex != nil {
		sprite.SpriteKey = fmt.Sprintf("%s_%d", tileRaw.SpriteRender.SpriteKey, *autoTileIndex)
	}
	entitySpec.SpriteRender = &sprite

	// BlockPassがtrueの場合は通行を遮断
	if tileRaw.BlockPass {
		entitySpec.BlockPass = &gc.BlockPass{}
	}

	// BlockViewがtrueの場合は視界を遮断
	if tileRaw.BlockView {
		entitySpec.BlockView = &gc.BlockView{}
	}

	entitySpec.Tile = &gc.Tile{}

	// タイル種別によらないので、ここでは初期化するだけ
	entitySpec.TileEnvironment = &gc.TileEnvironment{Shelter: oapi.Outdoor}

	return entitySpec, nil
}

// GetProp は指定された名前の置物の設定を取得する
func GetProp(raws oapi.Raws, name string) (oapi.Prop, error) {
	prop, ok := findByKey(raws.Props, func(p oapi.Prop) string { return p.Id }, name)
	if !ok {
		return oapi.Prop{}, NewKeyNotFoundError(name, "Props")
	}
	return prop, nil
}

// NewPropSpec は指定された名前の置物のEntitySpecを生成する
func NewPropSpec(raws oapi.Raws, name string) (gc.EntitySpec, error) {
	propRaw, err := GetProp(raws, name)
	if err != nil {
		return gc.EntitySpec{}, err
	}

	entitySpec := gc.EntitySpec{}
	entitySpec.Prop = &gc.Prop{}
	entitySpec.Name = &gc.Name{Name: propRaw.Name}
	entitySpec.RawID = &gc.RawID{ID: propRaw.Id}
	entitySpec.Description = &gc.Description{Description: propRaw.Description}

	// SpriteRenderの設定（AnimKeysを含む）
	spriteRender := toGCSpriteRender(propRaw.SpriteRender)
	if animKeys := PtrSlice(propRaw.AnimKeys); len(animKeys) > 0 {
		spriteRender.AnimKeys = animKeys
	}
	entitySpec.SpriteRender = &spriteRender

	if propRaw.BlockPass && propRaw.PassCost != nil {
		return gc.EntitySpec{}, fmt.Errorf("prop '%s': blockPass and passCost cannot both be set; passCost is unnecessary when impassable", name)
	}
	if propRaw.BlockPass {
		entitySpec.BlockPass = &gc.BlockPass{}
	}
	if propRaw.BlockView {
		entitySpec.BlockView = &gc.BlockView{}
	}
	if propRaw.PassCost != nil {
		entitySpec.PassCost = &gc.PassCost{Value: *propRaw.PassCost}
	}
	// 各条件に対応するインタラクションを蓄積する。
	// 1つのPropが複数のインタラクションを持てる
	var interactions []gc.InteractionKind

	if propRaw.Door != nil {
		entitySpec.Door = &gc.Door{
			IsOpen:      false,
			Orientation: gc.DoorOrientationHorizontal,
		}
		interactions = append(interactions, gc.InteractionDoor)
	}

	if propRaw.Hp != nil {
		entitySpec.HP = &gc.HP{Max: *propRaw.Hp, Current: *propRaw.Hp}
		interactions = append(interactions, gc.InteractionMelee)
	}

	entitySpec.LightSource = toGCLightSource(propRaw.LightSource)
	entitySpec.HeatSource = toGCHeatSource(propRaw.HeatSource)
	entitySpec.Bedding = toGCBedding(propRaw.Bedding)

	if propRaw.WarpNextTrigger != nil {
		interactions = append(interactions, gc.InteractionPortalNext)
	}

	if propRaw.WarpPrevTrigger != nil {
		interactions = append(interactions, gc.InteractionPortalPrev)
	}

	if propRaw.Storage != nil {
		mg, err := consts.ParseWeight(propRaw.Storage.MaxWeight)
		if err != nil {
			return gc.EntitySpec{}, fmt.Errorf("container '%s' max weight: %w", name, err)
		}
		entitySpec.WeightCapacity = &gc.WeightCapacity{Max: mg}
		interactions = append(interactions, gc.InteractionStorage)
	}

	if len(interactions) > 0 {
		entitySpec.Interactable = &gc.Interactable{Interactions: interactions}
	}

	return entitySpec, nil
}

// FindDisassembly は指定された名前の prop の分解定義を返す
func FindDisassembly(raws oapi.Raws, name string) (*oapi.Disassembly, bool) {
	prop, ok := findByKey(raws.Props, func(p oapi.Prop) string { return p.Id }, name)
	if !ok || prop.Disassembly == nil {
		return nil, false
	}
	return prop.Disassembly, true
}

// FindTool は指定された名前のアイテムの道具の性質を返す
func FindTool(raws oapi.Raws, name string) (*oapi.Tool, bool) {
	item, ok := findByKey(raws.Items, func(i oapi.Item) string { return i.Id }, name)
	if !ok || item.Tool == nil {
		return nil, false
	}
	return item.Tool, true
}

// GetProfession は指定されたIDの職業データを返す
func GetProfession(raws oapi.Raws, id string) (oapi.Profession, error) {
	prof, ok := findByKey(raws.Professions, func(p oapi.Profession) string { return p.Id }, id)
	if !ok {
		return oapi.Profession{}, NewKeyNotFoundError(id, "Professions")
	}
	return prof, nil
}

// SelectCommandByWeight はコマンドテーブルから重み付きランダム選択する
func SelectCommandByWeight(ct oapi.CommandTable, rng *rand.Rand) (string, error) {
	i, ok := consts.Pick(rng, ct.Entries, func(e oapi.CommandTableEntry) float64 { return e.Weight })
	if !ok {
		return "", nil
	}
	return ct.Entries[i].Id, nil
}

// SelectDropByWeight はドロップテーブルから重み付きランダム選択する。何も落とさないときは空文字を返す。
func SelectDropByWeight(dt oapi.DropTable, rng *rand.Rand) (string, error) {
	weights := make([]float64, 0, len(dt.Entries)+1)
	var nothing float64
	if dt.Nothing != nil {
		nothing = *dt.Nothing
	}
	weights = append(weights, nothing)
	for _, e := range dt.Entries {
		weights = append(weights, e.Weight)
	}
	i, ok := consts.Pick(rng, weights, func(w float64) float64 { return w })
	if !ok || i == 0 {
		return "", nil
	}
	return dt.Entries[i-1].Id, nil
}

// MinDanger は spawn テーブルが引ける最小の危険度。テーブルは1始まり設計で、これ未満は
// 危険度の指定漏れとしてエラーにする。
const MinDanger = 1

// errDangerBelowMinimum は MinDanger 未満の危険度で抽選したときに返す
var errDangerBelowMinimum = errors.New("danger is below the minimum")

// EnemyEntriesAt は敵テーブルのうち、危険度 danger で出る行を返す。
func EnemyEntriesAt(et oapi.EnemyTable, danger int) []oapi.EnemyTableEntry {
	return inDanger(et.Entries, danger, func(e oapi.EnemyTableEntry) (int, int) { return e.MinDanger, e.MaxDanger })
}

// ItemEntriesAt はアイテムテーブルのうち、危険度 danger で出る行を返す。
func ItemEntriesAt(it oapi.ItemTable, danger int) []oapi.ItemTableEntry {
	return inDanger(it.Entries, danger, func(e oapi.ItemTableEntry) (int, int) { return e.MinDanger, e.MaxDanger })
}

func inDanger[T any](entries []T, danger int, rangeOf func(T) (lo, hi int)) []T {
	out := make([]T, 0, len(entries))
	for _, e := range entries {
		if lo, hi := rangeOf(e); lo <= danger && danger <= hi {
			out = append(out, e)
		}
	}
	return out
}

func checkDanger(danger int) error {
	if danger < MinDanger {
		return fmt.Errorf("%w: danger %d, minimum %d; the caller must specify a valid danger", errDangerBelowMinimum, danger, MinDanger)
	}
	return nil
}

// DrawFromItemTable はアイテムテーブルから危険度に合うグループを重みで1つ選び、そのグループから1回ぶんを引く。
// グループの引き方は SelectFromItemGroup に従う。
func DrawFromItemTable(raws oapi.Raws, it oapi.ItemTable, rng *rand.Rand, danger int) ([]DrawnItem, error) {
	if err := checkDanger(danger); err != nil {
		return nil, err
	}
	entries := ItemEntriesAt(it, danger)
	i, ok := consts.Pick(rng, entries, func(e oapi.ItemTableEntry) float64 { return e.Weight })
	if !ok {
		return nil, nil
	}
	return SelectFromItemGroup(raws, entries[i].Id, rng)
}

// SelectEnemyByWeight は敵テーブルから危険度を考慮して重み付きランダム選択する
func SelectEnemyByWeight(et oapi.EnemyTable, rng *rand.Rand, danger int) (string, error) {
	if err := checkDanger(danger); err != nil {
		return "", err
	}
	entries := EnemyEntriesAt(et, danger)
	i, ok := consts.Pick(rng, entries, func(e oapi.EnemyTableEntry) float64 { return e.Weight })
	if !ok {
		return "", nil
	}
	return entries[i].Id, nil
}

// NewConditionDefs は raw の conditions、bodyParts、attackTypes から不調の定義の表を組み立てる。
// 参照の実在は ValidateReferences が確かめる前提で、見つからない部位はエラーにする
func NewConditionDefs(raws oapi.Raws) (gc.ConditionDefs, error) {
	var parts [gc.BodyPartCount]gc.BodyPartDef
	for _, row := range PtrSlice(raws.BodyParts) {
		bp, ok := gc.BodyPartByID(row.Id)
		if !ok {
			return gc.ConditionDefs{}, NewKeyNotFoundError(row.Id, "BodyParts")
		}
		parts[bp] = gc.BodyPartDef{Name: row.Name, HitWeight: row.HitWeight}
	}

	conditions := PtrSlice(raws.Conditions)
	conds := make(map[gc.ConditionType]gc.ConditionDef, len(conditions))
	for _, row := range conditions {
		def := gc.ConditionDef{
			Name:                    row.Name,
			Description:             row.Description,
			PainPerSeverity:         row.PainPerSeverity,
			BodyFuncDropPerSeverity: row.BodyFuncDropPerSeverity,
		}
		if row.Recovery != nil {
			def.Recovery = *row.Recovery
		}
		if row.WorsenPer != nil {
			def.WorsenPer = *row.WorsenPer
		}
		if row.RecoverPer != nil {
			def.RecoverPer = *row.RecoverPer
		}
		if row.BloodDropPerSeverity != nil {
			def.BloodDropPerSeverity = *row.BloodDropPerSeverity
		}
		if row.Cause != nil {
			def.Cause = gc.DeathCause(*row.Cause)
		}
		if row.Onset != nil {
			def.Onset = *row.Onset
		}
		for _, id := range PtrSlice(row.Parts) {
			bp, ok := gc.BodyPartByID(id)
			if !ok {
				return gc.ConditionDefs{}, NewKeyNotFoundError(id, "BodyParts")
			}
			def.Parts = append(def.Parts, bp)
		}
		conds[gc.ConditionType(row.Id)] = def
	}

	attackTypes := PtrSlice(raws.AttackTypes)
	injuries := make(map[string]gc.ConditionType, len(attackTypes))
	for _, row := range attackTypes {
		injuries[string(row.Id)] = gc.ConditionType(row.Injury)
	}
	return gc.NewConditionDefs(conds, parts, injuries), nil
}

// NewSkillDefs は raw の skills、skillCategories、attackTypes からスキルの定義の表を組み立てる。
// 命中の効果倍率は攻撃の種類の射程から、目減りさせる身体機能を決める。
// カテゴリは skillCategories の行の順に、カテゴリ内のスキルは skills の行の順に並べる
func NewSkillDefs(raws oapi.Raws) (gc.SkillDefs, error) {
	rows := PtrSlice(raws.Skills)
	skills := make(map[oapi.SkillId]gc.SkillDef, len(rows))
	proficiencies := make(map[oapi.ProficiencyId]gc.ProficiencyDef)
	idsByCategory := make(map[string][]oapi.SkillId)
	var resists []gc.ResistDef
	for i := range rows {
		row := &rows[i]
		id := row.Id
		skills[id] = gc.SkillDef{
			Name:     row.Name,
			Summary:  row.Summary,
			GainedBy: row.GainedBy,
			Effect:   row.Effect,
			Ability:  row.Ability,
		}
		for _, p := range PtrSlice(row.Proficiencies) {
			proficiencies[p.Id] = gc.ProficiencyDef{Skill: id, PerLevel: p.PerLevel}
			if p.Resists != nil {
				resists = append(resists, gc.ResistDef{Element: *p.Resists, Proficiency: p.Id})
			}
		}
		idsByCategory[row.Category] = append(idsByCategory[row.Category], id)
	}

	catRows := PtrSlice(raws.SkillCategories)
	categories := make([]gc.SkillCategory, 0, len(catRows))
	for _, c := range catRows {
		categories = append(categories, gc.SkillCategory{Name: c.Name, IDs: idsByCategory[c.Id]})
	}
	attackTypes := PtrSlice(raws.AttackTypes)
	weapons := make([]gc.WeaponDef, 0, len(attackTypes))
	for _, row := range attackTypes {
		at, err := gc.ParseAttackType(string(row.Id))
		if err != nil {
			return gc.SkillDefs{}, err
		}
		weapons = append(weapons, gc.WeaponDef{AttackType: at, Skill: row.Skill, Damage: row.Damage, Accuracy: row.Accuracy})
		accuracy, ok := proficiencies[row.Accuracy]
		if !ok {
			return gc.SkillDefs{}, NewKeyNotFoundError(string(row.Accuracy), "Proficiencies")
		}
		accuracy.BodyFunc = gc.BodyFuncManipulation
		if at.Range == gc.AttackRangeRanged {
			accuracy.BodyFunc = gc.BodyFuncSight
		}
		proficiencies[row.Accuracy] = accuracy
	}
	return gc.SkillDefs{Skills: skills, Categories: categories, Proficiencies: proficiencies, Weapons: weapons, Resists: resists}, nil
}

var (
	weatherKinds = map[oapi.WeatherId]gc.WeatherKind{
		oapi.Clear:    gc.WeatherClear,
		oapi.Cloudy:   gc.WeatherCloudy,
		oapi.Rain:     gc.WeatherRain,
		oapi.Snow:     gc.WeatherSnow,
		oapi.Blizzard: gc.WeatherBlizzard,
		oapi.ColdSnap: gc.WeatherColdSnap,
	}
	seasons = map[oapi.SeasonId]gc.Season{
		oapi.Spring: gc.SeasonSpring,
		oapi.Summer: gc.SeasonSummer,
		oapi.Autumn: gc.SeasonAutumn,
		oapi.Winter: gc.SeasonWinter,
	}
	severities = map[oapi.ConditionSeverity]gc.Severity{
		oapi.ConditionSeverityMinor:  gc.SeverityMinor,
		oapi.ConditionSeverityMedium: gc.SeverityMedium,
		oapi.ConditionSeveritySevere: gc.SeveritySevere,
	}
)

// NewTileEnvDefs は raw の shelterEffects、waterEffects、foliageEffects から、囲われの受け方と、水と植生がタイルの気温に足す補正の表を組み立てる
func NewTileEnvDefs(raws oapi.Raws) gc.TileEnvDefs {
	defs := gc.TileEnvDefs{Shelters: map[oapi.Shelter]gc.ShelterDef{}, Water: map[oapi.WaterExposure]int{}, Foliage: map[oapi.Foliage]int{}}
	for _, row := range PtrSlice(raws.ShelterEffects) {
		defs.Shelters[row.Id] = gc.ShelterDef{Name: row.Name, AnchorTemp: row.AnchorTemp, InfluencePercent: row.InfluencePercent}
	}
	for _, row := range PtrSlice(raws.WaterEffects) {
		defs.Water[row.Id] = row.TempModifier
	}
	for _, row := range PtrSlice(raws.FoliageEffects) {
		defs.Foliage[row.Id] = row.TempModifier
	}
	return defs
}

// NewWeatherDefs は raw の weathers から天候の定義の表を組み立てる。id と季節は enum なのでスキーマ検証で尽くされる
func NewWeatherDefs(raws oapi.Raws) gc.WeatherDefs {
	var kinds [gc.NumWeatherKind]gc.WeatherDef
	for _, row := range PtrSlice(raws.Weathers) {
		def := gc.WeatherDef{
			Effect:        gc.WeatherEffect{TempModifier: row.TempModifier, VisionDelta: row.VisionDelta},
			Severity:      row.Severity,
			SpellTurnsMin: row.SpellTurnsMin,
			SpellTurnsMax: row.SpellTurnsMax,
		}
		for _, sw := range row.Seasons {
			def.SeasonWeights[seasons[sw.Id]] = sw.Weight
		}
		kinds[weatherKinds[row.Id]] = def
	}
	return gc.WeatherDefs{Kinds: kinds}
}

// NewNeedDefs は raw の hungerStages と fatigueStages から空腹と疲労の段階の表を組み立てる
func NewNeedDefs(raws oapi.Raws) gc.NeedDefs {
	conditionOf := func(s *oapi.ConditionSeverity) gc.Severity {
		if s == nil {
			return gc.SeverityNone
		}
		return severities[*s]
	}
	hunger := make([]gc.HungerStage, 0, len(PtrSlice(raws.HungerStages)))
	for _, row := range PtrSlice(raws.HungerStages) {
		hunger = append(hunger, gc.HungerStage{Level: row.Id, Name: row.Name, From: row.From, Condition: conditionOf(row.Severity)})
	}
	fatigue := make([]gc.FatigueStage, 0, len(PtrSlice(raws.FatigueStages)))
	for _, row := range PtrSlice(raws.FatigueStages) {
		fatigue = append(fatigue, gc.FatigueStage{Level: row.Id, Name: row.Name, From: row.From, Condition: conditionOf(row.Severity)})
	}
	return gc.NeedDefs{Hunger: hunger, Fatigue: fatigue}
}
