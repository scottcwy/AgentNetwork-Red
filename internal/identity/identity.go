package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const privateKeyFile = "identity.key"

type Identity struct {
	DID        string
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

func LoadOrCreate(dataDir string) (*Identity, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}

	path := filepath.Join(dataDir, privateKeyFile)
	priv, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}

		_, generated, genErr := ed25519.GenerateKey(rand.Reader)
		if genErr != nil {
			return nil, genErr
		}
		if writeErr := os.WriteFile(path, generated, 0o600); writeErr != nil {
			return nil, writeErr
		}
		priv = generated
	}

	if len(priv) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid ed25519 private key length: %d", len(priv))
	}

	privateKey := ed25519.PrivateKey(priv)
	publicKey := privateKey.Public().(ed25519.PublicKey)

	return &Identity{
		DID:        DIDFromPublicKey(publicKey),
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

func KeyPath(dataDir string) string {
	return filepath.Join(dataDir, privateKeyFile)
}

func DIDFromPublicKey(pub ed25519.PublicKey) string {
	payload := make([]byte, 0, len(pub)+2)
	payload = append(payload, 0xed, 0x01)
	payload = append(payload, pub...)
	return "did:key:z" + base58BTCEncode(payload)
}

func base58BTCEncode(input []byte) string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	if len(input) == 0 {
		return ""
	}

	digits := []byte{0}
	for _, b := range input {
		carry := int(b)
		for i := 0; i < len(digits); i++ {
			carry += int(digits[i]) << 8
			digits[i] = byte(carry % 58)
			carry /= 58
		}
		for carry > 0 {
			digits = append(digits, byte(carry%58))
			carry /= 58
		}
	}

	zeros := 0
	for zeros < len(input) && input[zeros] == 0 {
		zeros++
	}

	encoded := make([]byte, 0, zeros+len(digits))
	for i := 0; i < zeros; i++ {
		encoded = append(encoded, alphabet[0])
	}
	for i := len(digits) - 1; i >= 0; i-- {
		encoded = append(encoded, alphabet[digits[i]])
	}
	return string(encoded)
}

func base58BTCDecode(input string) ([]byte, error) {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	index := make(map[rune]int, len(alphabet))
	for i, ch := range alphabet {
		index[ch] = i
	}

	bytes := []byte{0}
	for _, ch := range input {
		value, ok := index[ch]
		if !ok {
			return nil, fmt.Errorf("invalid base58 character %q", ch)
		}

		carry := value
		for i := 0; i < len(bytes); i++ {
			carry += int(bytes[i]) * 58
			bytes[i] = byte(carry & 0xff)
			carry >>= 8
		}
		for carry > 0 {
			bytes = append(bytes, byte(carry&0xff))
			carry >>= 8
		}
	}

	zeros := 0
	for zeros < len(input) && input[zeros] == alphabet[0] {
		zeros++
	}

	decoded := make([]byte, 0, zeros+len(bytes))
	for i := 0; i < zeros; i++ {
		decoded = append(decoded, 0)
	}
	for i := len(bytes) - 1; i >= 0; i-- {
		decoded = append(decoded, bytes[i])
	}
	return decoded, nil
}
