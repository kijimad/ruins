package interior

import (
	"math"
	"sort"
)

// Placement は「どこへ置くか」の意味論。散布をやめ各 stuff に置き場所を持たせる主装置。
// 実体は文字列。docs/design/20260725_70.md の placement 語彙の Stage 2 部分集合。
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
		// hash 由来のジッタ 0..1 を弱く混ぜ、同 density の並びを決定的かつ非規則に崩す
		j := norm01(hashTile(seed, t))
		cands = append(cands, scoredTile{pos: t, score: d + j*0.15})
	}
	// スコア降順。同点は座標で安定化し、再訪一致を保証する
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].score != cands[j].score {
			return cands[i].score > cands[j].score
		}
		if cands[i].pos.Y != cands[j].pos.Y {
			return cands[i].pos.Y < cands[j].pos.Y
		}
		return cands[i].pos.X < cands[j].pos.X
	})

	n := min(count, len(cands))
	out := make([]Vec, n)
	for i := range n {
		out[i] = cands[i].pos
	}
	return out
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
		e := dist(t, Vec(d))
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

func abs(v int) int {
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
