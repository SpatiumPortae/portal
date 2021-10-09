// Transfer.go specifies the required messaging for the transfer protocol.
package protocol

// TransferClientReadyMessage announces to the sender.Server that
// the receiver.Client is ready to receive the payload.
type TransferClientReadyMessage struct {
	Ready bool `json:"client_ready"`
}

// TransferClienReceivedMessage  announces to the sender.Server that
// the receiver.Client has received the payload.
type TransferClienReceivedMessage struct {
	Received bool `json:"client_receivied"`
}

// TransferClientClosingMessage announces to the sender.Server that
// the receiver.Client is ready to close the connection.
type TransferClientClosingMessage struct {
	Closning bool `json:"client_closing"`
}

// TransferServerReadyMessage announces to the receiver.Client that
// the sender.Server is ready to transfer the payload.
type TransferServerReadyMessage struct {
	Ready bool `json:"server_ready"`
}

// TransferServerClosingMessage announces to the receiver.Client that
// the sender.Server is closing the connection.
type TransferServerClosingMessage struct {
	Closing bool `json:"server_closing"`
}

// TransferServerErrorMessage announces a sender.Server error to the
// receiver.Client.
type TransferServerErrorMessage struct {
	Error error `json:"server_error"`
}
