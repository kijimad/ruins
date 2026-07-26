package overworld

import (
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner/interior"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
)

// buildingShell は建物外殻の footprint と入口。drawUrbanBuilding が計算し、内装 furnish が消費する。
type buildingShell struct {
	bx, by, bw, bh consts.Tile
	doorX, doorY   consts.Tile
}

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
	"pantry":        "dish_shelf",
	"barrel":        "barrel",
	"bathtub":       "bathtub",
	"toilet":        "toilet",
	"sink":          "sink",
	"desk":          "desk",
	"candle":        "candle",
	"carpet":        "carpet",
}

// furnishBuilding は建物外殻の内側を施設種別に応じた内装で満たす。interior.Furnish で決定的に配置を得て、
// raw prop 名へ写せる家具と装飾だけを外殻の壁の内側へ spawn する。内装の乱数は建物幾何と別ストリーム
// 0x3 にして、片方を変えても他方が動かないようにする。spawn したタイルの集合を返し、後段の敵配置が
// 家具の上に湧かないよう避けさせる。
func furnishBuilding(world w.World, g chunkGeom, shell buildingShell, fac facilityType, seed uint64) (map[consts.Coord[consts.Tile]]bool, error) {
	iseed := rand.New(rand.NewPCG(seed, 0x3)).Uint64()
	footprint := interior.Rect{X: int(shell.bx), Y: int(shell.by), W: int(shell.bw), H: int(shell.bh)}
	door := interior.Vec{X: int(shell.doorX), Y: int(shell.doorY)}

	occupied := make(map[consts.Coord[consts.Tile]]bool)
	for _, p := range interior.Furnish(iseed, footprint, door, string(fac)) {
		name, ok := interiorPropRaw[p.Ref]
		if !ok {
			continue // raw の無い戦利品や装飾は置かない
		}
		pos := consts.Coord[consts.Tile]{X: g.offsetX + consts.Tile(p.Pos.X), Y: g.offsetY + consts.Tile(p.Pos.Y)}
		if _, err := lifecycle.SpawnProp(world, name, pos.X, pos.Y); err != nil {
			return nil, fmt.Errorf("内装の配置に失敗 (%s at %d,%d): %w", name, pos.X, pos.Y, err)
		}
		occupied[pos] = true
	}
	return occupied, nil
}
