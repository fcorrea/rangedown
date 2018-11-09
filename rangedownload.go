package rangedownload

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

const MaxConcurrentDownloads = 16

// FileChunk holds the information regarding the download of a chunk of a file
type FileChunk struct {
	URL       string
	ID        int
	FileName  string
	ChunkSize int64
}

// RangeDownload holds information about a download that is performed in paralell
// using the 'Accept-Ranges' http header if supported by the server
type RangeDownload struct {
	URL                 string
	DestinationDir      string
	Chunks              []*FileChunk
	Size                int64
	ConcurrentDownloads int
	Progress            int
	wg                  sync.WaitGroup
	downloaded          chan int
}

// Download will download and write the content to a temporary file
func (f *FileChunk) Download(wg *sync.WaitGroup, out chan<- int64, start, end int64) error {
	defer wg.Done()
	var written int64

	client := &http.Client{}
	r := fmt.Sprintf("bytes=%v-%v", start, end)
	u, err := url.Parse(f.URL)
	if err != nil {
		return err
	}

	req := &http.Request{
		Method:     "GET",
		URL:        u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       nil,
	}
	req.Header.Add("Range", r)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Could not perform request to: %v", f.URL)
	}

	size, err := strconv.ParseInt(resp.Header["Content-Length"][0], 10, 64)

	abspath := os.Args[0] + string(filepath.Separator) + filepath.Base(u.Path) + fmt.Sprintf(".part%d", f.ID)
	fp := filepath.Dir(abspath)
	file, err := os.OpenFile(fp, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("Could not open file: %v", abspath)
	}
	defer file.Close()

	// Start consuming the response body and write it to the chunk file
	buf := make([]byte, 4*1024)
	for {
		data, err := resp.Body.Read(buf)
		if data > 0 {
			_, err := file.Write(buf)
			if err != nil {
				log.Fatalf("Could not write to file %v", abspath)
			}
			written += int64(data)
			out <- int64(data)
		}
		if err != nil {
			if err.Error() == "EOF" {
				if size != written {
					log.Fatal("Imcomplete download")
				}
				break
			}
		}

	}
	return nil
}

// NewRangeDownlaod initializes a RangeDownload struct with the url
func NewRangeDownlaod(u string, c int, d string) *RangeDownload {
	r := &RangeDownload{
		URL:                 u,
		DestinationDir:      d,
		ConcurrentDownloads: c,
	}
	for i := 0; i < c; i++ {
		fc := &FileChunk{
			URL: u,
			ID:  i,
		}
		r.Chunks = append(r.Chunks, fc)
	}
	return r
}

// GetRanges will create a map containing the download ranges for all chunks
func (r *RangeDownload) GetRanges() map[int][]int64 {
	chunkSize := r.Size / int64(len(r.Chunks))
	remainder := r.Size % int64(len(r.Chunks))
	chunks := len(r.Chunks) - 1 // reserve a slot for the remainder bytes

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

// Start initiates the download
func (r *RangeDownload) Start() error {
	chn := make(chan int64)
	ranges := r.GetRanges()
	for i := 0; i < len(r.Chunks); i++ {
		start, end := ranges[i][0], ranges[i][0]
		r.wg.Add(1)
		go r.Chunks[i].Download(&r.wg, chn, start, end)
	}
	return nil
}
