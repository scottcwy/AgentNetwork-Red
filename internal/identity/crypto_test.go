package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestDIDRoundTripAndDecrypt(t *testing.T) {
	_, senderPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_, recipientPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	recipientDID := DIDFromPublicKey(recipientPriv.Public().(ed25519.PublicKey))
	decodedPub, err := PublicKeyFromDID(recipientDID)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(decodedPub), string(recipientPriv.Public().(ed25519.PublicKey)); got != want {
		t.Fatalf("public key round-trip mismatch")
	}

	ciphertext, nonce, err := EncryptPlaintext(senderPriv, recipientDID, "hello dm")
	if err != nil {
		t.Fatal(err)
	}

	senderDID := DIDFromPublicKey(senderPriv.Public().(ed25519.PublicKey))
	plaintext, err := DecryptCiphertext(recipientPriv, senderDID, ciphertext, nonce)
	if err != nil {
		t.Fatal(err)
	}
	if plaintext != "hello dm" {
		t.Fatalf("unexpected plaintext: %q", plaintext)
	}
}
