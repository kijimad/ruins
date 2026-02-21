package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

const (
	// TotalAnimationDuration はアニメーション全体の時間（フレーム数）
	// 120フレーム = 60FPSで2秒
	TotalAnimationDuration = 120
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
	if world.Config.DisableAnimation {
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

		// アニメーション速度を計算（総時間を固定してフレーム数で割る）
		numFrames := int64(len(spriteRender.AnimKeys))
		frameInterval := TotalAnimationDuration / numFrames

		// フレームインデックスを計算
		frameIndex := (sys.animationCounter / frameInterval) % numFrames

		// SpriteKeyを更新
		spriteRender.SpriteKey = spriteRender.AnimKeys[frameIndex]
	}))
	return nil
}
