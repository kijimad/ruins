package states

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStateNoReuse はステートが再利用されないことを確認するテスト
func TestStateNoReuse(t *testing.T) {
	t.Parallel()

	t.Run("StateFactory[TestWorld]から毎回異なるインスタンスが作成される", func(t *testing.T) {
		t.Parallel()

		// ファクトリー関数を定義
		factory := func() State[TestWorld] {
			return &TestState{name: "TestState"}
		}

		// 複数回ファクトリー関数を実行
		state1 := factory()
		state2 := factory()
		state3 := factory()

		// 各インスタンスが異なることを確認（ポインタアドレスが異なる）
		assert.NotSame(t, state1, state2, "state1とstate2は異なるインスタンスである必要があります")
		assert.NotSame(t, state2, state3, "state2とstate3は異なるインスタンスである必要があります")
		assert.NotSame(t, state1, state3, "state1とstate3は異なるインスタンスである必要があります")
	})

	t.Run("TransitionのStateFactoriesが実行時に新しいインスタンスを作成する", func(t *testing.T) {
		t.Parallel()
		world := TestWorld{Name: "TestWorld"}

		// カウンターを使用して各インスタンスを追跡
		instanceCount := 0
		factory := func() State[TestWorld] {
			instanceCount++
			return &TestStateWithID{
				TestState: TestState{name: "TestState"},
				ID:        instanceCount,
			}
		}

		// Transitionを作成
		transition := Transition[TestWorld]{
			Type:          TransPush,
			NewStateFuncs: []StateFactory[TestWorld]{factory, factory},
		}

		// StateMachineを初期化
		initialState := &TestState{name: "Init"}
		sm, err := Init(initialState, world)
		assert.NoError(t, err)
		sm.lastTransition = transition

		// 最初の実行
		err = sm.Update(world)
		assert.NoError(t, err, "最初のUpdate でエラーが発生")
		assert.Equal(t, 2, instanceCount, "2つのステートが作成されるべき")

		// 同じTransitionで再実行
		sm.lastTransition = transition
		err = sm.Update(world)
		assert.NoError(t, err, "2回目のUpdateでエラーが発生")
		assert.Equal(t, 4, instanceCount, "さらに2つの新しいステートが作成されるべき")
	})

	t.Run("複数のPush操作で毎回新しいインスタンスが作成される", func(t *testing.T) {
		t.Parallel()
		world := TestWorld{Name: "TestWorld"}

		// 作成されたインスタンスを追跡
		createdStates := []*TestStateWithID{}
		idCounter := 0

		factory := func() State[TestWorld] {
			idCounter++
			state := &TestStateWithID{
				TestState: TestState{name: "TestState"},
				ID:        idCounter,
			}
			createdStates = append(createdStates, state)
			return state
		}

		// StateMachineを初期化
		sm, err := Init(&TestState{name: "Init"}, world)
		assert.NoError(t, err)

		// 複数回同じStateFactory[TestWorld]でPush
		for i := 0; i < 3; i++ {
			sm.lastTransition = Transition[TestWorld]{
				Type:          TransPush,
				NewStateFuncs: []StateFactory[TestWorld]{factory},
			}
			err := sm.Update(world)
			assert.NoError(t, err, "Update %d回目でエラーが発生", i+1)
		}

		// 3つの異なるインスタンスが作成されたことを確認
		assert.Equal(t, 3, len(createdStates), "3つのステートが作成されるべき")

		// 各インスタンスのIDが異なることを確認
		assert.Equal(t, 1, createdStates[0].ID)
		assert.Equal(t, 2, createdStates[1].ID)
		assert.Equal(t, 3, createdStates[2].ID)

		// ポインタが異なることを確認
		for i := 0; i < len(createdStates)-1; i++ {
			for j := i + 1; j < len(createdStates); j++ {
				assert.NotSame(t, createdStates[i], createdStates[j],
					"インスタンス%dとインスタンス%dは異なるべき", i, j)
			}
		}
	})
}

// TestStateWithID はIDを持つテスト用ステート
type TestStateWithID struct {
	TestState
	ID int
}
