// Package states はゲームステートの実装を提供する。
//
// # Actionベース入力処理の規約
//
// Actionベースの入力処理を行うステートは、es.ActionHandler インターフェースを実装してください。
// これはオプショナルなインターフェースで、すべてのステートが実装する必要はありません。
//
// ## 実装方法
//
// 1. es.ActionHandler インターフェースを実装
//
//	type YourState struct {
//	    es.BaseState[w.World]
//	}
//
//	// インターフェース実装の確認（コンパイル時チェック）
//	var _ es.State[w.World] = &YourState{}
//	var _ es.ActionHandler[w.World] = &YourState{}
//
// 2. HandleInput メソッドを実装（インターフェース要求）
//
//	func (st *YourState) HandleInput() (inputmapper.ActionID, bool) {
//	    if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
//	        return inputmapper.ActionMenuCancel, true
//	    }
//	    // 複数キー同時押しの判定もここで行う
//	    if inpututil.IsKeyJustPressed(ebiten.KeyW) && inpututil.IsKeyJustPressed(ebiten.KeyA) {
//	        return inputmapper.ActionMoveNorthWest, true
//	    }
//	    return "", false
//	}
//
// 3. DoAction メソッドを実装（インターフェース要求）
//
//	func (st *YourState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
//	    switch action {
//	    case inputmapper.ActionMenuCancel:
//	        return es.Transition[w.World]{Type: es.TransPop}, nil
//	    case inputmapper.ActionMoveNorthWest:
//	        // 移動処理
//	        return es.Transition[w.World]{Type: es.TransNone}, nil
//	    default:
//	        return es.Transition[w.World]{Type: es.TransNone}, nil
//	    }
//	}
//
// 4. Update メソッドで規約パターンを使用
//
//	func (st *YourState) Update(world w.World) (es.Transition[w.World], error) {
//	    // キー入力をActionに変換
//	    if action, ok := st.HandleInput(); ok {
//	        if transition, err := st.DoAction(world, action); err != nil {
//	            return es.Transition[w.World]{}, err
//	        } else if transition.Type != es.TransNone {
//	            return transition, nil
//	        }
//	    }
//
//	    // その他の処理
//	    // ...
//
//	    return st.ConsumeTransition(), nil
//	}
//
// ## テスト実装パターン
//
//	func TestDoAction(t *testing.T) {
//	    state := &YourState{}
//	    world := testutil.InitTestWorld(t)
//
//	    // インターフェース実装の検証
//	    var _ es.ActionHandler[w.World] = state
//
//	    transition, err := state.DoAction(world, inputmapper.ActionMenuCancel)
//	    require.NoError(t, err)
//	    assert.Equal(t, es.TransPop, transition.Type)
//	}
//
// ## 参考実装
//
// - DungeonState: 8方向移動とメニュー操作の実装例
// - InventoryMenuState: メニュー操作の実装例
//
// ## メリット
//
// - 入力処理とロジックを分離できる
// - テストからDoActionを直接呼び出せる
// - キー入力に依存しないテストが書ける
// - 複数キー同時押しの判定をhandleInputに集約できる
package states
