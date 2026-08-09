package activity

import "errors"

// アクティビティ関連のエラー定数
var (
	// アクティビティ一般エラー
	ErrActivityNil           = errors.New("activity is nil")
	ErrActorNotSet           = errors.New("actor is not set")
	ErrActivityNotFound      = errors.New("activity not found")
	ErrActivityActorNotFound = errors.New("activity actor not found")
	ErrActivityCannotPause   = errors.New("activity cannot be paused")
	ErrActivityCannotResume  = errors.New("activity cannot be resumed")
	ErrUnsupportedActivity   = errors.New("unsupported activity type")

	// 攻撃関連エラー
	ErrAttackTargetNotSet  = errors.New("attack target is not set")
	ErrAttackTargetInvalid = errors.New("attack target is invalid")
	ErrAttackerDead        = errors.New("attacker is dead")
	ErrTargetNoHPComponent = errors.New("target has no HP component")

	// 射撃関連エラー
	ErrShootNoFireWeapon = errors.New("no ranged weapon equipped")

	// 移動関連エラー
	ErrMoveTargetNotSet    = errors.New("move destination is not set")
	ErrMoveTargetInvalid   = errors.New("move destination is invalid")
	ErrGridElementNotFound = errors.New("GridElement component not found")

	// アイテム関連エラー
	ErrPositionNotFound = errors.New("position not found")
	ErrNoItemsToPickup  = errors.New("no items to pick up")
	ErrItemPickupFailed = errors.New("failed to pick up item")
	ErrItemNotSet       = errors.New("item is not set")

	// 休息関連エラー
	ErrRestEnemiesNearby   = errors.New("cannot rest because enemies are nearby")
	ErrRestInvalidDuration = errors.New("rest duration is invalid")

	// 待機関連エラー
	ErrWaitInvalidDuration = errors.New("wait duration is invalid")

	// 読書関連エラー
	ErrReadNeedsBook         = errors.New("reading requires a book")
	ErrReadTargetInvalid     = errors.New("read target is invalid")
	ErrReadTargetNotItem     = errors.New("read target is not an item")
	ErrReadBookNotInBackpack = errors.New("book is not in the backpack")
	ErrReadTargetNotSet      = errors.New("read target is not set")

	// クラフト関連エラー
	ErrCraftNeedsRecipe      = errors.New("crafting requires a recipe")
	ErrCraftTargetInvalid    = errors.New("craft target is invalid")
	ErrCraftTargetNotRecipe  = errors.New("craft target is not a recipe")
	ErrCraftRecipeNameGet    = errors.New("cannot get recipe name")
	ErrCraftMaterialShortage = errors.New("required materials or tools are insufficient")
	ErrCraftEntityNotSet     = errors.New("crafting entity is not set")

	// ワープ関連エラー
	ErrWarpHoleNotFound = errors.New("warp hole not found")
	ErrWarpUnknownType  = errors.New("unknown warp type")
	ErrWarpNoHole       = errors.New("no warp hole")
)
