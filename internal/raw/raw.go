package raw

import (
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/kijimaD/ruins/assets"
	gc "github.com/kijimaD/ruins/internal/components"
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
}

// Raws は全てのローデータを格納する構造体
type Raws struct {
	Items         []Item
	Recipes       []Recipe
	Members       []Member
	CommandTables []CommandTable
	DropTables    []DropTable
	ItemTables    []ItemTable
	EnemyTables   []EnemyTable
	SpriteSheets  []SpriteSheet
	Tiles         []TileRaw
	Props         []PropRaw
}

// Item はアイテムのローデータ
type Item struct {
	Name              string
	Description       string
	SpriteSheetName   string
	SpriteKey         string
	AnimKeys          []string
	Value             *int
	Weight            *float64 // 重量(kg)
	InflictsDamage    *int
	ProvidesNutrition *int  // 空腹度回復量
	Stackable         *bool // スタック可能かどうか
	Consumable        *Consumable
	ProvidesHealing   *ProvidesHealing
	Wearable          *Wearable
	EquipBonus        *EquipBonus
	Weapon            *Weapon
	Attack            *Attack
}

// ProvidesHealing は回復効果を提供する構造体
type ProvidesHealing struct {
	ValueType ValueType
	Amount    int
	Ratio     float64
}

// Consumable は消費可能なアイテムの設定
type Consumable struct {
	UsableScene string
	TargetGroup string
	TargetNum   string
}

// Weapon は武器アイテムの設定
type Weapon struct {
	Cost        int
	TargetGroup string
	TargetNum   string
}

// Attack は攻撃性能の設定
type Attack struct {
	Accuracy       int    // 命中率
	Damage         int    // 攻撃力
	AttackCount    int    // 攻撃回数
	Element        string // 攻撃属性
	AttackCategory string // 攻撃種別
}

// Wearable は装備可能アイテムの設定
type Wearable struct {
	Defense           int
	EquipmentCategory string
}

// EquipBonus は装備ボーナスの設定
type EquipBonus struct {
	Vitality  int
	Strength  int
	Sensation int
	Dexterity int
	Agility   int
}

// Recipe はレシピの情報
type Recipe struct {
	Name   string
	Inputs []RecipeInput
}

// RecipeInput は合成の元になる素材
type RecipeInput struct {
	Name   string
	Amount int
}

// Member はメンバーの情報
type Member struct {
	Name            string
	Player          *bool
	Attributes      Attributes
	SpriteSheetName string
	SpriteKey       string
	AnimKeys        []string
	LightSource     *gc.LightSource
	FactionType     string
	Dialog          *DialogRaw
}

// DialogRaw は会話データのローデータ
type DialogRaw struct {
	MessageKey string // メッセージキー
}

// Attributes はキャラクターの能力値
type Attributes struct {
	Vitality  int
	Strength  int
	Sensation int
	Dexterity int
	Agility   int
	Defense   int
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

// Load は文字列からローデータを読み込む
func Load(entityMetadataContent string) (Master, error) {
	rw := Master{}
	rw.ItemIndex = map[string]int{}
	rw.RecipeIndex = map[string]int{}
	rw.MemberIndex = map[string]int{}
	rw.CommandTableIndex = map[string]int{}
	rw.DropTableIndex = map[string]int{}
	rw.ItemTableIndex = map[string]int{}
	rw.EnemyTableIndex = map[string]int{}
	rw.SpriteSheetIndex = map[string]int{}
	rw.TileIndex = map[string]int{}
	rw.PropIndex = map[string]int{}

	metaData, err := toml.Decode(entityMetadataContent, &rw.Raws)
	if err != nil {
		return Master{}, fmt.Errorf("TOML decode error: %w", err)
	}
	// 未知のキーがあった場合はエラーにする
	undecoded := metaData.Undecoded()
	if len(undecoded) > 0 {
		return Master{}, fmt.Errorf("unknown keys found in TOML: %v", undecoded)
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

	return rw, nil
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
		if err := item.ProvidesHealing.ValueType.Valid(); err != nil {
			return gc.EntitySpec{}, fmt.Errorf("%s: %w", "invalid value type", err)
		}
		switch item.ProvidesHealing.ValueType {
		case PercentageType:
			entitySpec.ProvidesHealing = &gc.ProvidesHealing{Amount: gc.RatioAmount{Ratio: item.ProvidesHealing.Ratio}}
		case NumeralType:
			entitySpec.ProvidesHealing = &gc.ProvidesHealing{Amount: gc.NumeralAmount{Numeral: item.ProvidesHealing.Amount}}
		default:
			return gc.EntitySpec{}, fmt.Errorf("不明なValueType: %v", item.ProvidesHealing.ValueType)
		}
	}
	if item.ProvidesNutrition != nil {
		entitySpec.ProvidesNutrition = &gc.ProvidesNutrition{Amount: *item.ProvidesNutrition}
	}
	if item.InflictsDamage != nil {
		entitySpec.InflictsDamage = &gc.InflictsDamage{Amount: *item.InflictsDamage}
	}

	if item.Weapon != nil {
		if err := gc.TargetGroupType(item.Weapon.TargetGroup).Valid(); err != nil {
			return gc.EntitySpec{}, fmt.Errorf("%s: %w", "invalid weapon target group type", err)
		}
		if err := gc.TargetNumType(item.Weapon.TargetNum).Valid(); err != nil {
			return gc.EntitySpec{}, fmt.Errorf("%s: %w", "invalid weapon target num type", err)
		}

		entitySpec.Weapon = &gc.Weapon{
			TargetType: gc.TargetType{
				TargetGroup: gc.TargetGroupType(item.Weapon.TargetGroup),
				TargetNum:   gc.TargetNumType(item.Weapon.TargetNum),
			},
			Cost: item.Weapon.Cost,
		}
	}

	if item.Attack != nil {
		if err := gc.ElementType(item.Attack.Element).Valid(); err != nil {
			return gc.EntitySpec{}, err
		}
		attackType, err := gc.ParseAttackType(item.Attack.AttackCategory)
		if err != nil {
			return gc.EntitySpec{}, err
		}
		if err := attackType.Valid(); err != nil {
			return gc.EntitySpec{}, err
		}

		entitySpec.Attack = &gc.Attack{
			Accuracy:       item.Attack.Accuracy,
			Damage:         item.Attack.Damage,
			AttackCount:    item.Attack.AttackCount,
			Element:        gc.ElementType(item.Attack.Element),
			AttackCategory: attackType,
		}
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
		if err := gc.EquipmentType(item.Wearable.EquipmentCategory).Valid(); err != nil {
			return gc.EntitySpec{}, err
		}
		entitySpec.Wearable = &gc.Wearable{
			Defense:           item.Wearable.Defense,
			EquipmentCategory: gc.EquipmentType(item.Wearable.EquipmentCategory),
			EquipBonus:        bonus,
		}
	}

	if item.Value != nil {
		entitySpec.Value = &gc.Value{Value: *item.Value}
	}

	if item.Weight != nil {
		entitySpec.Weight = &gc.Weight{Kg: *item.Weight}
	}

	// Stackableフラグがtrueの場合は空のStackableコンポーネントを追加
	if item.Stackable != nil && *item.Stackable {
		entitySpec.Stackable = &gc.Stackable{}
	}

	// すべてのアイテムにInteractableを追加（所持状態に関わらず）
	entitySpec.Interactable = &gc.Interactable{Data: gc.ItemInteraction{}}

	return entitySpec, nil
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
		entitySpec.Recipe.Inputs = append(entitySpec.Recipe.Inputs, gc.RecipeInput{Name: input.Name, Amount: input.Amount})
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
	if itemSpec.Attack != nil {
		entitySpec.Attack = itemSpec.Attack
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
	entitySpec.TurnBased = &gc.TurnBased{AP: gc.Pool{Current: 100, Max: 100}} // TODO: Attributesから計算する
	entitySpec.SpriteRender = &gc.SpriteRender{
		SpriteSheetName: member.SpriteSheetName,
		SpriteKey:       member.SpriteKey,
		AnimKeys:        member.AnimKeys,
		Depth:           gc.DepthNumPlayer,
	}
	entitySpec.Attributes = &gc.Attributes{
		Vitality:  gc.Attribute{Base: member.Attributes.Vitality},
		Strength:  gc.Attribute{Base: member.Attributes.Strength},
		Sensation: gc.Attribute{Base: member.Attributes.Sensation},
		Dexterity: gc.Attribute{Base: member.Attributes.Dexterity},
		Agility:   gc.Attribute{Base: member.Attributes.Agility},
		Defense:   gc.Attribute{Base: member.Attributes.Defense},
	}
	entitySpec.Pools = &gc.Pools{}
	if member.Player != nil && *member.Player {
		entitySpec.Player = &gc.Player{}
	}

	commandTableIdx, ok := rw.CommandTableIndex[name]
	if ok && commandTableIdx < len(rw.Raws.CommandTables) {
		commandTable := rw.Raws.CommandTables[commandTableIdx]
		entitySpec.CommandTable = &gc.CommandTable{Name: commandTable.Name}
	}

	dropTableIdx, ok := rw.DropTableIndex[name]
	if ok && dropTableIdx < len(rw.Raws.DropTables) {
		dropTable := rw.Raws.DropTables[dropTableIdx]
		entitySpec.DropTable = &gc.DropTable{Name: dropTable.Name}
	}

	if member.LightSource != nil {
		entitySpec.LightSource = member.LightSource
	}

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
func (rw *Master) GetCommandTable(name string) (CommandTable, error) {
	ctIdx, ok := rw.CommandTableIndex[name]
	if !ok {
		return CommandTable{}, fmt.Errorf("キーが存在しない: %s", name)
	}
	if ctIdx >= len(rw.Raws.CommandTables) {
		return CommandTable{}, fmt.Errorf("コマンドテーブルインデックスが範囲外: %d (長さ: %d)", ctIdx, len(rw.Raws.CommandTables))
	}
	commandTable := rw.Raws.CommandTables[ctIdx]

	return commandTable, nil
}

// GetDropTable は指定された名前のドロップテーブルを取得する
func (rw *Master) GetDropTable(name string) (DropTable, error) {
	dtIdx, ok := rw.DropTableIndex[name]
	if !ok {
		return DropTable{}, fmt.Errorf("キーが存在しない: %s", name)
	}
	if dtIdx >= len(rw.Raws.DropTables) {
		return DropTable{}, fmt.Errorf("ドロップテーブルインデックスが範囲外: %d (長さ: %d)", dtIdx, len(rw.Raws.DropTables))
	}
	dropTable := rw.Raws.DropTables[dtIdx]

	return dropTable, nil
}

// GetItemTable は指定された名前のアイテムテーブルを取得する
func (rw *Master) GetItemTable(name string) (ItemTable, error) {
	itIdx, ok := rw.ItemTableIndex[name]
	if !ok {
		return ItemTable{}, fmt.Errorf("キーが存在しない: %s", name)
	}
	if itIdx >= len(rw.Raws.ItemTables) {
		return ItemTable{}, fmt.Errorf("アイテムテーブルインデックスが範囲外: %d (長さ: %d)", itIdx, len(rw.Raws.ItemTables))
	}
	itemTable := rw.Raws.ItemTables[itIdx]

	return itemTable, nil
}

// GetEnemyTable は指定された名前の敵テーブルを取得する
func (rw *Master) GetEnemyTable(name string) (EnemyTable, error) {
	etIdx, ok := rw.EnemyTableIndex[name]
	if !ok {
		return EnemyTable{}, fmt.Errorf("キーが存在しない: %s", name)
	}
	if etIdx >= len(rw.Raws.EnemyTables) {
		return EnemyTable{}, fmt.Errorf("敵テーブルインデックスが範囲外: %d (長さ: %d)", etIdx, len(rw.Raws.EnemyTables))
	}
	enemyTable := rw.Raws.EnemyTables[etIdx]

	return enemyTable, nil
}

// TileRaw はタイルのローデータ定義
type TileRaw struct {
	Name         string
	Description  string
	BlockPass    bool // 通行を遮断するか
	SpriteRender gc.SpriteRender
	BlockView    bool // 視界を遮断するか
}

// PropRaw は置物のローデータ定義
type PropRaw struct {
	Name         string
	Description  string
	SpriteRender gc.SpriteRender
	AnimKeys     []string
	BlockPass    bool
	BlockView    bool
	LightSource  *gc.LightSource
	Door         *DoorRaw
}

// DoorRaw はドアのローデータ
type DoorRaw struct{}

// GetTile は指定された名前のタイルを取得する
// 計画段階でタイルの性質（Walkableなど）を参照する場合に使用する
func (rw *Master) GetTile(name string) (TileRaw, error) {
	tileIdx, ok := rw.TileIndex[name]
	if !ok {
		return TileRaw{}, NewKeyNotFoundError(name, "TileIndex")
	}
	if tileIdx >= len(rw.Raws.Tiles) {
		return TileRaw{}, fmt.Errorf("タイルインデックスが範囲外: %d (長さ: %d)", tileIdx, len(rw.Raws.Tiles))
	}

	return rw.Raws.Tiles[tileIdx], nil
}

// NewTileSpec は指定された名前のタイルのEntitySpecを生成する
// 実際にエンティティを生成する際に使用する
func (rw *Master) NewTileSpec(name string, x, y gc.Tile, autoTileIndex *int) (gc.EntitySpec, error) {
	tileRaw, err := rw.GetTile(name)
	if err != nil {
		return gc.EntitySpec{}, err
	}

	entitySpec := gc.EntitySpec{}
	entitySpec.Name = &gc.Name{Name: tileRaw.Name}
	entitySpec.Description = &gc.Description{Description: tileRaw.Description}
	entitySpec.GridElement = &gc.GridElement{X: x, Y: y}

	// SpriteRenderを設定
	sprite := tileRaw.SpriteRender
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

	return entitySpec, nil
}

// GetProp は指定された名前の置物の設定を取得する
func (rw *Master) GetProp(name string) (PropRaw, error) {
	propIdx, ok := rw.PropIndex[name]
	if !ok {
		return PropRaw{}, NewKeyNotFoundError(name, "PropIndex")
	}
	if propIdx >= len(rw.Raws.Props) {
		return PropRaw{}, fmt.Errorf("置物インデックスが範囲外: %d (長さ: %d)", propIdx, len(rw.Raws.Props))
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
	spriteRender := propRaw.SpriteRender
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

	if propRaw.LightSource != nil {
		entitySpec.LightSource = propRaw.LightSource
	}

	if propRaw.Door != nil {
		// Door componentを追加（向きは初期値、spawn時に設定される）
		entitySpec.Door = &gc.Door{
			IsOpen:      false,
			Orientation: gc.DoorOrientationHorizontal,
		}
		// ドアは相互作用可能
		entitySpec.Interactable = &gc.Interactable{Data: gc.DoorInteraction{}}
	}

	return entitySpec, nil
}
