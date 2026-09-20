package states

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatPlayTime_時と分と秒で表す(t *testing.T) {
	t.Parallel()
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0:00:00"},
		{59*time.Minute + 59*time.Second, "0:59:59"},
		{time.Hour + time.Minute + time.Second, "1:01:01"},
		{101*time.Hour + 34*time.Minute + 7*time.Second, "101:34:07"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, formatPlayTime(c.d), "%v", c.d)
	}
}
