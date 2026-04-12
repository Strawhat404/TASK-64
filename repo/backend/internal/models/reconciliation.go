package models

import (
	"time"

	"github.com/google/uuid"
)

// TransactionFeed represents an imported batch of transactions from a CSV/Excel file.
type TransactionFeed struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TenantID    uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Filename    string    `json:"filename" db:"filename"`
	FeedType    string    `json:"feed_type" db:"feed_type"`
	RecordCount int       `json:"record_count" db:"record_count"`
	ImportedBy  uuid.UUID `json:"imported_by" db:"imported_by"`
	ImportedAt  time.Time `json:"imported_at" db:"imported_at"`
	Status      string    `json:"status" db:"status"`
}

// Transaction represents an individual financial transaction from an imported feed.
type Transaction struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	FeedID          uuid.UUID  `json:"feed_id" db:"feed_id"`
	TenantID        uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	TransactionDate time.Time  `json:"transaction_date" db:"transaction_date"`
	Amount          float64    `json:"amount" db:"amount"`
	Counterparty        *string    `json:"counterparty,omitempty" db:"counterparty"`
	CounterpartyAccount *string    `json:"counterparty_account,omitempty" db:"counterparty_account"`
	Memo                *string    `json:"memo,omitempty" db:"memo"`
	ReferenceNumber *string    `json:"reference_number,omitempty" db:"reference_number"`
	Source          string     `json:"source" db:"source"`
	Matched         bool       `json:"matched" db:"matched"`
	MatchID         *uuid.UUID `json:"match_id,omitempty" db:"match_id"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

// TransactionMatch represents a matched pair of internal and external transactions.
type TransactionMatch struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	TenantID            uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	InternalTxID        uuid.UUID  `json:"internal_tx_id" db:"internal_tx_id"`
	ExternalTxID        uuid.UUID  `json:"external_tx_id" db:"external_tx_id"`
	MatchConfidence     float64    `json:"match_confidence" db:"match_confidence"`
	MatchMethod         string     `json:"match_method" db:"match_method"`
	AmountVariance      float64    `json:"amount_variance" db:"amount_variance"`
	TimeVarianceMinutes int        `json:"time_variance_minutes" db:"time_variance_minutes"`
	MatchedBy           *uuid.UUID `json:"matched_by,omitempty" db:"matched_by"`
	MatchedAt           time.Time  `json:"matched_at" db:"matched_at"`
	Status              string     `json:"status" db:"status"`
}

// ReconciliationException represents an exception item such as unmatched, duplicate, or variance.
type ReconciliationException struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	TenantID        uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	TransactionID   *uuid.UUID `json:"transaction_id,omitempty" db:"transaction_id"`
	MatchID         *uuid.UUID `json:"match_id,omitempty" db:"match_id"`
	ExceptionType   string     `json:"exception_type" db:"exception_type"`
	Severity        string     `json:"severity" db:"severity"`
	Amount          *float64   `json:"amount,omitempty" db:"amount"`
	VarianceAmount  *float64   `json:"variance_amount,omitempty" db:"variance_amount"`
	Description     *string    `json:"description,omitempty" db:"description"`
	AssignedTo      *uuid.UUID `json:"assigned_to,omitempty" db:"assigned_to"`
	Status          string     `json:"status" db:"status"`
	Disposition     *string    `json:"disposition,omitempty" db:"disposition"`
	ResolutionNotes *string    `json:"resolution_notes,omitempty" db:"resolution_notes"`
	ResolvedBy      *uuid.UUID `json:"resolved_by,omitempty" db:"resolved_by"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// ImportRequest is the payload for importing a transaction feed.
type ImportRequest struct {
	FeedType string `json:"feed_type"`
}

// ImportResult contains the results of a feed import operation.
type ImportResult struct {
	FeedID              uuid.UUID `json:"feed_id"`
	RecordsImported     int       `json:"records_imported"`
	RecordsMatched      int       `json:"records_matched"`
	ExceptionsGenerated int       `json:"exceptions_generated"`
}

// MatchResult contains the results of a matching operation.
type MatchResult struct {
	TotalInternal  int                       `json:"total_internal"`
	TotalExternal  int                       `json:"total_external"`
	MatchedCount   int                       `json:"matched_count"`
	ExceptionCount int                       `json:"exception_count"`
	Matches        []TransactionMatch        `json:"matches"`
	Exceptions     []ReconciliationException `json:"exceptions"`
}

// ExceptionResolution is the payload for resolving a reconciliation exception.
type ExceptionResolution struct {
	Disposition     string `json:"disposition"`
	ResolutionNotes string `json:"resolution_notes"`
}

// ReconciliationSummary contains overall reconciliation statistics.
type ReconciliationSummary struct {
	TotalFeeds          int     `json:"total_feeds"`
	TotalTransactions   int     `json:"total_transactions"`
	TotalMatched        int     `json:"total_matched"`
	TotalExceptions     int     `json:"total_exceptions"`
	OpenExceptions      int     `json:"open_exceptions"`
	TotalOpen           int     `json:"total_open"`
	UnmatchedItems      int     `json:"unmatched_items"`
	SuspectedDuplicates int     `json:"suspected_duplicates"`
	VarianceAlerts      int     `json:"variance_alerts"`
	MatchRatePct        float64 `json:"match_rate_pct"`
	MatchRate           float64 `json:"match_rate"`
}
