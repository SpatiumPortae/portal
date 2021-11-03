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

// ArchiveAndCompressFiles tars and gzip-compresses files into a byte buffer
func ArchiveAndCompressFiles(files []*os.File) (*bytes.Buffer, error) {
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

// DecompressAndUnarchiveBytes gzip-decompresses and un-tars files into the current working directory
// and returns the names of the created files
func DecompressAndUnarchiveBytes(buffer *bytes.Buffer) ([]string, error) {
	// chained readers -> gr reads from buffer -> tr reades from gr
	gr, err := pgzip.NewReader(buffer)
	if err != nil {
		return nil, err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	var createdFiles []string
	for {
		header, err := tr.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if header == nil {
			continue
		}

		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}

		fileTarget := filepath.Join(cwd, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(fileTarget); err != nil {
				if err := os.MkdirAll(fileTarget, 0755); err != nil {
					return nil, err
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(fileTarget, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(f, tr); err != nil {
				return nil, err
			}
			createdFiles = append(createdFiles, header.Name)
			f.Close()
		}
	}

	return createdFiles, nil
}

// Traverses files and directories (recursively) for total size in bytes
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

// Credits to (legendary Mr. Nilsson): https://yourbasic.org/golang/formatting-byte-size-to-human-readable-format/
func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
