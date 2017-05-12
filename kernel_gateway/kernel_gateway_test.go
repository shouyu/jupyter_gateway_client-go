package kernel_gateway

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(client.Kernel.ID)
}
