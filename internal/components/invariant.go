package components

import (
	"reflect"
	"strings"

	"github.com/mlange-42/ark/ecs"
)

// hasser はエンティティが特定コンポーネントを持つか判定できる型。
// Ark の *ecs.Map[T] はいずれも Has(ecs.Entity) bool を満たす
type hasser interface {
	Has(ecs.Entity) bool
}

// Pred はエンティティに対する述語を表すインターフェース。
// Has, And, Or, Not を組み合わせてコンポーネント間の関係を宣言的に記述する
type Pred interface {
	Eval(entity ecs.Entity) bool
	String() string
}

// Has は指定コンポーネントの存在を検査する述語
type Has struct {
	Label string
	Comp  hasser
}

// Eval はPredインターフェースを実装する
func (h Has) Eval(entity ecs.Entity) bool {
	return h.Comp.Has(entity)
}

func (h Has) String() string {
	return h.Label
}

// factionPred は指定した派閥種別かを判定する述語。
// Factionは単一コンポーネントにKindを持つため、種別判定はKind比較で行う
type factionPred struct {
	comp *ecs.Map[Faction]
	kind FactionKind
}

func (p factionPred) Eval(entity ecs.Entity) bool {
	return p.comp.Has(entity) && p.comp.Get(entity).Kind == p.kind
}

func (p factionPred) String() string {
	return p.kind.String()
}

// factionIs は指定派閥種別の述語を返す
func (c *Components) factionIs(kind FactionKind) Pred {
	return factionPred{comp: c.Faction, kind: kind}
}

// And はすべての条件を満たすことを要求する述語
type And []Pred

// Eval はPredインターフェースを実装する
func (a And) Eval(entity ecs.Entity) bool {
	for _, p := range a {
		if !p.Eval(entity) {
			return false
		}
	}
	return true
}

func (a And) String() string {
	parts := make([]string, len(a))
	for i, p := range a {
		parts[i] = p.String()
	}
	return "(" + strings.Join(parts, " AND ") + ")"
}

// Or はいずれかの条件を満たすことを要求する述語
type Or []Pred

// Eval はPredインターフェースを実装する
func (o Or) Eval(entity ecs.Entity) bool {
	for _, p := range o {
		if p.Eval(entity) {
			return true
		}
	}
	return false
}

func (o Or) String() string {
	parts := make([]string, len(o))
	for i, p := range o {
		parts[i] = p.String()
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

// Not は条件の否定を表す述語
type Not struct {
	Pred Pred
}

// Eval はPredインターフェースを実装する
func (n Not) Eval(entity ecs.Entity) bool {
	return !n.Pred.Eval(entity)
}

func (n Not) String() string {
	return "NOT " + n.Pred.String()
}

// Category はコンポーネントの組み合わせでエンティティのカテゴリを定義する。
// Pred を実装しているので、他の述語と同様に組み合わせて使える
type Category struct {
	Name string
	Pred Pred
}

// Eval はPredインターフェースを実装する
func (cat Category) Eval(entity ecs.Entity) bool {
	return cat.Pred.Eval(entity)
}

func (cat Category) String() string {
	return cat.Name
}

// CategoryGroupKey はカテゴリグループを識別するキー
type CategoryGroupKey string

const (
	// InventoryCategoryKey はインベントリのタブに対応する分類
	InventoryCategoryKey CategoryGroupKey = "inventory"
	// ItemTypeCategoryKey はアイテム欄に表示する細分類
	ItemTypeCategoryKey CategoryGroupKey = "item_type"
	// FieldLookCategoryKey はフィールド上の観察で表示する分類
	FieldLookCategoryKey CategoryGroupKey = "field_look"
)

// インベントリカテゴリ名の定数
const (
	CategoryGoods  = "道具"
	CategoryWeapon = "武器"
	CategoryArmor  = "防具"
)

// アイテム種別カテゴリ名の定数
const (
	CategoryMaterial   = "素材"
	CategoryAmmo       = "弾薬"
	CategoryBook       = "本"
	CategoryProp       = "置物"
	CategoryConsumable = "消耗品"
	CategoryMelee      = "近接武器"
	CategoryFire       = "射撃武器"
)

// フィールド観察カテゴリ名の定数
const (
	CategoryPlayer = "自分"
	CategoryEnemy  = "敵"
	CategoryNPC    = "NPC"
	CategoryTile   = "タイル"
)

// has は Components のフィールド名を自動解決して Has を生成する。
// Map ハンドルのポインタ同一性でフィールドを特定する
func (c *Components) has(comp hasser) Has {
	val := reflect.ValueOf(c).Elem()
	typ := val.Type()
	for i := range val.NumField() {
		if val.Field(i).Interface() == comp {
			return Has{Label: typ.Field(i).Name, Comp: comp}
		}
	}
	panic("component not found in Components")
}

// Categories は観点ごとにグループ化されたカテゴリ定義を返す
func (c *Components) Categories() map[CategoryGroupKey][]Category {
	return map[CategoryGroupKey][]Category{
		InventoryCategoryKey: {
			{Name: CategoryGoods, Pred: Or{c.has(c.Material), c.has(c.Ammo), c.has(c.Book), c.has(c.Prop), c.has(c.Consumable)}},
			{Name: CategoryWeapon, Pred: Or{c.has(c.Melee), c.has(c.Fire)}},
			{Name: CategoryArmor, Pred: c.has(c.Wearable)},
		},
		ItemTypeCategoryKey: {
			{Name: CategoryMaterial, Pred: c.has(c.Material)},
			{Name: CategoryAmmo, Pred: c.has(c.Ammo)},
			{Name: CategoryBook, Pred: c.has(c.Book)},
			{Name: CategoryProp, Pred: c.has(c.Prop)},
			{Name: CategoryConsumable, Pred: c.has(c.Consumable)},
			// Fire は Melee より先に判定する。射撃武器は殴打性能として Melee も持つため
			{Name: CategoryFire, Pred: c.has(c.Fire)},
			{Name: CategoryMelee, Pred: c.has(c.Melee)},
			{Name: CategoryArmor, Pred: c.has(c.Wearable)},
		},
		FieldLookCategoryKey: {
			// Player を先に判定する。Player は FactionAlly も持つため
			{Name: CategoryPlayer, Pred: c.has(c.Player)},
			{Name: CategoryEnemy, Pred: c.factionIs(FactionEnemy)},
			{Name: CategoryNPC, Pred: Or{c.factionIs(FactionAlly), c.factionIs(FactionNeutral)}},
			{Name: CategoryProp, Pred: c.has(c.Prop)},
			{Name: CategoryTile, Pred: c.has(c.Tile)},
		},
	}
}

// CategoryOf は指定グループ内でエンティティが属するカテゴリ名を返す。
// どのカテゴリにも属さない場合は空文字とfalseを返す
func (c *Components) CategoryOf(key CategoryGroupKey, entity ecs.Entity) (string, bool) {
	for _, cat := range c.Categories()[key] {
		if cat.Eval(entity) {
			return cat.Name, true
		}
	}
	return "", false
}

// CategoryOfSpec は EntitySpec に対して CategoryOf と同等の判定を行う。
// Has.Label が EntitySpec のフィールド名と一致する規約を利用し、
// リフレクションでフィールドの nil 判定を行う
func (c *Components) CategoryOfSpec(key CategoryGroupKey, spec *EntitySpec) (string, bool) {
	specVal := reflect.ValueOf(spec).Elem()
	for _, cat := range c.Categories()[key] {
		if evalPredSpec(cat.Pred, specVal) {
			return cat.Name, true
		}
	}
	return "", false
}

// evalPredSpec は Pred を EntitySpec のリフレクション値に対して評価する
func evalPredSpec(pred Pred, specVal reflect.Value) bool {
	switch p := pred.(type) {
	case Has:
		field := specVal.FieldByName(p.Label)
		if !field.IsValid() {
			return false
		}
		// ポインタ型のフィールドのみ nil 判定する。値型フィールドは常に「存在する」とみなす
		if field.Kind() == reflect.Pointer || field.Kind() == reflect.Interface {
			return !field.IsNil()
		}
		return true
	case And:
		for _, sub := range p {
			if !evalPredSpec(sub, specVal) {
				return false
			}
		}
		return true
	case Or:
		for _, sub := range p {
			if evalPredSpec(sub, specVal) {
				return true
			}
		}
		return false
	case Not:
		return !evalPredSpec(p.Pred, specVal)
	case Category:
		return evalPredSpec(p.Pred, specVal)
	default:
		return false
	}
}
