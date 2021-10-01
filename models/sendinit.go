package models

type File struct {
	FileName string `json:"fileName"`
	Bytes int64 `json:"bytes"`
}