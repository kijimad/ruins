package raw

import (
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// errInvalidPackDice は pack のダイス表記が解釈できないときに返す。
var errInvalidPackDice = errors.New("invalid pack dice")

// DrawnItem は item group から抽選した1件。名前と個数を持つ。
type DrawnItem struct {
	Name  string
	Count int
}

// SelectFromItemGroup は id が指すアイテムグループかアイテムコレクションから1回ぶんの loot を抽選する。グループは
// 重み比で1エントリを選び、コレクションは各エントリを独立に確率判定する。個数は entry の pack ダイスで振る。
// 危険度を扱わず直接引くので、地上の床 loot と家具の収納 loot が同じ抽選を共有できる。
func SelectFromItemGroup(raws rawschema.Raws, id string, rng *rand.Rand) ([]DrawnItem, error) {
	if g, err := GetItemGroup(raws, id); err == nil {
		return drawGroup(g.Entries, rng)
	}
	c, err := GetItemCollection(raws, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errItemGroupNotExist, id)
	}
	return drawCollection(c.Entries, rng)
}

// drawGroup は重み比で1エントリを選び、その pack ぶんを返す。
func drawGroup(entries []rawschema.ItemGroupEntry, rng *rand.Rand) ([]DrawnItem, error) {
	i, ok := consts.Pick(rng, entries, func(e rawschema.ItemGroupEntry) float64 { return e.Weight })
	if !ok {
		return nil, nil
	}
	entry := entries[i]
	count, err := rollPack(entry.Pack, rng)
	if err != nil {
		return nil, err
	}
	return expandDraw(entry.Id, count), nil
}

// drawCollection は各エントリを chance の確率で独立に判定し、当選ぶんを返す。両方出ることも、どちらも出ない
// こともある。
func drawCollection(entries []rawschema.ItemCollectionEntry, rng *rand.Rand) ([]DrawnItem, error) {
	var out []DrawnItem
	for _, e := range entries {
		if rng.Float64()*100 < e.Chance {
			count, err := rollPack(e.Pack, rng)
			if err != nil {
				return nil, err
			}
			out = append(out, expandDraw(e.Id, count)...)
		}
	}
	return out, nil
}

// rollPack は pack のダイス表記を振って個数を返す。pack は必須で、空表記や不正表記は ParseDice が弾く。
// ロード時に Dice スカラーの @pattern が全 pack を検証するので、正常データでは実行時にエラーにならない。
func rollPack(pack rawschema.Dice, rng *rand.Rand) (int, error) {
	d, err := consts.ParseDice(pack)
	if err != nil {
		return 0, fmt.Errorf("%w %q: %w", errInvalidPackDice, pack, err)
	}
	return d.Roll(rng), nil
}

// expandDraw は抽選結果を DrawnItem に落とす。個数は spawn 側が1個1エンティティに展開するので、
// ここでは個数を1エントリにまとめて返す。
func expandDraw(name string, count int) []DrawnItem {
	if count <= 0 {
		return nil
	}
	return []DrawnItem{{Name: name, Count: count}}
}
