package interior

// 民家は廊下型のテンプレートで間取りを作る。玄関という狭い前室から廊下が背骨として伸び、各室は廊下に
// 面して開く。浴室・脱衣所・トイレは小部屋クラスタに寄せる。純 BSP は均一な部屋しか作れず、廊下という
// 多数の部屋に面する通路も狭い前室も表現できないため、住居の believability には間取りの階層を保証する
// テンプレートを使う。汎用 BSP の SubdivideBuilding はテンプレの無い施設や狭い footprint のフォールバック。
//
// 小部屋は廊下の向きに縛られない。要は各室が主廊下に面する必要はなく、兄弟部屋を介して入れれば
// 小部屋を作れる。脱衣所だけ廊下に面させ、その奥に浴室・トイレを再帰的に分割して nest する。この
// 「入れ子分割＋兄弟経由アクセス」を使えば横廊下でも縦廊下でも同じ道具で小部屋を作れる。PlanHouse は
// 横廊下、PlanHouseVertical は縦廊下でそれを示す。

// PlannedRoom は役割付きの部屋。廊下型の間取りは幾何と一緒に役割まで決める。ゾーン分類のように距離から
// 役割を推すのでなく、テンプレートが玄関や浴室を名指しする。
type PlannedRoom struct {
	Room Room
	Role string
}

// roomRole は返す部屋の順序と役割ラベルの対応。寝室2室を同じ bedroom に、納戸を storage にまとめる。
type roomRole struct{ key, role string }

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

// wireHouse は部屋矩形と接続指定から PlannedRoom 列を組む。conns の各対を戸口で繋ぐだけで、建物入口は
// 開けない。入口は敷地計画 Site が街路向きの辺へ1つ開けるので、テンプレが別に入口を焼き込むと裏壁に
// 穴が二重に開く。横型と縦型の間取りが幾何だけ差し替えて同じ組み立てを共有する。
func wireHouse(rectOf map[string]Rect, seed uint64, conns [][2]string, order []roomRole) []PlannedRoom {
	return assembleRooms(rectOf, wireDoorways(rectOf, seed, conns), order)
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
func PlanHouse(footprint Rect, seed uint64) []PlannedRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	right, bottom := x0+w-1, y0+h-1

	// 上段の底 兼 廊下の上壁。前庭ぶん建物高が縮むと下段の水回りが内側床0に潰れるので、下段が H>=3 を
	// 保つよう topBot に上限 bottom-5 を掛ける。廊下は topBot+3、下段は corrBot..bottom で最低 H=3 になる
	topBot := min(jitterSplit(seed, 0, y0+h*13/20), bottom-5)
	corrBot := topBot + 3 // 廊下の底 兼 下段の上壁。廊下の内側高は 2

	// 上段を縦線で4室に割る。居間を広めに取る。分割線を seed で ±1 揺らし、部屋幅を seed ごとに変える
	tc1 := jitterSplit(seed, 1, x0+w*9/28)
	tc2 := jitterSplit(seed, 2, x0+w*15/28)
	tc3 := jitterSplit(seed, 3, x0+w*21/28)
	// 下段を縦線で5室に割る。玄関を中央に置き、脱衣所と浴室を隣り合わせる
	bc1 := jitterSplit(seed, 4, x0+w*6/28)
	bc2 := jitterSplit(seed, 5, x0+w*11/28)
	bc3 := jitterSplit(seed, 6, x0+w*17/28)
	bc4 := jitterSplit(seed, 7, x0+w*22/28)

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
	return wireHouse(rectOf, seed, conns, houseOrder)
}

// PlanHouseVertical は縦廊下の民家間取りを決定的に生成する。縦廊下だと左右の翼は横長の帯になり、素朴に
// 割ると水回りが広くなる。そこで右下の水回りを入れ子に再帰分割し、脱衣所だけ廊下に面させ、浴室とトイレ
// は脱衣所の奥へ nest する。全室が主廊下に面する必要はなく、兄弟経由で入れれば縦廊下でも小部屋を作れる。
func PlanHouseVertical(footprint Rect, seed uint64) []PlannedRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	right, bottom := x0+w-1, y0+h-1

	cxL := jitterSplit(seed, 10, x0+w*3/7) // 廊下の左壁
	cxR := cxL + 3                         // 廊下の右壁。内側幅 2
	genkanTop := bottom - 4

	// 主要な分割線だけ seed で揺らす。水回りポケットの入れ子分割は小部屋なのでジッタで潰れないよう固定する
	leftMid := jitterSplit(seed, 11, y0+h*2/5) // 左翼を寝室(上)と居間(下)に割る
	kMid := jitterSplit(seed, 12, y0+h*7/20)   // 右翼の台所(上)の底
	bMid := jitterSplit(seed, 13, y0+h*12/20)  // 右翼の寝室(中)の底 兼 水回りポケットの上
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
	return wireHouse(rectOf, seed, conns, houseOrder)
}

// jitterSplit は分割線を seed で ±1 揺らす。固定比率のテンプレでも部屋サイズが seed ごとに変わり、
// 同一スケルトンの見えを崩す。分割線は隣接2室が共有するので、1本を動かしても両室が揃って動き連結は
// 保たれる。index は分割線ごとに変え、ジッタ同士・戸口抽選・型選択の相関を避けるため 5_000_000 番台へ
// 閉じる。戸口抽選は childSeed(seed, 1..9)、型選択は childSeed(seed, 0) を使うのでそれらと衝突しない。
func jitterSplit(seed uint64, index, base int) int {
	return base + int(childSeed(seed, 5_000_000+index)%3) - 1
}

// mirrorHouse は民家プランを footprint 内で左右反転する。玄関や水回り・居室の左右が入れ替わり、同じ
// テンプレから鏡像の間取りが得られる。矩形の左端と戸口の X を折り返し、幅と Y はそのままにするので、
// 連結と部屋の役割は保たれたまま向きだけが変わる。
func mirrorHouse(footprint Rect, plan []PlannedRoom) []PlannedRoom {
	out := make([]PlannedRoom, len(plan))
	for i, hr := range plan {
		r := hr.Room.Rect
		mr := Rect{X: footprint.X + footprint.W - (r.X - footprint.X) - r.W, Y: r.Y, W: r.W, H: r.H}
		doors := make([]Doorway, len(hr.Room.Doorways))
		for j, d := range hr.Room.Doorways {
			doors[j] = Doorway{X: footprint.X + footprint.W - 1 - (d.X - footprint.X), Y: d.Y}
		}
		out[i] = PlannedRoom{Room: Room{Rect: mr, Doorways: doors}, Role: hr.Role}
	}
	return out
}

// houseVariants は民家の間取りプランナの一覧。生成時に seed で1つ選ぶ。横廊下・縦廊下と、その左右反転で
// 4型ある。分割比のジッタと合わさり、同じ型でも seed ごとに部屋サイズが変わる。型を足すとここへ加える。
var houseVariants = []func(Rect, uint64) []PlannedRoom{
	PlanHouse,
	PlanHouseVertical,
	func(f Rect, s uint64) []PlannedRoom { return mirrorHouse(f, PlanHouse(f, s)) },
	func(f Rect, s uint64) []PlannedRoom { return mirrorHouse(f, PlanHouseVertical(f, s)) },
}

// PlanHouseCompact は狭い footprint 向けの小さな民家。廊下型の10室は 24x16 未満に入らないので、居間・
// 寝室・台所・浴室の田の字4室にする。本番の市街地チャンク(20x20)が生む ~14x12 の建物でも、BSP の
// のっぺりした main/back でなく民家らしい部屋の作り分けを保つ。入口は敷地計画 Site が居間側へ開ける。
func PlanHouseCompact(footprint Rect, seed uint64) []PlannedRoom {
	x0, y0, w, h := footprint.X, footprint.Y, footprint.W, footprint.H
	right, bottom := x0+w-1, y0+h-1

	mx := jitterSplit(seed, 40, x0+w/2) // 縦の仕切り。左右を分ける
	my := jitterSplit(seed, 41, y0+h/2) // 横の仕切り。上下を分ける

	rectOf := map[string]Rect{
		"living":  {X: x0, Y: y0, W: mx - x0 + 1, H: my - y0 + 1},
		"bedroom": {X: mx, Y: y0, W: right - mx + 1, H: my - y0 + 1},
		"kitchen": {X: x0, Y: my, W: mx - x0 + 1, H: bottom - my + 1},
		"bath":    {X: mx, Y: my, W: right - mx + 1, H: bottom - my + 1},
	}
	// 田の字の隣接を戸口で繋ぐ。居間から寝室・台所へ、台所から浴室へ。全室が居間から到達できる
	conns := [][2]string{
		{"living", "bedroom"}, {"living", "kitchen"}, {"kitchen", "bath"},
	}
	order := []roomRole{
		{"living", "living"}, {"bedroom", "bedroom"}, {"kitchen", "kitchen"}, {"bath", "bath"},
	}
	return assembleRooms(rectOf, wireDoorways(rectOf, seed, conns), order)
}

// PlanHouseAny は seed から間取りの型を1つ決定的に選び、その民家を生成する。24x16 以上あれば横廊下・縦廊下と
// その鏡像の廊下型4種から seed で選び、間取りに変化を出す。それ未満は廊下型が入らないので田の字のコンパクト
// 民家にする。本番の建物は狭いのでたいていコンパクトになる。型の選択は childSeed(seed, 0) に閉じ、各プランナ
// 内部の戸口抽選や分割比ジッタと相関しないようにする。
func PlanHouseAny(footprint Rect, seed uint64) []PlannedRoom {
	if footprint.W < 24 || footprint.H < 16 {
		return PlanHouseCompact(footprint, seed)
	}
	v := houseVariants[int(childSeed(seed, 0)%uint64(len(houseVariants)))]
	return v(footprint, seed)
}
