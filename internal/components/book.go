package components

// Book は読書可能な本のコンポーネント
type Book struct {
	Effort Pool             // Max=読了に必要な総工数, Current=蓄積した工数
	Skill  *SkillBookEffect // スキル経験値を毎ターン獲得する効果
}

// IsCompleted は読了済みかを返す
func (b *Book) IsCompleted() bool {
	return b.Effort.Current >= b.Effort.Max
}

// SkillBookEffect はスキル経験値を毎ターン獲得する効果
type SkillBookEffect struct {
	TargetSkill   SkillID // 経験値を獲得するスキル
	MaxLevel      int     // この本で上げられるスキル上限
	RequiredLevel int     // 読むのに必要なスキルレベル。0なら誰でも読める
}

// ReadingEfficiency は本とプレイヤーのスキルレベル差に基づく経験値効率を返す（0-100）
// diff = bookLevel - playerLevel（正=本が難しい）
func ReadingEfficiency(playerLevel, bookLevel int) int {
	diff := bookLevel - playerLevel

	const (
		maxDifficulty   = 5  // これ以上難しいと理解できない
		maxEase         = 5  // これ以上易しいと得るものがほぼない
		minEfficiency   = 10 // 易しすぎる場合の最低効率
		hardPenaltyRate = 10 // 難しい側の効率低下率（1レベル差あたり）
		easePenaltyRate = 18 // 易しい側の効率低下率（1レベル差あたり）
	)

	switch {
	case diff > maxDifficulty:
		return 0
	case diff >= 0:
		// 難しい側: 100→50（線形）
		return 100 - diff*hardPenaltyRate
	case diff >= -maxEase:
		// 易しい側: 100→10（線形）
		return 100 + diff*easePenaltyRate
	default:
		return minEfficiency
	}
}
