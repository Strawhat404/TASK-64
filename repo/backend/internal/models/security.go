package models

import (
	"time"

	"github.com/google/uuid"
)

// EncryptionKey represents a key in the encryption key registry.
type EncryptionKey struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	KeyAlias       string     `json:"key_alias" db:"key_alias"`
	EncryptedKey   []byte     `json:"-" db:"encrypted_key"`
	Nonce          []byte     `json:"-" db:"nonce"`
	Algorithm      string     `json:"algorithm" db:"algorithm"`
	Status         string     `json:"status" db:"status"`
	RotationNumber int        `json:"rotation_number" db:"rotation_number"`
	ActivatedAt    time.Time  `json:"activated_at" db:"activated_at"`
	RotatedAt      *time.Time `json:"rotated_at,omitempty" db:"rotated_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// EncryptedField tracks which table/column is encrypted with which key.
type EncryptedField struct {
	ID         uuid.UUID `json:"id" db:"id"`
	TableName  string    `json:"table_name" db:"table_name"`
	ColumnName string    `json:"column_name" db:"column_name"`
	KeyAlias   string    `json:"key_alias" db:"key_alias"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// SensitiveData represents an encrypted sensitive data record.
type SensitiveData struct {
	ID             uuid.UUID `json:"id" db:"id"`
	TenantID       uuid.UUID `json:"tenant_id" db:"tenant_id"`
	OwnerID        uuid.UUID `json:"owner_id" db:"owner_id"`
	DataType       string    `json:"data_type" db:"data_type"`
	EncryptedValue []byte    `json:"-" db:"encrypted_value"`
	Nonce          []byte    `json:"-" db:"nonce"`
	KeyAlias       string    `json:"-" db:"key_alias"`
	Label          string    `json:"label" db:"label"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// SensitiveDataResponse is an API response with masked sensitive data.
type SensitiveDataResponse struct {
	ID          uuid.UUID `json:"id"`
	DataType    string    `json:"data_type"`
	Label       string    `json:"label"`
	MaskedValue string    `json:"masked_value"`
	CreatedAt   time.Time `json:"created_at"`
}

// AuditLedgerEntry represents a single entry in the immutable audit ledger.
type AuditLedgerEntry struct {
	ID           int64                  `json:"id" db:"id"`
	EntryHash    string                 `json:"entry_hash" db:"entry_hash"`
	PreviousHash string                 `json:"previous_hash" db:"previous_hash"`
	TenantID     *uuid.UUID             `json:"tenant_id,omitempty" db:"tenant_id"`
	UserID       *uuid.UUID             `json:"user_id,omitempty" db:"user_id"`
	Action       string                 `json:"action" db:"action"`
	ResourceType string                 `json:"resource_type" db:"resource_type"`
	ResourceID   string                 `json:"resource_id" db:"resource_id"`
	Details      map[string]interface{} `json:"details,omitempty" db:"details"`
	IPAddress    string                 `json:"ip_address" db:"ip_address"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
}

// RetentionPolicy defines how long data in a given table should be retained.
type RetentionPolicy struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	TenantID       *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	TableName      string     `json:"table_name" db:"table_name"`
	RetentionYears int        `json:"retention_years" db:"retention_years"`
	LastPurgeAt    *time.Time `json:"last_purge_at,omitempty" db:"last_purge_at"`
	NextPurgeAt    *time.Time `json:"next_purge_at,omitempty" db:"next_purge_at"`
	IsActive       bool       `json:"is_active" db:"is_active"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// DeletionLogEntry records each secure deletion performed.
type DeletionLogEntry struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	TableName      string     `json:"table_name" db:"table_name"`
	RecordID       string     `json:"record_id" db:"record_id"`
	DeletionReason string     `json:"deletion_reason" db:"deletion_reason"`
	DeletedBy      *uuid.UUID `json:"deleted_by,omitempty" db:"deleted_by"`
	RetentionMet   bool       `json:"retention_met" db:"retention_met"`
	DeletedAt      time.Time  `json:"deleted_at" db:"deleted_at"`
}

// RateLimitStatus represents the current rate limit state for a client.
type RateLimitStatus struct {
	Identifier        string    `json:"identifier"`
	Type              string    `json:"type"`
	RequestsRemaining int       `json:"requests_remaining"`
	WindowReset       time.Time `json:"window_reset"`
}

// LegalHold represents a legal hold preventing data deletion.
type LegalHold struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	TenantID       uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	HoldReason     string     `json:"hold_reason" db:"hold_reason"`
	HeldBy         uuid.UUID  `json:"held_by" db:"held_by"`
	HoldStart      time.Time  `json:"hold_start" db:"hold_start"`
	HoldEnd        *time.Time `json:"hold_end,omitempty" db:"hold_end"`
	IsActive       bool       `json:"is_active" db:"is_active"`
	TargetTable    string     `json:"target_table" db:"target_table"`
	TargetRecordID *string    `json:"target_record_id,omitempty" db:"target_record_id"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// StoreSensitiveDataRequest is the payload for storing sensitive data.
type StoreSensitiveDataRequest struct {
	DataType string `json:"data_type"`
	Value    string `json:"value"`
	Label    string `json:"label"`
}

// KeyRotationRequest is the payload for triggering key rotation.
type KeyRotationRequest struct {
	KeyAlias string `json:"key_alias"`
}
