package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestRoadChunkSegs(t *testing.T) {
	t.Parallel()
	// チャンク (1,2) の中心は (36,60)。幅4の帯は中心線の -2 から +1 なので、中心の四角は 34 から 37
	c := consts.Coord[consts.Chunk]{X: 1, Y: 2}
	tests := []struct {
		name  string
		ports []roadPort
		want  []roadSeg
	}{
		{"縁の中央の出入り口は中心の四角を含めて縁まで伸ばす", []roadPort{{dir: RoadN}, {dir: RoadE}}, []roadSeg{
			{orient: orientVertical, fixed: 36, lo: 48, hi: 61},
			{orient: orientHorizontal, fixed: 60, lo: 34, hi: 47},
		}},
		{"区域の門に面する出入り口は門の通りの位置から中心へ折れる", []roadPort{{dir: RoadS, gate: true}}, []roadSeg{
			{orient: orientVertical, fixed: 24 + urbanStreetW/2, lo: 58, hi: 71},
			{orient: orientHorizontal, fixed: 60, lo: 24 + urbanStreetW/2 - 2, hi: 37},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, roadChunkSegs(c, tt.ports, 4, 24, 24))
		})
	}
}
