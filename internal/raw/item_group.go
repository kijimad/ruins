package raw

import (
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
)

// item group 抽選のエラー。呼び出し側とテストが errors.Is で種類を同定できるようにする。
var (
	// errUnknownItemGroupSubtype は group.Subtype が既知の distribution/collection いずれでもないときに返す。
	errUnknownItemGroupSubtype = errors.New("unknown item group subtype")
	// errInvalidPackDice は pack のダイス表記が解釈できないときに返す。
	errInvalidPackDice = errors.New("invalid pack dice")
)

// DrawnItem は item group から抽選した1件。名前と個数を持つ。
type DrawnItem struct {
	Name  string
	Count int
}

// SelectFromItemGroup は item group から1回ぶんの loot を抽選する。distribution は重み比で1エントリを選び、
// collection は各エントリを独立に確率判定する。個数は entry の pack ダイスで振る。stackable は個数を1エントリ
// にまとめ、非 stackable は1個ずつのエントリに展開する。深度を扱わず group を直接引くので、地上の床 loot と
// 家具の収納 loot が同じ抽選を共有できる。
func SelectFromItemGroup(raws oapi.Raws, groupID string, rng *rand.Rand) ([]DrawnItem, error) {
	group, err := GetItemGroup(raws, groupID)
	if err != nil {
		return nil, err
	}
	switch group.Subtype {
	case oapi.Distribution:
		return drawDistribution(raws, group.Entries, rng)
	case oapi.Collection:
		return drawCollection(raws, group.Entries, rng)
	default:
		// group.Subtype は raw 由来で未知値が来うる。データ不整合として呼び出し側へ返す
		return nil, fmt.Errorf("%w for %q: %s", errUnknownItemGroupSubtype, groupID, group.Subtype)
	}
}

// drawDistribution は重み比で1エントリを選び、その pack ぶんを返す。
func drawDistribution(raws oapi.Raws, entries []oapi.ItemGroupEntry, rng *rand.Rand) ([]DrawnItem, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	entry, err := SelectByWeightFunc(
		entries,
		func(e oapi.ItemGroupEntry) float64 { return e.Weight },
		func(e oapi.ItemGroupEntry) oapi.ItemGroupEntry { return e },
		rng,
	)
	if err != nil {
		return nil, err
	}
	if entry.Id == "" {
		return nil, nil
	}
	count, err := rollPack(entry.Pack, rng)
	if err != nil {
		return nil, err
	}
	return expandDraw(raws, entry.Id, count), nil
}

// drawCollection は各エントリを 0-100 の確率で独立に判定し、当選ぶんを返す。両方出ることも、どちらも出ない
// こともある。
func drawCollection(raws oapi.Raws, entries []oapi.ItemGroupEntry, rng *rand.Rand) ([]DrawnItem, error) {
	var out []DrawnItem
	for _, e := range entries {
		if e.Weight <= 0 {
			continue
		}
		if rng.Float64()*100 < e.Weight {
			count, err := rollPack(e.Pack, rng)
			if err != nil {
				return nil, err
			}
			out = append(out, expandDraw(raws, e.Id, count)...)
		}
	}
	return out, nil
}

// rollPack は pack のダイス表記を振って個数を返す。pack は必須で、空表記や不正表記は ParseDice が弾く。
// ロード時の validateSpawnDice が同じ ParseDice で全 pack を検証するので、正常データでは実行時にエラーにならない。
func rollPack(pack oapi.Dice, rng *rand.Rand) (int, error) {
	d, err := consts.ParseDice(pack)
	if err != nil {
		return 0, fmt.Errorf("%w %q: %w", errInvalidPackDice, pack, err)
	}
	return d.Roll(rng), nil
}

// expandDraw は stackable なら個数を1エントリにまとめ、非 stackable なら1個ずつのエントリに展開する。
func expandDraw(raws oapi.Raws, name string, count int) []DrawnItem {
	if count <= 0 {
		return nil
	}
	if isStackableItem(raws, name) {
		return []DrawnItem{{Name: name, Count: count}}
	}
	out := make([]DrawnItem, count)
	for i := range out {
		out[i] = DrawnItem{Name: name, Count: 1}
	}
	return out
}

// isStackableItem はアイテムが stackable かを raw で判定する。未知アイテムは false。
func isStackableItem(raws oapi.Raws, name string) bool {
	item, err := FindItem(raws, name)
	if err != nil {
		return false
	}
	return item.Stackable != nil && *item.Stackable
}
