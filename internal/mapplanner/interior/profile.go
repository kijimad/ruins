package interior

// 建物単位の生活感プロファイル。CDDA の palette 直交合成の翻案で、変種・散らかり・損傷・密度の独立軸を
// 建物ごとに1回引き、全室へ一様に効かせる。少量の語彙を数千通りの見えに乗算し「同じ家を二度見ない」感を
// 機構で作る。docs/design/20260725_70.md 追記その4 収穫4・追記その12。各軸は独立の childSeed ストリームで
// 引き相関を避ける。比率は CDDA 実測を初期値にする。

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

// rollProfile は seed から建物の生活感プロファイルを決定的に引く。各軸を独立ストリームで引いた後、variant を
// 事情として重ねて他軸を上書きする。放棄は損傷・散らかりを底上げ、ためこみは散らかり最大、建築中は無傷で
// 家具まばら。
func rollProfile(seed uint64) buildingProfile {
	p := buildingProfile{
		variant: rollVariant(childSeed(seed, 11_100_000)),
		clutter: rollClutter(childSeed(seed, 11_200_000)),
		damage:  rollDamage(childSeed(seed, 11_000_000)),
		density: densityFactor(childSeed(seed, 10_000_000)),
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

// rollDamage は損傷軸を引く。CDDA 実測 無傷2:小破3:大破1。3分の2の建物が何らかの損傷を持つ。
func rollDamage(seed uint64) damageLevel {
	switch r := childSeed(seed, 1) % 6; {
	case r < 2:
		return dmgIntact
	case r < 5:
		return dmgMinor
	default:
		return dmgMajor
	}
}

// rollClutter は散らかり軸を引く。CDDA 実測 整頓6:乱雑3:汚部屋1。
func rollClutter(seed uint64) clutterLevel {
	switch r := childSeed(seed, 1) % 10; {
	case r < 6:
		return clutterTidy
	case r < 9:
		return clutterMessy
	default:
		return clutterFilthy
	}
}

// rollVariant は変種軸を引く。CDDA 実測を per-mille に写す。標準970・放棄6・建築中10・サバイバリスト9・
// ためこみ5。数十軒に1軒だけ事情のある家が出る。
func rollVariant(seed uint64) variantKind {
	switch r := childSeed(seed, 1) % 1000; {
	case r < 970:
		return varStandard
	case r < 976:
		return varAbandoned
	case r < 986:
		return varUnderConstruction
	case r < 995:
		return varSurvivalist
	default:
		return varHoarder
	}
}

// densityFactor は密度軸を引く。疎6・普通10・密14 の ×/10。既存 scaleDensity の係数と同じ。
func densityFactor(seed uint64) int {
	factors := []int{6, 10, 14}
	return factors[int(childSeed(seed, 1)%uint64(len(factors)))]
}

// clutterPct は散らかりレベルごとに、空き床タイルへ小物を落とす割合を返す。整頓は0で、汚部屋ほど多い。
// 通行と可読性を損なわないよう汚部屋でも上限を抑える。
func clutterPct(level clutterLevel) int {
	switch level {
	case clutterMessy:
		return 8
	case clutterFilthy:
		return 20
	default:
		return 0
	}
}

// applyClutter は散らかりの小物を空き床へ撒く。あるべきでない場所に落ちた物を表し、既存 decor の debris を
// 使うので新規アセットは要らない。装飾で通行は阻まず、戸口前は避ける。呼び出し側が廊下・狭室を除外する。
func applyClutter(seed uint64, room Room, placed []Placed, level clutterLevel) []Placed {
	pct := clutterPct(level)
	if pct == 0 {
		return placed
	}
	occupied := occupiedSet(placed)
	added := make([]Placed, 0)
	for i, t := range room.Rect.interiorTiles() {
		if occupied[t] || isDoorwayAdjacent(room, t) {
			continue
		}
		if dropChance(childSeed(seed, i), 0, pct) {
			occupied[t] = true
			added = append(added, Placed{Kind: KindDecor, Ref: "debris", Pos: t})
		}
	}
	return append(placed, added...)
}
