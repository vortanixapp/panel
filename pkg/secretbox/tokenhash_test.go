package secretbox

import "testing"

func TestTokenHashIsDeterministic(t *testing.T) {
	const token = "agt_32f0a4fc9e1b4d7a8c0f"
	first := TokenHash(token)
	if first != TokenHash(token) {
		t.Fatal("отпечаток должен быть детерминированным")
	}
	if len(first) != 64 {
		t.Errorf("длина отпечатка %d, ожидалось 64 (sha256 в hex)", len(first))
	}
	if first == token {
		t.Error("отпечаток не должен совпадать с самим токеном")
	}
}

func TestTokenHashDiffersPerToken(t *testing.T) {
	if TokenHash("agt_aaa") == TokenHash("agt_bbb") {
		t.Fatal("разные токены дают разные отпечатки")
	}
}

func TestTokenHashTrimsWhitespace(t *testing.T) {
	want := TokenHash("agt_token")
	for _, v := range []string{" agt_token", "agt_token ", "\tagt_token\n"} {
		if TokenHash(v) != want {
			t.Errorf("отпечаток %q не совпал с нетронутым", v)
		}
	}
}

func TestTokenHashEmptyStaysEmpty(t *testing.T) {
	for _, v := range []string{"", "   ", "\n"} {
		if TokenHash(v) != "" {
			t.Errorf("пустое значение %q должно давать пустой отпечаток", v)
		}
	}
}

func TestEncryptIsNotSearchableButHashIs(t *testing.T) {
	box, err := New("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	const token = "agt_secret_value"
	a, err := box.Encrypt(token)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	b, err := box.Encrypt(token)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if a == b {
		t.Fatal("шифротекст обязан отличаться при каждом вызове (случайный nonce)")
	}
	if TokenHash(token) != TokenHash(token) {
		t.Fatal("отпечаток обязан совпадать — по нему идёт поиск")
	}
	if got := box.MustDecrypt(a); got != token {
		t.Errorf("расшифровка вернула %q вместо %q", got, token)
	}
}
