package file

import (
	"archive/tar"
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/pgzip"
)

const SEND_TEMP_FILE_NAME_PREFIX = "portal-send-temp"
const RECEIVE_TEMP_FILE_NAME_PREFIX = "portal-receive-temp"

// ----------------------------------------------------- Pack Files ----------------------------------------------------

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

// PackFiles tars and gzip-compresses files into a temporary file, returning it
// along with the resulting size
func PackFiles(files []*os.File) (*os.File, int64, error) {
	// chained writers -> writing to tw writes to gw -> writes to temporary file
	tempFile, err := os.CreateTemp(os.TempDir(), SEND_TEMP_FILE_NAME_PREFIX)
	if err != nil {
		return nil, 0, err
	}
	tempFileWriter := bufio.NewWriter(tempFile)
	gw := pgzip.NewWriter(tempFileWriter)
	tw := tar.NewWriter(gw)

	for _, file := range files {
		err := addToTarArchive(tw, file)
		if err != nil {
			return nil, 0, err
		}
	}
	tw.Close()
	gw.Close()
	tempFileWriter.Flush()
	fileInfo, err := tempFile.Stat()
	if err != nil {
		return nil, 0, err
	}

	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		return nil, 0, err
	}
	return tempFile, fileInfo.Size(), nil
}

// ---------------------------------------------------- Unpack Files ---------------------------------------------------

var ErrUnpackNoHeader = errors.New("no header in tar archive")
var ErrUnpackFileExists = errors.New("file exists")
var ErrUninitialized = errors.New("unpacker is uninitialized")

// Unpacker defines an encapsulated unit for unpacking a compressed
// tar archive
type Unpacker struct {
	prompt bool // prompt defines whether we should prompt the user to overwrite files
	cwd    string

	gr *pgzip.Reader
	tr *tar.Reader
	r  io.ReadCloser
}

func NewUnpacker(prompt bool, r io.ReadCloser) (*Unpacker, error) {
	gr, err := pgzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gr)

	return &Unpacker{
		prompt: prompt,
		cwd:    cwd,
		gr:     gr,
		tr:     tr,
		r:      r,
	}, nil
}

// Close closes all underlying readers of the unpacker.
func (u *Unpacker) Close() error {
	if u.gr != nil {
		if err := u.gr.Close(); err != nil {
			return err
		}
	}
	if u.r != nil {
		if err := u.r.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Unpack will decompress and unpack the archive. Resolves a Committer
// which can be used to write file to disk. If the unpacker is configured to prompt
// it will return a ErrUnpackFileExists along with the committer. Returns a io.EOF
// once the archive has been fully consumed.
func (u *Unpacker) Unpack() (Committer, error) {
	if u.tr == nil {
		return nil, ErrUninitialized
	}
	header, err := u.tr.Next()
	switch {
	case err != nil:
		return nil, err
	case header == nil:
		return nil, ErrUnpackNoHeader
	}
	path := filepath.Join(u.cwd, header.Name)
	commiter := committer{
		cwd:    u.cwd,
		name:   header.Name,
		tr:     u.tr,
		header: header,
	}

	if u.prompt && header.Typeflag == tar.TypeReg && fileExists(path) {
		return &commiter, ErrUnpackFileExists
	}
	return &commiter, nil
}

// Committer defines a unit that can commit a file to disk
type Committer interface {
	FileName() string
	Commit() (int64, error)
}

type committer struct {
	cwd    string
	name   string
	tr     *tar.Reader
	header *tar.Header
}

func (c *committer) FileName() string {
	return c.name
}

func (c *committer) Commit() (int64, error) {
	path := filepath.Join(c.cwd, c.name)
	switch c.header.Typeflag {
	case tar.TypeDir:
		if _, err := os.Stat(path); err != nil {
			if err := os.MkdirAll(path, 0755); err != nil {
				return 0, err
			}
		}
		return 0, nil
	case tar.TypeReg:
		f, err := os.Create(path)
		if err != nil {
			return 0, err
		}
		defer f.Close()
		if _, err := io.Copy(f, c.tr); err != nil {
			return 0, err
		}
		info, err := f.Stat()
		if err != nil {
			return 0, err
		}
		return info.Size(), nil
	default:
		return 0, errors.New("unsupported file type")
	}
}

// ----------------------------------------------------- Utilities -----------------------------------------------------

// Traverses a file or directory recursively for total size in bytes.
func FileSize(filePath string) (int64, error) {
	var size int64
	err := filepath.Walk(filePath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		size += info.Size()
		return err
	})
	if err != nil {
		return 0, err
	}
	return size, nil
}

// optimistically remove files created by portal with the specified prefix
func RemoveTemporaryFiles(prefix string) {
	tempFiles, err := os.ReadDir(os.TempDir())
	if err != nil {
		return
	}
	for _, tempFile := range tempFiles {
		fileInfo, err := tempFile.Info()
		if err != nil {
			continue
		}
		fileName := fileInfo.Name()
		if strings.HasPrefix(fileName, prefix) {
			os.Remove(filepath.Join(os.TempDir(), fileName))
		}
	}
}

// ------------------------------------------------------- Helper ------------------------------------------------------

// addToTarArchive adds a file/folder to a tar archive.
// Handles symlinks by replacing them with the files that they point to.
func addToTarArchive(tw *tar.Writer, file *os.File) error {
	var absoluteBase string
	absPath, err := filepath.Abs(file.Name())
	if err != nil {
		return err
	}
	absoluteBase = filepath.Dir(absPath)

	return filepath.Walk(file.Name(), func(path string, fi os.FileInfo, err error) error {
		if (fi.Mode() & os.ModeSymlink) == os.ModeSymlink {
			// read path that the symlink is pointing to
			var link string
			if link, err = filepath.EvalSymlinks(path); err != nil {
				return err
			}

			// replace fileinfo with symlink pointee, essentially treating the symlink as the real file
			fi, err = os.Stat(link)
			if err != nil {
				return err
			}
		}

		// tar.FileInfoHeader handles path as pointee if path is a symlink
		header, e := tar.FileInfoHeader(fi, path)
		if e != nil {
			return err
		}

		// use absolute paths to handle both relative and absolute input paths identically
		targetPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		// remove the absolute root from the filename, leaving only the desired filename
		header.Name = filepath.ToSlash(strings.TrimPrefix(targetPath, absoluteBase))
		header.Name = strings.TrimPrefix(header.Name, string(os.PathSeparator))

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !fi.IsDir() {
			data, err := os.Open(path)
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

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
