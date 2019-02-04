package rangedown

// GetRanges will create a map containing the download ranges for all chunks
func GetRanges(size int64, count int) map[int][]int64 {
	chunkSize := size / int64(count)
	remainder := size % int64(count)
	chunks := count - 1 // reserve a slot for the remainder bytes
	ranges := make(map[int][]int64, chunks)
	var from, to int64
	for i := 0; i < chunks; i++ {
		to += chunkSize
		ranges[i] = append(ranges[i], from, to)
		from = to + 1
	}
	// Insert the remainder bytes at the end of the map
	to = to + chunkSize + remainder
	ranges[chunks] = append(ranges[chunks], from, to)
	return ranges
}
