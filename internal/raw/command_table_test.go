package raw_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
)

// 検証する方法がわからんのでprintして確かめる用
func TestSelectByWeight(t *testing.T) {
	t.Parallel()
	ct := raw.CommandTable{
		Name: "TEST",
		Entries: []raw.CommandTableEntry{
			raw.CommandTableEntry{
				Weapon: "A",
				Weight: 0.5,
			},
			raw.CommandTableEntry{
				Weapon: "B",
				Weight: 0.2,
			},
			raw.CommandTableEntry{
				Weapon: "C",
				Weight: 0.3,
			},
		},
	}

	_ = ct.SelectByWeight()
}
