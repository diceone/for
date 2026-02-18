package vault

import "testing"

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	plaintext := "super-secret-password"
	password := "my-vault-password"

	enc, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if !IsEncrypted(enc) {
		t.Errorf("expected encrypted string to have prefix %q", Prefix)
	}

	dec, err := Decrypt(enc, password)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if dec != plaintext {
		t.Errorf("expected %q, got %q", plaintext, dec)
	}
}

func TestDecrypt_WrongPassword(t *testing.T) {
	enc, err := Encrypt("secret", "correct-password")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	_, err = Decrypt(enc, "wrong-password")
	if err == nil {
		t.Error("expected error when decrypting with wrong password")
	}
}

func TestDecrypt_PlainText_Passthrough(t *testing.T) {
	plain := "not-encrypted"
	result, err := Decrypt(plain, "any-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != plain {
		t.Errorf("expected %q, got %q", plain, result)
	}
}

func TestIsEncrypted(t *testing.T) {
	enc, _ := Encrypt("x", "pw")
	if !IsEncrypted(enc) {
		t.Error("expected IsEncrypted=true for encrypted string")
	}
	if IsEncrypted("plain") {
		t.Error("expected IsEncrypted=false for plain string")
	}
}

func TestEncrypt_DifferentNonce(t *testing.T) {
	// Two encryptions of the same plaintext must produce different ciphertexts (random nonce).
	e1, _ := Encrypt("hello", "pw")
	e2, _ := Encrypt("hello", "pw")
	if e1 == e2 {
		t.Error("expected different ciphertexts for each encryption (random nonce)")
	}
}

func TestDecryptMap(t *testing.T) {
	m := map[string]string{"key": "plain"}
	enc, _ := Encrypt("plain", "pw")
	m["enc"] = enc

	if err := DecryptMap(m, "pw"); err != nil {
		t.Fatalf("DecryptMap: %v", err)
	}
	if m["key"] != "plain" {
		t.Errorf("plain-text value changed: %q", m["key"])
	}
	if m["enc"] != "plain" {
		t.Errorf("expected decrypted value 'plain', got %q", m["enc"])
	}
}
