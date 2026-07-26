package interior

// 民家は廊下型のテンプレートで間取りを作る。玄関という狭い前室から廊下が背骨として伸び、各室は廊下に
// 面して開く。浴室・脱衣所・トイレは小部屋クラスタに寄せる。純 BSP は均一な部屋しか作れず、廊下という
// 多数の部屋に面する通路も狭い前室も表現できないため、住居の believability には間取りの階層を保証する
// テンプレートを使う。汎用 BSP の SplitBuilding は診療所など他用途に残す。
//
// 小部屋は廊下の向きに縛られない。要は各室が主廊下に面する必要はなく、兄弟部屋を介して入れれば
// 小部屋を作れる。脱衣所だけ廊下に面させ、その奥に浴室・トイレを再帰的に分割して nest する。この
// 「入れ子分割＋兄弟経由アクセス」を使えば横廊下でも縦廊下でも同じ道具で小部屋を作れる。PlanHouse は
// 横廊下、PlanHouseVertical は縦廊下でそれを示す。

// HouseRoom は役割付きの部屋。廊下型の間取りは幾何と一緒に役割まで決める。ゾーン分類のように距離から
// 役割を推すのでなく、テンプレートが玄関や浴室を名指しする。
type HouseRoom struct {
	Room Room
	Role string
}

// roomRole は返す部屋の順序と役割ラベルの対応。寝室2室を同じ bedroom に、納戸を storage にまとめる。
type roomRole struct{ key, role string }

// wireHouse は部屋矩形と接続指定から HouseRoom 列を組む。conns の各対を戸口で繋ぎ、entrance の部屋の
// 下辺中央に建物入口を開ける。横型と縦型の間取りが幾何だけ差し替えて同じ組み立てを共有する。
func wireHouse(rectOf map[string]Rect, seed uint64, conns [][2]string, entrance string, bottom int, order []roomRole) []HouseRoom {
	doors := map[string][]Doorway{}
	for i, c := range conns {
		if d, ok := sharedDoorway(rectOf[c[0]], rectOf[c[1]], childSeed(seed, i+1)); ok {
			doors[c[0]] = append(doors[c[0]], d)
			doors[c[1]] = append(doors[c[1]], d)
		}
	}
	gk := rectOf[entrance]
	doors[entrance] = append(doors[entrance], Doorway{X: gk.X + gk.W/2, Y: bottom})

	rooms := make([]HouseRoom, 0, len(order))
	for _, o := range order {
		rooms = append(rooms, HouseRoom{
			Room: Room{Rect: rectOf[o.key], Doorways: doors[o.key]},
			Role: o.role,
		})
	}
	return rooms
}

// houseOrder は横型と縦型で共通の返却順と役割ラベル。動線の手前から奥へ並べる。
var houseOrder = []roomRole{
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

// PlanHouse は横廊下の民家間取りを決定的に生成する。上段を居室、下段を玄関と水回りにし、いずれも縦線で
// 分割する。縦線で割ると各室は廊下に面したまま幅を狭められるので、トイレや浴室を小部屋にできる。返す
// 部屋は戸口で連結され、どの部屋にも入口から到達できる。footprint は概ね 24x16 以上を前提にする。
func PlanHouse(footprint Rect, seed uint64) []HouseRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	right, bottom := x0+w-1, y0+h-1

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
	// 廊下が背骨。上段の居室と下段の玄関・トイレ・脱衣所・納戸が面する。浴室は脱衣所の奥
	conns := [][2]string{
		{"corridor", "living"}, {"corridor", "kitchen"}, {"corridor", "bedroom_a"},
		{"corridor", "bedroom_b"}, {"corridor", "genkan"}, {"corridor", "dressing"},
		{"corridor", "toilet"}, {"corridor", "storage"}, {"dressing", "bath"},
	}
	return wireHouse(rectOf, seed, conns, "genkan", bottom, houseOrder)
}

// PlanHouseVertical は縦廊下の民家間取りを決定的に生成する。縦廊下だと左右の翼は横長の帯になり、素朴に
// 割ると水回りが広くなる。そこで右下の水回りを入れ子に再帰分割し、脱衣所だけ廊下に面させ、浴室とトイレ
// は脱衣所の奥へ nest する。全室が主廊下に面する必要はなく、兄弟経由で入れれば縦廊下でも小部屋を作れる。
func PlanHouseVertical(footprint Rect, seed uint64) []HouseRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	right, bottom := x0+w-1, y0+h-1

	cxL := x0 + w*3/7 // 廊下の左壁
	cxR := cxL + 3    // 廊下の右壁。内側幅 2
	genkanTop := bottom - 4

	leftMid := y0 + h*2/5 // 左翼を寝室(上)と居間(下)に割る
	kMid := y0 + h*7/20   // 右翼の台所(上)の底
	bMid := y0 + h*12/20  // 右翼の寝室(中)の底 兼 水回りポケットの上
	// 水回りポケットを入れ子分割する。脱衣所を廊下沿いの縦長に、浴室とトイレをその奥の小部屋に
	dwX := cxR + (right-cxR)*2/5 // 脱衣所の右壁 兼 浴室トイレの左壁
	stX := cxR + (right-cxR)*3/4 // 浴室トイレの右壁 兼 納戸の左壁
	wMid := y0 + h*16/20         // 浴室(上)とトイレ(下)を割る

	rectOf := map[string]Rect{
		"bedroom_a": {X: x0, Y: y0, W: cxL - x0 + 1, H: leftMid - y0 + 1},
		"living":    {X: x0, Y: leftMid, W: cxL - x0 + 1, H: bottom - leftMid + 1},
		"corridor":  {X: cxL, Y: y0, W: cxR - cxL + 1, H: genkanTop - y0 + 1},
		"genkan":    {X: cxL, Y: genkanTop, W: cxR - cxL + 1, H: bottom - genkanTop + 1},
		"kitchen":   {X: cxR, Y: y0, W: right - cxR + 1, H: kMid - y0 + 1},
		"bedroom_b": {X: cxR, Y: kMid, W: right - cxR + 1, H: bMid - kMid + 1},
		"dressing":  {X: cxR, Y: bMid, W: dwX - cxR + 1, H: bottom - bMid + 1},
		"bath":      {X: dwX, Y: bMid, W: stX - dwX + 1, H: wMid - bMid + 1},
		"toilet":    {X: dwX, Y: wMid, W: stX - dwX + 1, H: bottom - wMid + 1},
		"storage":   {X: stX, Y: bMid, W: right - stX + 1, H: bottom - bMid + 1},
	}
	// 廊下が背骨。左右の居室と玄関・脱衣所が面する。浴室とトイレは脱衣所の奥、納戸は浴室の隣に nest
	conns := [][2]string{
		{"corridor", "genkan"}, {"corridor", "living"}, {"corridor", "bedroom_a"},
		{"corridor", "kitchen"}, {"corridor", "bedroom_b"}, {"corridor", "dressing"},
		{"dressing", "bath"}, {"dressing", "toilet"}, {"bath", "storage"},
	}
	return wireHouse(rectOf, seed, conns, "genkan", bottom, houseOrder)
}
