package styled_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/stretchr/testify/assert"
)

func TestName_伸びる左寄せ列を返す(t *testing.T) {
	t.Parallel()
	assert.Equal(t, styled.Col{Mode: styled.ColGrow, Align: styled.AlignLeft}, styled.Name())
}

func TestFit_実測幅の左寄せ列を返す(t *testing.T) {
	t.Parallel()
	assert.Equal(t, styled.Col{Mode: styled.ColFit, Align: styled.AlignLeft}, styled.Fit())
}

func TestNum_実測幅の右寄せ列を返す(t *testing.T) {
	t.Parallel()
	assert.Equal(t, styled.Col{Mode: styled.ColFit, Align: styled.AlignRight}, styled.Num())
}

func TestIcon_正方の左寄せ列を返す(t *testing.T) {
	t.Parallel()
	assert.Equal(t, styled.Col{Mode: styled.ColIcon, Align: styled.AlignLeft}, styled.Icon())
}

func TestDesc_伸びる右寄せ列を返す(t *testing.T) {
	t.Parallel()
	assert.Equal(t, styled.Col{Mode: styled.ColGrow, Align: styled.AlignRight}, styled.Desc())
}

func TestCols(t *testing.T) {
	t.Parallel()

	t.Run("受け取った列をそのまま並びとして返す", func(t *testing.T) {
		t.Parallel()
		got := styled.Cols(styled.Name(), styled.Num(), styled.Icon())
		assert.Equal(t, []styled.Col{styled.Name(), styled.Num(), styled.Icon()}, got)
	})

	t.Run("空で呼ぶと空スライスを返す", func(t *testing.T) {
		t.Parallel()
		got := styled.Cols()
		assert.Empty(t, got)
	})
}

func TestAligns(t *testing.T) {
	t.Parallel()

	t.Run("各列の揃えだけを順に取り出す", func(t *testing.T) {
		t.Parallel()
		cols := []styled.Col{styled.Name(), styled.Num(), styled.Icon(), styled.Desc()}
		got := styled.Aligns(cols)
		assert.Equal(t, []styled.TextAlign{styled.AlignLeft, styled.AlignRight, styled.AlignLeft, styled.AlignRight}, got)
	})

	t.Run("空列なら空スライスを返す", func(t *testing.T) {
		t.Parallel()
		got := styled.Aligns(nil)
		assert.Empty(t, got)
	})
}
