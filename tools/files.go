package tools

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"

	gzip "github.com/klauspost/pgzip"
)

func ReadFiles(fileNames []string) ([]*os.File, error) {
	files := make([]*os.File, len(fileNames))
	for _, fileName := range fileNames {
		f, err := os.Open(fileName)
		if err != nil {
			return nil, fmt.Errorf("file '%s' not found", fileName)
		}
		files = append(files, f)
	}
	return files, nil
}

func CompressFiles(files []*os.File) (bytes.Buffer, error) {
	// chained writers -> writing to tw writes to gw -> writes to buffer
	var b bytes.Buffer
	gw := gzip.NewWriter(&b)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	totalSize, err := FilesTotalSize(files)
	if err != nil {
		return bytes.Buffer{}, err
	}

	cores := runtime.NumCPU()
	blocks := 2 * cores
	blockSize := int(totalSize) / blocks
	gw.SetConcurrency(blockSize, blocks)

	for _, file := range files {
		err := addToTarArchive(tw, file.Name())
		if err != nil {
			return bytes.Buffer{}, err
		}
	}

	return b, nil
}

func FilesTotalSize(files []*os.File) (int64, error) {
	var totalSize int64
	for _, file := range files {
		info, err := file.Stat()
		if err != nil {
			return 0, fmt.Errorf("file info for '%s' could not be read", err)
		}
		totalSize += info.Size()
	}
	return totalSize, nil
}

// Credits to: https://www.arthurkoziel.com/writing-tar-gz-files-in-go/
func addToTarArchive(tw *tar.Writer, filename string) error {
	// Open the file which will be written into the archive
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get FileInfo about our file providing file size, mode, etc.
	info, err := file.Stat()
	if err != nil {
		return err
	}

	// Create a tar Header from the FileInfo data
	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}

	// Use full path as name (FileInfoHeader only takes the basename)
	// If we don't do this the directory strucuture would
	// not be preserved https://golang.org/src/archive/tar/common.go?#L626
	header.Name = filename

	// Write file header to the tar archive
	err = tw.WriteHeader(header)
	if err != nil {
		return err
	}

	// Copy file content to tar archive
	_, err = io.Copy(tw, file)
	if err != nil {
		return err
	}

	return nil
}
