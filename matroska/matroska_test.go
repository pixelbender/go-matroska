package matroska

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMatroskaTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Matroska Test Suite in short mode")
	}
	_, err := os.Stat("testdata")
	if err != nil && os.IsNotExist(err) {
		if err = download("https://downloads.sourceforge.net/project/matroska/test_files/matroska_test_w1_1.zip", "testdata.zip"); err != nil {
			t.Fatal(err)
		}
		defer os.Remove("testdata.zip")
		if err = unpack("testdata.zip", "testdata"); err != nil {
			t.Fatal(err)
		}
		_, err = os.Stat("testdata")
	}
	if err != nil {
		t.Fatal(err)
	}
	tags := map[string][]*Tag{
		"test1.mkv": newTestTags("Big Buck Bunny - test 1", "Matroska Validation File1, basic MPEG4.2 and MP3 with only SimpleBlock"),
		"test2.mkv": newTestTags("Elephant Dream - test 2", "Matroska Validation File 2, 100,000 timecode scale, odd aspect ratio, and CRC-32. Codecs are AVC and AAC"),
		"test3.mkv": newTestTags("Elephant Dream - test 3", "Matroska Validation File 3, header stripping on the video track and no SimpleBlock"),
		"test4.mkv": nil,
		"test5.mkv": newTestTags("Big Buck Bunny - test 8", "Matroska Validation File 8, secondary audio commentary track, misc subtitle tracks"),
		"test6.mkv": newTestTags("Big Buck Bunny - test 6", "Matroska Validation File 6, random length to code the size of Clusters and Blocks, no Cues for seeking"),
		"test7.mkv": newTestTags("Big Buck Bunny - test 7", "Matroska Validation File 7, junk elements are present at the beggining or end of clusters, the parser should skip it. There is also a damaged element at 451418"),
		"test8.mkv": newTestTags("Big Buck Bunny - test 8", "Matroska Validation File 8, audio missing between timecodes 6.019s and 6.360s"),
	}
	for name, it := range tags {
		file := filepath.Join("testdata", name)
		want := it
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			doc, err := Decode(file)
			if err != nil {
				t.Fatal(err)
			}
			got := doc.Segment.Tags
			if !reflect.DeepEqual(want, got) {
				t.Errorf("Unexpected tags, want: %s\ngot: %s", dump(want), dump(got))
			}
			//for _, c := range doc.Segment.Cluster {
			//	log.Printf(">>>>>> %s", dump(c))
			//}
		})
	}
}

func newTestTags(title, comment string) []*Tag {
	return []*Tag{{
		Targets: []*Target{
			{TypeValue: 50},
		},
		SimpleTags: []*SimpleTag{
			NewSimpleTag("TITLE", title),
			NewSimpleTag("DATE_RELEASED", "2010"),
			NewSimpleTag("COMMENT", comment),
		},
	}}
}

func dump(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func download(url, file string) error {
	out, err := os.Create(file)
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func unpack(file, dir string) error {
	in, err := zip.OpenReader(file)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, it := range in.File {
		path := filepath.Join(dir, it.Name)
		if it.FileInfo().IsDir() {
			os.MkdirAll(path, it.Mode())
			continue
		}
		in, err := it.Open()
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, it.Mode())
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, in); err != nil {
			return err
		}
	}
	return nil
}
