package states

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatPlayTime_時と分で表す(t *testing.T) {
	t.Parallel()
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0:00"},
		{59 * time.Minute, "0:59"},
		{time.Hour + time.Minute, "1:01"},
		{101*time.Hour + 34*time.Minute, "101:34"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, formatPlayTime(c.d), "%v", c.d)
	}
}
