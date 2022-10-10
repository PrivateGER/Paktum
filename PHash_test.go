package main

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

func BenchmarkPHash(b *testing.B) {
	res, err := http.Get("https://img3.gelbooru.com/images/de/aa/deaa746c4de029e7539eb767b36d7f40.png")
	if err != nil {
		b.Error(err)
	}
	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, res.Body)
	if err != nil {
		b.Error(err)
	}
	reader := bytes.NewReader(buffer.Bytes())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res := GeneratePHash(DecodeImage(reader))
		if res == 0 {
			b.Error("Failed to generate pHash")
		}
		_, _ = reader.Seek(0, 0)
	}
}
