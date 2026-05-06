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

// Master はローデータを管理し、効率的な検索のためのインデックスを提供する
type Master struct {
	Raws              Raws
	ItemIndex         map[string]int
	RecipeIndex       map[string]int
	MemberIndex       map[string]int
	CommandTableIndex map[string]int
	DropTableIndex    map[string]int
	ItemTableIndex    map[string]int
	EnemyTableIndex   map[string]int
	SpriteSheetIndex  map[string]int
	TileIndex         map[string]int
	PropIndex         map[string]int
	ProfessionIndex   map[string]int
}

// Raws は全てのローデータを格納する構造体
type Raws struct {
	Items         []oapi.Item
	Recipes       []oapi.Recipe
	Members       []oapi.Member
	CommandTables []oapi.CommandTable
	DropTables    []oapi.DropTable
	ItemTables    []oapi.ItemTable
	EnemyTables   []oapi.EnemyTable
	SpriteSheets  []oapi.SpriteSheet
	Tiles         []oapi.Tile
	Props         []oapi.Prop
	Professions   []oapi.Profession
}

// LoadFromFile はファイルからローデータを読み込む
func LoadFromFile(path string) (Master, error) {
	bs, err := assets.FS.ReadFile(path)
	if err != nil {
		return Master{}, err
	}
	rw, err := Load(string(bs))
	if err != nil {
		return Master{}, err
	}
	return rw, nil
}

// DecodeRaws はTOML文字列をRaws構造体にデコードする。
// 未知のキーが含まれる場合はエラーを返す
func DecodeRaws(content string) (Raws, error) {
	var raws Raws
	metaData, err := toml.Decode(content, &raws)
	if err != nil {
		return Raws{}, fmt.Errorf("TOML decode error: %w", err)
	}
	if undecoded := metaData.Undecoded(); len(undecoded) > 0 {
		return Raws{}, fmt.Errorf("unknown keys found in TOML: %v", undecoded)
	}
	return raws, nil
}

// Load は文字列からローデータを読み込む
func Load(entityMetadataContent string) (Master, error) {
	raws, err := DecodeRaws(entityMetadataContent)
	if err != nil {
		return Master{}, err
	}

	rw := Master{
		Raws:              raws,
		ItemIndex:         map[string]int{},
		RecipeIndex:       map[string]int{},
		MemberIndex:       map[string]int{},
		CommandTableIndex: map[string]int{},
		DropTableIndex:    map[string]int{},
		ItemTableIndex:    map[string]int{},
		EnemyTableIndex:   map[string]int{},
		SpriteSheetIndex:  map[string]int{},
		TileIndex:         map[string]int{},
		PropIndex:         map[string]int{},
		ProfessionIndex:   map[string]int{},
	}

	for i, item := range rw.Raws.Items {
		rw.ItemIndex[item.Name] = i
	}
	for i, recipe := range rw.Raws.Recipes {
		rw.RecipeIndex[recipe.Name] = i
	}
	for i, member := range rw.Raws.Members {
		rw.MemberIndex[member.Name] = i
	}
	for i, commandTable := range rw.Raws.CommandTables {
		rw.CommandTableIndex[commandTable.Name] = i
	}
	for i, dropTable := range rw.Raws.DropTables {
		rw.DropTableIndex[dropTable.Name] = i
	}
	for i, itemTable := range rw.Raws.ItemTables {
		rw.ItemTableIndex[itemTable.Name] = i
	}
	for i, enemyTable := range rw.Raws.EnemyTables {
		rw.EnemyTableIndex[enemyTable.Name] = i
	}
	for i, spriteSheet := range rw.Raws.SpriteSheets {
		rw.SpriteSheetIndex[spriteSheet.Name] = i
	}
	for i, tile := range rw.Raws.Tiles {
		rw.TileIndex[tile.Name] = i
	}
	for i, prop := range rw.Raws.Props {
		rw.PropIndex[prop.Name] = i
	}
	for i, prof := range rw.Raws.Professions {
		rw.ProfessionIndex[prof.Id] = i
	}

	return rw, nil
}

// toGCSpriteRender はoapi.SpriteRenderからgc.SpriteRenderに変換する
func toGCSpriteRender(s oapi.SpriteRender) gc.SpriteRender {
	return gc.SpriteRender{
		SpriteSheetName: s.SpriteSheetName,
		SpriteKey:       s.SpriteKey,
		Depth:           gc.DepthNum(s.Depth),
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
func parseTargetType(targetGroup, targetNum string) (gc.TargetType, error) {
	if targetGroup == "" {
		return gc.TargetType{}, nil
	}
	if err := gc.TargetGroupType(targetGroup).Valid(); err != nil {
		return gc.TargetType{}, fmt.Errorf("invalid attack target group: %w", err)
	}
	if err := gc.TargetNumType(targetNum).Valid(); err != nil {
		return gc.TargetType{}, fmt.Errorf("invalid attack target num: %w", err)
	}
	return gc.TargetType{
		TargetGroup: gc.TargetGroupType(targetGroup),
		TargetNum:   gc.TargetNumType(targetNum),
	}, nil
}

// parseAttackType はAttackCategory文字列をパース・検証する
func parseAttackType(category string) (gc.AttackType, error) {
	attackType, err := gc.ParseAttackType(category)
	if err != nil {
		return gc.AttackType{}, err
	}
	if err := attackType.Valid(); err != nil {
		return gc.AttackType{}, err
	}
	return attackType, nil
}

// parseMelee はoapi.Meleeからgc.Meleeを生成する
func parseMelee(m *oapi.Melee) (*gc.Melee, error) {
	if err := gc.ElementType(m.Element).Valid(); err != nil {
		return nil, err
	}
	attackType, err := parseAttackType(m.AttackCategory)
	if err != nil {
		return nil, err
	}
	targetType, err := parseTargetType(m.TargetGroup, m.TargetNum)
	if err != nil {
		return nil, err
	}
	return &gc.Melee{
		Accuracy:       int(m.Accuracy),
		Damage:         int(m.Damage),
		AttackCount:    int(m.AttackCount),
		Element:        gc.ElementType(m.Element),
		AttackCategory: attackType,
		Cost:           int(m.Cost),
		TargetType:     targetType,
	}, nil
}

// parseFire はoapi.Fireからgc.Fireを生成する
func parseFire(f *oapi.Fire) (*gc.Fire, error) {
	if err := gc.ElementType(f.Element).Valid(); err != nil {
		return nil, err
	}
	attackType, err := parseAttackType(f.AttackCategory)
	if err != nil {
		return nil, err
	}
	targetType, err := parseTargetType(f.TargetGroup, f.TargetNum)
	if err != nil {
		return nil, err
	}
	return &gc.Fire{
		Accuracy:       int(f.Accuracy),
		Damage:         int(f.Damage),
		AttackCount:    int(f.AttackCount),
		Element:        gc.ElementType(f.Element),
		AttackCategory: attackType,
		Cost:           int(f.Cost),
		TargetType:     targetType,
		Magazine:       int(f.MagazineSize),
		MagazineSize:   int(f.MagazineSize),
		ReloadEffort:   int(f.ReloadEffort),
		AmmoTag:        f.AmmoTag,
	}, nil
}

// newProvidesHealingFromAPI はoapi.ProvidesHealingからProvidesHealingコンポーネントを生成する
func newProvidesHealingFromAPI(h *oapi.ProvidesHealing) (*gc.ProvidesHealing, error) {
	vt := ValueType(h.ValueType)
	if err := vt.Valid(); err != nil {
		return nil, fmt.Errorf("%s: %w", "invalid value type", err)
	}
	switch vt {
	case PercentageType:
		return &gc.ProvidesHealing{Amount: gc.RatioAmount{Ratio: h.Ratio}}, nil
	case NumeralType:
		return &gc.ProvidesHealing{Amount: gc.NumeralAmount{Numeral: int(h.Amount)}}, nil
	default:
		return nil, fmt.Errorf("不明なValueType: %v", vt)
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
		Effort: gc.Pool{Max: int(b.TotalEffort)},
		Skill: &gc.SkillBookEffect{
			TargetSkill:   skillID,
			MaxLevel:      int(b.Skill.MaxLevel),
			RequiredLevel: int(b.Skill.RequiredLevel),
		},
	}, nil
}

// NewItemSpec は指定された名前のアイテムのEntitySpecを生成する
func (rw *Master) NewItemSpec(name string) (gc.EntitySpec, error) {
	itemIdx, ok := rw.ItemIndex[name]
	if !ok {
		return gc.EntitySpec{}, NewKeyNotFoundError(name, "ItemIndex")
	}
	if itemIdx >= len(rw.Raws.Items) {
		return gc.EntitySpec{}, fmt.Errorf("アイテムインデックスが範囲外: %d (長さ: %d)", itemIdx, len(rw.Raws.Items))
	}
	item := rw.Raws.Items[itemIdx]

	entitySpec := gc.EntitySpec{}
	entitySpec.Item = &gc.Item{Count: 1}
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
		AnimKeys:        item.AnimKeys,
		Depth:           gc.DepthNumRug,
	}

	if item.Consumable != nil {
		if err := gc.TargetGroupType(item.Consumable.TargetGroup).Valid(); err != nil {
			return gc.EntitySpec{}, fmt.Errorf("%s: %w", "invalid target group type", err)
		}
		if err := gc.TargetNumType(item.Consumable.TargetNum).Valid(); err != nil {
			return gc.EntitySpec{}, fmt.Errorf("%s: %w", "invalid target num type", err)
		}
		targetType := gc.TargetType{
			TargetGroup: gc.TargetGroupType(item.Consumable.TargetGroup),
			TargetNum:   gc.TargetNumType(item.Consumable.TargetNum),
		}

		if err := gc.UsableSceneType(item.Consumable.UsableScene).Valid(); err != nil {
			return gc.EntitySpec{}, fmt.Errorf("%s: %w", "invalid usable scene type", err)
		}
		entitySpec.Consumable = &gc.Consumable{
			UsableScene: gc.UsableSceneType(item.Consumable.UsableScene),
			TargetType:  targetType,
		}
	}

	if item.ProvidesHealing != nil {
		healing, err := newProvidesHealingFromAPI(item.ProvidesHealing)
		if err != nil {
			return gc.EntitySpec{}, err
		}
		entitySpec.ProvidesHealing = healing
	}
	if item.ProvidesNutrition != nil {
		entitySpec.ProvidesNutrition = &gc.ProvidesNutrition{Amount: int(*item.ProvidesNutrition)}
	}
	if item.InflictsDamage != nil {
		entitySpec.InflictsDamage = &gc.InflictsDamage{Amount: int(*item.InflictsDamage)}
	}

	applyWeaponSpec(item, &entitySpec)

	if item.Ammo != nil {
		entitySpec.Ammo = &gc.Ammo{
			AmmoTag:       item.Ammo.AmmoTag,
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
		if err := gc.EquipmentType(item.Wearable.EquipmentCategory).Valid(); err != nil {
			return gc.EntitySpec{}, err
		}
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

	// すべてのアイテムにInteractableを追加（所持状態に関わらず）
	entitySpec.Interactable = &gc.Interactable{Data: gc.ItemInteraction{}}

	return entitySpec, nil
}

// applyWeaponSpec はItemのWeaponマーカーをEntitySpecに適用する
func applyWeaponSpec(item oapi.Item, spec *gc.EntitySpec) {
	if item.Weapon != nil {
		spec.Weapon = &gc.Weapon{}
	}
}

// NewRecipeSpec は指定された名前のレシピのEntitySpecを生成する
func (rw *Master) NewRecipeSpec(name string) (gc.EntitySpec, error) {
	recipeIdx, ok := rw.RecipeIndex[name]
	if !ok {
		return gc.EntitySpec{}, NewKeyNotFoundError(name, "RecipeIndex")
	}
	if recipeIdx >= len(rw.Raws.Recipes) {
		return gc.EntitySpec{}, fmt.Errorf("レシピインデックスが範囲外: %d (長さ: %d)", recipeIdx, len(rw.Raws.Recipes))
	}
	recipe := rw.Raws.Recipes[recipeIdx]
	entitySpec := gc.EntitySpec{}
	entitySpec.Name = &gc.Name{Name: recipe.Name}
	entitySpec.Recipe = &gc.Recipe{}
	for _, input := range recipe.Inputs {
		entitySpec.Recipe.Inputs = append(entitySpec.Recipe.Inputs, gc.RecipeInput{Name: input.Name, Amount: int(input.Amount)})
	}

	// 説明文や分類のため、マッチしたitemの定義から持ってくる
	// マスターデータのため位置を指定しない
	itemSpec, err := rw.NewItemSpec(recipe.Name)
	if err != nil {
		return gc.EntitySpec{}, fmt.Errorf("%s: %w", "failed to generate item for recipe", err)
	}
	entitySpec.Description = &gc.Description{Description: itemSpec.Description.Description}
	if itemSpec.Weapon != nil {
		entitySpec.Weapon = itemSpec.Weapon
	}
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
// 武器はマスターデータとして位置なしで生成される
func (rw *Master) NewWeaponSpec(name string) (gc.EntitySpec, error) {
	// 武器はアイテムの一種なので、ItemIndexから検索
	_, ok := rw.ItemIndex[name]
	if !ok {
		return gc.EntitySpec{}, NewKeyNotFoundError(name, "ItemIndex")
	}

	// マスターデータのため位置を指定しない
	itemSpec, err := rw.NewItemSpec(name)
	if err != nil {
		return gc.EntitySpec{}, fmt.Errorf("failed to generate weapon spec: %w", err)
	}

	// Weaponコンポーネントがない場合はエラー
	if itemSpec.Weapon == nil {
		return gc.EntitySpec{}, fmt.Errorf("%s is not a weapon (Weapon component missing)", name)
	}

	return itemSpec, nil
}

// NewMemberSpec は指定された名前のメンバーのEntitySpecを生成する
func (rw *Master) NewMemberSpec(name string) (gc.EntitySpec, error) {
	memberIdx, ok := rw.MemberIndex[name]
	if !ok {
		return gc.EntitySpec{}, fmt.Errorf("キーが存在しない: %s", name)
	}
	if memberIdx >= len(rw.Raws.Members) {
		return gc.EntitySpec{}, fmt.Errorf("メンバーインデックスが範囲外: %d (長さ: %d)", memberIdx, len(rw.Raws.Members))
	}
	member := rw.Raws.Members[memberIdx]

	entitySpec := gc.EntitySpec{}
	entitySpec.Name = &gc.Name{Name: member.Name}
	entitySpec.TurnBased = &gc.TurnBased{AP: gc.Pool{Current: 100, Max: 100}} // TODO: Abilitiesから計算する
	entitySpec.SpriteRender = &gc.SpriteRender{
		SpriteSheetName: member.SpriteSheetName,
		SpriteKey:       member.SpriteKey,
		AnimKeys:        member.AnimKeys,
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
	entitySpec.Pools = &gc.Pools{}
	if member.Player != nil && *member.Player {
		entitySpec.Player = &gc.Player{}
	}

	if member.CommandTableName != "" {
		commandTableIdx, ok := rw.CommandTableIndex[member.CommandTableName]
		if ok && commandTableIdx < len(rw.Raws.CommandTables) {
			commandTable := rw.Raws.CommandTables[commandTableIdx]
			entitySpec.CommandTable = &gc.CommandTable{Name: commandTable.Name}
		}
	}

	if member.DropTableName != "" {
		dropTableIdx, ok := rw.DropTableIndex[member.DropTableName]
		if ok && dropTableIdx < len(rw.Raws.DropTables) {
			dropTable := rw.Raws.DropTables[dropTableIdx]
			entitySpec.DropTable = &gc.DropTable{Name: dropTable.Name}
		}
	}

	entitySpec.LightSource = toGCLightSource(member.LightSource)

	// 派閥タイプの処理
	if member.FactionType != "" {
		switch member.FactionType {
		case gc.FactionAlly.String():
			entitySpec.FactionType = &gc.FactionAlly
		case gc.FactionEnemy.String():
			entitySpec.FactionType = &gc.FactionEnemy
		case gc.FactionNeutral.String():
			entitySpec.FactionType = &gc.FactionNeutral
		default:
			return gc.EntitySpec{}, fmt.Errorf("無効な派閥タイプ '%s' が指定されています: %s", member.FactionType, name)
		}
	}

	if member.Dialog != nil {
		entitySpec.Dialog = &gc.Dialog{
			MessageKey: member.Dialog.MessageKey,
		}
		entitySpec.Interactable = &gc.Interactable{Data: gc.TalkInteraction{}}
	}

	return entitySpec, nil
}

// NewPlayerSpec は指定された名前のプレイヤーのEntitySpecを生成する
func (rw *Master) NewPlayerSpec(name string) (gc.EntitySpec, error) {
	entitySpec, err := rw.NewMemberSpec(name)
	if err != nil {
		return gc.EntitySpec{}, err
	}
	entitySpec.FactionType = &gc.FactionAlly
	entitySpec.Player = &gc.Player{}
	entitySpec.Hunger = gc.NewHunger()
	return entitySpec, nil
}

// NewEnemySpec は指定された名前の敵のEntitySpecを生成する
func (rw *Master) NewEnemySpec(name string) (gc.EntitySpec, error) {
	entitySpec, err := rw.NewMemberSpec(name)
	if err != nil {
		return gc.EntitySpec{}, err
	}
	entitySpec.FactionType = &gc.FactionEnemy

	return entitySpec, nil
}

// GetCommandTable は指定された名前のコマンドテーブルを取得する
func (rw *Master) GetCommandTable(name string) (oapi.CommandTable, error) {
	ctIdx, ok := rw.CommandTableIndex[name]
	if !ok {
		return oapi.CommandTable{}, fmt.Errorf("キーが存在しない: %s", name)
	}
	if ctIdx >= len(rw.Raws.CommandTables) {
		return oapi.CommandTable{}, fmt.Errorf("コマンドテーブルインデックスが範囲外: %d (長さ: %d)", ctIdx, len(rw.Raws.CommandTables))
	}
	return rw.Raws.CommandTables[ctIdx], nil
}

// GetDropTable は指定された名前のドロップテーブルを取得する
func (rw *Master) GetDropTable(name string) (oapi.DropTable, error) {
	dtIdx, ok := rw.DropTableIndex[name]
	if !ok {
		return oapi.DropTable{}, fmt.Errorf("キーが存在しない: %s", name)
	}
	if dtIdx >= len(rw.Raws.DropTables) {
		return oapi.DropTable{}, fmt.Errorf("ドロップテーブルインデックスが範囲外: %d (長さ: %d)", dtIdx, len(rw.Raws.DropTables))
	}
	return rw.Raws.DropTables[dtIdx], nil
}

// GetItemTable は指定された名前のアイテムテーブルを取得する
func (rw *Master) GetItemTable(name string) (oapi.ItemTable, error) {
	itIdx, ok := rw.ItemTableIndex[name]
	if !ok {
		return oapi.ItemTable{}, fmt.Errorf("キーが存在しない: %s", name)
	}
	if itIdx >= len(rw.Raws.ItemTables) {
		return oapi.ItemTable{}, fmt.Errorf("アイテムテーブルインデックスが範囲外: %d (長さ: %d)", itIdx, len(rw.Raws.ItemTables))
	}
	return rw.Raws.ItemTables[itIdx], nil
}

// GetEnemyTable は指定された名前の敵テーブルを取得する
func (rw *Master) GetEnemyTable(name string) (oapi.EnemyTable, error) {
	etIdx, ok := rw.EnemyTableIndex[name]
	if !ok {
		return oapi.EnemyTable{}, fmt.Errorf("キーが存在しない: %s", name)
	}
	if etIdx >= len(rw.Raws.EnemyTables) {
		return oapi.EnemyTable{}, fmt.Errorf("敵テーブルインデックスが範囲外: %d (長さ: %d)", etIdx, len(rw.Raws.EnemyTables))
	}
	return rw.Raws.EnemyTables[etIdx], nil
}

// GetTile は指定された名前のタイルを取得する
// 計画段階でタイルの性質（Walkableなど）を参照する場合に使用する
func (rw *Master) GetTile(name string) (oapi.Tile, error) {
	tileIdx, ok := rw.TileIndex[name]
	if !ok {
		return oapi.Tile{}, NewKeyNotFoundError(name, "TileIndex")
	}
	if tileIdx >= len(rw.Raws.Tiles) {
		return oapi.Tile{}, fmt.Errorf("タイルインデックスが範囲外: %d (長さ: %d)", tileIdx, len(rw.Raws.Tiles))
	}
	return rw.Raws.Tiles[tileIdx], nil
}

// NewTileSpec は指定された名前のタイルのEntitySpecを生成する
// 実際にエンティティを生成する際に使用する
func (rw *Master) NewTileSpec(name string, x, y consts.Tile, autoTileIndex *int) (gc.EntitySpec, error) {
	tileRaw, err := rw.GetTile(name)
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

	// タイル種別によらないので、ここでは初期化するだけ
	entitySpec.TileTemperature = &gc.TileTemperature{}

	return entitySpec, nil
}

// GetProp は指定された名前の置物の設定を取得する
func (rw *Master) GetProp(name string) (oapi.Prop, error) {
	propIdx, ok := rw.PropIndex[name]
	if !ok {
		return oapi.Prop{}, NewKeyNotFoundError(name, "PropIndex")
	}
	if propIdx >= len(rw.Raws.Props) {
		return oapi.Prop{}, fmt.Errorf("置物インデックスが範囲外: %d (長さ: %d)", propIdx, len(rw.Raws.Props))
	}
	return rw.Raws.Props[propIdx], nil
}

// NewPropSpec は指定された名前の置物のEntitySpecを生成する
func (rw *Master) NewPropSpec(name string) (gc.EntitySpec, error) {
	propRaw, err := rw.GetProp(name)
	if err != nil {
		return gc.EntitySpec{}, err
	}

	entitySpec := gc.EntitySpec{}
	entitySpec.Prop = &gc.Prop{}
	entitySpec.Name = &gc.Name{Name: propRaw.Name}
	entitySpec.Description = &gc.Description{Description: propRaw.Description}

	// SpriteRenderの設定（AnimKeysを含む）
	spriteRender := toGCSpriteRender(propRaw.SpriteRender)
	if len(propRaw.AnimKeys) > 0 {
		spriteRender.AnimKeys = propRaw.AnimKeys
	}
	entitySpec.SpriteRender = &spriteRender

	if propRaw.BlockPass {
		entitySpec.BlockPass = &gc.BlockPass{}
	}
	if propRaw.BlockView {
		entitySpec.BlockView = &gc.BlockView{}
	}

	entitySpec.LightSource = toGCLightSource(propRaw.LightSource)

	if propRaw.Door != nil {
		// Door componentを追加（向きは初期値、spawn時に設定される）
		entitySpec.Door = &gc.Door{
			IsOpen:      false,
			Orientation: gc.DoorOrientationHorizontal,
		}
		// 扉は相互作用可能
		entitySpec.Interactable = &gc.Interactable{Data: gc.DoorInteraction{}}
	}

	// 扉ロックトリガー
	if propRaw.DoorLockTrigger != nil {
		entitySpec.Interactable = &gc.Interactable{Data: gc.DoorLockInteraction{}}
	}

	// 次階層ワープトリガー
	if propRaw.WarpNextTrigger != nil {
		entitySpec.Interactable = &gc.Interactable{
			Data: gc.PortalInteraction{PortalType: gc.PortalTypeNext},
		}
	}

	// 脱出ワープトリガー
	if propRaw.WarpEscapeTrigger != nil {
		entitySpec.Interactable = &gc.Interactable{
			Data: gc.PortalInteraction{PortalType: gc.PortalTypeTown},
		}
	}

	// ダンジョン選択ゲートトリガー
	if propRaw.DungeonGateTrigger != nil {
		entitySpec.Interactable = &gc.Interactable{
			Data: gc.DungeonGateInteraction{},
		}
	}

	return entitySpec, nil
}

// GetProfession は指定されたIDの職業データを返す
func (rw *Master) GetProfession(id string) (oapi.Profession, error) {
	idx, ok := rw.ProfessionIndex[id]
	if !ok {
		return oapi.Profession{}, NewKeyNotFoundError(id, "ProfessionIndex")
	}
	if idx >= len(rw.Raws.Professions) {
		return oapi.Profession{}, fmt.Errorf("職業インデックスが範囲外: %d (長さ: %d)", idx, len(rw.Raws.Professions))
	}
	return rw.Raws.Professions[idx], nil
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

// SelectItemByWeight はアイテムテーブルから深度を考慮して重み付きランダム選択する
func SelectItemByWeight(it oapi.ItemTable, rng *rand.Rand, depth int) (string, error) {
	filtered := make([]oapi.ItemTableEntry, 0, len(it.Entries))
	for _, entry := range it.Entries {
		if depth < int(entry.MinDepth) || depth > int(entry.MaxDepth) {
			continue
		}
		filtered = append(filtered, entry)
	}

	return SelectByWeightFunc(
		filtered,
		func(e oapi.ItemTableEntry) float64 { return e.Weight },
		func(e oapi.ItemTableEntry) string { return e.ItemName },
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
