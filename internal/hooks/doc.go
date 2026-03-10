// Package hooks はReact風の状態管理フックを提供する。
//
// # 概要
//
// 状態管理のみを担当する。UIウィジェットの構築や描画は行わない。
// Reactのフックパターンを参考に、宣言的な状態管理を実現する。
//
// # 責務
//
//   - Store: 状態とreducerを保持する
//   - UseState: 状態を取得・登録する
//   - UseRef: 再レンダリングしても保持される参照を提供する
//   - UseTimer: タイマー状態を管理する
//   - UseTabMenu: タブメニュー用の状態を一括登録する
//   - Mount: Propsの変更を検出する
//
// # 使い分け
//
// 状態管理のみを担当する。UIウィジェットの構築は internal/widgets パッケージ、
// 実際の描画はアプリケーション層（states パッケージ）の責務である。
//
// # 使用例
//
//	// Props定義
//	type MenuProps struct {
//	    Items []string
//	}
//
//	// Mountの作成
//	mount := hooks.NewMount[MenuProps]()
//	mount.SetProps(MenuProps{Items: items})
//
//	// UseStateで状態を登録
//	hooks.UseState(mount.Store(), "selected", 0, func(v int, a inputmapper.ActionID) int {
//	    switch a {
//	    case inputmapper.ActionMenuUp:
//	        return max(0, v-1)
//	    case inputmapper.ActionMenuDown:
//	        return v + 1
//	    }
//	    return v
//	})
//
//	// Dispatchで状態を更新
//	mount.Dispatch(action)
//
//	// 変更があれば再描画
//	if mount.Update() {
//	    widget = buildWidget()
//	}
package hooks
