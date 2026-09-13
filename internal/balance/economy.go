package balance

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/world/query"
)

// AuctionTakeHomeRate は基準価値どおりに落札されたときの競売の手取り率を返す。
// 手取り = 落札額 − 手数料 − 発送料を落札額で割る。集荷料は集荷1回ごとで品単位でないためここには含めない。
// 重い安物ほど発送料が手取りを食い、率が下がる。query.AuctionNetProceeds を単一出典で参照する。
//
// 落札額の分散や入札の伸びは確率過程で、実際の落札額分布はモンテカルロで測る領域。ここは基準価値で
// 売れた場合の決定論的な手取り率だけを見る。
func AuctionTakeHomeRate(saleValue consts.Currency, weightKg float64) float64 {
	if saleValue <= 0 {
		return 0
	}
	net := query.AuctionNetProceeds(saleValue, weightKg)
	return float64(net) / float64(saleValue)
}
