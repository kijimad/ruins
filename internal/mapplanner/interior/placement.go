package interior

import (
	"math"
	"sort"

	"github.com/kijimaD/ruins/internal/consts"
)

// Placement は「どこへ置くか」の意味論。散布をやめ各 stuff に置き場所を持たせる主装置。
// 実体は文字列。
type Placement string

const (
	// PlaceCenter は部屋中心寄り。中央の島什器など
	PlaceCenter Placement = "center"
	// PlaceWall は壁際。戸口隣接面は避ける。棚・冷蔵ケース
	PlaceWall Placement = "wall"
	// PlaceFullArea は全域に散らす。ゴミ・小物
	PlaceFullArea Placement = "full_area"
	// PlaceNearDoor は入口近く。レジ・カート
	PlaceNearDoor Placement = "near_door"
	// PlaceFarFromDoor は入口から遠い奥。冷蔵ケース・貴重品
	PlaceFarFromDoor Placement = "far_from_door"
	// PlaceRow は棚を1列おきの平行列に置き、間の列を通路として空ける。ゴンドラ・ラック
	PlaceRow Placement = "row"
)

// scoredTile は候補タイルと、placement が与えた密度と決定的ジッタの合成スコア。
type scoredTile struct {
	pos   Vec
	score float64
}

// selectTiles は placement 意味論に従い、部屋の空き床から count 枚のタイルを決定的に選ぶ。
// 各候補に density を与え、hash 由来のジッタで tie を崩し、スコア降順に安定 sort して上位を取る。
// 密度場 + 決定的間引きの最小実装。map を抽選に使わず、候補は slice に集めて sort する。
func selectTiles(room Room, p Placement, occupied map[Vec]bool, seed uint64, count int) []Vec {
	if count <= 0 {
		return nil
	}
	c := room.Rect.center()
	maxDist := float64(room.Rect.W + room.Rect.H)

	cands := make([]scoredTile, 0, room.Rect.W*room.Rect.H)
	for _, t := range room.Rect.interiorTiles() {
		if occupied[t] || isDoorwayAdjacent(room, t) {
			continue
		}
		d := placementDensity(room, p, t, c, maxDist)
		if d <= 0 {
			continue
		}
		cands = append(cands, scoredTile{pos: t, score: d})
	}

	if p == PlaceRow {
		// 棚は列優先で詰め、連続した平行列にする。散らさず x→y の順に埋めるので棚が線に見える
		sortColumnMajor(cands)
	} else {
		// 密度場 + hash ジッタ。同 density の並びを決定的かつ非規則に崩し、スコア降順に安定 sort する
		for i := range cands {
			cands[i].score += norm01(hashTile(seed, cands[i].pos)) * 0.15
		}
		sortByScore(cands)
	}

	n := min(count, len(cands))
	out := make([]Vec, n)
	for i := range n {
		out[i] = cands[i].pos
	}
	return out
}

// sortColumnMajor は候補を x→y の順に並べる。棚を左から列ごとに連続で詰め、平行列を線に見せる。
func sortColumnMajor(cands []scoredTile) {
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].pos.X != cands[j].pos.X {
			return cands[i].pos.X < cands[j].pos.X
		}
		return cands[i].pos.Y < cands[j].pos.Y
	})
}

// sortByScore はスコア降順に並べ、同点は座標で安定化して再訪一致を保証する。
func sortByScore(cands []scoredTile) {
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].score != cands[j].score {
			return cands[i].score > cands[j].score
		}
		if cands[i].pos.Y != cands[j].pos.Y {
			return cands[i].pos.Y < cands[j].pos.Y
		}
		return cands[i].pos.X < cands[j].pos.X
	})
}

// placementDensity は placement ごとの「タイル→スカラ密度」純関数。0 は置かない、大きいほど置きやすい。
func placementDensity(room Room, p Placement, t, center Vec, maxDist float64) float64 {
	switch p {
	case PlaceCenter:
		return 1 - dist(t, center)/maxDist
	case PlaceWall:
		if !nextToPerimeter(room.Rect, t) {
			return 0
		}
		return 1
	case PlaceFullArea:
		// パリティ格子に限定して通行を残す
		if (t.X+t.Y)%2 != 0 {
			return 0
		}
		return 0.5
	case PlaceNearDoor:
		return 1 - nearestDoorDist(room, t)/maxDist
	case PlaceFarFromDoor:
		return nearestDoorDist(room, t) / maxDist
	case PlaceRow:
		// 内側の左端から1列おきを棚列にし、間の列は通路として density 0 で空ける
		if (t.X-(room.Rect.X+1))%2 != 0 {
			return 0
		}
		return 1
	}
	panic("未知の Placement: " + string(p))
}

// nextToPerimeter は t が外周の壁に隣接する内側タイルかを返す。
func nextToPerimeter(r Rect, t Vec) bool {
	return t.X == r.X+1 || t.X == r.X+r.W-2 || t.Y == r.Y+1 || t.Y == r.Y+r.H-2
}

// isDoorwayAdjacent は t が戸口そのものか戸口の直前かを返す。通路を塞がないため候補から外す。
func isDoorwayAdjacent(room Room, t Vec) bool {
	for _, d := range room.Doorways {
		dx := abs(t.X - d.X)
		dy := abs(t.Y - d.Y)
		if dx <= 1 && dy <= 1 {
			return true
		}
	}
	return false
}

// nearestDoorDist は t から最も近い戸口までのユークリッド距離。戸口が無ければ大きな値。
func nearestDoorDist(room Room, t Vec) float64 {
	if len(room.Doorways) == 0 {
		return float64(room.Rect.W + room.Rect.H)
	}
	best := -1.0
	for _, d := range room.Doorways {
		e := dist(t, d)
		if best < 0 || e < best {
			best = e
		}
	}
	return best
}

func dist(a, b Vec) float64 {
	dx := float64(a.X - b.X)
	dy := float64(a.Y - b.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

func abs(v consts.Tile) consts.Tile {
	if v < 0 {
		return -v
	}
	return v
}

// hashTile は seed とタイル座標から決定的な 64bit を撹拌する。splitmix64 の finalizer を使う。
func hashTile(seed uint64, t Vec) uint64 {
	x := seed
	x ^= uint64(uint32(t.X)) * 0x9E3779B97F4A7C15
	x ^= uint64(uint32(t.Y)) * 0xC2B2AE3D27D4EB4F
	x ^= x >> 30
	x *= 0xBF58476D1CE4E5B9
	x ^= x >> 27
	x *= 0x94D049BB133111EB
	x ^= x >> 31
	return x
}

// norm01 は uint64 を 0..1 の float へ正規化する。
func norm01(v uint64) float64 {
	return float64(v>>11) / float64(1<<53)
}
