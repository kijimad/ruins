package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeOfDay_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tod  TimeOfDay
		want string
	}{
		{TimeDawn, "Dawn"},
		{TimeMorning, "Morning"},
		{TimeDay, "Noon"},
		{TimeEvening, "Evening"},
		{TimeNight, "Night"},
		{TimeMidnight, "Midnight"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.tod.String())
		})
	}
}

func TestTimeOfDay_String_不正な値はpanicする(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		_ = TimeOfDay(99).String()
	})
}

// TestGameTime_startsAtDawn は新規ゲームの GameTime が経過0・夜明け・1日目から始まることを確認する。
// turn 0 が暦の1日の始まりである夜明けなので、日数も季節もオフセットなしで導ける
func TestGameTime_startsAtDawn(t *testing.T) {
	t.Parallel()

	gt := &GameTime{} // 新規ゲームの初期値
	assert.Equal(t, consts.Turn(0), gt.TotalTurns, "経過ターンは0始まり")
	assert.Equal(t, TimeDawn, gt.GetTimeOfDay(), "夜明けから始まる")
	assert.Equal(t, 1, gt.GetDayNumber(), "1日目から始まる")
	assert.Equal(t, 0, gt.GetTemperatureModifier(), "夜明けの気温修正")
}

// TestGameTime_GetTimeOfDay は経過ターンから時間帯を導けることを確認する。
// 経過0は夜明けで、そこから朝・昼・夕・夜・深夜と進む
func TestGameTime_GetTimeOfDay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalTurns consts.Turn
		expected   TimeOfDay
	}{
		{"経過0は夜明け", 0, TimeDawn},
		{"経過249は夜明け", 249, TimeDawn},
		{"経過250は朝", 250, TimeMorning},
		{"経過499は朝", 499, TimeMorning},
		{"経過500は昼", 500, TimeDay},
		{"経過749は昼", 749, TimeDay},
		{"経過750は夕", 750, TimeEvening},
		{"経過999は夕", 999, TimeEvening},
		{"経過1000は夜", 1000, TimeNight},
		{"経過1249は夜", 1249, TimeNight},
		{"経過1250は深夜", 1250, TimeMidnight},
		{"経過1499は深夜", 1499, TimeMidnight},
		{"経過1500は夜明け（2日目）", 1500, TimeDawn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.totalTurns}
			assert.Equal(t, tt.expected, gt.GetTimeOfDay())
		})
	}
}

// TestGameTime_GetDayNumber は経過ターンから経過日数を導けることを確認する。
// turn 0 が1日目の夜明けなので、日付は turnsPerDay ごと、経過1500 で2日目に繰り上がる
func TestGameTime_GetDayNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalTurns consts.Turn
		expected   int
	}{
		{"経過0は1日目", 0, 1},
		{"経過1499は1日目", 1499, 1},
		{"経過1500は2日目", 1500, 2},
		{"経過3000は3日目", 3000, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.totalTurns}
			assert.Equal(t, tt.expected, gt.GetDayNumber())
		})
	}
}

// TestGameTime_GetTemperatureModifier は経過ターンに対応する時間帯の気温修正値を確認する
func TestGameTime_GetTemperatureModifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalTurns consts.Turn
		expected   int
	}{
		{"夜明けは+0°C", 0, 0},
		{"朝は+5°C", 250, 5},
		{"昼は+10°C", 500, 10},
		{"夕は+5°C", 750, 5},
		{"夜は-5°C", 1000, -5},
		{"深夜は-10°C", 1250, -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.totalTurns}
			assert.Equal(t, tt.expected, gt.GetTemperatureModifier())
		})
	}
}

// TestGameTime_GetSeasonalTemperature は経過日数に対応する季節の世界温度ベースを確認する。
// TotalTurns=(日-1)*1500 で当該日の始まりを指す。
func TestGameTime_GetSeasonalTemperature(t *testing.T) {
	t.Parallel()

	const turnsPerDay consts.Turn = 1500
	tests := []struct {
		name     string
		day      int
		expected int
	}{
		{"1日目は春の中点", 1, -4},
		{"9日目は夏のピーク", 9, 22},
		{"17日目は秋の中点", 17, -4},
		{"25日目は冬の底", 25, -30},
		{"33日目は翌年の春の中点", 33, -4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: consts.Turn(tt.day-1) * turnsPerDay}
			assert.Equal(t, tt.day, gt.GetDayNumber(), "前提: 当該日を指す")
			assert.Equal(t, tt.expected, gt.GetSeasonalTemperature())
		})
	}
}

// TestGameTime_GetSeason は経過日数に対応する季節を確認する。
func TestGameTime_GetSeason(t *testing.T) {
	t.Parallel()

	const turnsPerDay consts.Turn = 1500
	tests := []struct {
		name     string
		day      int
		expected Season
	}{
		{"1日目は春", 1, SeasonSpring},
		{"8日目も春", 8, SeasonSpring},
		{"9日目は夏", 9, SeasonSummer},
		{"17日目は秋", 17, SeasonAutumn},
		{"25日目は冬", 25, SeasonWinter},
		{"33日目は翌年の春", 33, SeasonSpring},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: consts.Turn(tt.day-1) * turnsPerDay}
			assert.Equal(t, tt.day, gt.GetDayNumber(), "前提: 当該日を指す")
			assert.Equal(t, tt.expected, gt.GetSeason())
		})
	}
}

func TestGameTime_Advance(t *testing.T) {
	t.Parallel()

	gt := &GameTime{TotalTurns: 0}
	gt.Advance()
	assert.Equal(t, 1, int(gt.TotalTurns))

	gt.Advance()
	assert.Equal(t, 2, int(gt.TotalTurns))
}

// TestGameTime_AdvanceToNextTimeOfDay は次の時間帯の開始ターンまで進むことを確認する。
// start は経過ターン。区切りは turnsPerTimeOfDay ごとなので、次の区切りへ丸める
func TestGameTime_AdvanceToNextTimeOfDay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		start     consts.Turn
		wantTurns consts.Turn
	}{
		{"区切りの途中から次の区切りへ", 100, 250},
		{"区切りちょうどから次の区切りへ", 250, 500},
		{"区切りの途中から次の区切りへ2", 800, 1000},
		{"日をまたぐ区切りへ", 1300, 1500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.start}
			gt.AdvanceToNextTimeOfDay()
			assert.Equal(t, int(tt.wantTurns), int(gt.TotalTurns))
		})
	}
}

// TestGameTime_AdvanceToNextTimeOfDay_常に1つ進む はどの経過ターンから呼んでも
// 時間帯がちょうど1つ進むことを保証する。どの時間帯からでも差は変わらない
func TestGameTime_AdvanceToNextTimeOfDay_常に1つ進む(t *testing.T) {
	t.Parallel()

	for start := consts.Turn(0); start < turnsPerDay*2; start += 37 {
		gt := &GameTime{TotalTurns: start}
		before := gt.GetTimeOfDay()
		gt.AdvanceToNextTimeOfDay()
		after := gt.GetTimeOfDay()
		// 時間帯の総数は1日のターン数を時間帯あたりのターン数で割った値
		timeOfDayCount := int(turnsPerDay / turnsPerTimeOfDay)
		want := TimeOfDay((int(before) + 1) % timeOfDayCount)
		assert.Equal(t, want, after, "start=%d では1つ次の時間帯になるべき", start)
	}
}

func TestGameTime_AdvanceToNextSeason_季節を1つ進める(t *testing.T) {
	t.Parallel()

	gt := &GameTime{}
	assert.Equal(t, SeasonSpring, gt.GetSeason(), "初期は春")

	gt.AdvanceToNextSeason()
	assert.Equal(t, SeasonSummer, gt.GetSeason())
	gt.AdvanceToNextSeason()
	assert.Equal(t, SeasonAutumn, gt.GetSeason())
	gt.AdvanceToNextSeason()
	assert.Equal(t, SeasonWinter, gt.GetSeason())
	gt.AdvanceToNextSeason()
	assert.Equal(t, SeasonSpring, gt.GetSeason(), "冬の次は翌年の春へ折り返す")
}

// TestGameTime_AdvanceToNextSeason_周期途中から呼んでも1つ進む は、季節周期の境界ちょうどでない
// 経過ターンから呼んでも、ちょうど次の季節の開始へ進むことを固定する。
func TestGameTime_AdvanceToNextSeason_周期途中から呼んでも1つ進む(t *testing.T) {
	t.Parallel()

	// 経過6000は春の周期 [0,12000) の途中（5日目）
	gt := &GameTime{TotalTurns: 6000}
	require.Equal(t, SeasonSpring, gt.GetSeason(), "前提: 6000は春")

	gt.AdvanceToNextSeason()
	assert.Equal(t, SeasonSummer, gt.GetSeason(), "1度で次の季節へ進む")
}

func TestGameTime_SeasonJustChanged(t *testing.T) {
	t.Parallel()

	// 春から夏へ切り替わるのは経過ターン12000。ここで GetDayNumber が8日目から9日目へ繰り上がる
	tests := []struct {
		name  string
		turns consts.Turn
		want  bool
	}{
		{"開始ターンは変化なし", 0, false},
		{"季節が切り替わるターン", 12000, true},
		{"切り替わりの直前は変化なし", 11999, false},
		{"切り替わりの次は変化なし", 12001, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.turns}
			assert.Equal(t, tt.want, gt.SeasonJustChanged())
		})
	}
}

func TestGameTime_TimeOfDayJustChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		turns       consts.Turn
		wantTOD     TimeOfDay
		wantChanged bool
	}{
		{"開始ターンは変化なし", 0, TimeDawn, false},
		{"朝へ入るターン", 250, TimeMorning, true},
		{"昼へ入るターン", 500, TimeDay, true},
		{"時間帯の途中は変化なし", 501, TimeDay, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.turns}
			tod, changed := gt.TimeOfDayJustChanged()
			assert.Equal(t, tt.wantTOD, tod)
			assert.Equal(t, tt.wantChanged, changed)
		})
	}
}
