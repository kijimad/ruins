package consts

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
)

// Dice は個数抽選。Base 個の Sides 面ダイスの和に Bonus を足す。1d3+1 は {Base:1, Sides:3, Bonus:1}。
// 定数個数は Sides<=0 とし、そのとき値は Bonus になる。5固定は {Sides:0, Bonus:5}。
// フィールドは int のみなので serde 安全で、保存対象コンポーネントに載っても壊れない。
type Dice struct {
	Base  int
	Sides int
	Bonus int
}

// ParseDice は "1d3+1" "2d4" "d6" "5" のような表記を Dice へ変換する。
// d/D の前が個数、後ろが面数、続く +/- 以降が Bonus になる。個数を省くと1、d が無ければ定数として扱う。
// 動的入力用。固定リテラルには MustParseDice を使う。regexp は使わず strings で足りる。ParseWeight と同じ方針。
func ParseDice(s string) (Dice, error) {
	str := strings.TrimSpace(s)
	if str == "" {
		return Dice{}, fmt.Errorf("ダイスが空です")
	}

	// d/D が無ければ定数。"5" は {Sides:0, Bonus:5}
	di := strings.IndexAny(str, "dD")
	if di < 0 {
		n, err := strconv.Atoi(str)
		if err != nil {
			return Dice{}, fmt.Errorf("ダイスの数値が不正です: %q（例: \"1d3+1\" \"5\"）", s)
		}
		return Dice{Bonus: n}, nil
	}

	// 個数部。省略時は1
	base := 1
	if basePart := str[:di]; basePart != "" {
		b, err := strconv.Atoi(basePart)
		if err != nil {
			return Dice{}, fmt.Errorf("ダイスの個数が不正です: %q", s)
		}
		base = b
	}

	// 面数部と Bonus 部。sides[+/-bonus]
	rest := str[di+1:]
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

	if base < 0 {
		return Dice{}, fmt.Errorf("ダイスの個数は負にできません: %q", s)
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

// Roll は Base 個の Sides 面ダイスの和に Bonus を足す。Sides<=0 は定数 Bonus。
// rng は呼び出し側が渡す。共有 RNG を汚さず決定的に抽選するため、内部で乱数器を持たない。
func (d Dice) Roll(rng *rand.Rand) int {
	if d.Sides <= 0 {
		return d.Bonus
	}
	sum := d.Bonus
	for range d.Base {
		sum += rng.IntN(d.Sides) + 1
	}
	return sum
}

// Min は取りうる最小値。Sides<=0 なら Bonus、そうでなければ Base 個の最小目1 の和に Bonus を足す。
func (d Dice) Min() int {
	if d.Sides <= 0 {
		return d.Bonus
	}
	return d.Base + d.Bonus
}

// Max は取りうる最大値。Sides<=0 なら Bonus、そうでなければ Base 個の最大目 Sides の和に Bonus を足す。
func (d Dice) Max() int {
	if d.Sides <= 0 {
		return d.Bonus
	}
	return d.Base*d.Sides + d.Bonus
}

// String は正規化表記を返す。ParseDice(d.String()) が元の Dice に戻る往復性を保つ。
// 表示・save 往復・ゴールデンで表記が揺れないようにするため、個数は常に明示する。
func (d Dice) String() string {
	if d.Sides <= 0 {
		return strconv.Itoa(d.Bonus)
	}
	s := fmt.Sprintf("%dd%d", d.Base, d.Sides)
	switch {
	case d.Bonus > 0:
		s += fmt.Sprintf("+%d", d.Bonus)
	case d.Bonus < 0:
		s += strconv.Itoa(d.Bonus) // マイナス符号を含む
	}
	return s
}
