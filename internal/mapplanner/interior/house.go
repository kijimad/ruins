package interior

// 民家は廊下型のテンプレートで間取りを作る。玄関という狭い前室から廊下が背骨として伸び、各室は廊下に
// 面して開く。浴室・脱衣所・トイレは奥の小部屋クラスタに寄せる。純 BSP は均一な部屋しか作れず、廊下と
// いう多数の部屋に面する通路も狭い前室も表現できないため、住居の believability には間取りの階層を保証
// するテンプレートを使う。汎用 BSP の SplitBuilding は診療所など他用途に残す。
//
// 動線: 入口 → 玄関 → 横廊下 → 居間・台所・寝室・トイレ・脱衣所・納戸。脱衣所 → 浴室。廊下は建物を
// 横断する背骨で、上段の居室と下段の玄関・水回りがともに面する。どの部屋も玄関から辿れる連結木になる。

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

	// 横の廊下を中央に通す。上段を居室、下段を玄関と水回りにし、いずれも縦線で分割する。縦線で割ると
	// 各室は廊下に面したまま幅を狭められるので、トイレや浴室を小部屋にできる。
	topBot := y0 + h*13/20 // 上段の底 兼 廊下の上壁
	corrBot := topBot + 3  // 廊下の底 兼 下段の上壁。廊下の内側高は 2

	// 上段を縦線で4室に割る。居間を広めに取る
	tc1 := x0 + w*9/28
	tc2 := x0 + w*15/28
	tc3 := x0 + w*21/28
	// 下段を縦線で5室に割る。玄関を中央に置き、脱衣所と浴室を隣り合わせる
	bc1 := x0 + w*6/28
	bc2 := x0 + w*11/28
	bc3 := x0 + w*17/28
	bc4 := x0 + w*22/28

	rectOf := map[string]Rect{
		"living":    {X: x0, Y: y0, W: tc1 - x0 + 1, H: topBot - y0 + 1},
		"kitchen":   {X: tc1, Y: y0, W: tc2 - tc1 + 1, H: topBot - y0 + 1},
		"bedroom_a": {X: tc2, Y: y0, W: tc3 - tc2 + 1, H: topBot - y0 + 1},
		"bedroom_b": {X: tc3, Y: y0, W: right - tc3 + 1, H: topBot - y0 + 1},
		"corridor":  {X: x0, Y: topBot, W: w, H: corrBot - topBot + 1},
		"dressing":  {X: x0, Y: corrBot, W: bc1 - x0 + 1, H: bottom - corrBot + 1},
		"bath":      {X: bc1, Y: corrBot, W: bc2 - bc1 + 1, H: bottom - corrBot + 1},
		"genkan":    {X: bc2, Y: corrBot, W: bc3 - bc2 + 1, H: bottom - corrBot + 1},
		"toilet":    {X: bc3, Y: corrBot, W: bc4 - bc3 + 1, H: bottom - corrBot + 1},
		"storage":   {X: bc4, Y: corrBot, W: right - bc4 + 1, H: bottom - corrBot + 1},
	}

	doors := map[string][]Doorway{}
	connect := func(a, b string, i int) {
		if d, ok := sharedDoorway(rectOf[a], rectOf[b], childSeed(seed, i)); ok {
			doors[a] = append(doors[a], d)
			doors[b] = append(doors[b], d)
		}
	}
	// 廊下が背骨。上段の居室と下段の玄関・トイレ・脱衣所・納戸が廊下に面する
	connect("corridor", "living", 1)
	connect("corridor", "kitchen", 2)
	connect("corridor", "bedroom_a", 3)
	connect("corridor", "bedroom_b", 4)
	connect("corridor", "genkan", 5)
	connect("corridor", "dressing", 6)
	connect("corridor", "toilet", 7)
	connect("corridor", "storage", 8)
	// 浴室は廊下に面さず脱衣所の奥。廊下→脱衣所→浴室の順に通る
	connect("dressing", "bath", 9)

	// 建物入口。玄関の下辺中央、footprint の外周上
	gk := rectOf["genkan"]
	doors["genkan"] = append(doors["genkan"], Doorway{X: gk.X + gk.W/2, Y: bottom})

	// 役割ラベルは寝室2室を同じ bedroom に、納戸は storage にまとめる。順序は動線の手前から奥へ
	order := []struct{ key, role string }{
		{"genkan", "genkan"},
		{"corridor", "corridor"},
		{"living", "living"},
		{"kitchen", "kitchen"},
		{"bedroom_a", "bedroom"},
		{"bedroom_b", "bedroom"},
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
