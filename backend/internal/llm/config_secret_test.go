package llm

import "testing"

func TestConfigSecretCipherRoundTrip(t *testing.T) {
	cipher := NewConfigSecretCipher("test-server-secret")
	encrypted, err := cipher.Encrypt("gw_test_private")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "gw_test_private" {
		t.Fatal("credential must not remain plaintext")
	}
	plain, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "gw_test_private" {
		t.Fatalf("unexpected plaintext %q", plain)
	}
	if _, err := NewConfigSecretCipher("different-secret").Decrypt(encrypted); err == nil {
		t.Fatal("ciphertext must be bound to the configured server secret")
	}
}
