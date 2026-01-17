package actions

import "errors"

// アクティビティ関連のエラー定数
var (
	// アクティビティ一般エラー
	ErrActivityNil           = errors.New("アクティビティがnilです")
	ErrActorNotSet           = errors.New("アクターが設定されていません")
	ErrActivityNotFound      = errors.New("アクティビティが見つかりません")
	ErrActivityActorNotFound = errors.New("アクティビティアクターが見つかりません")
	ErrActivityCannotPause   = errors.New("アクティビティは中断できません")
	ErrActivityCannotResume  = errors.New("アクティビティは再開できません")
	ErrUnsupportedActivity   = errors.New("未対応のアクティビティタイプです")

	// 攻撃関連エラー
	ErrAttackTargetNotSet     = errors.New("攻撃対象が設定されていません")
	ErrAttackTargetInvalid    = errors.New("攻撃対象が無効です")
	ErrAttackerDead           = errors.New("攻撃者が死亡しています")
	ErrAttackTargetNotExists  = errors.New("攻撃対象が存在しません")
	ErrAttackTargetDead       = errors.New("攻撃対象が既に死亡しています")
	ErrAttackOutOfRange       = errors.New("攻撃対象が射程外です")
	ErrAttackNoWeapon         = errors.New("攻撃手段がありません")
	ErrTargetNoPoolsComponent = errors.New("ターゲットにPoolsコンポーネントがありません")

	// 移動関連エラー
	ErrMoveTargetNotSet       = errors.New("移動先が設定されていません")
	ErrMoveTargetInvalid      = errors.New("移動先が無効です")
	ErrMoveTargetCoordInvalid = errors.New("移動先の座標が無効です")
	ErrMoveNoGridElement      = errors.New("移動するエンティティにGridElementが見つかりません")
	ErrGridElementNotFound    = errors.New("GridElementコンポーネントが見つかりません")

	// アイテム関連エラー
	ErrPositionNotFound = errors.New("位置情報が見つかりません")
	ErrNoItemsToPickup  = errors.New("拾えるアイテムがありません")
	ErrItemPickupFailed = errors.New("アイテムの拾得に失敗しました")
	ErrItemNotSet       = errors.New("アイテムが指定されていません")
	ErrInvalidItem      = errors.New("無効なアイテムです")
	ErrItemNoEffect     = errors.New("このアイテムには効果がありません")
	ErrActorNoPools     = errors.New("アクターにPoolsコンポーネントがありません")

	// 休息関連エラー
	ErrRestEnemiesNearby   = errors.New("周囲に敵がいるため休息できません")
	ErrRestInvalidDuration = errors.New("休息時間が無効です")
	ErrRestEntityNotSet    = errors.New("休息するエンティティが指定されていません")

	// 待機関連エラー
	ErrWaitEntityNotSet    = errors.New("待機するエンティティが指定されていません")
	ErrWaitInvalidDuration = errors.New("待機時間が無効です")

	// 読書関連エラー
	ErrReadNeedsBook         = errors.New("読書には本が必要です")
	ErrReadTargetInvalid     = errors.New("読書対象が無効です")
	ErrReadTargetNotItem     = errors.New("読書対象がアイテムではありません")
	ErrReadBookNotInBackpack = errors.New("本がバックパック内にありません")
	ErrReadTargetNotSet      = errors.New("読書対象が指定されていません")

	// クラフト関連エラー
	ErrCraftNeedsRecipe      = errors.New("クラフトにはレシピが必要です")
	ErrCraftTargetInvalid    = errors.New("クラフト対象が無効です")
	ErrCraftTargetNotRecipe  = errors.New("クラフト対象がレシピではありません")
	ErrCraftRecipeNameGet    = errors.New("レシピ名が取得できません")
	ErrCraftMaterialShortage = errors.New("必要な材料やツールが不足しています")
	ErrCraftEntityNotSet     = errors.New("クラフトするエンティティが指定されていません")

	// ワープ関連エラー
	ErrWarpHoleNotFound = errors.New("ワープホールが見つかりません")
	ErrWarpUnknownType  = errors.New("不明なワープタイプです")
	ErrWarpNoHole       = errors.New("ワープホールがありません")
)
