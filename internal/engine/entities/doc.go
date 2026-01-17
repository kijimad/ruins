// Package entities はエンティティとコンポーネントの管理機能を提供する。
//
// このパッケージは以下の機能を提供する：
// - エンティティの作成と管理
// - コンポーネントの動的な追加
// - 事前定義されたエンティティの読み込み
// - ECSワールドへのエンティティ追加
//
// 使用例：
//
//	// エンティティの作成
//	componentList := entities.ComponentList[gc.EntitySpec]{}
//	componentList.Entities = append(componentList.Entities, gc.EntitySpec{
//	    Position: &gc.Position{X: 100, Y: 200},
//	})
//
//	// ワールドへの追加
//	entitiesResult := entities.AddEntities(world, componentList)
//
// パッケージの責務：
// - エンティティ作成: コンポーネントリストからエンティティを作成
// - 動的追加: リフレクションを使用したコンポーネントの動的追加
// - 事前定義: 特定のエンティティタイプの事前定義
// - ECS統合: ECSワールドとの統合
package entities
