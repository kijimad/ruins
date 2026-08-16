package components

// 通信販売オークションの状態モデル。
// 1つの品はオークションを一方向に進む。状態は専用の enum でなく、どのコンポーネントを持つかと、
// 持ち物か出荷場所の収納かの位置で表す。
//
//	未出品            AuctionListing も AuctionSold も無い持ち物
//	出品中            AuctionListing。入札のたび CurrentBid が上がり延長する
//	落札済み・手元      AuctionSold で持ち物にある。出荷期限の対象
//	落札済み・集荷待ち  AuctionSold でステーションの収納にある。集荷タイマーで自動集荷される
//
// 集荷すると品エンティティは消え、以後は AuctionHistory の台帳へ移る。
//
//	受取金・未精算  AuctionHistory.Entries の受取金明細。精算で所持金へ入る
//	出荷実績        AuctionHistory.Records。精算済みの記録
//
// 手元と集荷待ちは位置で区別する。集荷待ちは持ち物クエリに掛からないので出荷期限で罰しない。
// AuctionSold.Penalized は手元で期限超過した記録であり、状態遷移ではない。

// AuctionListing は出品中の品に付く実行時状態。タグを貼った時点で採番し、以後その出品を Number で指す。
// 入札が来るたび CurrentBid が上がり競売が延長する。入札が止まったターンに落札が確定し AuctionSold へ移る。
type AuctionListing struct {
	Number     int // 出品の連番。アイテム詳細で確認できる一意の識別子
	CurrentBid int // 現在の入札額。入札が来るたびに上がる
	LastTurn   int // 直近に入札判定した総ターン数。1ターンに1回だけ判定する
}

// AuctionSold は落札済みで未出荷の品に付く実行時状態。出荷場所での出荷を待つ。
// 落札から出荷には期限があり、期限までに積荷へ渡さないと店の評判が下がる。
// 出荷すると手取りを入金し履歴へ記録して品を手放す。
type AuctionSold struct {
	Number    int  // 出品時の連番を引き継ぐ
	Bid       int  // 確定した落札額。手取りは出荷時に発送料と手数料を引いて求める
	DueTurn   int  // 出荷期限のターン。このターンまでに積荷へ渡さないと評判ペナルティを負う
	Penalized bool // 期限超過のペナルティを既に課したか。二重に課さないための印
}

// AuctionStation はオークションの出荷場所。積荷はこの prop の収納に置き、集荷タイマーもここが持つ。
// 荷物は物理的にその場所にあるのでステーションごとに持つ。出品や金銭などの情報はシングルトンで共通。
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

// AuctionEntryKind は金銭明細の種別。受取金か請求か。
type AuctionEntryKind string

const (
	// AuctionEntryReceipt は受取金。精算すると所持金へ加える
	AuctionEntryReceipt AuctionEntryKind = "receipt"
	// AuctionEntryInvoice は請求。精算すると所持金から引く
	AuctionEntryInvoice AuctionEntryKind = "invoice"
)

// AuctionEntry は金銭タブに並ぶ明細1件。受取金か請求のどちらかで、精算すると所持金へ足し引きする。
// 金額はいずれも正の額で持ち、足すか引くかは Kind で決める。
type AuctionEntry struct {
	Kind   AuctionEntryKind // 受取金か請求か
	Number int              // 受取金のとき出品の連番。請求は0
	Name   string           // 表示名。品名または請求名
	Amount int              // 受取金は手取り、請求は請求額
	Bid    int              // 明細の内訳。受取金のとき意味を持つ落札額
	Ship   int              // 明細の内訳。配送料
	Fee    int              // 明細の内訳。手数料
}

// AuctionHistory は金銭明細と出荷実績の履歴、採番カウンタ、店の評判を持つシングルトン。
// 出品や落札、金銭はどの出荷場所から見ても共通なのでここに集約する。
type AuctionHistory struct {
	NextNumber int            // 次に貼るタグへ振る連番
	Reputation int            // 店の評判。出荷期限を破ると下がる
	Entries    []AuctionEntry // 金銭タブに並ぶ未精算の受取金と請求
	Records    []AuctionRecord
}

// auctionStartingReputation は店の初期評判。出荷期限を破ると下がる。
const auctionStartingReputation = 100

// NewAuctionHistory は初期状態の履歴シングルトンを返す。評判を初期値にし、明細と実績は空で始める。
func NewAuctionHistory() *AuctionHistory {
	return &AuctionHistory{Reputation: auctionStartingReputation}
}
