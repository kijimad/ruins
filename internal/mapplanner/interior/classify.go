package interior

// classifyRoom は配置された stuff から部屋の役割を逆推定する。生成の検証に使い、店のはずが店に見えない
// 部屋を機械的に検出する。生成が役割から家具を置くのに対し、これは家具から役割を読む逆関数で、両者が
// 一致すれば「意図が読める部屋」ができている。特徴的な家具の一致数で採点し最も高い役割を返す。どの役割の
// 特徴も無ければ unknown を返す。

// roomSignature は役割を特徴づける家具の集合。これらが多く在るほどその役割らしい。抽象度を上げず、店なら
// レジと棚、診療所なら診察台と薬棚、といった「その施設を施設たらしめる物」を並べる。
type roomSignature struct {
	role string
	refs []string
}

// roomSignatures は役割の判定表。採点が同点なら先に並んだ役割を優先する。診療所は待合の受付什器も併せ持つ
// ので、待合より先に置いて診察室側へ倒す。
var roomSignatures = []roomSignature{
	{"store", []string{"register", "gondola", "walkin_cooler"}},
	{"clinic", []string{"exam_bed", "medcabinet"}},   // 診察室。exam_bed を持つので薬局より先に置いても奪われない
	{"pharmacy", []string{"medcabinet", "meds"}},     // 薬局。薬棚＋薬。exam_bed の無い薬品庫
	{"waiting", []string{"reception", "waitchair"}},
	{"dressing", []string{"washer", "sink"}},
	{"kitchen", []string{"pantry", "sink"}}, // 流しと食器棚の両方。流し単独は脱衣所と曖昧なので棚で分ける
	{"bath", []string{"bathtub"}},
	{"toilet", []string{"toilet"}},             // 便器のみは民家のトイレ
	{"restroom", []string{"toilet", "sink"}},   // 便器＋流しは店・診療所のトイレ。toilet より後で、両方揃えば勝つ
	{"office", []string{"desk"}},               // 机は事務所の署名
	{"bedroom", []string{"bed"}},
	{"living", []string{"sofa"}},
	{"storage", []string{"barrel"}},
}

// classifyRoom は placed の家具から役割を返す。present は membership 判定にのみ使うので map でも決定性を
// 損なわない。採点は roomSignatures の並び順に走るので同点は先の役割が勝つ。
func classifyRoom(placed []Placed) string {
	present := make(map[string]bool)
	for _, p := range placed {
		present[p.Ref] = true
	}

	best, bestScore := "unknown", 0
	for _, sig := range roomSignatures {
		score := 0
		for _, ref := range sig.refs {
			if present[ref] {
				score++
			}
		}
		if score > bestScore {
			best, bestScore = sig.role, score
		}
	}
	return best
}
