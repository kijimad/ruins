package interior

// 敷地計画。footprint を矩形のまま全部屋で埋めず、建物を内側へ取り、差分を庭にする。外形が clean な矩形
// でなくなり、広すぎる部屋の余剰を庭で引き算する。docs/design/20260725_70.md 追記その8。
//
//   - 軸B セットバック前庭: 入口側に前庭を空け、建物を footprint より内側へ取る
//   - 軸A 坪庭: 建物内の1室を庭にして家具を置かない。囲われた光庭になり広い1室の余剰を消す
//   - 軸C 玄関ポーチの凹み: 入口の壁を1マス凹ませ、外形を非矩形にして玄関を読ませる

// frontYard は入口側に空ける前庭の奥行き。sideYard は他辺の余白。建物を内寄せして庭を作る。
const (
	frontYard = 3
	sideYard  = 1
)

// roleGarden は坪庭の部屋の役割名。内側を庭タイルにして家具を置かない。
const roleGarden = "garden"

// Site は footprint 内の敷地計画。建物輪郭・庭・部屋・入口を分ける。overworld と VRT が同じ Site を描き、
// footprint をそのまま埋めず建物を内側へ取る。Garden は建物外のタイルで dirt として描き観葉を点在させる。
// ExtraWall はポーチの凹みで足す壁で、Building 内側だが床でなく壁にするタイル。
type Site struct {
	Footprint Rect
	Building  Rect
	Garden    map[Vec]bool
	ExtraWall map[Vec]bool
	Door      Vec
	Rooms     []PlannedRoom
}

// planSite は footprint を建物と庭に分ける。入口側に前庭を空けて建物を内寄せし、入口を建物辺へ寄せ、玄関を
// 凹ませ、建物内の1室を坪庭にする。建物が施設テンプレに満たない狭さなら軸を抑え、最低限の建物は必ず作る。
func planSite(footprint Rect, seed uint64, door Vec, facility string) Site {
	building := insetBuilding(footprint, door)
	side := doorSide(footprint, door)

	garden := footprintMinusBuilding(footprint, building)
	extra := make(map[Vec]bool)

	rooms, roles := planRooms(building, seed, facility)
	labeled := make([]PlannedRoom, len(rooms))
	for i := range rooms {
		labeled[i] = PlannedRoom{Room: rooms[i], Role: roles[i]}
	}

	// 入口は建物辺のうち前室の内側に面する位置へ。仕切り列に当たると部屋でなく壁に開くので、前室の内側の
	// 帯へ寄せる。玄関を凹ませるかは seed で選ぶ。全ての玄関が凹むと単調なので、半分は直線の開口部にする
	bdoor := chooseDoor(building, labeled, side, door)
	if childSeed(seed, 8_100_000)%2 == 0 {
		bdoor = carvePorch(building, bdoor, side, garden, extra)
	}
	attachDoor(labeled, bdoor, side)

	// 軸A 坪庭。入口の部屋と廊下を除く広めの1室を庭へ振り替える。家具を置かず内側を庭タイルにする
	if gi := pickGardenRoom(labeled, bdoor, seed); gi >= 0 {
		labeled[gi].Role = roleGarden
		for _, v := range labeled[gi].Room.Rect.interiorTiles() {
			garden[v] = true
		}
	}

	return Site{Footprint: footprint, Building: building, Garden: garden, ExtraWall: extra, Door: bdoor, Rooms: labeled}
}

// attachDoor は入口の内側の部屋へ door を戸口として足す。入口の1マス内側を含む部屋を探す。ポーチで下げた
// 入口でも、内向きの隣が最前列の部屋の内側に来るので同じ判定で拾える。
func attachDoor(rooms []PlannedRoom, door Vec, s side) {
	step := porchStep(s)
	inner := Vec{X: door.X + step.X, Y: door.Y + step.Y}
	for i := range rooms {
		if rooms[i].Room.Rect.containsInterior(inner) {
			rooms[i].Room.Doorways = append(rooms[i].Room.Doorways, Doorway(door))
			return
		}
	}
}

// insetBuilding は footprint を入口側に前庭ぶん、他辺に余白ぶん内寄せした建物矩形を返す。狭くて前庭を
// 取ると建物がテンプレ下限を割る footprint では、内寄せを諦めて footprint をそのまま建物にする。
func insetBuilding(footprint Rect, door Vec) Rect {
	f := footprint
	var b Rect
	switch doorSide(footprint, door) {
	case sideNorth:
		b = Rect{X: f.X + sideYard, Y: f.Y + frontYard, W: f.W - 2*sideYard, H: f.H - frontYard - sideYard}
	case sideSouth:
		b = Rect{X: f.X + sideYard, Y: f.Y + sideYard, W: f.W - 2*sideYard, H: f.H - frontYard - sideYard}
	case sideWest:
		b = Rect{X: f.X + frontYard, Y: f.Y + sideYard, W: f.W - frontYard - sideYard, H: f.H - 2*sideYard}
	default: // sideEast
		b = Rect{X: f.X + sideYard, Y: f.Y + sideYard, W: f.W - frontYard - sideYard, H: f.H - 2*sideYard}
	}
	// 建物が小さすぎると部屋も戸口も作れない。最小 5x5 を割るなら内寄せをやめる
	if b.W < 5 || b.H < 5 {
		return footprint
	}
	return b
}

// chooseDoor は建物辺の入口位置を、その辺に面する部屋の内側の帯へ落ちるよう選ぶ。overworld が街路向きに
// 決めた door の横位置を望みとし、前室の内側に入ればそのまま、仕切り列に当たるなら最も近い前室の内側へ
// 寄せる。これで入口が壁でなく部屋へ開く。
func chooseDoor(building Rect, rooms []PlannedRoom, s side, desired Vec) Vec {
	bottom, right := building.Y+building.H-1, building.X+building.W-1
	switch s {
	case sideNorth:
		return Vec{X: frontSlot(building, rooms, s, desired.X), Y: building.Y}
	case sideSouth:
		return Vec{X: frontSlot(building, rooms, s, desired.X), Y: bottom}
	case sideWest:
		return Vec{X: building.X, Y: frontSlot(building, rooms, s, desired.Y)}
	default: // sideEast
		return Vec{X: right, Y: frontSlot(building, rooms, s, desired.Y)}
	}
}

// frontSlot は辺 s に面する部屋の内側の帯のうち、望みの横位置 desired に最も近い座標を返す。望みが前室の
// 内側にあればそのまま返す。前室が無ければ望みをそのまま返す。
func frontSlot(building Rect, rooms []PlannedRoom, s side, desired int) int {
	bestLo, bestHi, found := 0, 0, false
	for _, hr := range rooms {
		lo, hi, ok := frontSpan(hr.Room.Rect, building, s)
		if !ok {
			continue
		}
		if desired >= lo && desired <= hi {
			return desired
		}
		if !found || spanDist(lo, hi, desired) < spanDist(bestLo, bestHi, desired) {
			bestLo, bestHi, found = lo, hi, true
		}
	}
	if !found {
		return desired
	}
	return clamp(desired, bestLo, bestHi)
}

// frontSpan は矩形が建物の辺 s に接するとき、その内側の帯 [lo, hi] を辺に沿う軸で返す。接さなければ ok=false。
// 接する条件は矩形の該当辺が建物の該当辺と同じ座標にあること。入口はこの帯に落とすと部屋の内側へ開く。
func frontSpan(r, building Rect, s side) (lo, hi int, ok bool) {
	switch s {
	case sideNorth:
		return r.X + 1, r.X + r.W - 2, r.Y == building.Y && r.W >= 3
	case sideSouth:
		return r.X + 1, r.X + r.W - 2, r.Y+r.H-1 == building.Y+building.H-1 && r.W >= 3
	case sideWest:
		return r.Y + 1, r.Y + r.H - 2, r.X == building.X && r.H >= 3
	default: // sideEast
		return r.Y + 1, r.Y + r.H - 2, r.X+r.W-1 == building.X+building.W-1 && r.H >= 3
	}
}

// spanDist は帯 [lo, hi] から点 v までの距離。帯の中なら 0。
func spanDist(lo, hi, v int) int {
	if v < lo {
		return lo - v
	}
	if v > hi {
		return v - hi
	}
	return 0
}

// carvePorch は入口を建物内へ1マス下げて玄関ポーチの凹みを作る。元の入口の1マスだけを庭に開け、両隣の
// 前壁は残し、下げた入口の両隣を壁にして 1幅1奥のポケットにする。開口を1幅にすると前壁と側壁が直交で
// 繋がり、角を斜めに視線や移動が抜けない。開口を3幅にすると口の角で前壁と側壁が斜め隣接になり漏れる。
// 凹みを作れない小さな建物では下げず元の door を返す。
func carvePorch(building Rect, door Vec, s side, garden, extra map[Vec]bool) Vec {
	step := porchStep(s)
	inner := Vec{X: door.X + step.X, Y: door.Y + step.Y}
	if !building.containsInterior(inner) {
		return door // 凹みを作る余地がない
	}
	// ポケットの側壁は下げた入口の両隣。角に寄りすぎて片側が建物外に出るなら凹ませない
	along := porchAlong(s)
	for d := -1; d <= 1; d += 2 {
		if !building.contains(Vec{X: inner.X + along.X*d, Y: inner.Y + along.Y*d}) {
			return door
		}
	}
	garden[door] = true                                           // 元の入口の1マスだけをポケットの口として庭に開ける。両隣の前壁は残す
	extra[Vec{X: inner.X + along.X, Y: inner.Y + along.Y}] = true // 下げた入口の両隣を側壁に
	extra[Vec{X: inner.X - along.X, Y: inner.Y - along.Y}] = true
	return inner
}

// floorSet は坪庭でない部屋の内側床タイルの集合を返す。坪庭の室の内側は Garden に入るのでここには来ない。
func (s Site) floorSet() map[Vec]bool {
	floor := make(map[Vec]bool)
	for _, hr := range s.Rooms {
		if hr.Role == roleGarden {
			continue
		}
		for _, v := range hr.Room.Rect.interiorTiles() {
			floor[v] = true
		}
	}
	return floor
}

// doorSet は全部屋の戸口と建物入口の集合を返す。
func (s Site) doorSet() map[Vec]bool {
	door := map[Vec]bool{s.Door: true}
	for _, hr := range s.Rooms {
		for _, d := range hr.Room.Doorways {
			door[Vec(d)] = true
		}
	}
	return door
}

// Walls は Site の全壁タイルを footprint 内で返す。建物外周・間仕切り・ポーチの側壁。庭・床・戸口は除く。
// overworld が壁タイルを描き、敵配置が壁を避けるのに使う。庭は footprint から建物を引いた集合なので、
// 非庭タイルは必ず建物内側にあり、床でも戸口でもなければ壁になる。ポーチの側壁は床の上に足す壁。
func (s Site) Walls() []Vec {
	floor := s.floorSet()
	door := s.doorSet()
	var walls []Vec
	for y := s.Footprint.Y; y < s.Footprint.Y+s.Footprint.H; y++ {
		for x := s.Footprint.X; x < s.Footprint.X+s.Footprint.W; x++ {
			v := Vec{X: x, Y: y}
			if s.Garden[v] || door[v] || (floor[v] && !s.ExtraWall[v]) {
				continue
			}
			walls = append(walls, v)
		}
	}
	return walls
}

// footprintMinusBuilding は footprint 内で建物矩形の外側のタイル集合を庭として返す。前庭と余白がここに入る。
func footprintMinusBuilding(footprint, building Rect) map[Vec]bool {
	garden := make(map[Vec]bool)
	if building == footprint {
		return garden
	}
	for y := footprint.Y; y < footprint.Y+footprint.H; y++ {
		for x := footprint.X; x < footprint.X+footprint.W; x++ {
			v := Vec{X: x, Y: y}
			if !building.contains(v) {
				garden[v] = true
			}
		}
	}
	return garden
}

// pickGardenRoom は坪庭にする部屋の添字を返す。入口の部屋・廊下・狭い部屋は避け、面積最大の居室を選ぶ。
// 該当が無ければ -1。1棟に坪庭は1つまでにして、部屋を庭で潰しすぎないようにする。
func pickGardenRoom(rooms []PlannedRoom, door Vec, seed uint64) int {
	// 4室に満たない小さな建物は坪庭を作らない。1室を庭に振ると部屋が減りすぎる
	if len(rooms) < 4 {
		return -1
	}
	// 半分の建物だけ坪庭を持つ。全棟に庭があると単調
	if childSeed(seed, 8_000_000)%2 != 0 {
		return -1
	}
	best, bestArea := -1, 0
	for i, hr := range rooms {
		r := hr.Room.Rect
		if hr.Role == "corridor" || containsDoor(r, door) {
			continue // 入口の部屋と廊下は残す
		}
		if r.W < 6 || r.H < 6 {
			continue // 狭い部屋は坪庭にしない。水回りを潰さない
		}
		if a := r.W * r.H; a > bestArea {
			best, bestArea = i, a
		}
	}
	return best
}

// containsDoor は矩形が door を戸口として面するかを、外周1マスの近さで判定する。
func containsDoor(r Rect, door Vec) bool {
	return door.X >= r.X-1 && door.X <= r.X+r.W && door.Y >= r.Y-1 && door.Y <= r.Y+r.H
}

// side は建物・区画の辺の向き。
type side int

const (
	sideNorth side = iota
	sideSouth
	sideWest
	sideEast
)

// doorSide は footprint のどの辺に door が乗るかを返す。
func doorSide(footprint Rect, door Vec) side {
	switch {
	case door.Y == footprint.Y:
		return sideNorth
	case door.Y == footprint.Y+footprint.H-1:
		return sideSouth
	case door.X == footprint.X:
		return sideWest
	default:
		return sideEast
	}
}

// porchStep は入口を建物内へ下げる向き。辺の内向き。
func porchStep(s side) Vec {
	switch s {
	case sideNorth:
		return Vec{X: 0, Y: 1}
	case sideSouth:
		return Vec{X: 0, Y: -1}
	case sideWest:
		return Vec{X: 1, Y: 0}
	default:
		return Vec{X: -1, Y: 0}
	}
}

// porchAlong はポーチの走る向き。辺に沿う方向で、北・南は横、西・東は縦。
func porchAlong(s side) Vec {
	if s == sideNorth || s == sideSouth {
		return Vec{X: 1, Y: 0}
	}
	return Vec{X: 0, Y: 1}
}

// contains は矩形が v を含むかを返す。外周を含む。
func (r Rect) contains(v Vec) bool {
	return v.X >= r.X && v.X < r.X+r.W && v.Y >= r.Y && v.Y < r.Y+r.H
}

// clamp は v を [lo, hi] に収める。
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
