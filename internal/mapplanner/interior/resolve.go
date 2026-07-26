package interior

import "math/rand/v2"

// Resolve は Content の各 Group を解決し、置くべき Selection の列を決定的に返す。
// Group ごとに childSeed で独立した乱数ストリームを引くので、Group を1つ足しても他 Group の
// 結果は変わらない。同じ seed で2回呼べば完全に一致する。
func (c Content) Resolve(seed uint64) []Selection {
	out := make([]Selection, 0, len(c.Groups))
	for i, g := range c.Groups {
		rng := rand.New(rand.NewPCG(childSeed(seed, i), 0))
		out = append(out, resolveGroup(rng, g)...)
	}
	return out
}

// resolveGroup は1つの Group を Style に従って解決する。GroupStyle は内部で定義する信頼できる値なので、
// 網羅漏れは default を置かず exhaustive linter に強制させ、到達しない末尾は panic で落とす。
func resolveGroup(rng *rand.Rand, g Group) []Selection {
	switch g.Style {
	case PickEach:
		return pickEach(rng, g.Items)
	case PickOne:
		return pickDistinct(rng, g.Items, 1)
	case PickN:
		return pickDistinct(rng, g.Items, g.Pick)
	}
	panic("未知の GroupStyle: " + string(g.Style))
}

// pickEach は Items を全部置く保証枠。各 Stuff は Chance で個別に gate する。
func pickEach(rng *rand.Rand, items []Stuff) []Selection {
	out := make([]Selection, 0, len(items))
	for _, it := range items {
		if !chancePass(rng, it.Chance) {
			continue
		}
		if n := it.Amount.roll(rng); n > 0 {
			out = append(out, selectionOf(it, n))
		}
	}
	return out
}

// pickDistinct は Items から重複なく n 個を重みで引く。map を使わず slice のプールで管理し、
// 引いた要素をプールから外して重複を防ぐ。Items が n 未満なら在るだけ引く。
func pickDistinct(rng *rand.Rand, items []Stuff, n int) []Selection {
	pool := make([]Stuff, len(items))
	copy(pool, items)
	out := make([]Selection, 0, n)
	for len(out) < n && len(pool) > 0 {
		i := weightedIndex(rng, pool)
		it := pool[i]
		pool = append(pool[:i], pool[i+1:]...)
		count := it.Amount.roll(rng)
		if count <= 0 {
			count = 1
		}
		out = append(out, selectionOf(it, count))
	}
	return out
}

// selectionOf は Stuff と個数から Selection を組む。置き方は archetype 既定で補い、Placement を空にした
// レシピもここで具体化する。衛星束も引き継ぐ。
func selectionOf(it Stuff, count int) Selection {
	return Selection{
		Kind:       it.Kind,
		Ref:        it.Ref,
		Count:      count,
		Placement:  placementOf(it.Ref, it.Placement),
		Satellites: it.Satellites,
	}
}

// weightedIndex は重みに比例して1つの添字を選ぶ。重み0は1とみなす。
func weightedIndex(rng *rand.Rand, items []Stuff) int {
	total := 0
	for _, it := range items {
		total += weightOf(it)
	}
	r := rng.IntN(total)
	for i, it := range items {
		r -= weightOf(it)
		if r < 0 {
			return i
		}
	}
	return len(items) - 1 // 重み合計と減算が整合すれば到達しない。安全側で末尾を返す
}

func weightOf(it Stuff) int {
	if it.Weight <= 0 {
		return 1
	}
	return it.Weight
}

// chancePass は Chance% で真を返す。0 以下は常に真。
func chancePass(rng *rand.Rand, chance int) bool {
	if chance <= 0 {
		return true
	}
	return rng.IntN(100) < chance
}

// roll は Base 個の Sides 面ダイスの和に Bonus を足す。Sides<=0 は定数 Bonus。
func (d Dice) roll(rng *rand.Rand) int {
	if d.Sides <= 0 {
		return d.Bonus
	}
	sum := d.Bonus
	for range d.Base {
		sum += rng.IntN(d.Sides) + 1
	}
	return sum
}

// childSeed は親 seed と添字から子 seed を導く。splitmix64 の finalizer で撹拌し、添字の1違いでも
// 出力を無相関へ散らす。可変な乱数器を持ち回らず、seed を引数導出で分岐させるための純関数。
func childSeed(parent uint64, index int) uint64 {
	x := parent + uint64(index+1)*0x9E3779B97F4A7C15
	x ^= x >> 30
	x *= 0xBF58476D1CE4E5B9
	x ^= x >> 27
	x *= 0x94D049BB133111EB
	x ^= x >> 31
	return x
}
