package lifecycle

import (
	"fmt"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// merchantStockItems は商人が初期在庫として並べるアイテム。1品1個ずつ在庫実体にする
var merchantStockItems = []string{
	"wooden_sword",
	"handgun",
	"western_armor",
	"work_helmet",
	"leather_boots",
	"healing_potion",
	"army_shooting_manual",
}

// recruitNamePool は隊員候補名のプール。値は抽選と生成時翻訳に使う msgid。訳は
// internal/i18n/locale/ja.po に持つ。名前は生成時に query.T で確定し、以後は表示名として扱う
var recruitNamePool = []string{
	"Jin", "Kai", "Ren", "Mira", "Sei",
	"Noa", "Riku", "Yu", "Haru", "Sora",
}

// recruitSpritePool は隊員候補スプライトのプール
var recruitSpritePool = []string{
	"general",
}

// PopulateMerchantStock は商人の品揃えを決める。アイテムと隊員候補を商人所有の LocationInStorage で持たせる。
// 商人生成時に一度だけ呼ぶ。品揃えは呼び出しごとの乱数で決まる
func PopulateMerchantStock(world w.World, merchant ecs.Entity, rng *rand.Rand) error {
	for _, itemName := range merchantStockItems {
		if _, err := SpawnStorageItem(world, itemName, 1, merchant); err != nil {
			return fmt.Errorf("failed to stock item %s: %w", itemName, err)
		}
	}

	count := 3 + rng.IntN(3) // 3〜5人
	used := make(map[string]bool)
	for range count {
		if len(used) >= len(recruitNamePool) {
			break
		}
		// 名前の重複を避ける
		var name string
		for {
			name = recruitNamePool[rng.IntN(len(recruitNamePool))]
			if !used[name] {
				used[name] = true
				break
			}
		}
		abilities := randomRecruitAbilities(rng)
		sprite := recruitSpritePool[rng.IntN(len(recruitSpritePool))]
		// 名前は生成時に現在言語へ確定する。抽選と重複排除は msgid の name で行う
		if _, err := SpawnStorageRecruit(world, merchant, query.T(world, name), abilities, sprite); err != nil {
			return fmt.Errorf("failed to stock recruit %s: %w", name, err)
		}
	}

	return nil
}

// randomRecruitAbilities はランダムな能力値を生成する
func randomRecruitAbilities(rng *rand.Rand) gc.Abilities {
	randStat := func() int { return 4 + rng.IntN(8) } // 4〜11
	return gc.Abilities{
		Vitality:  gc.Ability{Base: randStat()},
		Strength:  gc.Ability{Base: randStat()},
		Sensation: gc.Ability{Base: randStat()},
		Dexterity: gc.Ability{Base: randStat()},
		Agility:   gc.Ability{Base: randStat()},
		Defense:   gc.Ability{Base: randStat()},
	}
}

// SpawnStorageRecruit は商人の在庫に inert な隊員候補を生成する。
// GridElement を持たないためフィールドに出ず、描画も戦闘もAIも対象にならない。雇用で活性化する。
// 名前・能力・スプライトだけ持たせ、雇用時にこの3つから隊員を復元する。
// name は生成時に確定した表示名。以後そのまま表示し、翻訳し直さない
func SpawnStorageRecruit(world w.World, merchant ecs.Entity, name string, abilities gc.Abilities, spriteKey string) (ecs.Entity, error) {
	// 将来は全エンティティ共通の価値算出へ寄せる
	const valuePerAbility = 30
	value := (abilities.Vitality.Base + abilities.Strength.Base + abilities.Sensation.Base +
		abilities.Dexterity.Base + abilities.Agility.Base + abilities.Defense.Base) * valuePerAbility

	recruit := world.Components.AddEntity(world.ECS, &gc.EntitySpec{
		Name:      &gc.Name{Name: name},
		Abilities: &abilities,
		Weight:    &gc.Weight{Milligram: abilities.BodyWeight()},
		Value:     &gc.Value{Value: value},
		SpriteRender: &gc.SpriteRender{
			SpriteSheetName: fieldSpriteSheet,
			SpriteKey:       spriteKey,
			Depth:           gc.DepthNumPlayer,
		},
	})
	if err := MoveToStorage(world, recruit, merchant); err != nil {
		return gc.InvalidEntity, fmt.Errorf("failed to move recruit to storage: %w", err)
	}
	return recruit, nil
}

// ActivateRecruit は在庫の隊員候補を隊員として活性化する。候補実体を消し、隊員をプレイヤー隣へ生成する。
// 候補は Name・Abilities・SpriteRender を持つので、そこから SpawnSquadMember の引数を復元する
func ActivateRecruit(world w.World, player ecs.Entity, recruit ecs.Entity) (ecs.Entity, error) {
	if !world.Components.Abilities.Has(recruit) {
		return gc.InvalidEntity, fmt.Errorf("entity is not a recruit")
	}
	name := world.Components.Name.Get(recruit).Name
	abilities := *world.Components.Abilities.Get(recruit)
	spriteKey := world.Components.SpriteRender.Get(recruit).SpriteKey

	// 先に隊員を生成し、成功してから候補実体を消す。生成が失敗しても候補を在庫に残し、
	// 「代金は返ったのに候補も消えた」という不整合を避ける
	member, err := SpawnSquadMember(world, player, name, abilities, spriteKey)
	if err != nil {
		return gc.InvalidEntity, fmt.Errorf("failed to activate recruit: %w", err)
	}
	world.ECS.RemoveEntity(recruit)
	return member, nil
}
