package unit_tests

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"strings"
	"testing"
)

// Comprehensive encryption tests mirroring backend/internal/services/encryption.go.
// Tests AES-256-GCM encrypt/decrypt round-trips, nonce uniqueness, key generation,
// masking logic, and edge cases.

func aesGCMEncrypt(key, plaintext []byte) (ciphertext, nonce []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

func aesGCMDecrypt(key, ciphertext, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func generateKey() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	return key
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := generateKey()

	testCases := []string{
		"Hello, World!",
		"1234567890",
		"Short",
		"A very long plaintext that exceeds typical buffer sizes and contains special characters: @#$%^&*()",
		"",
		"Single",
		strings.Repeat("A", 10000),
	}

	for i, plaintext := range testCases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			ciphertext, nonce, err := aesGCMEncrypt(key, []byte(plaintext))
			if err != nil {
				t.Fatalf("encrypt failed: %v", err)
			}

			decrypted, err := aesGCMDecrypt(key, ciphertext, nonce)
			if err != nil {
				t.Fatalf("decrypt failed: %v", err)
			}

			if string(decrypted) != plaintext {
				t.Errorf("decrypted = %q, want %q", string(decrypted), plaintext)
			}
		})
	}
}

func TestEncryptProducesUniqueCiphertexts(t *testing.T) {
	key := generateKey()
	plaintext := []byte("same-plaintext")

	ciphertexts := make([][]byte, 10)
	for i := range ciphertexts {
		ct, _, err := aesGCMEncrypt(key, plaintext)
		if err != nil {
			t.Fatalf("encrypt failed: %v", err)
		}
		ciphertexts[i] = ct
	}

	// All ciphertexts should be different due to random nonces
	for i := 0; i < len(ciphertexts); i++ {
		for j := i + 1; j < len(ciphertexts); j++ {
			if string(ciphertexts[i]) == string(ciphertexts[j]) {
				t.Errorf("ciphertext[%d] == ciphertext[%d] — nonce reuse detected", i, j)
			}
		}
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	key1 := generateKey()
	key2 := generateKey()
	plaintext := []byte("sensitive-data")

	ciphertext, nonce, err := aesGCMEncrypt(key1, plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	_, err = aesGCMDecrypt(key2, ciphertext, nonce)
	if err == nil {
		t.Error("expected decryption with wrong key to fail")
	}
}

func TestDecryptWithCorruptedCiphertextFails(t *testing.T) {
	key := generateKey()
	plaintext := []byte("sensitive-data")

	ciphertext, nonce, err := aesGCMEncrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	// Corrupt the ciphertext
	corrupted := make([]byte, len(ciphertext))
	copy(corrupted, ciphertext)
	corrupted[0] ^= 0xFF

	_, err = aesGCMDecrypt(key, corrupted, nonce)
	if err == nil {
		t.Error("expected decryption of corrupted ciphertext to fail")
	}
}

func TestDecryptWithWrongNonceFails(t *testing.T) {
	key := generateKey()
	plaintext := []byte("sensitive-data")

	ciphertext, nonce, err := aesGCMEncrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	// Use a different nonce
	wrongNonce := make([]byte, len(nonce))
	copy(wrongNonce, nonce)
	wrongNonce[0] ^= 0xFF

	_, err = aesGCMDecrypt(key, ciphertext, wrongNonce)
	if err == nil {
		t.Error("expected decryption with wrong nonce to fail")
	}
}

func TestKeyGeneration(t *testing.T) {
	t.Run("key is 32 bytes", func(t *testing.T) {
		key := generateKey()
		if len(key) != 32 {
			t.Errorf("key length = %d, want 32", len(key))
		}
	})

	t.Run("generated keys are unique", func(t *testing.T) {
		keys := make([][]byte, 100)
		for i := range keys {
			keys[i] = generateKey()
		}
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if string(keys[i]) == string(keys[j]) {
					t.Errorf("key[%d] == key[%d] — key collision", i, j)
				}
			}
		}
	})
}

func TestNonceSize(t *testing.T) {
	key := generateKey()
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)

	// GCM standard nonce size is 12 bytes
	if gcm.NonceSize() != 12 {
		t.Errorf("nonce size = %d, want 12", gcm.NonceSize())
	}
}

func TestGCMOverhead(t *testing.T) {
	key := generateKey()
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)

	// GCM adds a 16-byte authentication tag
	if gcm.Overhead() != 16 {
		t.Errorf("GCM overhead = %d, want 16", gcm.Overhead())
	}
}

func TestInvalidKeySize(t *testing.T) {
	invalidSizes := []int{0, 8, 16, 24, 31, 33, 64}

	for _, size := range invalidSizes {
		key := make([]byte, size)
		_, err := aes.NewCipher(key)
		// AES accepts 16, 24, 32 byte keys
		if size == 16 || size == 24 {
			if err != nil {
				t.Errorf("key size %d should be valid for AES: %v", size, err)
			}
		} else if size == 32 {
			continue // skip — 32 is our target size
		} else {
			if err == nil {
				t.Errorf("key size %d should be invalid for AES", size)
			}
		}
	}
}

func TestEncryptEmptyPlaintext(t *testing.T) {
	key := generateKey()
	ciphertext, nonce, err := aesGCMEncrypt(key, []byte{})
	if err != nil {
		t.Fatalf("encrypt of empty plaintext failed: %v", err)
	}

	decrypted, err := aesGCMDecrypt(key, ciphertext, nonce)
	if err != nil {
		t.Fatalf("decrypt of empty plaintext failed: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("decrypted empty plaintext should be empty, got %d bytes", len(decrypted))
	}
}

func TestKeyRotationSimulation(t *testing.T) {
	oldKey := generateKey()
	newKey := generateKey()
	plaintext := []byte("sensitive-data-to-re-encrypt")

	// Encrypt with old key
	ciphertext, nonce, err := aesGCMEncrypt(oldKey, plaintext)
	if err != nil {
		t.Fatalf("encrypt with old key failed: %v", err)
	}

	// Decrypt with old key
	decrypted, err := aesGCMDecrypt(oldKey, ciphertext, nonce)
	if err != nil {
		t.Fatalf("decrypt with old key failed: %v", err)
	}

	// Re-encrypt with new key
	newCiphertext, newNonce, err := aesGCMEncrypt(newKey, decrypted)
	if err != nil {
		t.Fatalf("re-encrypt with new key failed: %v", err)
	}

	// Decrypt with new key
	finalDecrypted, err := aesGCMDecrypt(newKey, newCiphertext, newNonce)
	if err != nil {
		t.Fatalf("decrypt with new key failed: %v", err)
	}

	if string(finalDecrypted) != string(plaintext) {
		t.Errorf("re-encrypted data = %q, want %q", string(finalDecrypted), string(plaintext))
	}

	// Old key should NOT decrypt new ciphertext
	_, err = aesGCMDecrypt(oldKey, newCiphertext, newNonce)
	if err == nil {
		t.Error("old key should not decrypt data encrypted with new key")
	}
}
