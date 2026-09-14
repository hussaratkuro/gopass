package totp

import (
	"testing"
	"time"
)

func TestRFC6238Vector(t *testing.T) {
	const secret = "otpauth://totp/Test?secret=GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ&digits=8&period=30"
	code, err := Code(secret, time.Unix(59, 0))
	if err != nil {
		t.Fatal(err)
	}
	if code != "94287082" {
		t.Fatalf("code = %q, want 94287082", code)
	}
}

func TestPlainSecretUsesSixDigits(t *testing.T) {
	code, err := Code("GEZD GNBV-GY3TQOJQGEZDGNBVGY3TQOJQ", time.Unix(59, 0))
	if err != nil {
		t.Fatal(err)
	}
	if code != "287082" || Remaining("", time.Unix(59, 0)) != 1 {
		t.Fatalf("code/remaining = %q/%d", code, Remaining("", time.Unix(59, 0)))
	}
}
