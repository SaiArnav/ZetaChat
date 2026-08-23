package core

import "testing"

func TestGateSendAllowsWhenUnset(t *testing.T) {
	SetSendGate(nil)
	if err := GateSend("u1", "hi"); err != nil {
		t.Fatalf("GateSend with nil gate = %v, want nil", err)
	}
}

func TestGateSendBlocksAndPasses(t *testing.T) {
	calls := 0
	SetSendGate(func(chatID, text string) error {
		calls++
		if text == "blocked" {
			return ErrSendDenied
		}
		return nil
	})
	defer SetSendGate(nil)

	if err := GateSend("u1", "blocked"); err != ErrSendDenied {
		t.Fatalf("GateSend = %v, want ErrSendDenied", err)
	}
	if err := GateSend("u1", "ok"); err != nil {
		t.Fatalf("GateSend(ok) = %v, want nil", err)
	}
	if calls != 2 {
		t.Fatalf("gate called %d times, want 2", calls)
	}
}
