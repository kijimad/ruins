package activity

import "errors"

// アクティビティ関連のエラー定数
var (
	// アクティビティ一般エラー
	// ErrValidationFailed は Validate によるユーザ起因の検証失敗を包むセンチネル。
	// 弾切れ・敵接近・スキル不足など、プレイヤーに提示して操作をキャンセルすべき失敗を示す。
	// 呼び出し側は errors.Is で見分けて握りつぶし、それ以外のシステムエラーは伝播させる。
	// Validate は不変条件違反を error で返さず panic するため、これに紛れ込まない。契約は Behavior.Validate
	ErrValidationFailed      = errors.New("activity validation failed")
	ErrActivityNil           = errors.New("activity is nil")
	ErrActorNotSet           = errors.New("actor is not set")
	ErrActivityNotFound      = errors.New("activity not found")
	ErrActivityActorNotFound = errors.New("activity actor not found")
	ErrActivityCannotPause   = errors.New("activity cannot be paused")
	ErrActivityCannotResume  = errors.New("activity cannot be resumed")
	ErrUnsupportedActivity   = errors.New("unsupported activity type")

	// 攻撃関連エラー
	ErrAttackTargetNotSet    = errors.New("attack target is not set")
	ErrAttackTargetInvalid   = errors.New("attack target is invalid")
	ErrAttackerDead          = errors.New("attacker is dead")
	ErrAttackTargetNotExists = errors.New("attack target does not exist")
	ErrAttackTargetDead      = errors.New("attack target is already dead")
	ErrAttackOutOfRange      = errors.New("attack target is out of range")
	ErrAttackNoWeapon        = errors.New("no means of attack")
	ErrTargetNoHPComponent   = errors.New("target has no HP component")

	// 射撃関連エラー
	ErrShootNoFireWeapon       = errors.New("no ranged weapon equipped")
	ErrShootNoAmmo             = errors.New("out of ammo, please reload")
	ErrShootLineOfSightBlocked = errors.New("line of sight is blocked")
	ErrReloadNotNeeded         = errors.New("reload is not needed")
	ErrReloadNoAmmo            = errors.New("no ammo")

	// 移動関連エラー
	ErrMoveTargetNotSet       = errors.New("move destination is not set")
	ErrMoveTargetInvalid      = errors.New("move destination is invalid")
	ErrMoveTargetCoordInvalid = errors.New("move destination coordinate is invalid")
	ErrMoveOverweight         = errors.New("too heavy to move")
	ErrGridElementNotFound    = errors.New("GridElement component not found")

	// アイテム関連エラー
	ErrPositionNotFound = errors.New("position not found")
	ErrNoItemsToPickup  = errors.New("no items to pick up")
	ErrItemPickupFailed = errors.New("failed to pick up item")
	ErrItemNotSet       = errors.New("item is not set")
	ErrItemNoEffect     = errors.New("this item has no effect")

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
