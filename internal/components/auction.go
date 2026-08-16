package components

// AuctionListing はオークション箱に収納され競売中の品に付く実行時状態。
// 入札が来るたび CurrentBid が上がり競売が延長する。入札が止まったターンに落札が確定し、
// そのときの CurrentBid で売れる。デモ専用の一時状態であり保存しない。
type AuctionListing struct {
	CurrentBid int // 現在の入札額。入札が来るたびに上がる
	LastTurn   int // 直近に入札判定した総ターン数。1ターンに1回だけ判定する
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
