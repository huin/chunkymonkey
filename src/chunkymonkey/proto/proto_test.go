package proto

import (
	"os"
	"testing"
)

type NullWriter struct{}

func (w *NullWriter) Write(p []byte) (n int, err error) { return }

func TestWriteChatMessage(t *testing.T) {
	nullWriter := &NullWriter{}

	illegalChars := "This is a illegal char: ⁐"
	illegalColorTagMsg := "§1 This is a incorrect use of color tags §3"
	correctChars := "This is a proper chat message"
	correctColorTagMsg := "§1 This is a correct color usage in a message!"

	var err error

	err = WriteChatMessage(nullWriter, illegalChars)
	if err != nil {
		t.Errorf("%s", err)
	}

	err = WriteChatMessage(nullWriter, illegalColorTagMsg)
	if err == nil {
		t.Errorf("The test against message suffixes with illegal color tag failed.")
	}

	err = WriteChatMessage(nullWriter, correctChars)
	if err != nil {
		t.Errorf("correctChars shouldn't generate any errors: %s", err)
	}

	err = WriteChatMessage(nullWriter, correctColorTagMsg)
	if err != nil {
		t.Errorf("correctColorTagMsg shouldn't generate any errors: %s", err)
	}
}
