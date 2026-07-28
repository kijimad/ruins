package interior

import "github.com/kijimaD/ruins/internal/consts"

// 施設固有の間取りテンプレ。汎用 BSP は均一な部屋しか作れず、店も診療所も同じ骨格になってしまう。店なら
// 開けた売場＋奥のバックヤード、診療所なら入口の待合＋廊下＋診察室の列、という施設ごとの構造を保証する
// テンプレを持ち、「言われなくても何の施設か分かる」間取りにする。民家の PlanHouseAny と同じく PlannedRoom を
// 返し、planRooms が役割ごとに内装を敷く。帯分割と連結は split.go の layout builder へ委ね、テンプレは
// 前室＋背骨＋帯の合成として短く書く。奥室は倉庫・診察室の1種に潰さず、事務所・トイレ・薬局などへ役割を
// 振り分けて非民家も民家並みの room 多様性にする。

// storeBackRole は店の奥室 i 番目の役割を返す。0番は必ず倉庫にして店に物置が1つは在る不変条件を守り、
// 以降は事務所・従業員トイレ・冷蔵庫室・倉庫から seed で選んで奥室に多様な役割を出す。
func storeBackRole(seed uint64, i int) roleName {
	if i == 0 {
		return "storeroom"
	}
	pool := []roleName{"office", "restroom", "coldroom", "storeroom"}
	return pool[childSeed(seed, 6_100_000+i)%uint64(len(pool))]
}

// clinicBackRole は診療所の奥室 i 番目(0-indexed, total は奥室総数)の役割を返す。0番は必ず薬局、
// 3室以上なら末尾をトイレ、4室なら医師室も足し、残りを診察室で埋める。薬局と水回りと医師室を保証しつつ
// 診察室を主にする。
func clinicBackRole(i, total int) roleName {
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

// storeBackKeys は店のバックヤードの矩形表の鍵。個数ぶんの先頭を使う。
var storeBackKeys = []string{"back0", "back1", "back2"}

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

// storeBackCells は個数ぶんのバックヤードの cell を役割付きで返す。
func storeBackCells(seed uint64, n int) []cell {
	cells := make([]cell, n)
	for i := range cells {
		cells[i] = cell{key: storeBackKeys[i], role: storeBackRole(seed, i), weight: 1}
	}
	return cells
}

// storeBackBottom は売場を上いっぱいに取り、バックヤードを下の壁沿いに縦線で 2〜3 室へ並べる型。
func storeBackBottom(footprint Rect, seed uint64) []PlannedRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	bottom := y0 + h - 1
	l := newLayout(seed)

	salesBot := jitterSplit(seed, 20, y0+h*7/10) // 売場の底 兼 バックヤードの上壁
	n := 2 + int(childSeed(seed, 6_000_000)%2)   // バックヤードの個数
	l.room("sales", roleMain, Rect{X: x0, Y: y0, W: w, H: salesBot - y0 + 1})
	l.strip(Rect{X: x0, Y: salesBot, W: w, H: bottom - salesBot + 1}, splitCols, "sales", storeBackCells(seed, n))
	return l.build()
}

// storeBackSide は売場を縦いっぱいの高さで取り、バックヤードを横の壁沿いの柱に横線で 2〜3 室へ積む型。
// left なら柱が左、そうでなければ右。奥行きでなく横幅で店と物置を分け、下ストリップ型と別の平面にする。
func storeBackSide(footprint Rect, seed uint64, left bool) []PlannedRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	right := x0 + w - 1
	l := newLayout(seed)

	backW := clamp(jitterSplit(seed, 22, w*3/10), wetBlockSide, w/2) // バックヤードの柱の幅
	n := 2 + int(childSeed(seed, 6_000_000)%2)

	var backCol, sales Rect
	if left {
		backCol = Rect{X: x0, Y: y0, W: backW, H: h}
		sales = Rect{X: x0 + backW - 1, Y: y0, W: w - backW + 1, H: h} // 柱と列を共有する
	} else {
		backCol = Rect{X: right - backW + 1, Y: y0, W: backW, H: h}
		sales = Rect{X: x0, Y: y0, W: w - backW + 1, H: h}
	}
	l.room("sales", roleMain, sales)
	l.strip(backCol, splitRows, "sales", storeBackCells(seed, n))
	return l.build()
}

// PlanClinic は診療所の間取りを決定的に生成する。入口側に待合と受付の1室を横いっぱいに取り、その奥へ
// 中央の縦廊下を通し、廊下の左右に診察室を並べる。待合が手前・診察室が奥・廊下が背骨という動線で、店の
// 開けた売場とも民家の水回りとも違う「診療所の平面」にする。各翼を seed で上下2室へ割るか1室のまま
// にするかを変え、診察室が2室の広い診療所と4室の細かい診療所を出す。
func PlanClinic(footprint Rect, seed uint64) []PlannedRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	right, bottom := x0+w-1, y0+h-1
	l := newLayout(seed)

	waitBot := jitterSplit(seed, 30, y0+h*3/10) // 待合の底 兼 廊下と診察室の上壁
	cxL := jitterSplit(seed, 31, x0+w*2/5)      // 廊下の左壁
	cxR := cxL + 3                              // 廊下の右壁。内側幅 2
	split := childSeed(seed, 6_000_001)%2 == 0  // 各翼を上下2室へ割るか、1室のままにするか

	l.room("waiting", "waiting", Rect{X: x0, Y: y0, W: w, H: waitBot - y0 + 1})
	l.room("corridor", "corridor", Rect{X: cxL, Y: waitBot, W: cxR - cxL + 1, H: bottom - waitBot + 1})
	l.connect("waiting", "corridor")

	total := 2
	if split {
		total = 4
	}
	bi := 0
	for _, wing := range []struct {
		key      string
		xlo, xhi consts.Tile
	}{
		{"exam_l", x0, cxL},
		{"exam_r", cxR, right},
	} {
		base := Rect{X: wing.xlo, Y: waitBot, W: wing.xhi - wing.xlo + 1, H: bottom - waitBot + 1}
		if split {
			l.strip(base, splitRows, "corridor", []cell{
				{key: wing.key + "_a", role: clinicBackRole(bi, total), weight: 1},
				{key: wing.key + "_b", role: clinicBackRole(bi+1, total), weight: 1},
			})
			bi += 2
		} else {
			l.room(wing.key, clinicBackRole(bi, total), base)
			l.connect("corridor", wing.key)
			bi++
		}
	}
	return l.build()
}
