package overworld

import (
	"fmt"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner/interior"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
)

// interiorPropRaw は interior の抽象 Ref をゲームの raw prop 名へ写す。既存の prop へ寄せ、raw の無い抽象
// 什器は近い実物へ当てる。表に無い Ref は置かない。KindLoot の戦利品は含めない。urban の v1 は家具と
// 装飾だけを置き、施設固有の戦利品はアイテム設計が固まってから足す。
var interiorPropRaw = map[string]string{
	"register":      "register",
	"gondola":       "goods_shelf",
	"walkin_cooler": "refrigerator",
	"reception":     "desk",
	"waitchair":     "chair",
	"exam_bed":      "exam_bed",
	"medcabinet":    "medicine_cabinet",
	"bed":           "bed",
	"table":         "table",
	"chair":         "chair",
	"sofa":          "sofa",
	"closet":        "closet",
	"lantern":       "lantern",
	"plant":         "plant",
	"washer":        "washer",
	"pantry":        "dish_shelf",
	"barrel":        "barrel",
	"bathtub":       "bathtub",
	"toilet":        "toilet",
	"sink":          "sink",
	"desk":          "desk",
	"candle":        "candle",
	"carpet":        "carpet",
	"rubble":        "rubble",
	"debris":        "debris",
}

// furnishBuilding は建物区画 footprint を敷地計画し、Site が返す庭・壁・部屋を描いて内装で満たす。footprint を
// そのまま埋めず、interior.FurnishBuilding が前庭を空け坪庭を作り玄関を凹ませる。庭は土、壁は壁、残りの
// 部屋の床と戸口は床として描き、入口の扉と家具を spawn する。内装の乱数は建物幾何と別ストリーム 0x3 に
// する。壁判定の関数と占有タイルを返し、後段の敵配置が壁や家具の上に湧かないよう避けさせる。
func furnishBuilding(world w.World, g chunkGeom, footprint interior.Rect, door interior.Vec, orient gc.DoorOrientation, fac facilityType, seed uint64) (func(lx, ly consts.Tile) bool, map[consts.Coord[consts.Tile]]bool, error) {
	iseed := rand.New(rand.NewPCG(seed, 0x3)).Uint64()
	site, placed := interior.FurnishBuilding(iseed, footprint, door, string(fac))

	wallSet := make(map[interior.Vec]bool)
	for _, wv := range site.Walls() {
		wallSet[wv] = true
	}

	tiles := g.tiles.get()
	occupied := make(map[consts.Coord[consts.Tile]]bool)
	// 建物区画のタイルを Site から描く。庭→土、壁→壁、残り(部屋の床・戸口)→床。敵が湧かないよう壁は占有に入れる
	for y := footprint.Y; y < footprint.Y+footprint.H; y++ {
		for x := footprint.X; x < footprint.X+footprint.W; x++ {
			v := interior.Vec{X: x, Y: y}
			coord := consts.Coord[consts.Tile]{X: g.offsetX + consts.Tile(x), Y: g.offsetY + consts.Tile(y)}
			name := consts.TileNameFloor
			switch {
			case site.Garden[v]:
				name = consts.TileNameDirt
			case wallSet[v]:
				name = consts.TileNameDWall
			}
			if err := replaceTile(world, tiles, coord, name); err != nil {
				return nil, nil, fmt.Errorf("内装のタイル配置に失敗 (x=%d, y=%d): %w", coord.X, coord.Y, err)
			}
			if wallSet[v] {
				occupied[coord] = true
			}
		}
	}

	// 入口の扉を建物辺の site.Door へ置く。前庭ぶん内寄せした建物の辺にあり、前庭が街路との間に挟まる
	dcoord := consts.Coord[consts.Tile]{X: g.offsetX + consts.Tile(site.Door.X), Y: g.offsetY + consts.Tile(site.Door.Y)}
	if _, err := lifecycle.SpawnDoor(world, dcoord, orient); err != nil {
		return nil, nil, fmt.Errorf("内装の扉配置に失敗: %w", err)
	}

	// 家具と装飾を spawn する。写像できる Ref だけを建物の内側へ置く。坪庭の観葉もここで庭の土の上へ乗る
	for _, p := range placed {
		name, ok := interiorPropRaw[p.Ref]
		if !ok {
			continue // raw の無い戦利品や装飾は置かない
		}
		pos := consts.Coord[consts.Tile]{X: g.offsetX + consts.Tile(p.Pos.X), Y: g.offsetY + consts.Tile(p.Pos.Y)}
		if _, err := lifecycle.SpawnProp(world, name, pos.X, pos.Y); err != nil {
			return nil, nil, fmt.Errorf("内装の配置に失敗 (%s at %d,%d): %w", name, pos.X, pos.Y, err)
		}
		occupied[pos] = true
	}

	isWall := func(lx, ly consts.Tile) bool { return wallSet[interior.Vec{X: int(lx), Y: int(ly)}] }
	return isWall, occupied, nil
}
