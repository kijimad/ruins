package interior

import (
	"strconv"

	"github.com/kijimaD/ruins/internal/oapi"
)

// 建物単位の生活感プロファイル。変種・散らかり・損傷・密度の独立軸を建物ごとに1回引き、全室へ一様に
// 効かせる。少量の語彙を数千通りの見えに乗算し「同じ家を二度見ない」感を機構で作る。各軸は独立の childSeed ストリームで引き相関を
// 避ける。比率は実測ベースのチューニング初期値。

// damageLevel は建物の損傷段階。無傷は経年を掛けず、大破ほど略奪と瓦礫が増える。
type damageLevel int

const (
	dmgIntact damageLevel = iota // 無傷
	dmgMinor                     // 小破
	dmgMajor                     // 大破
)

// clutterLevel は散らかり段階。あるべきでない場所に落ちた小物の量。
type clutterLevel int

const (
	clutterTidy   clutterLevel = iota // 整頓
	clutterMessy                      // 乱雑
	clutterFilthy                     // 汚部屋
)

// variantKind は建物の事情。大多数は標準で、稀に放棄・建築中・サバイバリスト・ためこみが出る。他軸を
// 上書きし、rare な1軒へ物語を与える。専用 prop の要る建築中・サバイバリストは既存語彙で近似する。
type variantKind int

const (
	varStandard          variantKind = iota // 標準
	varAbandoned                            // 放棄
	varUnderConstruction                    // 建築中
	varSurvivalist                          // サバイバリスト
	varHoarder                              // ためこみ
)

// buildingProfile は建物1棟の生活感の直交軸。FurnishStages が全室へ一様に効かせる。
type buildingProfile struct {
	variant variantKind
	clutter clutterLevel
	damage  damageLevel
	density int // 家具量の倍率 ×/10
}

// profileTable は建物の生活感の各軸と hero 部屋の目玉の抽選表。raw の interiorProfile から組み立てる。
// 各軸の並びは raw の行の順で、抽選の結果はこの順に依る。
type profileTable struct {
	damages   []choice[damageLevel]
	clutters  []choice[clutterLevel]
	variants  []choice[variantKind]
	densities []choice[int]
	heroes    []choice[[]string]
}

var (
	damageLevels  = map[oapi.BuildingDamage]damageLevel{oapi.Intact: dmgIntact, oapi.Minor: dmgMinor, oapi.Major: dmgMajor}
	clutterLevels = map[oapi.BuildingClutter]clutterLevel{oapi.Tidy: clutterTidy, oapi.Messy: clutterMessy, oapi.Filthy: clutterFilthy}
	variantKinds  = map[oapi.BuildingVariant]variantKind{
		oapi.Standard:          varStandard,
		oapi.Abandoned:         varAbandoned,
		oapi.UnderConstruction: varUnderConstruction,
		oapi.Survivalist:       varSurvivalist,
		oapi.Hoarder:           varHoarder,
	}
)

// newProfileTable は raw の interiorProfile から抽選表を組み立てる。raw に未設定なら error を返す
func newProfileTable(raws oapi.Raws) (profileTable, error) {
	p := raws.InteriorProfile
	if p == nil {
		return profileTable{}, errInteriorProfileMissing
	}
	var t profileTable
	for _, c := range p.Damages {
		t.damages = append(t.damages, choice[damageLevel]{damageLevels[c.Id], c.Weight})
	}
	for _, c := range p.Clutters {
		t.clutters = append(t.clutters, choice[clutterLevel]{clutterLevels[c.Id], c.Weight})
	}
	for _, c := range p.Variants {
		t.variants = append(t.variants, choice[variantKind]{variantKinds[c.Id], c.Weight})
	}
	for _, c := range p.Densities {
		t.densities = append(t.densities, choice[int]{c.Factor, c.Weight})
	}
	for _, c := range p.Heroes {
		t.heroes = append(t.heroes, choice[[]string]{c.Refs, c.Weight})
	}
	return t, nil
}

// rollProfile は seed から建物の生活感プロファイルを決定的に引く。各軸を独立ストリームで引いた後、variant を
// 事情として重ねて他軸を上書きする。放棄は損傷・散らかりを底上げ、ためこみは散らかり最大、建築中は無傷で
// 家具まばら。
func rollProfile(t profileTable, seed uint64) buildingProfile {
	p := buildingProfile{
		variant: pickChoice(childSeed(childSeed(seed, 11_100_000), 1), t.variants),
		clutter: pickChoice(childSeed(childSeed(seed, 11_200_000), 1), t.clutters),
		damage:  pickChoice(childSeed(childSeed(seed, 11_000_000), 1), t.damages),
		density: pickChoice(childSeed(childSeed(seed, 10_000_000), 1), t.densities),
	}
	switch p.variant {
	case varAbandoned:
		p.damage = dmgMajor
		if p.clutter < clutterMessy {
			p.clutter = clutterMessy
		}
	case varHoarder:
		p.clutter = clutterFilthy
	case varUnderConstruction:
		p.damage = dmgIntact // 新築なので傷はない
		p.density = 6        // 家具まばら
		if p.clutter < clutterMessy {
			p.clutter = clutterMessy // 資材散乱の代用
		}
	case varSurvivalist:
		if p.clutter < clutterMessy {
			p.clutter = clutterMessy // 物資散乱の代用
		}
	case varStandard:
	}
	return p
}

// clutterFurniturePct は散らかりレベルごとに、家具1つの隣へ小物を落とす割合を返す。机の上・たんすの脇に物が
// 溜まる因果を作る。整頓は0で、汚部屋ほど多い。
func clutterFurniturePct(level clutterLevel) int {
	switch level {
	case clutterTidy:
		return 0
	case clutterMessy:
		return 35
	case clutterFilthy:
		return 70
	}
	panic("unknown clutterLevel: " + strconv.Itoa(int(level)))
}

// clutterRefs は散らかりの小物プールを部屋役割で寄せて返す。「あるべきでない場所の物」を、寝室には
// 洗濯かご、その他は木箱、というふうに役割へ寄せて believability を上げる。すべて SpawnProp が要る実在の prop で
// 仮画像でないものだけを使う。食器・写真などの拾える item を家具の上へ載せるのは item spawn 経路が要るので今後。
func clutterRefs(role roleName) []string {
	switch role {
	case "bedroom":
		return []string{"laundry", "crate", "debris"}
	default:
		return []string{"crate", "debris"}
	}
}

// applyClutter は散らかりの小物を配置する。物は床の孤立した位置だけに置くと不自然なので、主に家具の隣へ
// 落とし、机の上・たんすの脇に物が溜まる因果を作る。汚部屋のときだけ、加えて床にもまばらに散らす。役割別
// プールから引き、既存の実スプライトへ写る Ref だけを使う。装飾で通行は阻まず、戸口前は避ける。呼び出し側が
// 廊下・狭室を除外する。
func applyClutter(seed uint64, room Room, placed []Placed, level clutterLevel, role roleName) []Placed {
	furnPct := clutterFurniturePct(level)
	if furnPct == 0 {
		return placed
	}
	pool := clutterRefs(role)
	occupied := occupiedSet(placed)
	added := make([]Placed, 0)
	// 主。家具の隣の空きタイルへ小物を落とす。机やたんすの周りに物が集まる
	for i, p := range placed {
		if p.Kind != KindFurniture || !dropChance(seed, i, furnPct) {
			continue
		}
		for _, n := range neighbors4(p.Pos) {
			if room.Rect.containsInterior(n) && !occupied[n] && !isDoorwayAdjacent(room, n) {
				ref := pickOne(childSeed(seed, 2_000+i), pool)
				occupied[n] = true
				added = append(added, Placed{Kind: KindDecor, Ref: ref, Pos: n})
				break
			}
		}
	}
	// 補。汚部屋は家具から離れた床にもまばらに散らす。家具の無い部屋でも散らかるようにする
	if level == clutterFilthy {
		for i, t := range room.Rect.interiorTiles() {
			if occupied[t] || isDoorwayAdjacent(room, t) {
				continue
			}
			if dropChance(seed, 7_000+i, 7) {
				ref := pickOne(childSeed(seed, 8_000+i), pool)
				occupied[t] = true
				added = append(added, Placed{Kind: KindDecor, Ref: ref, Pos: t})
			}
		}
	}
	return append(placed, added...)
}
