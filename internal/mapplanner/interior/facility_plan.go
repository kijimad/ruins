package interior

// 施設固有の間取りテンプレ。汎用 BSP は均一な部屋しか作れず、店も診療所も同じ骨格になってしまう。店なら
// 開けた売場＋奥のバックヤード、診療所なら入口の待合＋廊下＋診察室の列、という施設ごとの構造を保証する
// テンプレを持ち、「言われなくても何の施設か分かる」間取りにする。民家の PlanHouse と同じく HouseRoom を
// 返し、planRooms が役割ごとに内装を敷く。

// PlanStore は店舗の間取りを決定的に生成する。入口側の広い売場を1室で取り、奥の壁沿いにバックヤードの
// 小部屋を並べる。売場は商品棚と冷蔵ケースの開けた空間、奥は樽の物置にして、民家の細かい間仕切りとも
// 診療所の廊下型とも違う「店の平面」にする。バックヤードの個数は seed で 2〜3 に変える。
func PlanStore(footprint Rect, seed uint64) []HouseRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	right, bottom := x0+w-1, y0+h-1

	salesBot := jitterSplit(seed, 20, y0+h*7/10) // 売場の底 兼 バックヤードの上壁

	// バックヤードを奥の壁沿いに縦線で 2〜3 室へ割る。個数を seed で変える
	n := 2 + int(childSeed(seed, 6_000_000)%2)
	rectOf := map[string]Rect{
		"sales": {X: x0, Y: y0, W: w, H: salesBot - y0 + 1},
	}
	order := make([]roomRole, 0, 1+n)
	order = append(order, roomRole{"sales", "main"})
	conns := make([][2]string, 0, n)

	// 各室は売場に面する。分割線は隣室と列を共有させて壁を通す。共有壁の列を一致させるため、前の室の
	// 右端をそのまま次の室の左端にする
	keys := []string{"back0", "back1", "back2"}
	prev := x0
	for i := range n {
		edge := right
		if i < n-1 {
			edge = x0 + w*(i+1)/n
		}
		rectOf[keys[i]] = Rect{X: prev, Y: salesBot, W: edge - prev + 1, H: bottom - salesBot + 1}
		conns = append(conns, [2]string{"sales", keys[i]})
		order = append(order, roomRole{keys[i], "back"})
		prev = edge
	}
	return assembleRooms(rectOf, wireDoorways(rectOf, seed, conns), order)
}

// PlanClinic は診療所の間取りを決定的に生成する。入口側に待合と受付の1室を横いっぱいに取り、その奥へ
// 中央の縦廊下を通し、廊下の左右に診察室を上下2段ずつ並べる。待合が手前・診察室が奥・廊下が背骨という
// 動線で、店の開けた売場とも民家の水回りとも違う「診療所の平面」にする。
func PlanClinic(footprint Rect, seed uint64) []HouseRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	right, bottom := x0+w-1, y0+h-1

	waitBot := jitterSplit(seed, 30, y0+h*3/10) // 待合の底 兼 廊下と診察室の上壁
	cxL := jitterSplit(seed, 31, x0+w*2/5)      // 廊下の左壁
	cxR := cxL + 3                              // 廊下の右壁。内側幅 2
	midY := waitBot + (bottom-waitBot)/2        // 左右の診察室を上下2段に割る

	rectOf := map[string]Rect{
		"waiting":  {X: x0, Y: y0, W: w, H: waitBot - y0 + 1},
		"corridor": {X: cxL, Y: waitBot, W: cxR - cxL + 1, H: bottom - waitBot + 1},
		"exam_a":   {X: x0, Y: waitBot, W: cxL - x0 + 1, H: midY - waitBot + 1},
		"exam_b":   {X: x0, Y: midY, W: cxL - x0 + 1, H: bottom - midY + 1},
		"exam_c":   {X: cxR, Y: waitBot, W: right - cxR + 1, H: midY - waitBot + 1},
		"exam_d":   {X: cxR, Y: midY, W: right - cxR + 1, H: bottom - midY + 1},
	}
	// 廊下が背骨。待合と4つの診察室が廊下に面する
	conns := [][2]string{
		{"waiting", "corridor"},
		{"corridor", "exam_a"}, {"corridor", "exam_b"},
		{"corridor", "exam_c"}, {"corridor", "exam_d"},
	}
	order := []roomRole{
		{"waiting", "main"},
		{"corridor", "corridor"},
		{"exam_a", "back"}, {"exam_b", "back"},
		{"exam_c", "back"}, {"exam_d", "back"},
	}
	return assembleRooms(rectOf, wireDoorways(rectOf, seed, conns), order)
}
