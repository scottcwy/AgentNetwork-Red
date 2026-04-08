package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"

	"filippo.io/edwards25519"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/crypto/nacl/box"
)

const didKeyPrefix = "did:key:z"

func PublicKeyFromDID(did string) (ed25519.PublicKey, error) {
	if len(did) <= len(didKeyPrefix) || did[:len(didKeyPrefix)] != didKeyPrefix {
		return nil, fmt.Errorf("invalid did:key: %q", did)
	}

	decoded, err := base58BTCDecode(did[len(didKeyPrefix):])
	if err != nil {
		return nil, err
	}
	if len(decoded) != 34 || decoded[0] != 0xed || decoded[1] != 0x01 {
		return nil, fmt.Errorf("invalid did:key payload")
	}

	pub := make([]byte, ed25519.PublicKeySize)
	copy(pub, decoded[2:])
	return ed25519.PublicKey(pub), nil
}

func PeerIDFromDID(did string) (peer.ID, error) {
	pub, err := PublicKeyFromDID(did)
	if err != nil {
		return "", err
	}

	libp2pPub, err := libp2pcrypto.UnmarshalEd25519PublicKey(pub)
	if err != nil {
		return "", err
	}
	return peer.IDFromPublicKey(libp2pPub)
}

func EncryptPlaintext(senderPriv ed25519.PrivateKey, recipientDID string, plaintext string) (ciphertextB64 string, nonceB64 string, err error) {
	recipientPub, err := PublicKeyFromDID(recipientDID)
	if err != nil {
		return "", "", err
	}

	recipientCurve, err := curve25519PublicFromEd25519(recipientPub)
	if err != nil {
		return "", "", err
	}
	senderCurve := curve25519PrivateFromEd25519(senderPriv)

	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", "", err
	}

	sealed := box.Seal(nil, []byte(plaintext), &nonce, recipientCurve, senderCurve)
	return base64.StdEncoding.EncodeToString(sealed), base64.StdEncoding.EncodeToString(nonce[:]), nil
}

func DecryptCiphertext(recipientPriv ed25519.PrivateKey, senderDID, ciphertextB64, nonceB64 string) (string, error) {
	senderPub, err := PublicKeyFromDID(senderDID)
	if err != nil {
		return "", err
	}

	senderCurve, err := curve25519PublicFromEd25519(senderPub)
	if err != nil {
		return "", err
	}
	recipientCurve := curve25519PrivateFromEd25519(recipientPriv)

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}

	nonceBytes, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return "", fmt.Errorf("decode nonce: %w", err)
	}
	if len(nonceBytes) != 24 {
		return "", errors.New("invalid nonce length")
	}

	var nonce [24]byte
	copy(nonce[:], nonceBytes)

	opened, ok := box.Open(nil, ciphertext, &nonce, senderCurve, recipientCurve)
	if !ok {
		return "", errors.New("decrypt message: authentication failed")
	}
	return string(opened), nil
}

func curve25519PublicFromEd25519(pub ed25519.PublicKey) (*[32]byte, error) {
	point, err := new(edwards25519.Point).SetBytes(pub)
	if err != nil {
		return nil, err
	}

	montgomery := point.BytesMontgomery()
	var out [32]byte
	copy(out[:], montgomery)
	return &out, nil
}

func curve25519PrivateFromEd25519(priv ed25519.PrivateKey) *[32]byte {
	sum := sha512.Sum512(priv.Seed())

	var out [32]byte
	copy(out[:], sum[:32])
	out[0] &= 248
	out[31] &= 127
	out[31] |= 64
	return &out
}
