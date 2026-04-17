package unit_tests

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/argon2"
)

// Tests for authentication helper functions.
// Production logic references:
//   - backend/internal/handlers/auth.go (verifyArgon2Hash, HashPasswordArgon2, constantTimeCompare)

func hashPasswordArgon2(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
	hash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, 64*1024, 3, 4,
		hex.EncodeToString(salt),
		hex.EncodeToString(key),
	)
	return hash, nil
}

func verifyArgon2Hash(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}
	var memory uint32
	var iterations uint32
	var parallelism uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return false
	}
	salt, err := hex.DecodeString(parts[4])
	if err != nil {
		salt = []byte(parts[4])
	}
	expectedHash, err := hex.DecodeString(parts[5])
	if err != nil {
		expectedHash = []byte(parts[5])
	}
	keyLen := uint32(len(expectedHash))
	if keyLen == 0 {
		keyLen = 32
	}
	derivedKey := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLen)
	return constantTimeCompare(derivedKey, expectedHash)
}

func constantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := range a {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

func TestHashAndVerifyRoundTrip(t *testing.T) {
	passwords := []string{
		"SecurePass123!",
		"MyP@ssw0rd!234",
		"a-very-long-password-that-exceeds-normal-length-requirements",
		"12char_pass!",
		"special!@#$%^&*()",
	}

	for _, pw := range passwords {
		t.Run(pw, func(t *testing.T) {
			hash, err := hashPasswordArgon2(pw)
			if err != nil {
				t.Fatalf("hashing failed: %v", err)
			}
			if !verifyArgon2Hash(pw, hash) {
				t.Error("password verification failed for correct password")
			}
		})
	}
}

func TestVerifyRejectsWrongPassword(t *testing.T) {
	hash, err := hashPasswordArgon2("CorrectPassword1")
	if err != nil {
		t.Fatalf("hashing failed: %v", err)
	}

	wrongPasswords := []string{
		"WrongPassword1!",
		"correctpassword1",
		"CorrectPassword",
		"CorrectPassword1!",
		"",
	}

	for _, pw := range wrongPasswords {
		if verifyArgon2Hash(pw, hash) {
			t.Errorf("wrong password %q should not verify", pw)
		}
	}
}

func TestHashFormat(t *testing.T) {
	hash, err := hashPasswordArgon2("TestPassword12")
	if err != nil {
		t.Fatalf("hashing failed: %v", err)
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Fatalf("hash should have 6 parts separated by $, got %d: %v", len(parts), parts)
	}

	t.Run("empty first part", func(t *testing.T) {
		if parts[0] != "" {
			t.Errorf("first part should be empty, got %q", parts[0])
		}
	})

	t.Run("algorithm is argon2id", func(t *testing.T) {
		if parts[1] != "argon2id" {
			t.Errorf("algorithm should be argon2id, got %q", parts[1])
		}
	})

	t.Run("version is v=19", func(t *testing.T) {
		if parts[2] != "v=19" {
			t.Errorf("version should be v=19, got %q", parts[2])
		}
	})

	t.Run("parameters are correct", func(t *testing.T) {
		expected := "m=65536,t=3,p=4"
		if parts[3] != expected {
			t.Errorf("parameters = %q, want %q", parts[3], expected)
		}
	})

	t.Run("salt is hex-encoded 16 bytes", func(t *testing.T) {
		salt, err := hex.DecodeString(parts[4])
		if err != nil {
			t.Fatalf("salt is not valid hex: %v", err)
		}
		if len(salt) != 16 {
			t.Errorf("salt length = %d bytes, want 16", len(salt))
		}
	})

	t.Run("hash is hex-encoded 32 bytes", func(t *testing.T) {
		hashBytes, err := hex.DecodeString(parts[5])
		if err != nil {
			t.Fatalf("hash is not valid hex: %v", err)
		}
		if len(hashBytes) != 32 {
			t.Errorf("hash length = %d bytes, want 32", len(hashBytes))
		}
	})
}

func TestHashUniqueness(t *testing.T) {
	// Same password should produce different hashes due to random salt
	hashes := make([]string, 10)
	for i := range hashes {
		hash, err := hashPasswordArgon2("SamePassword12")
		if err != nil {
			t.Fatalf("hashing failed: %v", err)
		}
		hashes[i] = hash
	}

	for i := 0; i < len(hashes); i++ {
		for j := i + 1; j < len(hashes); j++ {
			if hashes[i] == hashes[j] {
				t.Errorf("hash[%d] == hash[%d] — salt should make hashes unique", i, j)
			}
		}
	}
}

func TestConstantTimeCompare(t *testing.T) {
	t.Run("equal slices", func(t *testing.T) {
		a := []byte{1, 2, 3, 4}
		b := []byte{1, 2, 3, 4}
		if !constantTimeCompare(a, b) {
			t.Error("equal slices should compare as equal")
		}
	})

	t.Run("unequal slices", func(t *testing.T) {
		a := []byte{1, 2, 3, 4}
		b := []byte{1, 2, 3, 5}
		if constantTimeCompare(a, b) {
			t.Error("unequal slices should compare as unequal")
		}
	})

	t.Run("different lengths", func(t *testing.T) {
		a := []byte{1, 2, 3}
		b := []byte{1, 2, 3, 4}
		if constantTimeCompare(a, b) {
			t.Error("different length slices should compare as unequal")
		}
	})

	t.Run("empty slices", func(t *testing.T) {
		if !constantTimeCompare([]byte{}, []byte{}) {
			t.Error("two empty slices should compare as equal")
		}
	})

	t.Run("single byte difference", func(t *testing.T) {
		a := []byte{0xFF}
		b := []byte{0xFE}
		if constantTimeCompare(a, b) {
			t.Error("single byte difference should be detected")
		}
	})
}

func TestVerifyRejectsInvalidHashFormats(t *testing.T) {
	invalidHashes := []string{
		"",
		"not-a-hash",
		"$argon2id$v=19$m=65536,t=3,p=4",                  // too few parts
		"$argon2id$v=19$m=65536,t=3,p=4$salt$hash$extra$x", // too many parts
		"$argon2id$v=19$invalid-params$salt$hash",           // bad params
	}

	for _, hash := range invalidHashes {
		if verifyArgon2Hash("password1234", hash) {
			t.Errorf("invalid hash %q should not verify", hash)
		}
	}
}

func TestSessionTokenGeneration(t *testing.T) {
	// Mirrors the token generation in Login handler
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	token := hex.EncodeToString(tokenBytes)

	t.Run("token is 64 hex chars", func(t *testing.T) {
		if len(token) != 64 {
			t.Errorf("token length = %d, want 64", len(token))
		}
	})

	t.Run("token is valid hex", func(t *testing.T) {
		_, err := hex.DecodeString(token)
		if err != nil {
			t.Errorf("token is not valid hex: %v", err)
		}
	})

	t.Run("tokens are unique", func(t *testing.T) {
		tokens := make(map[string]bool)
		for i := 0; i < 100; i++ {
			b := make([]byte, 32)
			rand.Read(b)
			tok := hex.EncodeToString(b)
			if tokens[tok] {
				t.Error("duplicate token generated")
			}
			tokens[tok] = true
		}
	})
}
