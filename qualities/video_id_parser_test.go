package qualities

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVideoIDParser(t *testing.T) {
	assert.Equal(t, ParseVideoID("https://youtu.be/m7Mzgmpr-Qc"), "m7Mzgmpr-Qc")

	assert.Equal(t, ParseVideoID("https://www.youtube.com/watch?v=m7Mzgmpr-Qc"), "m7Mzgmpr-Qc")
	assert.Equal(t, ParseVideoID("https://m.youtube.com/watch?v=m7Mzgmpr-Qc"), "m7Mzgmpr-Qc")
	assert.Equal(t, ParseVideoID("https://youtube.com/watch?v=m7Mzgmpr-Qc"), "m7Mzgmpr-Qc")

	assert.Equal(t, ParseVideoID("https://www.youtube.com/live/m7Mzgmpr-Qc"), "m7Mzgmpr-Qc")
	assert.Equal(t, ParseVideoID("https://m.youtube.com/live/m7Mzgmpr-Qc"), "m7Mzgmpr-Qc")
	assert.Equal(t, ParseVideoID("https://youtube.com/live/m7Mzgmpr-Qc"), "m7Mzgmpr-Qc")

	assert.Empty(t, ParseVideoID("https://www.youtube.com/feed/subscriptions"), "")

	assert.Equal(t, ParseVideoID("https://dev.pomu.app"), "https://dev.pomu.app")
}
