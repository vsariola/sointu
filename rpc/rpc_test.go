package rpc_test

import (
	"testing"

	"github.com/vsariola/sointu/rpc"
)

func TestSendReceive(t *testing.T) {
	receiver, err := rpc.Receiver()
	if err != nil {
		t.Fatalf("rpc.Receiver error: %v", err)
	}
	sender, err := rpc.Sender("127.0.0.1")
	if err != nil {
		t.Fatalf("rpc.Sender error: %v", err)
	}
	value := []float32{42}
	sender <- value
	valueGot := <-receiver
	if valueGot[0] != value[0] {
		t.Fatalf("rpc.Sender error: %v", err)
	}
}
