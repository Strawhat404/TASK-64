package services

import (
	"bytes"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	svc := &EncryptionService{masterKey: []byte("CHANGE_ME_32_BYTE_DEV_KEY_00000!")}

	plaintext := []byte("sensitive-data-123")
	ciphertext, nonce, err := svc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext should not equal plaintext")
	}

	decrypted, err := svc.Decrypt(ciphertext, nonce)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptWithKeyDecryptWithKey_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	copy(key, []byte("test-dek-key-for-encryption-test"))

	plaintext := []byte("bank-account-number-1234567890")
	ciphertext, nonce, err := encryptWithKey(key, plaintext)
	if err != nil {
		t.Fatalf("encryptWithKey failed: %v", err)
	}

	decrypted, err := decryptWithKey(key, ciphertext, nonce)
	if err != nil {
		t.Fatalf("decryptWithKey failed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptWithWrongKey_Fails(t *testing.T) {
	key1 := make([]byte, 32)
	copy(key1, []byte("key-one-for-encrypt-test-1234567"))
	key2 := make([]byte, 32)
	copy(key2, []byte("key-two-for-decrypt-test-7654321"))

	plaintext := []byte("secret")
	ciphertext, nonce, err := encryptWithKey(key1, plaintext)
	if err != nil {
		t.Fatalf("encryptWithKey failed: %v", err)
	}

	_, err = decryptWithKey(key2, ciphertext, nonce)
	if err == nil {
		t.Fatal("expected decryption to fail with wrong key, but it succeeded")
	}
}

func TestEncryptProducesUniqueNonces(t *testing.T) {
	svc := &EncryptionService{masterKey: []byte("CHANGE_ME_32_BYTE_DEV_KEY_00000!")}

	plaintext := []byte("same-input")
	_, nonce1, _ := svc.Encrypt(plaintext)
	_, nonce2, _ := svc.Encrypt(plaintext)

	if bytes.Equal(nonce1, nonce2) {
		t.Error("two encryptions should produce different nonces")
	}
}

func TestEncryptProducesUniqueCiphertext(t *testing.T) {
	svc := &EncryptionService{masterKey: []byte("CHANGE_ME_32_BYTE_DEV_KEY_00000!")}

	plaintext := []byte("same-input")
	ct1, _, _ := svc.Encrypt(plaintext)
	ct2, _, _ := svc.Encrypt(plaintext)

	if bytes.Equal(ct1, ct2) {
		t.Error("two encryptions of the same plaintext should produce different ciphertext (GCM with random nonce)")
	}
}

func TestMaskValue(t *testing.T) {
	svc := &EncryptionService{}

	cases := []struct {
		input string
		want  string
	}{
		{"1234567890", "****7890"},
		{"abc", "****"},
		{"", "****"},
		{"12345", "****2345"},
	}

	for _, tc := range cases {
		got := svc.MaskValue(tc.input)
		if got != tc.want {
			t.Errorf("MaskValue(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestGenerateKey_Is32Bytes(t *testing.T) {
	svc := &EncryptionService{}
	key, err := svc.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("key length = %d, want 32", len(key))
	}
}

func TestGenerateKey_IsUnique(t *testing.T) {
	svc := &EncryptionService{}
	key1, _ := svc.GenerateKey()
	key2, _ := svc.GenerateKey()
	if bytes.Equal(key1, key2) {
		t.Error("two generated keys should be different")
	}
}

func TestDEKEncryptedWithDifferentKeyThanMasterKey(t *testing.T) {
	masterKey := []byte("CHANGE_ME_32_BYTE_DEV_KEY_00000!")
	svc := &EncryptionService{masterKey: masterKey}

	dek, err := svc.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// DEK should not equal master key
	if bytes.Equal(dek, masterKey) {
		t.Error("generated DEK should not equal master key")
	}

	// Data encrypted with DEK should not be decryptable with master key
	plaintext := []byte("test-data")
	ct, nonce, err := encryptWithKey(dek, plaintext)
	if err != nil {
		t.Fatalf("encryptWithKey(dek) failed: %v", err)
	}

	_, err = svc.Decrypt(ct, nonce) // Try with master key
	if err == nil {
		t.Error("data encrypted with DEK should not be decryptable with master key")
	}
}

// ============================================================================
// Envelope encryption / key rotation integration tests
// ============================================================================

// TestEnvelopeEncryption_DEKWrappedByMasterKey validates the full envelope encryption
// workflow: generate DEK, wrap it with master key, unwrap, use DEK to encrypt/decrypt data.
func TestEnvelopeEncryption_DEKWrappedByMasterKey(t *testing.T) {
	masterKey := []byte("CHANGE_ME_32_BYTE_DEV_KEY_00000!")
	svc := &EncryptionService{masterKey: masterKey}

	// Step 1: Generate a DEK
	dek, err := svc.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// Step 2: Wrap (encrypt) the DEK with the master key
	wrappedDEK, dekNonce, err := svc.Encrypt(dek)
	if err != nil {
		t.Fatalf("Failed to wrap DEK: %v", err)
	}

	// Step 3: Unwrap (decrypt) the DEK with the master key
	unwrappedDEK, err := svc.Decrypt(wrappedDEK, dekNonce)
	if err != nil {
		t.Fatalf("Failed to unwrap DEK: %v", err)
	}
	if !bytes.Equal(unwrappedDEK, dek) {
		t.Fatal("unwrapped DEK should equal original DEK")
	}

	// Step 4: Use the DEK to encrypt data
	plaintext := []byte("SSN-123-45-6789")
	dataCT, dataNonce, err := encryptWithKey(unwrappedDEK, plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt data with DEK: %v", err)
	}

	// Step 5: Use the DEK to decrypt data
	decrypted, err := decryptWithKey(unwrappedDEK, dataCT, dataNonce)
	if err != nil {
		t.Fatalf("Failed to decrypt data with DEK: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

// TestEnvelopeEncryption_DEKNonceMustBePreserved validates that discarding the
// DEK nonce makes the wrapped DEK unrecoverable.
func TestEnvelopeEncryption_DEKNonceMustBePreserved(t *testing.T) {
	masterKey := []byte("CHANGE_ME_32_BYTE_DEV_KEY_00000!")
	svc := &EncryptionService{masterKey: masterKey}

	dek, _ := svc.GenerateKey()
	wrappedDEK, _, err := svc.Encrypt(dek) // nonce is intentionally ignored
	if err != nil {
		t.Fatalf("Failed to wrap DEK: %v", err)
	}

	// Try to decrypt with a wrong nonce — should fail
	wrongNonce := make([]byte, 12) // GCM nonce is 12 bytes, all zeros
	_, err = svc.Decrypt(wrappedDEK, wrongNonce)
	if err == nil {
		t.Fatal("expected decryption to fail with wrong nonce, but it succeeded — nonce must be stored")
	}
}

// TestKeyRotation_ReEncryptProducesNewCiphertext simulates the rotation re-encryption
// path: data encrypted with old DEK is decrypted then re-encrypted with new DEK.
func TestKeyRotation_ReEncryptProducesNewCiphertext(t *testing.T) {
	masterKey := []byte("CHANGE_ME_32_BYTE_DEV_KEY_00000!")
	svc := &EncryptionService{masterKey: masterKey}

	// Generate old and new DEKs
	oldDEK, _ := svc.GenerateKey()
	newDEK, _ := svc.GenerateKey()

	if bytes.Equal(oldDEK, newDEK) {
		t.Fatal("old and new DEKs should be different")
	}

	// Encrypt with old DEK
	plaintext := []byte("bank-account-9876543210")
	oldCT, oldNonce, err := encryptWithKey(oldDEK, plaintext)
	if err != nil {
		t.Fatalf("encrypt with old DEK failed: %v", err)
	}

	// Simulate rotation: decrypt with old DEK
	decrypted, err := decryptWithKey(oldDEK, oldCT, oldNonce)
	if err != nil {
		t.Fatalf("decrypt with old DEK failed: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatal("decrypted data should match original")
	}

	// Re-encrypt with new DEK
	newCT, newNonce, err := encryptWithKey(newDEK, decrypted)
	if err != nil {
		t.Fatalf("encrypt with new DEK failed: %v", err)
	}

	// Verify new ciphertext is different from old
	if bytes.Equal(oldCT, newCT) {
		t.Error("re-encrypted ciphertext should differ from original")
	}

	// Verify new DEK can decrypt the rotated data
	finalDecrypted, err := decryptWithKey(newDEK, newCT, newNonce)
	if err != nil {
		t.Fatalf("decrypt with new DEK failed: %v", err)
	}
	if !bytes.Equal(finalDecrypted, plaintext) {
		t.Errorf("final decrypted = %q, want %q", finalDecrypted, plaintext)
	}

	// Verify old DEK cannot decrypt the rotated data
	_, err = decryptWithKey(oldDEK, newCT, newNonce)
	if err == nil {
		t.Error("old DEK should not be able to decrypt data re-encrypted with new DEK")
	}
}

// TestKeyRotation_OldDEKCannotDecryptRotatedData ensures post-rotation data isolation.
func TestKeyRotation_OldDEKCannotDecryptRotatedData(t *testing.T) {
	svc := &EncryptionService{masterKey: []byte("CHANGE_ME_32_BYTE_DEV_KEY_00000!")}

	oldDEK, _ := svc.GenerateKey()
	newDEK, _ := svc.GenerateKey()

	plaintext := []byte("routing-number-021000021")

	// Encrypt with old, then re-encrypt with new (simulating rotation)
	ct1, nonce1, _ := encryptWithKey(oldDEK, plaintext)
	decrypted, _ := decryptWithKey(oldDEK, ct1, nonce1)
	ct2, nonce2, _ := encryptWithKey(newDEK, decrypted)

	// Old key must not decrypt new ciphertext
	_, err := decryptWithKey(oldDEK, ct2, nonce2)
	if err == nil {
		t.Fatal("old DEK must not decrypt data encrypted with new DEK after rotation")
	}

	// New key must decrypt new ciphertext
	result, err := decryptWithKey(newDEK, ct2, nonce2)
	if err != nil {
		t.Fatalf("new DEK should decrypt rotated data: %v", err)
	}
	if !bytes.Equal(result, plaintext) {
		t.Errorf("result = %q, want %q", result, plaintext)
	}
}

// TestEnvelopeEncryption_MasterKeyRotationIsolation ensures that data encrypted
// under one master key's DEK hierarchy is not accessible under a different master key.
func TestEnvelopeEncryption_MasterKeyRotationIsolation(t *testing.T) {
	mk1 := []byte("master-key-one-32-bytes-long-ok!")
	mk2 := []byte("master-key-two-32-bytes-long-ok!")
	svc1 := &EncryptionService{masterKey: mk1}
	svc2 := &EncryptionService{masterKey: mk2}

	// Wrap a DEK under master key 1
	dek, _ := svc1.GenerateKey()
	wrappedDEK, dekNonce, _ := svc1.Encrypt(dek)

	// Master key 2 should not be able to unwrap it
	_, err := svc2.Decrypt(wrappedDEK, dekNonce)
	if err == nil {
		t.Fatal("different master key should not be able to unwrap DEK")
	}

	// Master key 1 should still be able to unwrap it
	unwrapped, err := svc1.Decrypt(wrappedDEK, dekNonce)
	if err != nil {
		t.Fatalf("original master key should unwrap DEK: %v", err)
	}
	if !bytes.Equal(unwrapped, dek) {
		t.Error("unwrapped DEK should match original")
	}
}

// TestMultipleDataRecords_SameKeyDifferentNonces validates that multiple records
// encrypted under the same DEK produce unique ciphertexts.
func TestMultipleDataRecords_SameKeyDifferentNonces(t *testing.T) {
	svc := &EncryptionService{masterKey: []byte("CHANGE_ME_32_BYTE_DEV_KEY_00000!")}
	dek, _ := svc.GenerateKey()

	plaintext := []byte("same-sensitive-value")
	ct1, nonce1, _ := encryptWithKey(dek, plaintext)
	ct2, nonce2, _ := encryptWithKey(dek, plaintext)

	if bytes.Equal(nonce1, nonce2) {
		t.Error("same-key encryptions must produce different nonces")
	}
	if bytes.Equal(ct1, ct2) {
		t.Error("same-key encryptions must produce different ciphertexts")
	}

	// Both should decrypt to the same plaintext
	d1, _ := decryptWithKey(dek, ct1, nonce1)
	d2, _ := decryptWithKey(dek, ct2, nonce2)
	if !bytes.Equal(d1, plaintext) || !bytes.Equal(d2, plaintext) {
		t.Error("both ciphertexts should decrypt to the original plaintext")
	}
}
