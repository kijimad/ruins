package components

// AuctionListing は通信販売デモで出品中の品に付く実行時状態。
// 出品タグを貼るとこれが付き、プレイヤーが StepsLeft 歩ぶん動くと落札して Bid が確定する。
// ID はタグ由来の一意識別子で、ピッキングのときに個体を指すのに使う。
// ターン制の進行に依存せず移動そのものを進行の単位にするので、どの場面でも歩けば必ず進む。
// デモ専用の一時状態であり保存しない。
type AuctionListing struct {
	ID        int  // 一意識別子。出品タグから受け継ぐ
	StepsLeft int  // 落札までに必要な残りの移動数
	Slow      bool // 粘り出品。落札まで長いが高値が付く
	Bid       int  // 落札額。落札後に確定する
	Won       bool // 落札済みか
	Announced int  // 直近に告知した残り歩数。カウントダウンの二重告知を防ぐ。初期は -1
}

// AuctionTag は発券機が発行する出品タグ。持ち物として持ち歩く同一の消耗品で、
// 品に貼ると消費され、貼った品にそのとき一意識別子が採番される。
// タグ自体は区別を持たず、どの物に貼るかだけが選択になる。デモ専用の一時状態であり保存しない。
type AuctionTag struct{}

// AuctionClock は通信販売デモの進行状態のシングルトン。プレイヤーの前回位置を覚えて
// 1歩動くたびに出品の残り歩数を減らし、発行するタグの一意識別子も採番する。
// デモ専用の一時状態であり保存しない。
type AuctionClock struct {
	LastX     int
	LastY     int
	HasLast   bool
	NextTagID int // 次に発行するタグの一意識別子
}
