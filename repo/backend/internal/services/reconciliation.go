package services

import (
	"database/sql"
	"math"
	"strings"
	"time"

	"compliance-console/internal/models"

	"github.com/google/uuid"
)

// MatchTransactions runs the core matching algorithm for a tenant's unmatched transactions.
// It loads unmatched internal and external transactions, scores potential matches, creates
// match records for high-confidence pairs, and generates exceptions for unmatched or
// problematic items.
func MatchTransactions(db *sql.DB, tenantID uuid.UUID, feedID uuid.UUID) (*models.MatchResult, error) {
	// Load unmatched internal transactions scoped to this feed
	internalTxs, err := loadUnmatchedTransactions(db, tenantID, feedID, "internal")
	if err != nil {
		return nil, err
	}

	// Load unmatched external transactions scoped to this feed
	externalTxs, err := loadUnmatchedTransactions(db, tenantID, feedID, "external")
	if err != nil {
		return nil, err
	}

	result := &models.MatchResult{
		TotalInternal: len(internalTxs),
		TotalExternal: len(externalTxs),
		Matches:       []models.TransactionMatch{},
		Exceptions:    []models.ReconciliationException{},
	}

	// Track which external transactions have been matched
	matchedExternal := make(map[uuid.UUID]bool)

	now := time.Now()

	for i := range internalTxs {
		intTx := &internalTxs[i]
		bestScore := 0.0
		bestIdx := -1

		for j := range externalTxs {
			extTx := &externalTxs[j]
			if matchedExternal[extTx.ID] {
				continue
			}

			score := scoreMatchInternal(intTx, extTx)
			if score > bestScore {
				bestScore = score
				bestIdx = j
			}
		}

		if bestScore >= 70 && bestIdx >= 0 {
			// Auto-match
			extTx := &externalTxs[bestIdx]
			matchedExternal[extTx.ID] = true

			amountVariance := math.Abs(intTx.Amount - extTx.Amount)
			timeVariance := int(math.Abs(intTx.TransactionDate.Sub(extTx.TransactionDate).Minutes()))

			matchMethod := "exact"
			if amountVariance > 0 || timeVariance > 0 {
				matchMethod = "fuzzy"
			}

			matchID := uuid.New()
			match := models.TransactionMatch{
				ID:                  matchID,
				TenantID:            tenantID,
				InternalTxID:        intTx.ID,
				ExternalTxID:        extTx.ID,
				MatchConfidence:     bestScore,
				MatchMethod:         matchMethod,
				AmountVariance:      amountVariance,
				TimeVarianceMinutes: timeVariance,
				MatchedAt:           now,
				Status:              "confirmed",
			}

			err := insertMatch(db, &match)
			if err != nil {
				return nil, err
			}

			// Mark both transactions as matched
			err = markTransactionMatched(db, intTx.ID, matchID)
			if err != nil {
				return nil, err
			}
			err = markTransactionMatched(db, extTx.ID, matchID)
			if err != nil {
				return nil, err
			}

			result.Matches = append(result.Matches, match)
			result.MatchedCount++

			// Generate variance exception if amount difference > $1.00
			if amountVariance > 1.00 {
				desc := "Amount variance exceeds $1.00 threshold"
				exc := models.ReconciliationException{
					ID:             uuid.New(),
					TenantID:       tenantID,
					TransactionID:  &intTx.ID,
					MatchID:        &matchID,
					ExceptionType:  "variance_over_threshold",
					Severity:       classifyVarianceSeverity(amountVariance),
					Amount:         &intTx.Amount,
					VarianceAmount: &amountVariance,
					Description:    &desc,
					Status:         "open",
					CreatedAt:      now,
					UpdatedAt:      now,
				}
				err := insertException(db, &exc)
				if err != nil {
					return nil, err
				}
				result.Exceptions = append(result.Exceptions, exc)
				result.ExceptionCount++
			}
		} else if bestScore >= 50 && bestScore < 70 && bestIdx >= 0 {
			// Flag for manual review - create exception but do not auto-match
			desc := "Potential match found but confidence below auto-match threshold"
			exc := models.ReconciliationException{
				ID:            uuid.New(),
				TenantID:      tenantID,
				TransactionID: &intTx.ID,
				ExceptionType: "unmatched",
				Severity:      "medium",
				Amount:        &intTx.Amount,
				Description:   &desc,
				Status:        "open",
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			err := insertException(db, &exc)
			if err != nil {
				return nil, err
			}
			result.Exceptions = append(result.Exceptions, exc)
			result.ExceptionCount++
		} else {
			// Unmatched internal transaction
			desc := "No matching external transaction found"
			exc := models.ReconciliationException{
				ID:            uuid.New(),
				TenantID:      tenantID,
				TransactionID: &intTx.ID,
				ExceptionType: "unmatched",
				Severity:      "high",
				Amount:        &intTx.Amount,
				Description:   &desc,
				Status:        "open",
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			err := insertException(db, &exc)
			if err != nil {
				return nil, err
			}
			result.Exceptions = append(result.Exceptions, exc)
			result.ExceptionCount++
		}
	}

	// Generate exceptions for unmatched external transactions
	for j := range externalTxs {
		extTx := &externalTxs[j]
		if matchedExternal[extTx.ID] {
			continue
		}
		desc := "No matching internal transaction found"
		exc := models.ReconciliationException{
			ID:            uuid.New(),
			TenantID:      tenantID,
			TransactionID: &extTx.ID,
			ExceptionType: "unmatched",
			Severity:      "high",
			Amount:        &extTx.Amount,
			Description:   &desc,
			Status:        "open",
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		err := insertException(db, &exc)
		if err != nil {
			return nil, err
		}
		result.Exceptions = append(result.Exceptions, exc)
		result.ExceptionCount++
	}

	// Detect duplicates within the specified feed
	dupes, err := DetectDuplicates(db, tenantID, feedID)
	if err != nil {
		return nil, err
	}
	result.Exceptions = append(result.Exceptions, dupes...)
	result.ExceptionCount += len(dupes)

	// Update feed status to completed
	_, err = db.Exec(`UPDATE transaction_feeds SET status = 'completed' WHERE id = $1 AND tenant_id = $2`, feedID, tenantID)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// DetectDuplicates finds transactions with the same amount and counterparty within
// 24 hours in the same feed.
func DetectDuplicates(db *sql.DB, tenantID, feedID uuid.UUID) ([]models.ReconciliationException, error) {
	rows, err := db.Query(`
		SELECT t1.id, t1.amount, t1.counterparty, t1.transaction_date
		FROM transactions t1
		JOIN transactions t2 ON t1.feed_id = t2.feed_id
			AND t1.id != t2.id
			AND t1.amount = t2.amount
			AND t1.counterparty IS NOT NULL
			AND t2.counterparty IS NOT NULL
			AND t1.counterparty = t2.counterparty
			AND ABS(EXTRACT(EPOCH FROM (t1.transaction_date - t2.transaction_date))) <= 86400
		WHERE t1.feed_id = $1 AND t1.tenant_id = $2
		GROUP BY t1.id, t1.amount, t1.counterparty, t1.transaction_date
	`, feedID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now()
	var exceptions []models.ReconciliationException
	seen := make(map[uuid.UUID]bool)

	for rows.Next() {
		var txID uuid.UUID
		var amount float64
		var counterparty *string
		var txDate time.Time

		if err := rows.Scan(&txID, &amount, &counterparty, &txDate); err != nil {
			return nil, err
		}

		if seen[txID] {
			continue
		}
		seen[txID] = true

		desc := "Suspected duplicate: same amount and counterparty within 24 hours"
		exc := models.ReconciliationException{
			ID:            uuid.New(),
			TenantID:      tenantID,
			TransactionID: &txID,
			ExceptionType: "duplicate_suspect",
			Severity:      "medium",
			Amount:        &amount,
			Description:   &desc,
			Status:        "open",
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if err := insertException(db, &exc); err != nil {
			return nil, err
		}
		exceptions = append(exceptions, exc)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return exceptions, nil
}

// ImportCSVFeed bulk-inserts transactions from parsed CSV data and creates a feed record.
func ImportCSVFeed(db *sql.DB, tenantID, userID uuid.UUID, feedType, filename string, records []models.Transaction) (*models.TransactionFeed, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	feedID := uuid.New()
	now := time.Now()

	feed := &models.TransactionFeed{
		ID:          feedID,
		TenantID:    tenantID,
		Filename:    filename,
		FeedType:    feedType,
		RecordCount: len(records),
		ImportedBy:  userID,
		ImportedAt:  now,
		Status:      "processing",
	}

	_, err = tx.Exec(`
		INSERT INTO transaction_feeds (id, tenant_id, filename, feed_type, record_count, imported_by, imported_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, feed.ID, feed.TenantID, feed.Filename, feed.FeedType, feed.RecordCount,
		feed.ImportedBy, feed.ImportedAt, feed.Status)
	if err != nil {
		return nil, err
	}

	// Bulk insert transactions
	for i := range records {
		rec := &records[i]
		rec.ID = uuid.New()
		rec.FeedID = feedID
		rec.TenantID = tenantID
		rec.Source = feedType
		rec.CreatedAt = now

		_, err = tx.Exec(`
			INSERT INTO transactions (id, feed_id, tenant_id, transaction_date, amount,
				counterparty, counterparty_account, memo, reference_number, source, matched, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, rec.ID, rec.FeedID, rec.TenantID, rec.TransactionDate, rec.Amount,
			rec.Counterparty, rec.CounterpartyAccount, rec.Memo, rec.ReferenceNumber, rec.Source, false, rec.CreatedAt)
		if err != nil {
			return nil, err
		}
	}

	// Update feed status to pending (ready for matching)
	_, err = tx.Exec(`UPDATE transaction_feeds SET status = 'pending' WHERE id = $1`, feedID)
	if err != nil {
		return nil, err
	}
	feed.Status = "pending"

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return feed, nil
}

// scoreMatch calculates a match score between an internal and external transaction.
// Maximum score is 100 points.
// ScoreMatch computes a match confidence score between two transactions.
// Exported for direct unit testing against production logic.
func ScoreMatch(internal, external *models.Transaction) float64 {
	return scoreMatchInternal(internal, external)
}

func scoreMatchInternal(internal, external *models.Transaction) float64 {
	score := 0.0

	// Amount scoring: exact = 40, within $1.00 = 20, else 0
	amountDiff := math.Abs(internal.Amount - external.Amount)
	if amountDiff == 0 {
		score += 40
	} else if amountDiff <= 1.00 {
		score += 20
	}

	// Timestamp scoring: strict ±10 min enforcement per spec
	timeDiff := math.Abs(internal.TransactionDate.Sub(external.TransactionDate).Minutes())
	if timeDiff <= 10 {
		score += 30
	}
	// Beyond ±10 minutes: 0 timestamp points (strict per spec)

	// Counterparty scoring: exact = 15, substring/contains = 7
	if internal.Counterparty != nil && external.Counterparty != nil {
		intCP := strings.TrimSpace(strings.ToLower(*internal.Counterparty))
		extCP := strings.TrimSpace(strings.ToLower(*external.Counterparty))
		if intCP != "" && extCP != "" {
			if intCP == extCP {
				score += 15
			} else if strings.Contains(intCP, extCP) || strings.Contains(extCP, intCP) {
				score += 7
			}
		}
	}

	// Counterparty account scoring: exact = 10, partial = 5
	if internal.CounterpartyAccount != nil && external.CounterpartyAccount != nil {
		intAcct := strings.TrimSpace(strings.ToLower(*internal.CounterpartyAccount))
		extAcct := strings.TrimSpace(strings.ToLower(*external.CounterpartyAccount))
		if intAcct != "" && extAcct != "" {
			if intAcct == extAcct {
				score += 10
			} else if strings.Contains(intAcct, extAcct) || strings.Contains(extAcct, intAcct) {
				score += 5
			}
		}
	}

	// Memo scoring: Levenshtein-based similarity, up to 10 points
	if internal.Memo != nil && external.Memo != nil {
		intMemo := strings.TrimSpace(*internal.Memo)
		extMemo := strings.TrimSpace(*external.Memo)
		if intMemo != "" && extMemo != "" {
			sim := levenshteinSimilarity(strings.ToLower(intMemo), strings.ToLower(extMemo))
			score += sim * 10
		}
	}

	return score
}

// levenshteinDistance computes the edit distance between two strings.
func levenshteinDistance(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

// levenshteinSimilarity returns a 0.0-1.0 score based on edit distance.
func levenshteinSimilarity(a, b string) float64 {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	if maxLen == 0 {
		return 1.0
	}
	dist := levenshteinDistance(a, b)
	return 1.0 - float64(dist)/float64(maxLen)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// classifyVarianceSeverity determines exception severity based on variance amount.
func classifyVarianceSeverity(variance float64) string {
	switch {
	case variance > 100:
		return "critical"
	case variance > 50:
		return "high"
	case variance > 10:
		return "medium"
	default:
		return "low"
	}
}

// loadUnmatchedTransactions loads all unmatched transactions for a tenant and feed with the given source type.
func loadUnmatchedTransactions(db *sql.DB, tenantID uuid.UUID, feedID uuid.UUID, source string) ([]models.Transaction, error) {
	rows, err := db.Query(`
		SELECT id, feed_id, tenant_id, transaction_date, amount,
			counterparty, counterparty_account, memo, reference_number, source, matched, match_id, created_at
		FROM transactions
		WHERE tenant_id = $1 AND feed_id = $2 AND source = $3 AND matched = FALSE
		ORDER BY transaction_date ASC
	`, tenantID, feedID, source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []models.Transaction
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(
			&t.ID, &t.FeedID, &t.TenantID, &t.TransactionDate, &t.Amount,
			&t.Counterparty, &t.CounterpartyAccount, &t.Memo, &t.ReferenceNumber, &t.Source,
			&t.Matched, &t.MatchID, &t.CreatedAt,
		); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return txs, nil
}

// insertMatch inserts a transaction_matches record.
func insertMatch(db *sql.DB, m *models.TransactionMatch) error {
	_, err := db.Exec(`
		INSERT INTO transaction_matches (id, tenant_id, internal_tx_id, external_tx_id,
			match_confidence, match_method, amount_variance, time_variance_minutes,
			matched_by, matched_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, m.ID, m.TenantID, m.InternalTxID, m.ExternalTxID,
		m.MatchConfidence, m.MatchMethod, m.AmountVariance, m.TimeVarianceMinutes,
		m.MatchedBy, m.MatchedAt, m.Status)
	return err
}

// markTransactionMatched updates a transaction to set matched = true and assigns the match_id.
func markTransactionMatched(db *sql.DB, txID, matchID uuid.UUID) error {
	_, err := db.Exec(`
		UPDATE transactions SET matched = TRUE, match_id = $1 WHERE id = $2
	`, matchID, txID)
	return err
}

// insertException inserts a reconciliation_exceptions record.
func insertException(db *sql.DB, exc *models.ReconciliationException) error {
	_, err := db.Exec(`
		INSERT INTO reconciliation_exceptions (id, tenant_id, transaction_id, match_id,
			exception_type, severity, amount, variance_amount, description,
			assigned_to, status, disposition, resolution_notes, resolved_by, resolved_at,
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, exc.ID, exc.TenantID, exc.TransactionID, exc.MatchID,
		exc.ExceptionType, exc.Severity, exc.Amount, exc.VarianceAmount, exc.Description,
		exc.AssignedTo, exc.Status, exc.Disposition, exc.ResolutionNotes,
		exc.ResolvedBy, exc.ResolvedAt, exc.CreatedAt, exc.UpdatedAt)
	return err
}
