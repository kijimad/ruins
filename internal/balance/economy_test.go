package balance

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestAuctionTakeHomeRate_手数料と発送料(t *testing.T) {
	t.Parallel()
	// 価値1000・1kg: 手数料120 + 発送料25 を引き 855。手取り率85.5%
	assert.InDelta(t, 0.855, AuctionTakeHomeRate(1000, 1), 1e-9)
	// 重い安物は発送料に食われる。価値1000・10kg: 手数料120 + 発送料250 → 630 = 63%
	assert.InDelta(t, 0.630, AuctionTakeHomeRate(1000, 10), 1e-9)
	// 価値0は0を返す
	assert.Equal(t, 0.0, AuctionTakeHomeRate(consts.Currency(0), 1))
}
