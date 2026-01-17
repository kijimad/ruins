/*
Package save は安定ID + リフレクションベースのECSシリアライゼーションシステムを提供する。

## 概要

このパッケージはBevy ECSの安定ID概念とGoのリフレクション機能を組み合わせて、
ECSワールドの状態を安全にセーブ・ロードする機能を提供します。

## 主要な特徴

### 1. 安定ID システム
- エンティティIDは世代管理により安定性を保証
- エンティティの削除・再作成時も参照の整合性を維持
- セーブファイル間でのエンティティ参照が安全

### 2. 自動リフレクション
- Goのreflectionを使用してコンポーネント型を自動検出
- 手動でのコンポーネント登録が不要
- 新しいコンポーネント追加時の修正が最小限

### 3. エンティティ参照の自動解決
- AIVision.TargetEntity などのエンティティ間参照を自動処理
- セーブ時は安定IDに変換、ロード時は実エンティティに復元

## 使用方法

### 基本的なセーブ・ロード

```go
// シリアライゼーションマネージャーを作成
manager := save.NewSerializationManager("./saves")

// ワールドを保存
err := manager.SaveWorld(world, "slot1")

	if err != nil {
	    log.Fatal(err)
	}

// ワールドを読み込み
err = manager.LoadWorld(world, "slot1")

	if err != nil {
	    log.Fatal(err)
	}

```

### 新しいコンポーネントの対応

新しいコンポーネント型を追加する場合は、reflection.goの
InitializeFromWorld関数に登録を追加してください：

```go

	func (r *ComponentRegistry) InitializeFromWorld(world w.World) error {
	    // ... 既存の登録 ...

	    // 新しいコンポーネントを登録
	    r.registerComponent(
	        reflect.TypeOf(&gc.NewComponent{}),
	        components.NewComponent,
	        r.extractNewComponent,
	        r.restoreNewComponent,
	        nil, // エンティティ参照がない場合
	    )

	    return nil
	}

```

そして対応する抽出・復元関数を実装：

```go

	func (r *ComponentRegistry) extractNewComponent(world w.World, entity ecs.Entity) (interface{}, bool) {
	    if !entity.HasComponent(world.Components.NewComponent) {
	        return nil, false
	    }
	    comp := world.Components.NewComponent.Get(entity).(*gc.NewComponent)
	    return *comp, true
	}

	func (r *ComponentRegistry) restoreNewComponent(world w.World, entity ecs.Entity, data interface{}) error {
	    comp, ok := data.(gc.NewComponent)
	    if !ok {
	        return fmt.Errorf("invalid NewComponent data type: %T", data)
	    }
	    world.Components.NewComponent.Set(entity, &comp)
	    return nil
	}

```

## 設計原則

### 型安全性
- Goの型システムを活用して実行時エラーを最小化
- リフレクションを使用しつつも型安全性を保持

### パフォーマンス
- コンポーネントレジストリの初期化は1回のみ
- 安定IDマッピングはハッシュマップで高速アクセス

### 拡張性
- 新しいコンポーネント型の追加が容易
- セーブデータフォーマットの後方互換性を考慮

### エラーハンドリング
- 不正なセーブデータに対して適切なエラーメッセージ
- 部分的な復元エラーでも可能な限り処理を継続

## 制限事項

### サポートされるコンポーネント型
現在サポートされているコンポーネント：
- Camera (カメラのみPositionコンポーネントを使用)
- GridElement (タイルベースの位置情報)
- AIVision, AIRoaming, AIChasing
- SpriteRender
- NullComponent (Player, BlockView, BlockPass)

### エンティティ参照
現在、AIVision.TargetEntityのみ自動処理されます。
他のエンティティ参照を持つコンポーネントを追加する場合は、
対応するResolveRefFunc を実装する必要があります。

## ファイル形式

セーブファイルはJSON形式で、以下の構造を持ちます：

```json

	{
	  "version": "1.0.0",
	  "timestamp": "2025-01-XX...",
	  "world": {
	    "entities": [
	      {
	        "stable_id": {"index": 1, "generation": 0},
	        "components": {
	          "GridElement": {
	            "type": "GridElement",
	            "data": {"x": 5, "y": 10}
	          }
	        }
	      }
	    ]
	  }
	}

```

## パッケージ責務

- **stable_id.go**: 安定ID管理システム
- **reflection.go**: コンポーネント型のリフレクション・レジストリ
- **manager.go**: セーブ・ロード処理の統合管理
- **manager_test.go**: 総合テストスイート

## 使い分け

このパッケージは以下の用途に適しています：
- ゲームの進行状況保存
- デバッグ用のワールド状態スナップショット
- レベルエディタでの作成データ保存

高頻度のスナップショット（ネットワーク同期等）には、
より軽量なバイナリ形式を検討することを推奨します。
*/
package save
