package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

const (
	// AnimationFrameInterval はアニメーションフレーム切替間隔（フレーム数）
	// 30フレームごとに切り替え（60FPSで0.5秒）
	AnimationFrameInterval = 30
)

// AnimationSystem は全エンティティのスプライトアニメーションを更新する
type AnimationSystem struct {
	animationCounter int64
}

// NewAnimationSystem はAnimationSystemを初期化する
func NewAnimationSystem() *AnimationSystem {
	return &AnimationSystem{}
}

// String はシステム名を返す
// w.Updater interfaceを実装
func (sys AnimationSystem) String() string {
	return "AnimationSystem"
}

// Update は全エンティティのスプライトアニメーションを更新する
// w.Updater interfaceを実装
func (sys *AnimationSystem) Update(world w.World) error {
	cfg := config.Get()
	if cfg.DisableAnimation {
		return nil
	}

	sys.animationCounter++

	world.Manager.Join(
		world.Components.SpriteRender,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		spriteRender := world.Components.SpriteRender.Get(entity).(*gc.SpriteRender)

		// AnimKeysが空ならアニメーションなし
		if len(spriteRender.AnimKeys) == 0 {
			return
		}

		// フレームインデックスを計算
		frameIndex := (sys.animationCounter / AnimationFrameInterval) % int64(len(spriteRender.AnimKeys))

		// SpriteKeyを更新
		spriteRender.SpriteKey = spriteRender.AnimKeys[frameIndex]
	}))
	return nil
}
