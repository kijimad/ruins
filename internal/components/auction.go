package components

// AuctionListing はオークション箱に収納され競売中の品に付く実行時状態。
// ResolveTurn の総ターン数に達すると今回の入札が決着する。売れれば品ごと消え、
// 売れなければ ResolveTurn を更新して再入札する。デモ専用の一時状態であり保存しない。
type AuctionListing struct {
	ResolveTurn int // この総ターン数で今回の入札が決着する
}

// AuctionBox はオークション箱を示すマーカー。この収納へ入れた品が競売にかかる。
// デモ専用の一時状態であり保存しない。
type AuctionBox struct{}

// AuctionRecord は1件の出荷実績。落札額から送料と手数料を引いた手取りまでを残す。
type AuctionRecord struct {
	Name string // 売れた品の表示名
	Bid  int    // 落札額
	Ship int    // 送料。重量に比例する
	Fee  int    // 手数料。落札額に比例する
	Net  int    // 手取り。落札額から送料と手数料を引いた額
	Turn int    // 売れた総ターン数
}

// AuctionHistory は出荷実績の履歴を保持するシングルトン。あとで閲覧するための専用の保存領域。
// デモ専用の一時状態であり保存しない。
type AuctionHistory struct {
	Records []AuctionRecord
}
