package interior

// 民家は廊下型のテンプレートで間取りを作る。玄関という狭い前室から廊下が背骨として伸び、各室は廊下に
// 面して開く。浴室・脱衣所・トイレは奥の小部屋クラスタに寄せる。純 BSP は均一な部屋しか作れず、廊下と
// いう多数の部屋に面する通路も狭い前室も表現できないため、住居の believability には間取りの階層を保証
// するテンプレートを使う。汎用 BSP の SplitBuilding は診療所など他用途に残す。
//
// 動線: 入口 → 玄関 → 廊下 → 居間・台所・寝室・脱衣所。脱衣所 → 浴室。玄関 → トイレ → 納戸。
// どの部屋も玄関から辿れる連結木になる。

// HouseRoom は役割付きの部屋。廊下型の間取りは幾何と一緒に役割まで決める。ゾーン分類のように距離から
// 役割を推すのでなく、テンプレートが玄関や浴室を名指しする。
type HouseRoom struct {
	Room Room
	Role string
}

// PlanHouse は footprint に廊下型の民家間取りを決定的に生成する。返す部屋は戸口で連結され、どの部屋にも
// 入口から到達できる。footprint は概ね 24x16 以上を前提にする。
func PlanHouse(footprint Rect, seed uint64) []HouseRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	right, bottom := x0+w-1, y0+h-1

	// 縦の廊下を中央やや左に通し、左右に部屋を並べる
	lx := x0 + w*2/5 // 左翼の右壁 兼 廊下の左壁
	rx := lx + 3     // 廊下の右壁 兼 右翼の左壁。廊下の内側幅は 2
	// 横の分割線
	leftSplit := y0 + h/3    // 左翼を寝室(上)と居間(下)に割る
	kitchenBot := y0 + h*2/5 // 右翼の台所(上)の底
	midBot := y0 + h*13/20   // 右翼の寝室(中)の底 兼 水回り(下)の上
	genkanTop := y0 + h - 5  // 玄関(下)の上
	waterMid := y0 + h - 4   // 水回りを上下段に割る
	splitX := rx + (right-rx)/2

	rectOf := map[string]Rect{
		"bedroom_a": {X: x0, Y: y0, W: lx - x0 + 1, H: leftSplit - y0 + 1},
		"living":    {X: x0, Y: leftSplit, W: lx - x0 + 1, H: bottom - leftSplit + 1},
		"corridor":  {X: lx, Y: y0, W: rx - lx + 1, H: genkanTop - y0 + 1},
		"genkan":    {X: lx, Y: genkanTop, W: rx - lx + 1, H: bottom - genkanTop + 1},
		"kitchen":   {X: rx, Y: y0, W: right - rx + 1, H: kitchenBot - y0 + 1},
		"bedroom_b": {X: rx, Y: kitchenBot, W: right - rx + 1, H: midBot - kitchenBot + 1},
		"dressing":  {X: rx, Y: midBot, W: splitX - rx + 1, H: waterMid - midBot + 1},
		"bath":      {X: splitX, Y: midBot, W: right - splitX + 1, H: waterMid - midBot + 1},
		"toilet":    {X: rx, Y: waterMid, W: splitX - rx + 1, H: bottom - waterMid + 1},
		"storage":   {X: splitX, Y: waterMid, W: right - splitX + 1, H: bottom - waterMid + 1},
	}

	doors := map[string][]Doorway{}
	connect := func(a, b string, i int) {
		if d, ok := sharedDoorway(rectOf[a], rectOf[b], childSeed(seed, i)); ok {
			doors[a] = append(doors[a], d)
			doors[b] = append(doors[b], d)
		}
	}
	connect("corridor", "genkan", 1)
	connect("corridor", "bedroom_a", 2)
	connect("corridor", "living", 3)
	connect("corridor", "kitchen", 4)
	connect("corridor", "bedroom_b", 5)
	connect("corridor", "dressing", 6)
	connect("dressing", "bath", 7)
	connect("genkan", "toilet", 8)
	connect("toilet", "storage", 9)

	// 建物入口。玄関の下辺中央、footprint の外周上
	gk := rectOf["genkan"]
	doors["genkan"] = append(doors["genkan"], Doorway{X: gk.X + gk.W/2, Y: bottom})

	// 役割ラベルは寝室2室を同じ bedroom に、納戸は storage にまとめる。順序は動線の手前から奥へ
	order := []struct{ key, role string }{
		{"genkan", "genkan"},
		{"corridor", "corridor"},
		{"living", "living"},
		{"bedroom_a", "bedroom"},
		{"bedroom_b", "bedroom"},
		{"kitchen", "kitchen"},
		{"dressing", "dressing"},
		{"bath", "bath"},
		{"toilet", "toilet"},
		{"storage", "storage"},
	}
	rooms := make([]HouseRoom, 0, len(order))
	for _, o := range order {
		rooms = append(rooms, HouseRoom{
			Room: Room{Rect: rectOf[o.key], Doorways: doors[o.key]},
			Role: o.role,
		})
	}
	return rooms
}
