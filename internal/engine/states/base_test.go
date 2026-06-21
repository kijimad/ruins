package states

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseState_SetTransition(t *testing.T) {
	t.Parallel()

	bs := &BaseState[TestWorld]{}
	trans := Transition[TestWorld]{Type: TransPush}
	bs.SetTransition(trans)

	got := bs.GetTransition()
	assert.NotNil(t, got)
	assert.Equal(t, TransPush, got.Type)
}

func TestBaseState_GetTransition_Initial(t *testing.T) {
	t.Parallel()

	bs := &BaseState[TestWorld]{}
	assert.Nil(t, bs.GetTransition(), "初期状態ではnilを返す")
}

func TestBaseState_ClearTransition(t *testing.T) {
	t.Parallel()

	bs := &BaseState[TestWorld]{}
	bs.SetTransition(Transition[TestWorld]{Type: TransReplace})
	bs.ClearTransition()
	assert.Nil(t, bs.GetTransition(), "クリア後はnilを返す")
}

func TestBaseState_ConsumeTransition(t *testing.T) {
	t.Parallel()

	t.Run("遷移がある場合は消費して返す", func(t *testing.T) {
		t.Parallel()
		bs := &BaseState[TestWorld]{}
		bs.SetTransition(Transition[TestWorld]{Type: TransPop})

		got := bs.ConsumeTransition()
		assert.Equal(t, TransPop, got.Type)
		assert.Nil(t, bs.GetTransition(), "消費後はnilになる")
	})

	t.Run("遷移がない場合はTransNoneを返す", func(t *testing.T) {
		t.Parallel()
		bs := &BaseState[TestWorld]{}

		got := bs.ConsumeTransition()
		assert.Equal(t, TransNone, got.Type)
	})
}
