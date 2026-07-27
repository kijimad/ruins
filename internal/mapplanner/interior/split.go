package interior

// 分割プリミティブ。施設テンプレに散っていた「帯をN分割し各室を背骨へ繋ぐ」反復と、退化を防ぐ最小サイズの
// 担保を1箇所へ集約する。docs/design/20260725_70.md 追記その11。分割文法の概念だけを翻案し、全文法エンジンは
// 作らない。house・store・clinic はこの builder の短い合成として書ける。

// splitAxis は帯を割る向き。列は縦線でX方向、行は横線でY方向に割る。
type splitAxis int

const (
	splitCols splitAxis = iota // 縦線で列へ割る
	splitRows                  // 横線で行へ割る
)

// minRoomSideSplit は分割後の各室の一辺の最小タイル数。内側床を1以上残す下限。散らばっていた clamp を
// この1定数へ集約し、strip がすべての分割でこの下限を保証する。
const minRoomSideSplit = 3

// wetBlockSide は浴室・トイレを寄せる水回り小ブロックの短辺。1〜2タイルの什器しか置かないので居室より
// 小さく固定し、建物が広がっても水回りだけは肥大させない。
const wetBlockSide = 4

// cell は帯の1室。key は矩形表の鍵、role は役割ラベル、weight は帯内の面積比。
type cell struct {
	key    string
	role   roleName
	weight int
}

// layout は rectOf・conns・order を溜める間取りの builder。build で既存 assembleRooms へ渡す。seed を保持し、
// strip のジッタ種を呼び出し順で導いて決定性を保つ。
type layout struct {
	seed   uint64
	rect   map[string]Rect
	conns  [][2]string
	order  []roomRole
	nsplit int
}

func newLayout(seed uint64) *layout {
	return &layout{seed: seed, rect: map[string]Rect{}}
}

// room は葉の部屋を1つ足す。
func (l *layout) room(key string, role roleName, r Rect) {
	l.rect[key] = r
	l.order = append(l.order, roomRole{key: key, role: role})
}

// connect は2部屋を戸口で繋ぐ指定を足す。実際の戸口位置は build の wireDoorways が共有壁から決める。
func (l *layout) connect(a, b string) {
	l.conns = append(l.conns, [2]string{a, b})
}

// strip は base を axis 方向に cells の weight 比で分割し、各室を spine へ扇状に連結する。spine が空なら連結
// しない。分割線をジッタで揺らし、各室が最小サイズを割らないよう clamp する。共有壁の座標一致をここで担保し、
// テンプレ側の分割算術と個別 clamp を無くす。
func (l *layout) strip(base Rect, axis splitAxis, spine string, cells []cell) {
	l.nsplit++
	weights := make([]int, len(cells))
	for i, c := range cells {
		weights[i] = max(1, c.weight)
	}
	lo, hi := base.X, base.X+base.W-1
	if axis == splitRows {
		lo, hi = base.Y, base.Y+base.H-1
	}
	bounds := l.splitSpan(lo, hi, weights)
	for i, c := range cells {
		r := Rect{X: bounds[i], Y: base.Y, W: bounds[i+1] - bounds[i] + 1, H: base.H}
		if axis == splitRows {
			r = Rect{X: base.X, Y: bounds[i], W: base.W, H: bounds[i+1] - bounds[i] + 1}
		}
		l.room(c.key, c.role, r)
		if spine != "" {
			l.connect(spine, c.key)
		}
	}
}

// splitSpan は [lo, hi] を weights の比で len(weights) 個へ割った境界列を返す。隣接室は境界座標を共有壁として
// 共有する。内部境界を seed で ±1 揺らし、各室が minRoomSideSplit を割らないよう左右から押し戻して clamp する。
func (l *layout) splitSpan(lo, hi int, weights []int) []int {
	n := len(weights)
	bounds := make([]int, n+1)
	bounds[0], bounds[n] = lo, hi
	span, total := hi-lo+1, 0
	for _, w := range weights {
		total += w
	}
	acc := 0
	for i := 1; i < n; i++ {
		acc += weights[i-1]
		bounds[i] = jitterSplit(l.seed, 5_100_000+l.nsplit*16+i, lo+span*acc/total)
	}
	// 各室 >= minRoomSideSplit を保つ。まず左から前室に押され、次に右から末尾室に押し戻す
	for i := 1; i <= n; i++ {
		if bounds[i]-bounds[i-1]+1 < minRoomSideSplit {
			bounds[i] = bounds[i-1] + minRoomSideSplit - 1
		}
	}
	for i := n - 1; i >= 0; i-- {
		if bounds[i+1]-bounds[i]+1 < minRoomSideSplit {
			bounds[i] = bounds[i+1] - minRoomSideSplit + 1
		}
	}
	return bounds
}

// wetBlock は block を長辺で2分し、浴室を手前、トイレを奥に小さく置く。parent→bath→toilet で連結し、
// トイレは浴室の奥へ nest する。水回りは1〜2タイルの什器しか置かないので居室より小さく保つ。扇状連結の
// strip と違い、トイレは spine に接せず浴室経由で入るので専用にする。
func (l *layout) wetBlock(block Rect, parent string) {
	l.nsplit++
	if block.W >= block.H {
		mid := (block.X + block.X + block.W) / 2 // 左右に割る
		l.room("bath", "bath", Rect{X: block.X, Y: block.Y, W: mid - block.X + 1, H: block.H})
		l.room("toilet", "toilet", Rect{X: mid, Y: block.Y, W: block.X + block.W - 1 - mid + 1, H: block.H})
	} else {
		mid := (block.Y + block.Y + block.H) / 2 // 上下に割る
		l.room("bath", "bath", Rect{X: block.X, Y: block.Y, W: block.W, H: mid - block.Y + 1})
		l.room("toilet", "toilet", Rect{X: block.X, Y: mid, W: block.W, H: block.Y + block.H - 1 - mid + 1})
	}
	l.connect(parent, "bath")
	l.connect("bath", "toilet")
}

// build は溜めた間取りを PlannedRoom 列へ組む。戸口の位置抽選は wireDoorways が共有壁から決める。
func (l *layout) build() []PlannedRoom {
	return assembleRooms(l.rect, wireDoorways(l.rect, l.seed, l.conns), l.order)
}
