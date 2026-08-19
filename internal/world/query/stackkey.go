package query

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// StackKey はアイテムのスタック同一性キー。キーが等しいアイテムだけが同一スタックに束ねられる。
// スタック数はこのキーで束ねた同値クラスの大きさとして導出し、保存しない。
// 個体差を足すときはキーに項を1つ加える。鮮度のような連続量は段階へ量子化すれば束ね、
// 生値のまま持てば個体ごとに分かれる。
type StackKey struct {
	RawID          string            // 生成元の同定キー。まず品種が一致すること
	FreshnessStage gc.FreshnessStage // 腐敗品の鮮度段階。非腐敗品は空文字
	Equip          string            // 装備の個体差の指紋。性能の違う同名装備を束ねない。非装備は空文字
	// solo は RawID を持たない実体の単独化。同定キーが無い実体は束ねようがないので、
	// 自分自身をキーに入れて他の誰とも等しくならないようにする。これにより GroupStacks は
	// タイルや壁が混ざった任意の実体列を受けても安全で、呼び出し側の事前選別が要らない
	solo ecs.Entity
}

// StackKeyOf は entity のスタック同一性キーを返す。スタック同一判定の唯一の権威。
// RawID に加え、腐敗品なら現在の鮮度段階を、装備なら性能の指紋を含める。
// RawID を持たない実体は自分自身だけの単独スタックになる。
func StackKeyOf(world w.World, entity ecs.Entity) StackKey {
	var key StackKey
	if !world.Components.RawID.Has(entity) {
		key.solo = entity
		return key
	}
	key.RawID = world.Components.RawID.Get(entity).ID
	if stage, ok := FreshnessStageOf(world, entity); ok {
		key.FreshnessStage = stage
	}
	key.Equip = equipFingerprint(world, entity)
	return key
}

// equipFingerprint は装備の性能を1つの文字列に畳む。クラフトの乱数化で同名でも性能が
// ばらつくため、生値をそのまま指紋にして性能の違う個体を別スタックに分ける。
// 完全一致の個体だけが束ねられる。非装備は空文字で、指紋は同一性に影響しない
func equipFingerprint(world w.World, entity ecs.Entity) string {
	var fp string
	if world.Components.Melee.Has(entity) {
		fp += fmt.Sprintf("m%+v", *world.Components.Melee.Get(entity))
	}
	if world.Components.Fire.Has(entity) {
		fp += fmt.Sprintf("f%+v", *world.Components.Fire.Get(entity))
	}
	if world.Components.Wearable.Has(entity) {
		fp += fmt.Sprintf("w%+v", *world.Components.Wearable.Get(entity))
	}
	return fp
}

// SameStack は a と b が同一スタックに束ねられるかを返す。StackKey の等価そのもの。
func SameStack(world w.World, a ecs.Entity, b ecs.Entity) bool {
	return StackKeyOf(world, a) == StackKeyOf(world, b)
}

// StackCountOf は candidates のうち entity と同一スタックのものを数える。スタック数は保存せず
// この数え上げで導出する。数える範囲は呼び出し側が候補集合として渡す。リロードやクラフトは
// バックパックの中を、総重量は装備や収納も含む所有全体を候補にする、というように範囲を選ぶ。
func StackCountOf(world w.World, entity ecs.Entity, candidates []ecs.Entity) int {
	key := StackKeyOf(world, entity)
	n := 0
	for _, c := range candidates {
		if StackKeyOf(world, c) == key {
			n++
		}
	}
	return n
}

// Stack は同一スタックに束ねたアイテム群。表示や一括操作の単位に使う。
type Stack struct {
	Rep     ecs.Entity   // 束の代表。入力での初出エンティティ
	Count   int          // 束に属する個数
	Members []ecs.Entity // 束に属する全エンティティ。初出順
}

// BackpackStacks は owner のバックパックの中身を、表示順に並べたスタックの列で返す。
// 収集・並べ替え・束ねをこの1呼び出しに集約し、一覧はこの結果をそのまま行にする。
// 生のエンティティ列挙から一覧を組むと、束ね忘れや並べ替え忘れが呼び出し側ごとに起きる
func BackpackStacks(world w.World, owner ecs.Entity) []Stack {
	var items []ecs.Entity
	q := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
	for q.Next() {
		entity := q.Entity()
		if world.Components.LocationInBackpack.Get(entity).Owner == owner {
			items = append(items, entity)
		}
	}
	return GroupStacks(world, SortEntities(world, items))
}

// StorageStacks は storage の中身を、表示順に並べたスタックの列で返す。BackpackStacks と対
func StorageStacks(world w.World, storage ecs.Entity) []Stack {
	return GroupStacks(world, SortEntities(world, GetStorageItems(world, storage)))
}

// StackMembers は entity と同じ所有者かつ同じ位置種別にある、同一スタックのエンティティを返す。
// 個数の数え上げ、一括消費、表示の束ねの範囲をこの1関数に集約する。バックパックと収納は
// 所有者一致で絞る。装備やフィールド、位置未設定は束ねず entity 単独を返す。
func StackMembers(world w.World, entity ecs.Entity) []ecs.Entity {
	key := StackKeyOf(world, entity)
	switch {
	case world.Components.LocationInBackpack.Has(entity):
		owner := world.Components.LocationInBackpack.Get(entity).Owner
		return sameStackInBackpack(world, owner, key)
	case world.Components.LocationInStorage.Has(entity):
		owner := world.Components.LocationInStorage.Get(entity).Owner
		return sameStackInStorage(world, owner, key)
	case world.Components.LocationOnField.Has(entity):
		if ge := world.Components.GridElement.Get(entity); ge != nil {
			return sameStackOnField(world, ge, key)
		}
		return []ecs.Entity{entity}
	default:
		return []ecs.Entity{entity}
	}
}

// sameStackOnField は床の同じタイルにある同一スタックのエンティティを返す。床アイテムは
// 所有者を持たず位置で束ねるため、同座標かつ同キーのものを数える。
func sameStackOnField(world w.World, grid *gc.GridElement, key StackKey) []ecs.Entity {
	var out []ecs.Entity
	q := ecs.NewFilter2[gc.LocationOnField, gc.GridElement](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		ge := world.Components.GridElement.Get(e)
		if ge.X == grid.X && ge.Y == grid.Y && StackKeyOf(world, e) == key {
			out = append(out, e)
		}
	}
	return out
}

func sameStackInBackpack(world w.World, owner ecs.Entity, key StackKey) []ecs.Entity {
	var out []ecs.Entity
	q := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.LocationInBackpack.Get(e).Owner == owner && StackKeyOf(world, e) == key {
			out = append(out, e)
		}
	}
	return out
}

func sameStackInStorage(world w.World, owner ecs.Entity, key StackKey) []ecs.Entity {
	var out []ecs.Entity
	q := ecs.NewFilter1[gc.LocationInStorage](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.LocationInStorage.Get(e).Owner == owner && StackKeyOf(world, e) == key {
			out = append(out, e)
		}
	}
	return out
}

// GroupStacks は entities を StackKey で束ね、各束の代表と個数を返す。
// 束の並びは入力での初出順にして決定的にする。表示の一覧はこの束1つを1行にする。
func GroupStacks(world w.World, entities []ecs.Entity) []Stack {
	index := make(map[StackKey]int, len(entities))
	stacks := make([]Stack, 0, len(entities))
	for _, e := range entities {
		key := StackKeyOf(world, e)
		if i, ok := index[key]; ok {
			stacks[i].Count++
			stacks[i].Members = append(stacks[i].Members, e)
			continue
		}
		index[key] = len(stacks)
		stacks = append(stacks, Stack{Rep: e, Count: 1, Members: []ecs.Entity{e}})
	}
	return stacks
}
