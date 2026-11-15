package systems

import (
	"testing"

	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/stretchr/testify/assert"
)

func TestAreaEntrySystem_observe_別のエリアに入ったときだけ入ったとみなす(t *testing.T) {
	t.Parallel()

	townA := overworld.Area{Key: "urban:town:0,0", Name: "Town"}
	townB := overworld.Area{Key: "urban:town:9,9", Name: "Town"}
	wasteland := overworld.Area{Name: "Wasteland"}

	sys := &AreaEntrySystem{}
	seq := []overworld.Area{townA, townA, wasteland, townA, townB, wasteland, wasteland}
	got := make([]bool, 0, len(seq))
	for _, a := range seq {
		got = append(got, sys.observe(a))
	}
	assert.Equal(t, []bool{
		false, // 最初に見たエリアは記憶するだけ
		false, // 同じ区域の中
		false, // 荒れ地に出た
		true,  // 荒れ地から町に戻った
		true,  // 同じ種類の別の区域に移った
		false, // 荒れ地に出た
		false, // 荒れ地の中
	}, got)
}
