package uicore_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/widgets/uicore"
	"github.com/stretchr/testify/assert"
)

func TestFitWidth_中身が無ければ下限になる(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 5, uicore.FitWidth(nil, 3, 5), "中身が無いときはlowerになる")
}

func TestFitWidth_最も広い中身にextraを足す(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 13, uicore.FitWidth([]int{4, 10, 7}, 3, 5), "最大値にextraを足した値がlowerを上回るのでそちらを採る")
}

func TestFitWidth_下限を下回らせない(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 20, uicore.FitWidth([]int{1, 2}, 3, 20), "widest+extraがlowerを下回るのでlowerになる")
}

func TestMeasureText_フェイスがnilなら0を返す(t *testing.T) {
	t.Parallel()
	w, h := uicore.MeasureText("hello", nil)
	assert.Equal(t, 0, w, "フェイス無しは幅0")
	assert.Equal(t, 0, h, "フェイス無しは高さ0")
}

func TestLineHeight_実フェイスでAgの高さを返す(t *testing.T) {
	t.Parallel()
	res := borrowRes()
	defer facePool.Put(res)

	_, wantH := uicore.MeasureText("Ag", res.Text.BodyFace)
	assert.Equal(t, wantH, uicore.LineHeight(res.Text.BodyFace), "MeasureTextで測ったAgの高さと一致する")
	assert.Positive(t, uicore.LineHeight(res.Text.BodyFace), "実フェイスなら正の高さになる")
}
