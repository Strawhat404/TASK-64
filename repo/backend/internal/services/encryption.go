package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"compliance-console/internal/models"

	"github.com/google/uuid"
)

// devMasterKey is a placeholder key used only in development when MASTER_KEY is not set.
// It is exactly 32 bytes for AES-256.
var devMasterKey = []byte("CHANGE_ME_32_BYTE_DEV_KEY_00000!")

// EncryptionService provides AES-256-GCM encryption and key management.
type EncryptionService struct {
	masterKey []byte
}

// NewEncryptionService creates a new EncryptionService.
// It tries the local keyring file first, then the MASTER_KEY environment variable (hex-encoded),
// and finally falls back to a development default with a warning.
func NewEncryptionService() *EncryptionService {
	// Try local keyring file first
	key := loadKeyFromKeyring()
	if key != nil {
		return &EncryptionService{masterKey: key}
	}
	// Fall back to environment variable
	envKey := os.Getenv("MASTER_KEY")
	if envKey != "" {
		decoded, err := hex.DecodeString(envKey)
		if err == nil && len(decoded) == 32 {
			return &EncryptionService{masterKey: decoded}
		}
		// Also accept raw 32-byte keys for backward compatibility
		if len(envKey) == 32 {
			return &EncryptionService{masterKey: []byte(envKey)}
		}
		log.Println("WARNING: MASTER_KEY is not valid hex-encoded 32-byte key or raw 32 bytes. Using dev-mode default.")
	}
	// Development fallback — reject in production
	env := os.Getenv("APP_ENV")
	if env == "production" || env == "prod" {
		log.Fatal("FATAL: No master key configured. Set MASTER_KEY env var or use keyring for production. Refusing to start with dev key.")
	}
	log.Println("WARNING: Using development master key. Set MASTER_KEY env var or use keyring for production.")
	devKey := make([]byte, 32)
	copy(devKey, []byte("dev-master-key-not-for-production"))
	return &EncryptionService{masterKey: devKey}
}

// loadKeyFromKeyring attempts to load a hex-encoded 32-byte master key from local keyring files.
func loadKeyFromKeyring() []byte {
	paths := []string{".keyring", os.Getenv("HOME") + "/.compliance-console-keyring"}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		trimmed := strings.TrimSpace(string(data))
		decoded, err := hex.DecodeString(trimmed)
		if err == nil && len(decoded) == 32 {
			log.Printf("Loaded master key from keyring file: %s", path)
			return decoded
		}
	}
	return nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns ciphertext and nonce.
func (s *EncryptionService) Encrypt(plaintext []byte) (ciphertext []byte, nonce []byte, err error) {
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce = make([]byte, gcm.NonceSize()) // 12 bytes for GCM
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with the provided nonce.
func (s *EncryptionService) Decrypt(ciphertext, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// decryptDEK retrieves and decrypts the data-encryption key (DEK) for a given alias and tenant.
// Returns the raw 32-byte DEK or an error.
func (s *EncryptionService) decryptDEK(db *sql.DB, tenantID uuid.UUID, keyAlias string) ([]byte, error) {
	var encryptedKey, nonce []byte
	err := db.QueryRow(`
		SELECT encrypted_key, nonce FROM encryption_keys
		WHERE key_alias = $1 AND tenant_id = $2 AND status = 'active'
	`, keyAlias, tenantID).Scan(&encryptedKey, &nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve DEK for alias %s: %w", keyAlias, err)
	}

	// If nonce is nil, the key was bootstrapped before envelope encryption was added.
	// Cannot decrypt the DEK without its nonce — caller should fall back to master key.
	if nonce == nil {
		return nil, fmt.Errorf("DEK for alias %s has no nonce (legacy key without envelope encryption)", keyAlias)
	}

	dek, err := s.Decrypt(encryptedKey, nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt DEK for alias %s: %w", keyAlias, err)
	}
	return dek, nil
}

// encryptWithKey encrypts plaintext using a specific key (not the master key).
func encryptWithKey(key, plaintext []byte) (ciphertext []byte, nonce []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// decryptWithKey decrypts ciphertext using a specific key (not the master key).
func decryptWithKey(key, ciphertext, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
}

// DecryptValue decrypts ciphertext using the DEK identified by keyAlias (envelope decryption).
// Falls back to master key if no DEK is registered for backward compatibility.
func (s *EncryptionService) DecryptValue(db *sql.DB, tenantID uuid.UUID, keyAlias string, ciphertext, nonce []byte) ([]byte, error) {
	dek, dekErr := s.decryptDEK(db, tenantID, keyAlias)
	if dekErr != nil {
		// Fallback to master key for backward compatibility
		return s.Decrypt(ciphertext, nonce)
	}
	return decryptWithKey(dek, ciphertext, nonce)
}

// StoreSensitiveField encrypts and stores a sensitive value in the sensitive_data table.
// Uses envelope encryption: retrieves the DEK by alias, encrypts data with the DEK.
// Falls back to master key if no DEK is registered (backward compatibility).
func (s *EncryptionService) StoreSensitiveField(db *sql.DB, tenantID, ownerID uuid.UUID, dataType, value, label, keyAlias string) error {
	var ciphertext, nonce []byte
	var err error

	dek, dekErr := s.decryptDEK(db, tenantID, keyAlias)
	if dekErr != nil {
		// Fallback to master key for backward compatibility (no DEK registered yet)
		ciphertext, nonce, err = s.Encrypt([]byte(value))
	} else {
		ciphertext, nonce, err = encryptWithKey(dek, []byte(value))
	}
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO sensitive_data (tenant_id, owner_id, data_type, encrypted_value, nonce, key_alias, label)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tenantID, ownerID, dataType, ciphertext, nonce, keyAlias, label)
	if err != nil {
		return fmt.Errorf("failed to store sensitive data: %w", err)
	}

	return nil
}

// RetrieveSensitiveField decrypts and returns the plaintext value for a sensitive_data record.
// Uses envelope decryption: retrieves the DEK by alias, decrypts data with the DEK.
// Falls back to master key if no DEK is registered (backward compatibility).
func (s *EncryptionService) RetrieveSensitiveField(db *sql.DB, id uuid.UUID, tenantID uuid.UUID) (string, error) {
	var encryptedValue, nonce []byte
	var keyAlias string
	err := db.QueryRow(`
		SELECT encrypted_value, nonce, key_alias FROM sensitive_data WHERE id = $1 AND tenant_id = $2
	`, id, tenantID).Scan(&encryptedValue, &nonce, &keyAlias)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve sensitive data: %w", err)
	}

	// Try envelope decryption with DEK first
	dek, dekErr := s.decryptDEK(db, tenantID, keyAlias)
	var plaintext []byte
	if dekErr != nil {
		// Fallback: try master key directly (data encrypted before envelope encryption was enabled)
		plaintext, err = s.Decrypt(encryptedValue, nonce)
	} else {
		plaintext, err = decryptWithKey(dek, encryptedValue, nonce)
	}
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintext), nil
}

// MaskValue returns a masked version of the value showing only the last 4 characters.
// For values shorter than 4 characters, the entire value is masked.
func (s *EncryptionService) MaskValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return "****" + value[len(value)-4:]
}

// RotateKey creates a new data-encryption key (DEK) and re-encrypts all sensitive_data rows
// that use the old key alias with the new DEK. The DEK is itself encrypted with the master key
// (envelope encryption) and stored alongside its nonce.
func (s *EncryptionService) RotateKey(db *sql.DB, tenantID uuid.UUID, oldAlias, newAlias string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate a new DEK
	newDEK, err := s.GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}

	// Encrypt the new DEK with the master key for storage (envelope encryption)
	encNewKey, encNonce, err := s.Encrypt(newDEK)
	if err != nil {
		return fmt.Errorf("failed to encrypt new key: %w", err)
	}

	// Retrieve and decrypt the old DEK to re-encrypt data
	var oldEncKey, oldNonce []byte
	err = tx.QueryRow(`
		SELECT encrypted_key, nonce FROM encryption_keys WHERE key_alias = $1 AND tenant_id = $2
	`, oldAlias, tenantID).Scan(&oldEncKey, &oldNonce)
	if err != nil {
		return fmt.Errorf("failed to retrieve old key: %w", err)
	}

	var oldDEK []byte
	if oldNonce != nil {
		// Old key was stored with envelope encryption — decrypt with master key
		oldDEK, err = s.Decrypt(oldEncKey, oldNonce)
		if err != nil {
			return fmt.Errorf("failed to decrypt old DEK: %w", err)
		}
	}

	// Get the old key's rotation number
	var rotationNumber int
	err = tx.QueryRow(`SELECT rotation_number FROM encryption_keys WHERE key_alias = $1 AND tenant_id = $2`, oldAlias, tenantID).Scan(&rotationNumber)
	if err != nil {
		return fmt.Errorf("failed to get old key rotation number: %w", err)
	}

	// Insert new key entry with both encrypted key and its nonce
	_, err = tx.Exec(`
		INSERT INTO encryption_keys (tenant_id, key_alias, encrypted_key, nonce, algorithm, status, rotation_number, activated_at)
		VALUES ($1, $2, $3, $4, 'AES-256-GCM', 'active', $5, NOW())
	`, tenantID, newAlias, encNewKey, encNonce, rotationNumber+1)
	if err != nil {
		return fmt.Errorf("failed to insert new key: %w", err)
	}

	// Re-encrypt all sensitive_data rows using the old alias for this tenant
	rows, err := tx.Query(`
		SELECT id, encrypted_value, nonce FROM sensitive_data WHERE key_alias = $1 AND tenant_id = $2
	`, oldAlias, tenantID)
	if err != nil {
		return fmt.Errorf("failed to query sensitive data for re-encryption: %w", err)
	}
	defer rows.Close()

	type reEncryptRow struct {
		id         uuid.UUID
		ciphertext []byte
		nonce      []byte
	}

	var toUpdate []reEncryptRow
	for rows.Next() {
		var r reEncryptRow
		if err := rows.Scan(&r.id, &r.ciphertext, &r.nonce); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}
		toUpdate = append(toUpdate, r)
	}
	rows.Close()

	for _, r := range toUpdate {
		// Decrypt with old DEK (or master key for legacy data)
		var plaintext []byte
		if oldDEK != nil {
			plaintext, err = decryptWithKey(oldDEK, r.ciphertext, r.nonce)
		} else {
			plaintext, err = s.Decrypt(r.ciphertext, r.nonce)
		}
		if err != nil {
			return fmt.Errorf("failed to decrypt row %s: %w", r.id, err)
		}

		// Re-encrypt with the new DEK
		newCiphertext, newNonce, err := encryptWithKey(newDEK, plaintext)
		if err != nil {
			return fmt.Errorf("failed to re-encrypt row %s: %w", r.id, err)
		}

		_, err = tx.Exec(`
			UPDATE sensitive_data SET encrypted_value = $1, nonce = $2, key_alias = $3, updated_at = NOW()
			WHERE id = $4 AND tenant_id = $5
		`, newCiphertext, newNonce, newAlias, r.id, tenantID)
		if err != nil {
			return fmt.Errorf("failed to update row %s: %w", r.id, err)
		}
	}

	// Mark old key as rotated
	_, err = tx.Exec(`
		UPDATE encryption_keys SET status = 'rotated', rotated_at = NOW() WHERE key_alias = $1 AND tenant_id = $2
	`, oldAlias, tenantID)
	if err != nil {
		return fmt.Errorf("failed to mark old key as rotated: %w", err)
	}

	return tx.Commit()
}

// GenerateKeyBytes returns a new random 32-byte encryption key.
func (e *EncryptionService) GenerateKeyBytes() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic("failed to generate key: " + err.Error())
	}
	return key
}

// GenerateKey generates a cryptographically secure 32-byte key.
func (s *EncryptionService) GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// GetQuarterlyRotationDue returns active keys whose activated_at is more than 90 days ago.
func (s *EncryptionService) GetQuarterlyRotationDue(db *sql.DB, tenantID uuid.UUID) ([]models.EncryptionKey, error) {
	cutoff := time.Now().AddDate(0, 0, -90)
	rows, err := db.Query(`
		SELECT id, key_alias, algorithm, status, rotation_number, activated_at, rotated_at, expires_at, created_at
		FROM encryption_keys
		WHERE status = 'active' AND activated_at < $1 AND tenant_id = $2
		ORDER BY activated_at ASC
	`, cutoff, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query keys due for rotation: %w", err)
	}
	defer rows.Close()

	var keys []models.EncryptionKey
	for rows.Next() {
		var k models.EncryptionKey
		if err := rows.Scan(
			&k.ID, &k.KeyAlias, &k.Algorithm, &k.Status, &k.RotationNumber,
			&k.ActivatedAt, &k.RotatedAt, &k.ExpiresAt, &k.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan key: %w", err)
		}
		keys = append(keys, k)
	}

	if keys == nil {
		keys = []models.EncryptionKey{}
	}

	return keys, nil
}
