package interior

import (
	"sort"

	"github.com/kijimaD/ruins/internal/consts"
)

// 建物内装のパイプラインと役割ルーティング。footprint を敷地計画して部屋へ割り、各部屋の役割から content を
// 引いて加工する。overworld の大きな建物外殻を、間仕切りと役割で構造の変化のある内装に変える。
// 加工の各段は FurnishStage で残し、段別 VRT でどの加工が犯人かを切り分けられるようにする。

// FurnishStage は建物内装の加工1段の累積配置。Label は段名、Placed はその段まで適用した全室の配置。
// plan は空の間取り、fill は什器、age は経年、flavor は装飾を足した段を表す。
type FurnishStage struct {
	Label  string
	Placed []Placed
}

// FurnishStages は敷地計画と内装を加工ステップごとに分けて返す。footprint を建物と庭に分ける planSite を
// 前段に置き、建物内の各室へ plan→fill→age→flavor を掛けた累積配置を返す。どの段で見た目が壊れたかを段別
// VRT で切り分けられるようにする。坪庭の室は家具を置かず観葉だけ置き、経年・flavor を掛けない。FurnishBuilding
// はこの最終段を返す薄い包みで、両者は同じパイプラインを共有するので段別 VRT と実生成は乖離しない。
func FurnishStages(seed uint64, footprint Rect, door Vec, facility FacilityKind) (Site, []FurnishStage) {
	site := planSite(footprint, seed, door, facility)

	prof := rollProfile(seed) // 生活感の直交軸は建物ごとに1つ。全室へ一様に効かせる
	var fill, decayed, flavored []Placed
	for i := range site.Rooms {
		hr := site.Rooms[i]
		roomSeed := childSeed(seed, 300+i)
		// 密度は全室へ一様に効かせる。同じ内装でもがらんとした家と物で埋まった家を出し分ける
		f := FillRoom(roomSeed, hr.Room, applyDensity(roleContent(facility, hr.Role, seed), prof.density))
		// 損傷レベルで略奪・生活痕・廃墟化の強度を変える。無傷なら素通し
		a := Age(roomSeed, hr.Room, f, prof.damage)
		fl := a
		// flavor と散らかりは到達性修復を通らないので、幅1の通路や狭室に置くと歩行を塞ぐ。廊下と、内側が
		// 1マス幅しかない狭室には足さない。通路を蝋燭や絨毯や小物で埋めない
		if hr.Role != roleCorridor && !isNarrowRoom(hr.Room.Rect) {
			fl = Flavor(roomSeed, hr.Room, a, facilityFlavor(facility))
			// 散らかりの小物を家具の隣へ落とし、生活感を足す。整頓の建物では何も足さない
			fl = applyClutter(childSeed(roomSeed, 11_300_000), hr.Room, fl, prof.clutter, hr.Role)
		}
		fill = append(fill, f...)
		decayed = append(decayed, a...)
		flavored = append(flavored, fl...)
	}
	// 外皮 FacadePass。街路側の前壁へ窓・シャッター・看板を付け、閉じた箱を正面のある建物にする。損傷レベルで
	// 廃業した店のシャッターを決める。壁の上に載る prop なので室内の充填とは別に足す
	facade := facadeElements(site, facility, prof.damage)
	flavored = append(flavored, facade...)
	// lot pass。敷地を塀で囲い門で開け、前庭に外構を置く。建物を裸で地面に置かない
	flavored = append(flavored, lotElements(site, facility)...)
	// hero 部屋。稀な1棟の主室中央へ landmark を1つ据え、記憶に残る見せ場にする。主室中央には食卓など
	// 中央配置の什器が既にあることが多いので、目玉は showpiece としてそのタイルの既存 prop を退けて1つだけ
	// 置き、スプライトの重なりを防ぐ。占有は1タイルのままなので歩行性は変わらない
	if ref, ok := heroCenterpiece(seed); ok {
		if pos, ok := heroSpot(site); ok {
			kept := flavored[:0]
			for _, p := range flavored {
				if p.Pos == pos {
					continue
				}
				kept = append(kept, p)
			}
			kept = append(kept, Placed{Kind: KindDecor, Ref: ref, Pos: pos})
			flavored = kept
		}
	}
	return site, []FurnishStage{
		{Label: "1 plan", Placed: nil},
		{Label: "2 fill", Placed: fill},
		{Label: "3 age", Placed: decayed},
		{Label: "4 flavor", Placed: flavored},
	}
}

// roleName は部屋の役割ラベル。主室・廊下・寝室・薬局といった部屋の意味を表す。facility と隣り合って
// roleContent に渡るので、型で分けて引数の取り違えを防ぐ。
type roleName string

// roleMain は主室の役割名。売場・待合など施設の顔の部屋。BSP フォールバックは面積最大をこれにする。
const roleMain roleName = "main"

// roleCorridor は廊下の役割名。通路として空け、フレーバーや hero の目玉を置かない。
const roleCorridor roleName = "corridor"

// isNarrowRoom は部屋の内側が幅1以下の通路状かを返す。1マス幅の廊下や薄い水回りにフレーバーを置くと
// 唯一の歩行帯を塞ぐので、その判定に使う。
func isNarrowRoom(r Rect) bool {
	return r.W-2 <= 1 || r.H-2 <= 1
}

// FurnishBuilding は footprint を敷地計画し、建物内の各室へ内装を敷いて、敷地と最終配置を返す。footprint を
// そのまま埋めず、入口側に前庭を空け、1室を坪庭にし、玄関を凹ませる。加工は FurnishStages が持ち、ここは
// その最終段 flavor を返す。呼び出し側は Site から Walls で壁タイル、Garden で庭タイルを導き、配置を spawn する。
func FurnishBuilding(seed uint64, footprint Rect, door Vec, facility FacilityKind) (Site, []Placed) {
	site, stages := FurnishStages(seed, footprint, door, facility)
	return site, stages[len(stages)-1].Placed
}

// planRooms は施設に応じて部屋群と各部屋の役割を返す。施設ごとに固有の間取りテンプレを持ち、民家は廊下型・
// 店は売場＋バックヤード・診療所は待合＋診察室の列にして、「何の施設か分かる」平面にする。テンプレに足りない
// 小さな footprint と、テンプレの無い施設は BSP へ落として面積最大を主室・残りを奥室にする。
func planRooms(footprint Rect, seed uint64, facility FacilityKind) ([]Room, []roleName) {
	if planner, minW, minH, ok := facilityPlanner(facility); ok && footprint.W >= minW && footprint.H >= minH {
		plan := planner(footprint, seed)
		rooms := make([]Room, len(plan))
		roles := make([]roleName, len(plan))
		for i, hr := range plan {
			rooms[i] = hr.Room
			roles[i] = hr.Role
		}
		return rooms, roles
	}
	rooms := SubdivideBuilding(footprint, seed)
	roles := make([]roleName, len(rooms))
	for rank, ri := range roomOrderByArea(rooms) {
		if rank == 0 {
			roles[ri] = roleMain
		} else {
			roles[ri] = "back"
		}
	}
	return rooms, roles
}

// facilityPlanner は施設種別に対応する間取りテンプレと、テンプレが破綻しない最小寸法を返す。本番の市街地
// チャンク 24x24 が生む建物は街路と前庭ぶん内寄せして概ね 17〜20 タイル角なので、下限を 12x9 まで下げ、
// その狭さでも施設テンプレを発火させる。民家は幅14・高さ13 のどちらかを欠くと PlanHouseAny が田の字の
// コンパクト民家へ切り替える。店・診療所は部屋数が少ないので狭くても成立する。下限を下回る建物とテンプレの
// 無い施設は BSP へ委ねる。骨董品店は店、研究施設は診療所のテンプレを共有する。
func facilityPlanner(facility FacilityKind) (fn func(Rect, uint64) []PlannedRoom, minW, minH consts.Tile, ok bool) {
	switch facility {
	case facHouse:
		return PlanHouseAny, 12, 9, true
	case facStore, facAntique:
		return PlanStore, 12, 9, true
	case facClinic, facLab:
		return PlanClinic, 12, 9, true
	default:
		return nil, 0, 0, false
	}
}

// roleContent は役割から content を引く。main は施設の顔、それ以外はまず施設の room カタログ、無ければ
// 民家の共有役割(corridor 等)、それも無ければ施設別の奥室既定へ落とす。民家だけでなく店・診療所も役割名で
// 部屋を作り分けられるよう、施設カタログを優先して引く。役割名は planRooms とテンプレが付ける。
func roleContent(facility FacilityKind, role roleName, seed uint64) Content {
	if role == roleMain {
		return facilityContent(facility, seed)
	}
	if c, ok := roomCatalog(facility)[role]; ok {
		return c
	}
	if c, ok := houseRoomContents()[role]; ok {
		return c
	}
	return backRoomContent(facility)
}

// roomCatalog は施設種別ごとの「役割名→content」表を返す。houseRoomContents を民家以外へ横展開したもので、
// テンプレが付けた役割名で各室の内装を引く。骨董品店は店、研究施設は診療所の表を共有する。
func roomCatalog(facility FacilityKind) map[roleName]Content {
	switch facility {
	case facHouse:
		return houseRoomContents()
	case facStore, facAntique:
		return storeRoomContents()
	case facClinic, facLab:
		return clinicRoomContents()
	default:
		return nil
	}
}

// backRoomContent は奥室の内装。施設ごとに、店は物置、民家は寝室、診療所は診察室にする。既存の家具を
// 使い回すので新しい content 語彙は要らない。役割カタログに無い役割のフォールバック。
func backRoomContent(facility FacilityKind) Content {
	switch facility {
	case facHouse:
		return bedroomContent()
	case facClinic, facLab:
		return examRoomContent()
	default:
		return storageRoomContent()
	}
}

// roomOrderByArea は部屋を面積降順の添字列で返す。主室に最大の部屋を選ぶための順序。
func roomOrderByArea(rooms []Room) []int {
	idx := make([]int, len(rooms))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		ra, rb := rooms[idx[a]].Rect, rooms[idx[b]].Rect
		return ra.W*ra.H > rb.W*rb.H
	})
	return idx
}
