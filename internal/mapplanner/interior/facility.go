package interior

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
)

// plannerDef は間取りテンプレの実装と、それが破綻しない最小寸法。
type plannerDef struct {
	fn         func(Rect, uint64) []PlannedRoom
	minW, minH consts.Tile
}

// planners は planner キーから間取り実装を引く。
var planners = map[oapi.PlannerKey]plannerDef{
	oapi.House:  {PlanHouseAny, 12, 9},
	oapi.Store:  {PlanStore, 12, 9},
	oapi.Clinic: {PlanClinic, 12, 9},
	oapi.Bsp:    {planBSP, 0, 0},
}

// planBSP は汎用の BSP 分割で面積最大を主室、残りを奥室にする。
func planBSP(footprint Rect, seed uint64) []PlannedRoom {
	rooms := SubdivideBuilding(footprint, seed)
	out := make([]PlannedRoom, len(rooms))
	for rank, ri := range roomOrderByArea(rooms) {
		role := roleBack
		if rank == 0 {
			role = roleMain
		}
		out[ri] = PlannedRoom{Room: rooms[ri], Role: role}
	}
	return out
}

// Furnish は建物の footprint と入口から、施設種別に応じた内装の配置を決定的に返す。footprint を外周が壁の
// 1部屋とみなし、door はその外周上の入口。多部屋の敷地計画は FurnishBuilding が担い、Furnish は単室で
// 施設種別の主室レシピ・密度・経年・flavor の直交軸を検証する単位になる。未登録の施設は error。
func Furnish(raws oapi.Raws, seed uint64, footprint Rect, door Vec, facility string) ([]Placed, error) {
	prof := rollProfile(seed)
	room := Room{Rect: footprint, Doorways: []Doorway{{X: door.X, Y: door.Y}}}
	main, err := facilityContent(raws, facility, seed)
	if err != nil {
		return nil, err
	}
	placed := FillRoom(seed, room, applyDensity(main, prof.density))
	// 時間の層。損傷レベルで略奪・生活痕・廃墟化の強度を変える。無傷の建物は新品のまま
	placed = Age(seed, room, placed, prof.damage)
	// 家具の隙間へ flavor machine を1つ置き、戦利品の無い空き箱部屋に character を与える
	flavor, err := flavorContent(raws)
	if err != nil {
		return nil, err
	}
	placed = Flavor(seed, room, placed, flavor)
	// 散らかりの小物を家具の隣へ落とし、生活感を足す
	return applyClutter(childSeed(seed, 11_300_000), room, placed, prof.clutter, roleMain), nil
}

// applyDensity は content の家具量を密度係数 factor(×/10)で増減する。個数1の必須什器は1を保ち、詰め物の
// 棚だけが増減する。密度は buildingProfile が建物ごとに引く直交軸で、同じ内装でもがらんとした店と品で
// 埋まった店を出し分ける。factor==10 は等倍で素通し。
func applyDensity(c Content, factor int) Content {
	if factor == 10 {
		return c
	}
	for gi := range c.Groups {
		for ii := range c.Groups[gi].Items {
			it := &c.Groups[gi].Items[ii]
			if it.Kind != KindFurniture {
				continue // 家具だけ密度を変える。戦利品・装飾はそのまま
			}
			it.Amount.Base = scaleAmount(it.Amount.Base, factor)
			it.Amount.Bonus = scaleAmount(it.Amount.Bonus, factor)
		}
	}
	return c
}

// scaleAmount は個数を f/10 倍する。元が1以上なら最低1を保ち、必須の1個が密度で消えないようにする。
func scaleAmount(v, f int) int {
	if v <= 0 {
		return v
	}
	if s := v * f / 10; s >= 1 {
		return s
	}
	return 1
}
