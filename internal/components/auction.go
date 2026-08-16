package components

// AuctionListing は出品中の品に付く実行時状態。タグを貼った時点で採番し、以後その出品を Number で指す。
// 入札が来るたび CurrentBid が上がり競売が延長する。入札が止まったターンに落札が確定し AuctionSold へ移る。
// デモ専用の一時状態であり保存しない。
type AuctionListing struct {
	Number     int // 出品の連番。アイテム詳細で確認できる一意の識別子
	CurrentBid int // 現在の入札額。入札が来るたびに上がる
	LastTurn   int // 直近に入札判定した総ターン数。1ターンに1回だけ判定する
}

// AuctionSold は落札済みで未出荷の品に付く実行時状態。出荷場所での出荷を待つ。
// 落札から出荷には期限があり、期限までに積荷へ渡さないと店の評判が下がる。
// 出荷すると手取りを入金し履歴へ記録して品を手放す。デモ専用の一時状態であり保存しない。
type AuctionSold struct {
	Number    int  // 出品時の連番を引き継ぐ
	Bid       int  // 確定した落札額。手取りは出荷時に発送料と手数料を引いて求める
	DueTurn   int  // 出荷期限のターン。このターンまでに積荷へ渡さないと評判ペナルティを負う
	Penalized bool // 期限超過のペナルティを既に課したか。二重に課さないための印
}

// AuctionStation はオークションの出荷場所を示す。積荷が入ると集荷タイマーが動き出し、
// 満了すると積荷をまとめて集荷する。ここで落札済みの品を出荷し、状況を確認する。
// デモ専用の一時状態であり保存しない。
type AuctionStation struct {
	ShipAtTurn int // 集荷するターン。0 は積荷が無くタイマー停止中を表す
}

// AuctionRecord は1件の出荷実績。落札額から送料と手数料を引いた手取りまでを残す。
type AuctionRecord struct {
	Number int    // 出品の連番
	Name   string // 売れた品の表示名
	Bid    int    // 落札額
	Ship   int    // 送料。重量に比例する
	Fee    int    // 手数料。落札額に比例する
	Net    int    // 手取り。落札額から送料と手数料を引いた額
	Turn   int    // 売れた総ターン数
}

// AuctionHistory は出荷実績の履歴と採番カウンタ、店の評判を持つシングルトン。
// あとで閲覧するための専用の保存領域。デモ専用の一時状態であり保存しない。
type AuctionHistory struct {
	NextNumber int // 次に貼るタグへ振る連番
	Reputation int // 店の評判。出荷期限を破ると下がる
	Records    []AuctionRecord
}
