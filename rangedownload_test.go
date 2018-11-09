package rangedownload

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRangeDownload(t *testing.T) {
	assert := assert.New(t)

	rangedownload := NewRangeDownload("http://foo.com/some.iso")
	assert.Equal(rangedownload.URL, "http://foo.com/some.iso")
}
