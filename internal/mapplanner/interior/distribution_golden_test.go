package interior

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

// facilityDist は1施設を多数 seed で生成したときの出現分布。単一 seed の画像 golden が「その1枚」しか
// 守れないのに対し、これは生成の統計的な形を固定する。レイアウトの多様性や家具の散らばりが崩れる退行を、
// 目視でなく数値で捕まえられる。生成は決定的なので固定 seed 列の集計も固定になり、golden が安定する。
type facilityDist struct {
	Runs      int            // 実行回数
	RoomCount map[string]int // 部屋数 → その部屋数になった建物の回数
	Roles     map[string]int // 部屋役割 → 総出現回数
	Furniture map[string]int // 家具 Ref → 総配置回数
	Decor     map[string]int // 装飾 Ref → 総配置回数
	Loot      map[string]int // 戦利品 Ref → 総配置回数
	Heroes    int            // hero の見せ場を持った建物の数
}

// TestGolden_Distribution は施設ごとに seed 0..N-1 で建物を生成し、部屋数・役割・家具・装飾・戦利品・hero の
// 出現回数を集計した JSON を golden にする。golden ファイル自体が「どの施設にどの部屋・家具がどれくらい出るか」
// の一覧になり、人が開いて分布を確認できる。生成を変えて分布が動けば golden が差分を出し、退行に気付ける。
func TestGolden_Distribution(t *testing.T) {
	t.Parallel()

	const runs = 100
	footprint := Rect{X: 0, Y: 0, W: prodFootprint, H: prodFootprint}
	door := Vec{X: prodFootprint / 2, Y: 0}

	// map のキー順は json.Marshal が整列するので golden は決定的。施設は overworld が生む全種を並べる
	facilities := []FacilityKind{facHouse, facStore, facClinic, facOffice, facDepot, facAntique, facLab}
	out := map[FacilityKind]*facilityDist{}
	for _, fac := range facilities {
		d := &facilityDist{
			Runs:      runs,
			RoomCount: map[string]int{},
			Roles:     map[string]int{},
			Furniture: map[string]int{},
			Decor:     map[string]int{},
			Loot:      map[string]int{},
		}
		for seed := range uint64(runs) {
			site, placed := FurnishBuilding(seed, footprint, door, fac)
			d.RoomCount[strconv.Itoa(len(site.Rooms))]++
			for _, r := range site.Rooms {
				d.Roles[string(r.Role)]++
			}
			for _, p := range placed {
				switch p.Kind {
				case KindFurniture:
					d.Furniture[p.Ref]++
				case KindDecor:
					d.Decor[p.Ref]++
				case KindLoot:
					d.Loot[p.Ref]++
				default: // KindBeing/KindTrap は現状 placed に出ないので集計しない
				}
			}
			if _, ok := heroCenterpiece(seed); ok {
				d.Heroes++
			}
		}
		out[fac] = d
	}

	buf, err := json.MarshalIndent(out, "", "  ")
	require.NoError(t, err)

	g := goldie.New(t, goldie.WithNameSuffix(".json"))
	g.Assert(t, t.Name(), buf)
}
