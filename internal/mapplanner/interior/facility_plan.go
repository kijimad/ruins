package interior

// 施設固有の間取りテンプレ。汎用 BSP は均一な部屋しか作れず、店も診療所も同じ骨格になってしまう。店なら
// 開けた売場＋奥のバックヤード、診療所なら入口の待合＋廊下＋診察室の列、という施設ごとの構造を保証する
// テンプレを持ち、「言われなくても何の施設か分かる」間取りにする。民家の PlanHouse と同じく PlannedRoom を
// 返し、planRooms が役割ごとに内装を敷く。奥室も倉庫・診察室の1種に潰さず、事務所・トイレ・薬局などへ
// 役割を振り分けて、非民家も民家並みの room 多様性にする。

// storeBackRole は店の奥室 i 番目の役割を返す。0番は必ず倉庫にして店に物置が1つは在る不変条件を守り、
// 以降は事務所・従業員トイレ・冷蔵庫室・倉庫から seed で選んで奥室に多様な役割を出す。
func storeBackRole(seed uint64, i int) string {
	if i == 0 {
		return "storeroom"
	}
	pool := []string{"office", "restroom", "coldroom", "storeroom"}
	return pool[childSeed(seed, 6_100_000+i)%uint64(len(pool))]
}

// clinicBackRole は診療所の奥室 i 番目(0-indexed, total は奥室総数)の役割を返す。0番は必ず施錠薬局、
// 3室以上なら末尾をトイレ、4室なら医師室も足し、残りを診察室で埋める。薬局と水回りと医師室を保証しつつ
// 診察室を主にする。
func clinicBackRole(i, total int) string {
	switch {
	case i == 0:
		return "pharmacy"
	case i == total-1 && total >= 3:
		return "restroom"
	case i == total-2 && total >= 4:
		return "office"
	default:
		return "exam"
	}
}

// PlanStore は店舗の間取りを決定的に生成する。入口側の広い売場を1室で取り、その奥にバックヤードの小部屋を
// 並べる。売場は商品棚と冷蔵ケースの開けた空間、奥は樽の物置にして、民家の細かい間仕切りとも診療所の廊下型
// とも違う「店の平面」にする。バックヤードを奥の壁沿いに置くか横の壁沿いに置くかを seed で選び、下ストリップ・
// 右柱・左柱の3型を出す。どの型でも売場は北の入口に面する。バックヤードの個数も seed で 2〜3 に変える。
func PlanStore(footprint Rect, seed uint64) []PlannedRoom {
	switch childSeed(seed, 6_000_002) % 3 {
	case 0:
		return storeBackBottom(footprint, seed)
	case 1:
		return storeBackSide(footprint, seed, false)
	default:
		return storeBackSide(footprint, seed, true)
	}
}

// storeBackBottom は売場を上いっぱいに取り、バックヤードを下の壁沿いに縦線で 2〜3 室へ並べる型。
func storeBackBottom(footprint Rect, seed uint64) []PlannedRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	right, bottom := x0+w-1, y0+h-1

	salesBot := jitterSplit(seed, 20, y0+h*7/10) // 売場の底 兼 バックヤードの上壁
	n := 2 + int(childSeed(seed, 6_000_000)%2)   // バックヤードの個数
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
		order = append(order, roomRole{keys[i], storeBackRole(seed, i)})
		prev = edge
	}
	return assembleRooms(rectOf, wireDoorways(rectOf, seed, conns), order)
}

// storeBackSide は売場を横いっぱいの高さで取り、バックヤードを横の壁沿いの柱に縦積みで 2〜3 室へ並べる型。
// left なら左の壁沿い、そうでなければ右の壁沿いに柱を置く。売場は入口のある上辺に面したまま、奥行きでなく
// 横幅で店と物置を分ける。下ストリップ型と別の平面になり、店の骨格に変種が出る。
func storeBackSide(footprint Rect, seed uint64, left bool) []PlannedRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	right, bottom := x0+w-1, y0+h-1

	backW := jitterSplit(seed, 22, w*3/10) // バックヤードの柱の幅
	n := 2 + int(childSeed(seed, 6_000_000)%2)

	// 売場とバックヤードの共有列 cx を決める。left なら柱が左、そうでなければ柱が右
	var cx, backX0 int
	if left {
		cx = x0 + backW
		backX0 = x0
	} else {
		cx = right - backW
		backX0 = cx
	}
	salesX0, salesW := x0, cx-x0+1
	if left {
		salesX0, salesW = cx, right-cx+1
	}

	rectOf := map[string]Rect{
		"sales": {X: salesX0, Y: y0, W: salesW, H: h},
	}
	order := make([]roomRole, 0, 1+n)
	order = append(order, roomRole{"sales", "main"})
	conns := make([][2]string, 0, n)

	// 柱を縦線でなく横線で 2〜3 室へ積む。各室は売場と列 cx を共有して面する。共有壁の行を一致させるため、
	// 前の室の下端をそのまま次の室の上端にする
	keys := []string{"back0", "back1", "back2"}
	prev := y0
	for i := range n {
		edge := bottom
		if i < n-1 {
			edge = y0 + h*(i+1)/n
		}
		rectOf[keys[i]] = Rect{X: backX0, Y: prev, W: backW + 1, H: edge - prev + 1}
		conns = append(conns, [2]string{"sales", keys[i]})
		order = append(order, roomRole{keys[i], storeBackRole(seed, i)})
		prev = edge
	}
	return assembleRooms(rectOf, wireDoorways(rectOf, seed, conns), order)
}

// PlanClinic は診療所の間取りを決定的に生成する。入口側に待合と受付の1室を横いっぱいに取り、その奥へ
// 中央の縦廊下を通し、廊下の左右に診察室を並べる。待合が手前・診察室が奥・廊下が背骨という動線で、店の
// 開けた売場とも民家の水回りとも違う「診療所の平面」にする。各翼を seed で上下2室へ割るか1室のまま
// にするかを変え、診察室が2室の広い診療所と4室の細かい診療所を出す。
func PlanClinic(footprint Rect, seed uint64) []PlannedRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	right, bottom := x0+w-1, y0+h-1

	waitBot := jitterSplit(seed, 30, y0+h*3/10) // 待合の底 兼 廊下と診察室の上壁
	cxL := jitterSplit(seed, 31, x0+w*2/5)      // 廊下の左壁
	cxR := cxL + 3                              // 廊下の右壁。内側幅 2
	split := childSeed(seed, 6_000_001)%2 == 0  // 各翼を上下2室へ割るか、1室のままにするか

	rectOf := map[string]Rect{
		"waiting":  {X: x0, Y: y0, W: w, H: waitBot - y0 + 1},
		"corridor": {X: cxL, Y: waitBot, W: cxR - cxL + 1, H: bottom - waitBot + 1},
	}
	conns := [][2]string{{"waiting", "corridor"}}
	order := []roomRole{{"waiting", "waiting"}, {"corridor", "corridor"}}

	// 廊下の左右の翼へ診察室を割り付ける。翼を上下2室に割ると4診察室、割らないと2診察室になる。奥室は
	// clinicBackRole で薬局・トイレ・診察室へ振り分ける
	total := 2
	if split {
		total = 4
	}
	midY := waitBot + (bottom-waitBot)/2
	bi := 0
	for _, wing := range []struct {
		key      string
		xlo, xhi int
	}{
		{"exam_l", x0, cxL},
		{"exam_r", cxR, right},
	} {
		if split {
			ka, kb := wing.key+"_a", wing.key+"_b"
			rectOf[ka] = Rect{X: wing.xlo, Y: waitBot, W: wing.xhi - wing.xlo + 1, H: midY - waitBot + 1}
			rectOf[kb] = Rect{X: wing.xlo, Y: midY, W: wing.xhi - wing.xlo + 1, H: bottom - midY + 1}
			conns = append(conns, [2]string{"corridor", ka}, [2]string{"corridor", kb})
			order = append(order, roomRole{ka, clinicBackRole(bi, total)}, roomRole{kb, clinicBackRole(bi+1, total)})
			bi += 2
		} else {
			rectOf[wing.key] = Rect{X: wing.xlo, Y: waitBot, W: wing.xhi - wing.xlo + 1, H: bottom - waitBot + 1}
			conns = append(conns, [2]string{"corridor", wing.key})
			order = append(order, roomRole{wing.key, clinicBackRole(bi, total)})
			bi++
		}
	}
	return assembleRooms(rectOf, wireDoorways(rectOf, seed, conns), order)
}
