package main

import (
	"bytes"
	"io"

	"github.com/schollz/progressbar/v3"
)

func main() {
	buf := bytes.NewBuffer([]byte("A frog walks into a bank..."))
	bar := progressbar.DefaultBytes(int64(buf.Len()), "downloading")
	to := &bytes.Buffer{}

	io.Copy(io.MultiWriter(to, bar), buf)
}
