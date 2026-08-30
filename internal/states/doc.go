// Package states はゲームステートの実装を提供する。
//
// # 入力処理の規約
//
// キーボードの直読みはしない。キーと Action の対応は keybind.Binding の束縛表で宣言し、
// 変換の実行と再生供給源の差し替えは keybind が一手に担う。こうすることで全 state が
// world.Resources.InputSource から Action 列で駆動でき、キー対応の単体テストも
// モックキーボードでまとめて書ける。例外はテキストチャンネルの CharacterNamingState だけで、
// IME の変換途中状態は Action に還元できないためキーを直読みする。
//
// ## メニュー画面
//
// menuloop.Screen に乗せ、menuloop.Model を実装する。カーソル移動系は Screen が吸うので、
// DoAction には画面の意味を持つ Action だけが届く。画面固有キーが要るときは
// menuloop.KeyBindings で束縛表を返す。
//
//	type YourMenuState struct {
//	    es.BaseState[w.World]
//	    screen *menuloop.Screen[YourProps]
//	}
//
//	var _ es.State[w.World] = &YourMenuState{}
//	var _ menuloop.KeyBindings = &YourMenuState{}
//
//	// KeyBindings は共通入力に加える独自キーの束縛表。x で選択中の詳細モーダルを開く
//	func (st *YourMenuState) KeyBindings() []keybind.Binding {
//	    return detailOpenBindings
//	}
//
//	func (st *YourMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
//	    switch action {
//	    case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
//	        return es.Transition[w.World]{Type: es.TransPop}, nil
//	    case inputmapper.ActionMenuSelect:
//	        return st.handleSelection(world)
//	    default:
//	        return es.Transition[w.World]{}, fmt.Errorf("yourMenu: unsupported action: %s", action)
//	    }
//	}
//
// default のエラーは再生シナリオの打ち間違いを検出する安全網なので省略しない。
// 通常 DoAction には画面の意味を持つ Action だけが届くのでこの経路は通らない。
// 返した error は StateMachine.Update から上へ伝播し、握りつぶさず fail-fast で表面化する。
//
// ## メニュー以外の画面
//
// 文脈の束縛表を宣言し、keybind.ReadInput で読んで DoAction へ流す。
// 2キー同時押しは Binding の Held、押しっぱなしのリピートは PressRepeat で表す。
//
//	var yourBindings = []keybind.Binding{
//	    {Key: ebiten.KeyEscape, Action: inputmapper.ActionCloseMenu},
//	    {Key: ebiten.KeyUp, Press: keybind.PressRepeat, Action: inputmapper.ActionMoveNorth},
//	}
//
//	func (st *YourState) Update(world w.World) (es.Transition[w.World], error) {
//	    if action, ok := keybind.ReadInput(world, yourBindings); ok {
//	        if transition, err := st.DoAction(world, action); err != nil {
//	            return es.Transition[w.World]{}, err
//	        } else if transition.Type != es.TransNone {
//	            return transition, nil
//	        }
//	    }
//	    return st.ConsumeTransition(), nil
//	}
//
// ## テスト実装パターン
//
// ロジックは DoAction を直接呼んで検証し、キー対応は keybind.Convert にモックキーボードと
// 束縛表を渡して検証する。画面をまたぐ流れは vrt/replay.PlayScenario で Action 列から
// 本番ループごと駆動する。
//
//	mock := input.NewMockKeyboardInput()
//	mock.SetKeyPressedWithRepeat(ebiten.KeyUp, true)
//	action, ok := keybind.Convert(mock, yourBindings)
//
// ## 参考実装
//
// - SettingsMenuState: menuloop に乗る最小のメニュー
// - ItemActionState: 束縛表を verbList から導出する動詞タブ画面
// - DungeonState: 移動の同時押しとデバッグ表の重ねを含むフィールド入力
package states
