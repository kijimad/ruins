package components

import (
	"strings"

	ecs "github.com/x-hgg-x/goecs/v2"
)

// Pred はエンティティに対する述語を表すインターフェース。
// Has, And, Or, Not を組み合わせてコンポーネント間の関係を宣言的に記述する
type Pred interface {
	Eval(entity ecs.Entity) bool
	String() string
}

// Has は指定コンポーネントの存在を検査する述語
type Has struct {
	Label string
	Comp  ecs.DataComponent
}

// Eval はPredインターフェースを実装する
func (h Has) Eval(entity ecs.Entity) bool {
	return entity.HasComponent(h.Comp)
}

func (h Has) String() string {
	return h.Label
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
)

// インベントリカテゴリ名の定数
const (
	CategoryGoods  = "道具"
	CategoryWeapon = "武器"
	CategoryArmor  = "防具"
)

// Categories は観点ごとにグループ化されたカテゴリ定義を返す
func (c *Components) Categories() map[CategoryGroupKey][]Category {
	return map[CategoryGroupKey][]Category{
		InventoryCategoryKey: {
			{Name: CategoryGoods, Pred: Or{Has{"Material", c.Material}, Has{"Ammo", c.Ammo}, Has{"Book", c.Book}, Has{"Prop", c.Prop}, Has{"Consumable", c.Consumable}}},
			{Name: CategoryWeapon, Pred: Or{Has{"Melee", c.Melee}, Has{"Fire", c.Fire}}},
			{Name: CategoryArmor, Pred: Has{"Wearable", c.Wearable}},
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
