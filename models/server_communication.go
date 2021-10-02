package models

type SenderEstablishMessage struct {
	File        File `json:"file"`
	DesiredPort int  `json:"desiredPort"`
}

type ServerGeneratedPasswordMessage struct {
	Password Password `json:"password"`
}
