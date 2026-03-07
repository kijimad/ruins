// Package ui は宣言的UIフレームワークを提供する。
//
// # 概要
//
// Reactに似た宣言的UIを実現する。外部データ（Props）と内部状態（UseState）を
// 分離し、状態管理を担当する。
//
// # 責務
//
//   - Store: 状態とreducerを保持する
//   - UseState: 状態を取得・登録する関数（ReactのuseStateに相当）
//   - UseTabMenu: タブメニュー用の状態を一括登録するヘルパー
//   - Reducer: 個別の状態を更新する関数型
//   - Mount: Props + State を管理し変更を検出する
//
// # 使い分け
//
// Mountは状態管理と変更検出のみを担当する。
// 実際の描画（Widget構築）はアプリケーション層の責務である。
//
// # 使用例
//
//	// Props定義
//	type MenuProps struct {
//	    Items []string
//	}
//
//	// Stateでの使用
//	mount := ui.NewMount[MenuProps]()
//	mount.SetProps(MenuProps{Items: items})
//
//	// UseStateで状態を登録
//	ui.UseState(mount.Store(), "selected", 0, func(v int, a inputmapper.ActionID) int {
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
package ui
