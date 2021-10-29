package tools

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/klauspost/pgzip"
)

func ReadFiles(fileNames []string) ([]*os.File, error) {
	var files []*os.File
	for _, fileName := range fileNames {
		f, err := os.Open(fileName)
		if err != nil {
			return nil, fmt.Errorf("file '%s' not found", fileName)
		}
		files = append(files, f)
	}
	return files, nil
}

func CompressFiles(files []*os.File) (*bytes.Buffer, error) {
	// chained writers -> writing to tw writes to gw -> writes to buffer
	b := new(bytes.Buffer)
	gw := pgzip.NewWriter(b)
	tw := tar.NewWriter(gw)
	defer tw.Close()
	defer gw.Close()

	for _, file := range files {
		err := addToTarArchive(tw, file)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

// Traverses files and directories (recursively) for total size
func FilesTotalSize(files []*os.File) (int64, error) {
	var size int64
	for _, file := range files {
		err := filepath.Walk(file.Name(), func(_ string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				size += info.Size()
			}
			return err
		})
		if err != nil {
			return 0, err
		}
	}
	return size, nil
}

// Credits to: https://gist.github.com/mimoo/25fc9716e0f1353791f5908f94d6e726
func addToTarArchive(tw *tar.Writer, file *os.File) error {
	return filepath.Walk(file.Name(), func(file string, fi os.FileInfo, err error) error {
		header, e := tar.FileInfoHeader(fi, file)
		if e != nil {
			return err
		}
		header.Name = filepath.ToSlash(file)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			defer data.Close()
			if _, err := io.Copy(tw, data); err != nil {
				return err
			}
		}
		return nil
	})
}
