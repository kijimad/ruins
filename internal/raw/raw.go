package raw

import (
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
		return oapi.Raws{}, fmt.Errorf("ローデータの検証に失敗(%s): %w", path, err)
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
	item, ok := findByKey(raws.Items, func(i oapi.Item) string { return i.Name }, name)
	if !ok {
		return oapi.Item{}, NewKeyNotFoundError(name, "Items")
	}
	return item, nil
}

// FindMember は指定された名前のメンバー定義を検索する
func FindMember(raws oapi.Raws, name string) (oapi.Member, error) {
	member, ok := findByKey(raws.Members, func(m oapi.Member) string { return m.Name }, name)
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

// parseTargetType はTargetGroup/TargetNumの文字列ペアをパースする
// enum値の妥当性はOpenAPIスキーマで検証済み
func parseTargetType(targetGroup oapi.TargetGroup, targetNum oapi.TargetNum) gc.TargetType {
	if targetGroup == "" {
		return gc.TargetType{}
	}
	return gc.TargetType{
		TargetGroup: gc.TargetGroupType(string(targetGroup)),
		TargetNum:   gc.TargetNumType(string(targetNum)),
	}
}

// parseMelee はoapi.Meleeからgc.Meleeを生成する
func parseMelee(m *oapi.Melee) (*gc.Melee, error) {
	attackType, err := gc.ParseAttackType(string(m.AttackCategory))
	if err != nil {
		return nil, err
	}
	return &gc.Melee{
		Accuracy:       int(m.Accuracy),
		Damage:         int(m.Damage),
		AttackCount:    int(m.AttackCount),
		Element:        gc.ElementType(string(m.Element)),
		AttackCategory: attackType,
		Cost:           int(m.Cost),
		TargetType:     parseTargetType(m.TargetGroup, m.TargetNum),
	}, nil
}

// parseFire はoapi.Fireからgc.Fireを生成する
func parseFire(f *oapi.Fire) (*gc.Fire, error) {
	attackType, err := gc.ParseAttackType(string(f.AttackCategory))
	if err != nil {
		return nil, err
	}
	var ammoTag string
	if f.AmmoTag != nil {
		ammoTag = string(*f.AmmoTag)
	}
	return &gc.Fire{
		Accuracy:       int(f.Accuracy),
		Damage:         int(f.Damage),
		AttackCount:    int(f.AttackCount),
		Element:        gc.ElementType(string(f.Element)),
		AttackCategory: attackType,
		Cost:           int(f.Cost),
		TargetType:     parseTargetType(f.TargetGroup, f.TargetNum),
		Magazine:       int(f.MagazineSize),
		MagazineSize:   int(f.MagazineSize),
		ReloadEffort:   int(f.ReloadEffort),
		AmmoTag:        ammoTag,
	}, nil
}

// newProvidesHealingFromAPI はoapi.ProvidesHealingからProvidesHealingコンポーネントを生成する
// enum値の妥当性はOpenAPIスキーマで検証済み
func newProvidesHealingFromAPI(h *oapi.ProvidesHealing) *gc.ProvidesHealing {
	switch h.ValueType {
	case oapi.PERCENTAGE:
		return &gc.ProvidesHealing{Amount: gc.RatioAmount{Ratio: h.Ratio}}
	default:
		return &gc.ProvidesHealing{Amount: gc.NumeralAmount{Numeral: int(h.Amount)}}
	}
}

// newBookFromAPI はoapi.Bookからgc.Bookを生成する
func newBookFromAPI(b *oapi.Book) (*gc.Book, error) {
	if b.Skill == nil {
		return nil, fmt.Errorf("BookにSkillの指定が必要です")
	}

	skillID := gc.SkillID(b.Skill.TargetSkill)
	if !gc.HasSkillName(skillID) {
		return nil, fmt.Errorf("未定義のスキルID: %q", b.Skill.TargetSkill)
	}

	return &gc.Book{
		Effort: gc.IntPool{Max: int(b.TotalEffort)},
		Skill: &gc.SkillBookEffect{
			TargetSkill:   skillID,
			MaxLevel:      int(b.Skill.MaxLevel),
			RequiredLevel: int(b.Skill.RequiredLevel),
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
	entitySpec.Name = &gc.Name{Name: item.Name}
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
			UsableScene: gc.UsableSceneType(item.Consumable.UsableScene),
			TargetType: gc.TargetType{
				TargetGroup: gc.TargetGroupType(item.Consumable.TargetGroup),
				TargetNum:   gc.TargetNumType(item.Consumable.TargetNum),
			},
		}
	}

	if item.ProvidesHealing != nil {
		entitySpec.ProvidesHealing = newProvidesHealingFromAPI(item.ProvidesHealing)
	}
	if item.ProvidesNutrition != nil {
		entitySpec.ProvidesNutrition = &gc.ProvidesNutrition{Amount: int(*item.ProvidesNutrition)}
	}
	if item.InflictsDamage != nil {
		entitySpec.InflictsDamage = &gc.InflictsDamage{Amount: int(*item.InflictsDamage)}
	}

	if item.Ammo != nil {
		var ammoAmmoTag string
		if item.Ammo.AmmoTag != nil {
			ammoAmmoTag = string(*item.Ammo.AmmoTag)
		}
		entitySpec.Ammo = &gc.Ammo{
			AmmoTag:       ammoAmmoTag,
			DamageBonus:   int(item.Ammo.DamageBonus),
			AccuracyBonus: int(item.Ammo.AccuracyBonus),
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
			Vitality:  int(item.EquipBonus.Vitality),
			Strength:  int(item.EquipBonus.Strength),
			Sensation: int(item.EquipBonus.Sensation),
			Dexterity: int(item.EquipBonus.Dexterity),
			Agility:   int(item.EquipBonus.Agility),
		}
	}

	if item.Wearable != nil {
		entitySpec.Wearable = &gc.Wearable{
			Defense:           int(item.Wearable.Defense),
			EquipmentCategory: gc.EquipmentType(item.Wearable.EquipmentCategory),
			EquipBonus:        bonus,
			InsulationCold:    int(item.Wearable.InsulationCold),
			InsulationHeat:    int(item.Wearable.InsulationHeat),
		}
	}

	entitySpec.Value = &gc.Value{Value: int(item.Value)}

	if item.Weight != nil {
		entitySpec.Weight = &gc.Weight{Kg: *item.Weight}
	}

	// Stackableフラグがtrueの場合は空のStackableコンポーネントを追加
	if item.Stackable != nil && *item.Stackable {
		entitySpec.Stackable = &gc.Stackable{}
	}

	if item.Book != nil {
		book, err := newBookFromAPI(item.Book)
		if err != nil {
			return gc.EntitySpec{}, err
		}
		entitySpec.Book = book
	}

	if item.Material != nil && *item.Material {
		entitySpec.Material = &gc.Material{}
	}

	// すべてのアイテムにInteractableを追加（所持状態に関わらず）
	entitySpec.Interactable = &gc.Interactable{Interactions: []gc.InteractionData{gc.ItemInteraction{}}}

	return entitySpec, nil
}

// NewRecipeSpec は指定された名前のレシピのEntitySpecを生成する
func NewRecipeSpec(raws oapi.Raws, name string) (gc.EntitySpec, error) {
	recipe, ok := findByKey(raws.Recipes, func(r oapi.Recipe) string { return r.Name }, name)
	if !ok {
		return gc.EntitySpec{}, NewKeyNotFoundError(name, "Recipes")
	}

	entitySpec := gc.EntitySpec{}
	entitySpec.Name = &gc.Name{Name: recipe.Name}
	entitySpec.Recipe = &gc.Recipe{}
	for _, input := range recipe.Inputs {
		entitySpec.Recipe.Inputs = append(entitySpec.Recipe.Inputs, gc.RecipeInput{Name: input.Name, Amount: int(input.Amount)})
	}

	// 説明文や分類のため、マッチしたitemの定義から持ってくる
	itemSpec, err := NewItemSpec(raws, recipe.Name)
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
	member, ok := findByKey(raws.Members, func(m oapi.Member) string { return m.Name }, name)
	if !ok {
		return gc.EntitySpec{}, fmt.Errorf("キーが存在しない: %s", name)
	}

	entitySpec := gc.EntitySpec{}
	entitySpec.Name = &gc.Name{Name: member.Name}
	entitySpec.TurnBased = &gc.TurnBased{AP: gc.IntPool{Current: 100, Max: 100}} // TODO: Abilitiesから計算する
	entitySpec.SpriteRender = &gc.SpriteRender{
		SpriteSheetName: member.SpriteSheetName,
		SpriteKey:       member.SpriteKey,
		AnimKeys:        PtrSlice(member.AnimKeys),
		Depth:           gc.DepthNumPlayer,
	}
	entitySpec.Abilities = &gc.Abilities{
		Vitality:  gc.Ability{Base: int(member.Abilities.Vitality)},
		Strength:  gc.Ability{Base: int(member.Abilities.Strength)},
		Sensation: gc.Ability{Base: int(member.Abilities.Sensation)},
		Dexterity: gc.Ability{Base: int(member.Abilities.Dexterity)},
		Agility:   gc.Ability{Base: int(member.Abilities.Agility)},
		Defense:   gc.Ability{Base: int(member.Abilities.Defense)},
	}
	entitySpec.HP = &gc.HP{}
	entitySpec.WeightCapacity = &gc.WeightCapacity{}
	if member.Player != nil && *member.Player {
		entitySpec.Player = &gc.Player{}
	}

	if member.CommandTableName != nil && *member.CommandTableName != "" {
		ct, err := GetCommandTable(raws, *member.CommandTableName)
		if err == nil {
			entitySpec.CommandTable = &gc.CommandTable{Name: ct.Name}
		}
	}

	if member.DropTableName != nil && *member.DropTableName != "" {
		dt, err := GetDropTable(raws, *member.DropTableName)
		if err == nil {
			entitySpec.DropTable = &gc.DropTable{Name: dt.Name}
		}
	}

	entitySpec.LightSource = toGCLightSource(member.LightSource)

	// 派閥タイプの処理
	if member.FactionType != nil && string(*member.FactionType) != "" {
		switch string(*member.FactionType) {
		case gc.FactionAlly.String():
			entitySpec.FactionType = &gc.FactionAlly
		case gc.FactionEnemy.String():
			entitySpec.FactionType = &gc.FactionEnemy
		case gc.FactionNeutral.String():
			entitySpec.FactionType = &gc.FactionNeutral
		default:
			return gc.EntitySpec{}, fmt.Errorf("無効な派閥タイプ '%s' が指定されています: %s", *member.FactionType, name)
		}
	}

	// 態度タイプの処理
	if member.Disposition != nil && string(*member.Disposition) != "" {
		dt := gc.DispositionType(*member.Disposition)
		entitySpec.Disposition = &gc.Disposition{Default: dt, Current: dt}
	}

	// 移動パターンの処理
	if member.MovementPattern != nil && string(*member.MovementPattern) != "" {
		mp := gc.MovementPattern(*member.MovementPattern)
		entitySpec.MovementPattern = &mp
	}

	if member.Dialog != nil {
		entitySpec.Dialog = &gc.Dialog{
			MessageKey: member.Dialog.MessageKey,
		}
		entitySpec.Interactable = &gc.Interactable{Interactions: []gc.InteractionData{gc.TalkInteraction{}}}
	}

	return entitySpec, nil
}

// NewPlayerSpec は指定された名前のプレイヤーのEntitySpecを生成する
func NewPlayerSpec(raws oapi.Raws, name string) (gc.EntitySpec, error) {
	entitySpec, err := NewMemberSpec(raws, name)
	if err != nil {
		return gc.EntitySpec{}, err
	}
	entitySpec.FactionType = &gc.FactionAlly
	entitySpec.Player = &gc.Player{}
	entitySpec.Hunger = gc.NewHunger()
	return entitySpec, nil
}

// NewEnemySpec は指定された名前の敵のEntitySpecを生成する
func NewEnemySpec(raws oapi.Raws, name string) (gc.EntitySpec, error) {
	entitySpec, err := NewMemberSpec(raws, name)
	if err != nil {
		return gc.EntitySpec{}, err
	}
	entitySpec.FactionType = &gc.FactionEnemy

	return entitySpec, nil
}

// GetCommandTable は指定された名前のコマンドテーブルを取得する
func GetCommandTable(raws oapi.Raws, name string) (oapi.CommandTable, error) {
	ct, ok := findByKey(raws.CommandTables, func(c oapi.CommandTable) string { return c.Name }, name)
	if !ok {
		return oapi.CommandTable{}, fmt.Errorf("キーが存在しない: %s", name)
	}
	return ct, nil
}

// GetDropTable は指定された名前のドロップテーブルを取得する
func GetDropTable(raws oapi.Raws, name string) (oapi.DropTable, error) {
	dt, ok := findByKey(raws.DropTables, func(d oapi.DropTable) string { return d.Name }, name)
	if !ok {
		return oapi.DropTable{}, fmt.Errorf("キーが存在しない: %s", name)
	}
	return dt, nil
}

// GetItemGroup は指定された名前のアイテムグループを取得する
func GetItemGroup(raws oapi.Raws, name string) (oapi.ItemGroup, error) {
	ig, ok := findByKey(raws.ItemGroups, func(g oapi.ItemGroup) string { return g.Name }, name)
	if !ok {
		return oapi.ItemGroup{}, fmt.Errorf("アイテムグループが存在しない: %s", name)
	}
	return ig, nil
}

// GetItemTable は指定された名前のアイテムテーブルを取得する
func GetItemTable(raws oapi.Raws, name string) (oapi.ItemTable, error) {
	it, ok := findByKey(raws.ItemTables, func(t oapi.ItemTable) string { return t.Name }, name)
	if !ok {
		return oapi.ItemTable{}, fmt.Errorf("キーが存在しない: %s", name)
	}
	return it, nil
}

// GetEnemyTable は指定された名前の敵テーブルを取得する
func GetEnemyTable(raws oapi.Raws, name string) (oapi.EnemyTable, error) {
	et, ok := findByKey(raws.EnemyTables, func(t oapi.EnemyTable) string { return t.Name }, name)
	if !ok {
		return oapi.EnemyTable{}, fmt.Errorf("キーが存在しない: %s", name)
	}
	return et, nil
}

// GetTile は指定された名前のタイルを取得する
// 計画段階でタイルの性質（Walkableなど）を参照する場合に使用する
func GetTile(raws oapi.Raws, name string) (oapi.Tile, error) {
	tile, ok := findByKey(raws.Tiles, func(t oapi.Tile) string { return t.Name }, name)
	if !ok {
		return oapi.Tile{}, NewKeyNotFoundError(name, "Tiles")
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
	entitySpec.Description = &gc.Description{Description: tileRaw.Description}
	entitySpec.GridElement = &gc.GridElement{X: x, Y: y}

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
	entitySpec.TileTemperature = &gc.TileTemperature{}

	return entitySpec, nil
}

// GetProp は指定された名前の置物の設定を取得する
func GetProp(raws oapi.Raws, name string) (oapi.Prop, error) {
	prop, ok := findByKey(raws.Props, func(p oapi.Prop) string { return p.Name }, name)
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
	entitySpec.Description = &gc.Description{Description: propRaw.Description}

	// SpriteRenderの設定（AnimKeysを含む）
	spriteRender := toGCSpriteRender(propRaw.SpriteRender)
	if animKeys := PtrSlice(propRaw.AnimKeys); len(animKeys) > 0 {
		spriteRender.AnimKeys = animKeys
	}
	entitySpec.SpriteRender = &spriteRender

	if propRaw.BlockPass && propRaw.PassCost != nil {
		return gc.EntitySpec{}, fmt.Errorf("prop '%s': blockPassとpassCostは同時に設定できません。通行不可ならpassCostは不要です", name)
	}
	if propRaw.BlockPass {
		entitySpec.BlockPass = &gc.BlockPass{}
	}
	if propRaw.BlockView {
		entitySpec.BlockView = &gc.BlockView{}
	}
	if propRaw.PassCost != nil {
		entitySpec.PassCost = &gc.PassCost{Value: int(*propRaw.PassCost)}
	}
	// 各条件に対応するインタラクションを蓄積する。
	// 1つのPropが複数のインタラクションを持てる
	var interactions []gc.InteractionData

	if propRaw.Hp != nil {
		hp := int(*propRaw.Hp)
		entitySpec.HP = &gc.HP{Max: hp, Current: hp}
		interactions = append(interactions, gc.MeleeInteraction{})
	}

	entitySpec.LightSource = toGCLightSource(propRaw.LightSource)

	if propRaw.Door != nil {
		entitySpec.Door = &gc.Door{
			IsOpen:      false,
			Orientation: gc.DoorOrientationHorizontal,
		}
		interactions = append(interactions, gc.DoorInteraction{})
	}

	if propRaw.DoorLockTrigger != nil {
		interactions = append(interactions, gc.DoorLockInteraction{})
	}

	if propRaw.WarpNextTrigger != nil {
		interactions = append(interactions, gc.PortalInteraction{PortalType: gc.PortalTypeNext})
	}

	if propRaw.WarpEscapeTrigger != nil {
		interactions = append(interactions, gc.PortalInteraction{PortalType: gc.PortalTypeTown})
	}

	if propRaw.DungeonGateTrigger != nil {
		interactions = append(interactions, gc.DungeonGateInteraction{})
	}

	if propRaw.Storage != nil {
		entitySpec.WeightCapacity = &gc.WeightCapacity{
			Max: propRaw.Storage.MaxWeight,
		}
		interactions = append(interactions, gc.StorageInteraction{})
	}

	if len(interactions) > 0 {
		entitySpec.Interactable = &gc.Interactable{Interactions: interactions}
	}

	return entitySpec, nil
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
	return SelectByWeightFunc(
		ct.Entries,
		func(e oapi.CommandTableEntry) float64 { return e.Weight },
		func(e oapi.CommandTableEntry) string { return e.Weapon },
		rng,
	)
}

// SelectDropByWeight はドロップテーブルから重み付きランダム選択する
func SelectDropByWeight(dt oapi.DropTable, rng *rand.Rand) (string, error) {
	return SelectByWeightFunc(
		dt.Entries,
		func(e oapi.DropTableEntry) float64 { return e.Weight },
		func(e oapi.DropTableEntry) string { return e.Material },
		rng,
	)
}

// SelectItemByWeight はアイテムテーブルから深度を考慮してグループ経由で重み付きランダム選択する
// テーブルエントリからグループを選び、グループ内からアイテムを選択して返す
func SelectItemByWeight(raws oapi.Raws, it oapi.ItemTable, rng *rand.Rand, depth int) (string, error) {
	filtered := make([]oapi.ItemTableEntry, 0, len(it.Entries))
	for _, entry := range it.Entries {
		if depth < int(entry.MinDepth) || depth > int(entry.MaxDepth) {
			continue
		}
		filtered = append(filtered, entry)
	}

	// テーブルエントリからグループ名を重み付き選択する
	groupName, err := SelectByWeightFunc(
		filtered,
		func(e oapi.ItemTableEntry) float64 { return e.Weight },
		func(e oapi.ItemTableEntry) string { return e.GroupName },
		rng,
	)
	if err != nil || groupName == "" {
		return "", err
	}

	// グループからアイテムを選択する
	group, err := GetItemGroup(raws, groupName)
	if err != nil {
		return "", err
	}
	return SelectByWeightFunc(
		group.Entries,
		func(e oapi.ItemGroupEntry) float64 { return e.Weight },
		func(e oapi.ItemGroupEntry) string { return e.ItemName },
		rng,
	)
}

// SelectEnemyByWeight は敵テーブルから深度を考慮して重み付きランダム選択する
func SelectEnemyByWeight(et oapi.EnemyTable, rng *rand.Rand, depth int) (string, error) {
	filtered := make([]oapi.EnemyTableEntry, 0, len(et.Entries))
	for _, entry := range et.Entries {
		if depth < int(entry.MinDepth) || depth > int(entry.MaxDepth) {
			continue
		}
		filtered = append(filtered, entry)
	}

	return SelectByWeightFunc(
		filtered,
		func(e oapi.EnemyTableEntry) float64 { return e.Weight },
		func(e oapi.EnemyTableEntry) string { return e.EnemyName },
		rng,
	)
}
