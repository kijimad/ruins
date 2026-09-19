package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAuctionHistory(t *testing.T) {
	t.Parallel()

	h := NewAuctionHistory()
	assert.Equal(t, auctionStartingReputation, h.Reputation)
	assert.Equal(t, 0, h.NextNumber)
	assert.Empty(t, h.Entries)
	assert.Empty(t, h.Records)
}
