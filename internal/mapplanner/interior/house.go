package interior

import "github.com/kijimaD/ruins/internal/consts"

// 民家の間取りテンプレ。玄関という狭い前室から廊下が背骨として伸び、各室は廊下に面して開く。浴室・トイレは
// 小ブロックに寄せる。純 BSP は均一な部屋しか作れず、廊下という多数の部屋に面する通路も狭い前室も表現できない
// ため、住居の believability には間取りの階層を保証するテンプレを使う。テンプレは split.go の layout builder の
// 短い合成で書き、帯分割・退化防止・連結を機構へ委ねる。

// PlannedRoom は役割付きの部屋。テンプレは幾何と一緒に役割まで決める。ゾーン分類のように距離から役割を
// 推すのでなく、テンプレが玄関や浴室を名指しする。
type PlannedRoom struct {
	Room Room
	Role roleName
}

// roomRole は返す部屋の順序と役割ラベルの対応。
type roomRole struct {
	key  string
	role roleName
}

// wireDoorways は conns の各対を共有壁の戸口で繋ぎ、部屋キー→戸口列の対応を返す。戸口の位置抽選は
// childSeed(seed, i+1) に閉じ、分割比ジッタの 5_000_000 番台や型選択の 0 と相関しない。民家・店・診療所の
// どのテンプレも部屋間の連結にこの1関数を共有する。
func wireDoorways(rectOf map[string]Rect, seed uint64, conns [][2]string) map[string][]Doorway {
	doors := map[string][]Doorway{}
	for i, c := range conns {
		if d, ok := sharedDoorway(rectOf[c[0]], rectOf[c[1]], childSeed(seed, i+1)); ok {
			doors[c[0]] = append(doors[c[0]], d)
			doors[c[1]] = append(doors[c[1]], d)
		}
	}
	return doors
}

// assembleRooms は矩形表と戸口表を order の順に PlannedRoom へ組む。返す順序が動線の手前から奥へ並ぶよう
// order を作る。
func assembleRooms(rectOf map[string]Rect, doors map[string][]Doorway, order []roomRole) []PlannedRoom {
	rooms := make([]PlannedRoom, 0, len(order))
	for _, o := range order {
		rooms = append(rooms, PlannedRoom{
			Room: Room{Rect: rectOf[o.key], Doorways: doors[o.key]},
			Role: o.role,
		})
	}
	return rooms
}

// jitterSplit は分割線を seed で ±1 揺らす。固定比率のテンプレでも部屋サイズが seed ごとに変わり、同一
// スケルトンの見えを崩す。分割線は隣接2室が共有するので、1本を動かしても両室が揃って動き連結は保たれる。
// index は分割線ごとに変え、ジッタ同士・戸口抽選・型選択の相関を避けるため 5_000_000 番台へ閉じる。
func jitterSplit(seed uint64, index int, base consts.Tile) consts.Tile {
	return base + consts.Tile(childSeed(seed, 5_000_000+index)%3) - 1
}

// PlanHouseMid は横廊下の中型民家。玄関を左上角に置き、上段(玄関・居間・台所)を全幅の横廊下で繋ぎ、下段は
// 寝室を広く取り、右下の水回り小ブロックへ浴室とトイレを寄せる。玄関を街路の当たる左上角に固定するので、
// 北入口でも西入口でも敷地計画の frontSlot が入口を玄関へスナップし、入口は必ず玄関へ開く。
func PlanHouseMid(footprint Rect, seed uint64) []PlannedRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	right, bottom := x0+w-1, y0+h-1
	l := newLayout(seed)

	topBot := min(jitterSplit(seed, 20, y0+h*9/20), bottom-6) // 上段の底 兼 廊下の上壁。下段に水回りの高さを残す
	corrBot := topBot + 2                                     // 廊下の底。内側高1の通路

	l.room("corridor", "corridor", Rect{X: x0, Y: topBot, W: w, H: corrBot - topBot + 1})
	// 玄関は左上角に幅4で固定する。strip 任せで幅3になると内側1で、西入口のポーチの凹み1マスで入口が隣室へ
	// ずれる。幅4なら内側2で凹みを吸える。残りの上段を居間・台所へ割り、いずれも下の廊下へ面させる
	gRight := x0 + wetBlockSide - 1
	l.room("genkan", "genkan", Rect{X: x0, Y: y0, W: gRight - x0 + 1, H: topBot - y0 + 1})
	l.connect("corridor", "genkan")
	l.strip(Rect{X: gRight, Y: y0, W: right - gRight + 1, H: topBot - y0 + 1}, splitCols, "corridor",
		[]cell{{"living", "living", 5}, {"kitchen", "kitchen", 4}})
	// 下段は寝室(広)と右下の水回り小ブロック。寝室と浴室は廊下へ面し、トイレは浴室の奥に nest する
	wcX := right - wetBlockSide + 1
	l.room("bedroom", "bedroom", Rect{X: x0, Y: corrBot, W: wcX - x0 + 1, H: bottom - corrBot + 1})
	l.connect("corridor", "bedroom")
	l.wetBlock(Rect{X: wcX, Y: corrBot, W: right - wcX + 1, H: bottom - corrBot + 1}, "corridor")
	return l.build()
}

// PlanHouseMidV は縦廊下の中型民家。玄関を左上角に置き、その下へ縦廊下を伸ばす。右エリアに居間・台所・寝室を
// 縦に積み、寝室の右端の水回り小ブロックへ浴室とトイレを寄せる。横型と骨格が違い、入口は必ず玄関へ開く。
// 右エリアの縦積みは建物高が要るので、PlanHouseAny は高い建物でのみ縦型を選ぶ。
func PlanHouseMidV(footprint Rect, seed uint64) []PlannedRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	right := x0 + w - 1
	l := newLayout(seed)

	cxR := x0 + wetBlockSide - 1 // 縦廊下の右壁。幅4、内側2。西入口のポーチの凹み1マスを吸える
	gBot := y0 + 3               // 玄関の底。左上角の玄関 4x4

	l.room("genkan", "genkan", Rect{X: x0, Y: y0, W: cxR - x0 + 1, H: gBot - y0 + 1})
	l.room("corridor", "corridor", Rect{X: x0, Y: gBot, W: cxR - x0 + 1, H: y0 + h - 1 - gBot + 1})
	l.connect("corridor", "genkan")
	// 右エリアを縦に居間・台所・寝室へ割り、各室を廊下へ面させる
	l.strip(Rect{X: cxR, Y: y0, W: right - cxR + 1, H: h}, splitRows, "corridor",
		[]cell{{"living", "living", 5}, {"kitchen", "kitchen", 4}, {"bedroom", "bedroom", 5}})
	// 居間の上部は廊下の上の玄関に面する。廊下との縦の重なりが浅い seed でも連結を保証する保険の戸口
	l.connect("genkan", "living")
	// 寝室の右端を水回り小ブロックへ割る。寝室を縮め、共有壁で浴室・トイレを隣接させる
	bed := l.rect["bedroom"]
	l.rect["bedroom"] = Rect{X: bed.X, Y: bed.Y, W: bed.W - wetBlockSide + 1, H: bed.H}
	l.wetBlock(Rect{X: bed.X + bed.W - wetBlockSide, Y: bed.Y, W: wetBlockSide, H: bed.H}, "bedroom")
	return l.build()
}

// PlanHouseCompact は狭い footprint 向けの小さな民家。中型が入らない ~13 未満の建物で、居間・寝室・台所・
// 浴室の田の字4室にする。本番のチャンク24では通常この経路に落ちないが、より小さな建物のフォールバックとして
// 民家らしい部屋の作り分けを保つ。入口は敷地計画 Site が居間側へ開ける。
func PlanHouseCompact(footprint Rect, seed uint64) []PlannedRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	bottom := y0 + h - 1
	l := newLayout(seed)

	my := jitterSplit(seed, 41, y0+h/2) // 上下の仕切り
	// 上段は居間・寝室、下段は台所・浴室。各段を左右に割る
	l.strip(Rect{X: x0, Y: y0, W: w, H: my - y0 + 1}, splitCols, "",
		[]cell{{"living", "living", 1}, {"bedroom", "bedroom", 1}})
	l.strip(Rect{X: x0, Y: my, W: w, H: bottom - my + 1}, splitCols, "",
		[]cell{{"kitchen", "kitchen", 1}, {"bath", "bath", 1}})
	// 田の字の隣接を戸口で繋ぐ。居間から寝室・台所へ、台所から浴室へ。全室が居間から到達できる
	l.connect("living", "bedroom")
	l.connect("living", "kitchen")
	l.connect("kitchen", "bath")
	return l.build()
}

// PlanHouseAny は建物サイズから中型かコンパクトを決定的に選ぶ。高さ16以上は縦積みの入る横型・縦型を seed で
// 選び骨格の変化を出す。それ未満(高さ13以上)は縦積みが入らないので横型に限る。さらに狭ければ田の字。型の選択は
// childSeed(seed, 0) に閉じ、各テンプレ内部の戸口抽選や分割比ジッタと相関しないようにする。
func PlanHouseAny(footprint Rect, seed uint64) []PlannedRoom {
	switch {
	case footprint.W >= 14 && footprint.H >= 16:
		if childSeed(seed, 0)%2 == 0 {
			return PlanHouseMid(footprint, seed)
		}
		return PlanHouseMidV(footprint, seed)
	case footprint.W >= 14 && footprint.H >= 13:
		return PlanHouseMid(footprint, seed)
	default:
		return PlanHouseCompact(footprint, seed)
	}
}
