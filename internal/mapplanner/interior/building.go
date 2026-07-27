package interior

// 建物は footprint を分割文法で複数の部屋へ割る。各部屋は自分の壁を持ち、隣接する部屋は戸口で繋ぐ。
// content システムはこの部屋群を受け、部屋ごとに中身を流し込む。
// ここでは決定的 BSP を既定にし、部屋の連結と建物入口までを担う。ゾーン分類は後続 Stage。

const (
	minRoomSide = 6 // 部屋の一辺の最小タイル数。内側床は最小 4
	maxSplitDep = 3 // BSP の分割深さ上限。26x18 で概ね 6 室に収まり、住居として読める粒度になる
)

// SubdivideBuilding は footprint を BSP で複数部屋へ分割し戸口で相互連結する。建物入口は開けない。返す
// 部屋は相互に連結し、外から入った部屋から全室へ到達できる。テンプレの無い施設や狭い footprint の
// フォールバックとして planRooms が使う。入口は敷地計画 Site が別に開ける。
func SubdivideBuilding(footprint Rect, seed uint64) []Room {
	rects := bspSplit(footprint, seed, 0)
	rooms := make([]Room, len(rects))
	for i, r := range rects {
		rooms[i] = Room{Rect: r}
	}
	connectRooms(rooms, seed)
	return rooms
}

// bspSplit は rect を長辺で再帰的に2分し、葉の矩形を部屋として返す。分割線は両子が共有する壁になる。
func bspSplit(rect Rect, seed uint64, depth int) []Rect {
	canV := rect.W >= 2*minRoomSide+1
	canH := rect.H >= 2*minRoomSide+1
	if depth >= maxSplitDep || (!canV && !canH) {
		return []Rect{rect}
	}
	if canV && (!canH || rect.W >= rect.H) {
		lo, hi := rect.X+minRoomSide, rect.X+rect.W-minRoomSide
		pos := lo + int(childSeed(seed, 0)%uint64(hi-lo+1))
		left := Rect{X: rect.X, Y: rect.Y, W: pos - rect.X + 1, H: rect.H}
		right := Rect{X: pos, Y: rect.Y, W: rect.X + rect.W - pos, H: rect.H}
		return append(bspSplit(left, childSeed(seed, 1), depth+1), bspSplit(right, childSeed(seed, 2), depth+1)...)
	}
	lo, hi := rect.Y+minRoomSide, rect.Y+rect.H-minRoomSide
	pos := lo + int(childSeed(seed, 3)%uint64(hi-lo+1))
	top := Rect{X: rect.X, Y: rect.Y, W: rect.W, H: pos - rect.Y + 1}
	bot := Rect{X: rect.X, Y: pos, W: rect.W, H: rect.Y + rect.H - pos}
	return append(bspSplit(top, childSeed(seed, 4), depth+1), bspSplit(bot, childSeed(seed, 5), depth+1)...)
}

// connectRooms は隣接する部屋対を union-find の全域木で最小限に戸口で繋ぐ。木なので過剰な扉を作らず、
// かつ全部屋が連結する。
func connectRooms(rooms []Room, seed uint64) {
	uf := newUnionFind(len(rooms))
	for i := range rooms {
		for j := i + 1; j < len(rooms); j++ {
			ri, rj := roomIndex(i), roomIndex(j)
			if uf.find(ri) == uf.find(rj) {
				continue
			}
			if d, ok := sharedDoorway(rooms[i].Rect, rooms[j].Rect, childSeed(seed, i*97+j)); ok {
				rooms[i].Doorways = append(rooms[i].Doorways, d)
				rooms[j].Doorways = append(rooms[j].Doorways, d)
				uf.union(ri, rj)
			}
		}
	}
}

// sharedDoorway は2部屋が壁を共有していれば、その壁上の1タイルを戸口として返す。両側が床になる位置を選ぶ。
func sharedDoorway(a, b Rect, seed uint64) (Doorway, bool) {
	// 縦の共有壁: a の右列 == b の左列、または逆
	if col, ok := sharedColumn(a, b); ok {
		lo := max(a.Y, b.Y) + 1
		hi := min(a.Y+a.H, b.Y+b.H) - 2
		if lo <= hi {
			y := lo + int(seed%uint64(hi-lo+1))
			return Doorway{X: col, Y: y}, true
		}
	}
	// 横の共有壁: a の下行 == b の上行、または逆
	if row, ok := sharedRow(a, b); ok {
		lo := max(a.X, b.X) + 1
		hi := min(a.X+a.W, b.X+b.W) - 2
		if lo <= hi {
			x := lo + int(seed%uint64(hi-lo+1))
			return Doorway{X: x, Y: row}, true
		}
	}
	return Doorway{}, false
}

func sharedColumn(a, b Rect) (int, bool) {
	if a.X+a.W-1 == b.X {
		return b.X, true
	}
	if b.X+b.W-1 == a.X {
		return a.X, true
	}
	return 0, false
}

func sharedRow(a, b Rect) (int, bool) {
	if a.Y+a.H-1 == b.Y {
		return b.Y, true
	}
	if b.Y+b.H-1 == a.Y {
		return a.Y, true
	}
	return 0, false
}

// roomIndex は rooms スライスの添字で、部屋を1つ指す。連結・BFS・union-find が部屋を参照するのに使い、
// 距離や個数など他の int と混ざらないよう型で区別する。
type roomIndex int

// noRoom は該当する部屋が無いことを表す番兵。入口が見つからない、到達不能などに使う。
const noRoom roomIndex = -1

// roomDepth は入口部屋からの戸口グラフ上のホップ数。タイル距離ではなく、戸口を1つ通るごとに1増える位相量。
// 0 が入口の間で、増えるほど奥まった私的な部屋になる。
type roomDepth int

// depthUnreachable は入口から戸口で辿れない部屋の深さを表す番兵。
const depthUnreachable roomDepth = -1

// roomDepths は建物入口を持つ部屋を起点に、戸口グラフ上の各部屋の入口からの距離を BFS で返す。
// これがゾーン分類の基礎で、距離 0 は入口の間、距離が増えるほど奥まった私的な部屋になる。手前を公共、
// 奥を私的とする住居の動線を、面積という幾何量でなく到達構造という位相量で表す。入口が無い、または
// 入口から孤立した部屋は depthUnreachable にする。SubdivideBuilding は全部屋を連結するので通常は出ない。
func roomDepths(rooms []Room) []roomDepth {
	depths := make([]roomDepth, len(rooms))
	for i := range depths {
		depths[i] = depthUnreachable
	}

	// 戸口タイルからそれを共有する部屋群を引く。部屋どうしの内部戸口は2部屋が持ち、建物入口は1部屋だけが
	// 持つ。だから共有されない戸口を持つ部屋が入口の間になる。玄関ポーチで入口が外周から1マス内へ下がっても、
	// 入口はやはり1部屋しか持たないので、外周判定でなく共有数で入口を特定すればポーチに強い
	doorRooms := make(map[Vec][]roomIndex)
	for i, r := range rooms {
		for _, d := range r.Doorways {
			doorRooms[Vec(d)] = append(doorRooms[Vec(d)], roomIndex(i))
		}
	}
	start := noRoom
	for i, r := range rooms {
		for _, d := range r.Doorways {
			if len(doorRooms[Vec(d)]) == 1 {
				start = roomIndex(i)
				break
			}
		}
		if start != noRoom {
			break
		}
	}
	if start == noRoom {
		return depths
	}

	depths[start] = 0
	queue := []roomIndex{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, d := range rooms[cur].Doorways {
			for _, nb := range doorRooms[Vec(d)] {
				if depths[nb] == depthUnreachable {
					depths[nb] = depths[cur] + 1
					queue = append(queue, nb)
				}
			}
		}
	}
	return depths
}

// --- union-find ---

type unionFind struct{ parent []roomIndex }

func newUnionFind(n int) *unionFind {
	p := make([]roomIndex, n)
	for i := range p {
		p[i] = roomIndex(i)
	}
	return &unionFind{parent: p}
}

func (u *unionFind) find(i roomIndex) roomIndex {
	for u.parent[i] != i {
		u.parent[i] = u.parent[u.parent[i]]
		i = u.parent[i]
	}
	return i
}

func (u *unionFind) union(i, j roomIndex) { u.parent[u.find(i)] = u.find(j) }
