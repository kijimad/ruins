package raw_test

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/require"
)

func TestSelectByWeight(t *testing.T) {
	t.Parallel()
	ct := oapi.CommandTable{
		Name: "TEST",
		Entries: []oapi.CommandTableEntry{
			{
				Weapon: "A",
				Weight: 0.5,
			},
			{
				Weapon: "B",
				Weight: 0.2,
			},
			{
				Weapon: "C",
				Weight: 0.3,
			},
		},
	}

	rng := rand.New(rand.NewPCG(12345, 67890))
	_, err := raw.SelectCommandByWeight(ct, rng)
	require.NoError(t, err)
}
