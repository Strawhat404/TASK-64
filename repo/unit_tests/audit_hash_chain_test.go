package unit_tests

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

// Mirrors the hash-chain logic in backend/internal/services/auditledger.go.
// Each entry's hash = SHA256(previousHash + tenantID + userID + action +
//   resourceType + resourceID + details + timestamp).
// The first entry links to "GENESIS".

const genesisHash = "GENESIS"

type ledgerEntry struct {
	id           int64
	entryHash    string
	previousHash string
	tenantID     string
	userID       string
	action       string
	resourceType string
	resourceID   string
	details      string
	createdAt    time.Time
}

func computeEntryHash(prev, tenantID, userID, action, resourceType, resourceID, details string, ts time.Time) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		prev, tenantID, userID, action, resourceType, resourceID, details, ts.UTC().Format(time.RFC3339Nano))
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

func buildChain(entries []ledgerEntry) []ledgerEntry {
	result := make([]ledgerEntry, len(entries))
	prev := genesisHash
	for i, e := range entries {
		e.previousHash = prev
		e.entryHash = computeEntryHash(prev, e.tenantID, e.userID, e.action,
			e.resourceType, e.resourceID, e.details, e.createdAt)
		e.id = int64(i + 1)
		result[i] = e
		prev = e.entryHash
	}
	return result
}

func verifyChain(chain []ledgerEntry) (bool, int64) {
	prev := genesisHash
	for _, e := range chain {
		expected := computeEntryHash(prev, e.tenantID, e.userID, e.action,
			e.resourceType, e.resourceID, e.details, e.createdAt)
		if e.entryHash != expected {
			return false, e.id
		}
		if e.previousHash != prev {
			return false, e.id
		}
		prev = e.entryHash
	}
	return true, 0
}

func TestAuditChainGenesisLink(t *testing.T) {
	entries := buildChain([]ledgerEntry{
		{tenantID: "t1", userID: "u1", action: "login", resourceType: "session",
			resourceID: "s1", details: "{}", createdAt: time.Now()},
	})

	if entries[0].previousHash != genesisHash {
		t.Errorf("first entry previousHash = %q, want %q", entries[0].previousHash, genesisHash)
	}
}

func TestAuditChainIntegrity(t *testing.T) {
	base := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	raw := []ledgerEntry{
		{tenantID: "t1", userID: "u1", action: "login", resourceType: "session",
			resourceID: "s1", details: "{}", createdAt: base},
		{tenantID: "t1", userID: "u1", action: "create_user", resourceType: "user",
			resourceID: "u2", details: `{"username":"new"}`, createdAt: base.Add(time.Minute)},
		{tenantID: "t1", userID: "u2", action: "login", resourceType: "session",
			resourceID: "s2", details: "{}", createdAt: base.Add(2 * time.Minute)},
	}

	chain := buildChain(raw)

	valid, brokenAt := verifyChain(chain)
	if !valid {
		t.Errorf("chain should be valid, broken at entry %d", brokenAt)
	}
}

func TestAuditChainTamperDetection(t *testing.T) {
	base := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	raw := []ledgerEntry{
		{tenantID: "t1", userID: "u1", action: "login", resourceType: "session",
			resourceID: "s1", details: "{}", createdAt: base},
		{tenantID: "t1", userID: "u1", action: "deactivate_user", resourceType: "user",
			resourceID: "u2", details: `{"reason":"termination"}`, createdAt: base.Add(time.Minute)},
		{tenantID: "t1", userID: "u1", action: "logout", resourceType: "session",
			resourceID: "s1", details: "{}", createdAt: base.Add(2 * time.Minute)},
	}

	chain := buildChain(raw)

	t.Run("tamper action field", func(t *testing.T) {
		tampered := make([]ledgerEntry, len(chain))
		copy(tampered, chain)
		tampered[1].action = "update_user" // changed from deactivate_user
		valid, brokenAt := verifyChain(tampered)
		if valid {
			t.Error("tampered chain should fail verification")
		}
		if brokenAt != 2 {
			t.Errorf("broken at entry %d, want 2", brokenAt)
		}
	})

	t.Run("tamper details field", func(t *testing.T) {
		tampered := make([]ledgerEntry, len(chain))
		copy(tampered, chain)
		tampered[1].details = `{"reason":"resignation"}` // changed
		valid, brokenAt := verifyChain(tampered)
		if valid {
			t.Error("tampered chain should fail verification")
		}
		if brokenAt != 2 {
			t.Errorf("broken at entry %d, want 2", brokenAt)
		}
	})

	t.Run("tamper timestamp", func(t *testing.T) {
		tampered := make([]ledgerEntry, len(chain))
		copy(tampered, chain)
		tampered[0].createdAt = tampered[0].createdAt.Add(time.Second)
		valid, _ := verifyChain(tampered)
		if valid {
			t.Error("timestamp-tampered chain should fail verification")
		}
	})

	t.Run("delete middle entry", func(t *testing.T) {
		shortened := []ledgerEntry{chain[0], chain[2]}
		valid, _ := verifyChain(shortened)
		if valid {
			t.Error("chain with deleted entry should fail verification")
		}
	})

	t.Run("swap entry order", func(t *testing.T) {
		swapped := []ledgerEntry{chain[0], chain[2], chain[1]}
		valid, _ := verifyChain(swapped)
		if valid {
			t.Error("reordered chain should fail verification")
		}
	})
}

func TestAuditChainHashDeterminism(t *testing.T) {
	ts := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	h1 := computeEntryHash("prev", "t1", "u1", "login", "session", "s1", "{}", ts)
	h2 := computeEntryHash("prev", "t1", "u1", "login", "session", "s1", "{}", ts)
	if h1 != h2 {
		t.Error("same inputs should produce identical hashes")
	}

	// Different input should produce different hash
	h3 := computeEntryHash("prev", "t1", "u1", "logout", "session", "s1", "{}", ts)
	if h1 == h3 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestAuditChainHashLength(t *testing.T) {
	ts := time.Now()
	h := computeEntryHash("prev", "t1", "u1", "login", "session", "s1", "{}", ts)
	// SHA-256 produces 64 hex characters
	if len(h) != 64 {
		t.Errorf("hash length = %d, want 64", len(h))
	}
}
