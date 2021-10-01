package models

type ReceiveRequest struct {
	FileName string `json:"fileName"`
	Bytes int64 `json:"bytes"`
}