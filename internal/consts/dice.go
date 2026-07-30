package consts

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
)

// Dice は個数抽選。Base 個の Sides 面ダイスの和に Bonus を足す。1d3+1 は {Base:1, Sides:3, Bonus:1}。
// 固定値は Nd1 で表す。5固定は {Base:5, Sides:1}。数字だけの定数表記は使わず、常にダイス形にする。
type Dice struct {
	Base  int
	Sides int
	Bonus int
}

// ParseDice は "1d3+1" "2d4" "1d1" のような表記を Dice へ変換する。
// d の前が個数、後ろが面数、続く +/- 以降が Bonus になる。個数・面数は必ず書く。
// 固定値も "1d1" のように書き、数字だけの省略表記は許さない。表記が一意になり読み手が迷わない。
// 区切りは小文字 d に統一する。大文字 D は受け付けない。
// 動的入力用。固定リテラルには MustParseDice を使う。regexp は使わず strings で足りる。ParseWeight と同じ方針。
func ParseDice(s string) (Dice, error) {
	if s == "" {
		return Dice{}, fmt.Errorf("ダイスが空です")
	}

	// 前後空白も含め厳密に扱う。空白混じりは各数値の Atoi が弾く
	// d は必須。数字だけの定数表記は許さず、固定値も "1d1" と書かせる
	before, after, ok := strings.Cut(s, "d")
	if !ok {
		return Dice{}, fmt.Errorf("ダイス表記には d が必要です。固定値も \"1d1\" のように書く: %q", s)
	}

	// 個数は省略できない。"d6" でなく "1d6" と明示的に書かせて表記を一意にする。
	if before == "" {
		return Dice{}, fmt.Errorf("ダイスの個数を省略できません。例: \"1d6\": %q", s)
	}
	base, err := strconv.Atoi(before)
	if err != nil {
		return Dice{}, fmt.Errorf("ダイスの個数が不正です: %q", s)
	}

	// 面数部と Bonus 部。sides[+/-bonus]
	rest := after
	bonus := 0
	sidesPart := rest
	if bi := strings.IndexAny(rest, "+-"); bi >= 0 {
		sidesPart = rest[:bi]
		b, err := strconv.Atoi(rest[bi:])
		if err != nil {
			return Dice{}, fmt.Errorf("ダイスのボーナスが不正です: %q", s)
		}
		bonus = b
	}
	sides, err := strconv.Atoi(sidesPart)
	if err != nil {
		return Dice{}, fmt.Errorf("ダイスの面数が不正です: %q", s)
	}

	// d を書いたなら個数は1以上を要求する。"0d6" は定数のつもりの誤記とみなして弾く。
	// 定数は "5" のように d を書かずに表す。
	if base < 1 {
		return Dice{}, fmt.Errorf("ダイスの個数は1以上です: %q", s)
	}
	if sides < 1 {
		return Dice{}, fmt.Errorf("ダイスの面数は1以上です: %q", s)
	}
	return Dice{Base: base, Sides: sides, Bonus: bonus}, nil
}

// MustParseDice は ParseDice のパニック版。固定リテラルやテスト値のように失敗があり得ない箇所で使う。
// regexp.MustCompile と同じ発想。動的な入力には error を返す ParseDice を使う。
func MustParseDice(s string) Dice {
	d, err := ParseDice(s)
	if err != nil {
		panic(err)
	}
	return d
}

// Roll は Base 個の Sides 面ダイスの和に Bonus を足す。
// 有効な Dice は Base>=1・Sides>=1。ParseDice が保証する。ゼロ値など不正な Dice を
// 直接渡したバグを即座に気づけるよう panic する。
// rng は呼び出し側が渡す。共有 RNG を汚さず決定的に抽選するため、内部で乱数器を持たない。
func (d Dice) Roll(rng *rand.Rand) int {
	if d.Base < 1 || d.Sides < 1 {
		panic(fmt.Sprintf("不正な Dice です。ParseDice 経由で作る: %+v", d))
	}
	sum := d.Bonus
	for range d.Base {
		sum += rng.IntN(d.Sides) + 1
	}
	return sum
}

// Min は取りうる最小値。Base 個の最小目1 の和に Bonus を足す。
func (d Dice) Min() int {
	return d.Base + d.Bonus
}

// Max は取りうる最大値。Base 個の最大目 Sides の和に Bonus を足す。
func (d Dice) Max() int {
	return d.Base*d.Sides + d.Bonus
}

// String は正規化表記を返す。ParseDice(d.String()) が元の Dice に戻る往復性を保つ。
// 表示・save 往復・ゴールデンで表記が揺れないようにするため、常に NdM 形で出す。
func (d Dice) String() string {
	s := fmt.Sprintf("%dd%d", d.Base, d.Sides)
	switch {
	case d.Bonus > 0:
		s += fmt.Sprintf("+%d", d.Bonus)
	case d.Bonus < 0:
		s += strconv.Itoa(d.Bonus) // マイナス符号を含む
	}
	return s
}
