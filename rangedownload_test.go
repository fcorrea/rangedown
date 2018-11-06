package rangedownload

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type RangeDownloadTestSuite struct {
	suite.Suite
}

func (suite *RangeDownloadTestSuite) TestGetRanges() {
	expected := make(map[int][]int64)
	expected[0] = append(expected[0], 0, 40)
	expected[1] = append(expected[1], 41, 80)
	rngd := RangeDownload{
		TotalSize: 80,
		Chunks:    MakeFileChunks(2),
	}
	result := rngd.GetRanges()
	suite.Equal(expected, result)
}

func (suite *RangeDownloadTestSuite) TestGetRangesGrowsTheLastChunk() {
	expected := make(map[int][]int64)
	expected[0] = append(expected[0], 0, 41)
	expected[1] = append(expected[1], 42, 83)
	rngd := RangeDownload{
		TotalSize: 83,
		Chunks:    MakeFileChunks(2),
	}
	result := rngd.GetRanges()
	suite.Equal(expected, result)
}

func (suite *RangeDownloadTestSuite) TestGetRangesOneChunk() {
	expected := make(map[int][]int64)
	expected[0] = append(expected[0], 0, 80)
	rngd := RangeDownload{
		TotalSize: 80,
		Chunks:    MakeFileChunks(1),
	}
	result := rngd.GetRanges()
	suite.Equal(expected, result)
}

func MakeFileChunks(num int) []*FileChunk {
	chunks := make([]*FileChunk, num)
	for i := 0; i < num; i++ {
		chunks[i] = &FileChunk{ID: i}
	}
	return chunks
}

func TestSuiteRangeDownloadSuite(t *testing.T) {
	suite.Run(t, new(RangeDownloadTestSuite))
}
