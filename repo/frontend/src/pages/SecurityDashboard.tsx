import { useState, useEffect, useCallback } from "react";
import axios from "axios";
import Sidebar from "../components/Sidebar";
import { SortableHeader, useSortableData } from "../components/SortableHeader";

interface EncryptionKey {
  id: string;
  key_alias: string;
  algorithm: string;
  status: string;
  rotation_number: number;
  activated_at: string;
}

interface SensitiveData {
  id: string;
  masked_value: string;
  data_type: string;
  label: string;
  revealed_value?: string;
}

interface AuditEntry {
  id: string;
  action: string;
  user_id: string;
  resource_type: string;
  resource_id: string;
  created_at: string;
  details: any;
  ip_address: string;
}

interface RateLimit {
  identifier: string;
  type: string;
  requests_remaining: number;
  window_reset: string;
}

interface RetentionPolicy {
  id: string;
  table_name: string;
  retention_years: number;
  last_purge_at: string | null;
  next_purge_at: string | null;
  is_active: boolean;
  created_at: string;
}

interface ChainStatus {
  valid: boolean;
  entries_checked: number;
  broken_at?: number;
}

export default function SecurityDashboard() {
  const [keys, setKeys] = useState<EncryptionKey[]>([]);
  const [sensitiveData, setSensitiveData] = useState<SensitiveData[]>([]);
  const [auditEntries, setAuditEntries] = useState<AuditEntry[]>([]);
  const [rateLimits, setRateLimits] = useState<RateLimit[]>([]);
  const [retentionPolicies, setRetentionPolicies] = useState<RetentionPolicy[]>([]);
  const [loading, setLoading] = useState(true);

  // Chain verification
  const [chainStatus, setChainStatus] = useState<ChainStatus | null>(null);
  const [verifyingChain, setVerifyingChain] = useState(false);

  // Reveal confirmation
  const [revealModal, setRevealModal] = useState<{ id: string; label: string } | null>(null);
  const [revealing, setRevealing] = useState(false);

  // Key rotation
  const [rotatingKeyId, setRotatingKeyId] = useState<string | null>(null);

  // Cleanup
  const [cleaningPolicyId, setCleaningPolicyId] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [keysRes, sensitiveRes, auditRes, limitsRes, retentionRes] = await Promise.all([
        axios.get("/api/security/keys").catch(() => ({ data: [] })),
        axios.get("/api/security/sensitive").catch(() => ({ data: [] })),
        axios.get("/api/security/audit-ledger").catch(() => ({ data: [] })),
        axios.get("/api/security/rate-limits").catch(() => ({ data: [] })),
        axios.get("/api/security/retention").catch(() => ({ data: [] })),
      ]);
      const keysData = keysRes.data?.data ?? keysRes.data;
      setKeys(Array.isArray(keysData) ? keysData : []);
      const sensitiveDataArr = sensitiveRes.data?.data ?? sensitiveRes.data;
      setSensitiveData(Array.isArray(sensitiveDataArr) ? sensitiveDataArr : []);
      const auditData = auditRes.data?.data ?? auditRes.data;
      setAuditEntries(Array.isArray(auditData) ? auditData : []);
      const limitsData = limitsRes.data?.data ?? limitsRes.data;
      setRateLimits(Array.isArray(limitsData) ? limitsData : []);
      const retentionData = retentionRes.data?.data ?? retentionRes.data;
      setRetentionPolicies(Array.isArray(retentionData) ? retentionData : []);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const daysSinceActivation = (activatedAt: string): number => {
    const activated = new Date(activatedAt).getTime();
    const now = Date.now();
    return Math.floor((now - activated) / (1000 * 60 * 60 * 24));
  };

  const rotateKey = async (keyId: string, keyAlias: string) => {
    setRotatingKeyId(keyId);
    try {
      await axios.post(`/api/security/keys/rotate`, { key_alias: keyAlias });
      fetchData();
    } catch {
      alert("Failed to rotate key.");
    } finally {
      setRotatingKeyId(null);
    }
  };

  const revealSensitive = async (id: string) => {
    setRevealing(true);
    try {
      const res = await axios.post(`/api/security/sensitive/${id}/reveal`);
      setSensitiveData((prev) =>
        prev.map((d) =>
          d.id === id ? { ...d, revealed_value: res.data.value || res.data.revealed_value } : d
        )
      );
      setRevealModal(null);
    } catch {
      alert("Failed to reveal data.");
    } finally {
      setRevealing(false);
    }
  };

  const verifyChain = async () => {
    setVerifyingChain(true);
    try {
      const res = await axios.post("/api/security/audit-ledger/verify");
      setChainStatus(res.data);
    } catch {
      alert("Failed to verify chain.");
    } finally {
      setVerifyingChain(false);
    }
  };

  const runCleanup = async (policyId: string) => {
    setCleaningPolicyId(policyId);
    try {
      await axios.post(`/api/security/retention/cleanup`);
      fetchData();
    } catch {
      alert("Failed to run cleanup.");
    } finally {
      setCleaningPolicyId(null);
    }
  };

  const { sortedItems: sortedKeys, sortConfig: keysSortConfig, requestSort: requestKeysSort } = useSortableData(keys);
  const { sortedItems: sortedRetentionPolicies, sortConfig: retentionSortConfig, requestSort: requestRetentionSort } = useSortableData(retentionPolicies);

  if (loading) {
    return (
      <div className="flex h-screen bg-slate-950">
        <Sidebar />
        <main className="flex-1 overflow-auto p-6 flex items-center justify-center">
          <div className="animate-spin w-8 h-8 border-2 border-slate-600 border-t-blue-500 rounded-full" />
        </main>
      </div>
    );
  }

  return (
    <div className="flex h-screen bg-slate-950">
      <Sidebar />
      <main className="flex-1 overflow-auto p-6">
        {/* Header */}
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-slate-100">Security & Data Lifecycle</h1>
          <p className="text-slate-400 mt-1">Encryption, audit ledger, rate limits, and data retention</p>
        </div>

        {/* 1. Encryption Keys */}
        <section className="mb-6">
          <h2 className="text-lg font-semibold text-slate-200 mb-4">Encryption Keys</h2>
          <div className="bg-slate-800 rounded-xl border border-slate-700 overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700">
                  <SortableHeader label="Alias" sortKey="key_alias" currentSort={keysSortConfig} onSort={requestKeysSort} />
                  <SortableHeader label="Algorithm" sortKey="algorithm" currentSort={keysSortConfig} onSort={requestKeysSort} />
                  <SortableHeader label="Status" sortKey="status" currentSort={keysSortConfig} onSort={requestKeysSort} />
                  <SortableHeader label="Rotation #" sortKey="rotation_number" currentSort={keysSortConfig} onSort={requestKeysSort} align="right" />
                  <SortableHeader label="Activated" sortKey="activated_at" currentSort={keysSortConfig} onSort={requestKeysSort} />
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Rotation Due</th>
                  <th className="text-right px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700">
                {sortedKeys.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="px-4 py-8 text-center text-slate-500">No encryption keys found</td>
                  </tr>
                ) : (
                  sortedKeys.map((key) => {
                    const days = daysSinceActivation(key.activated_at);
                    const overdue = days > 90;
                    return (
                      <tr key={key.id} className="bg-slate-800 hover:bg-slate-750 transition-colors">
                        <td className="px-4 py-3 text-sm text-slate-200 font-medium font-mono">{key.key_alias}</td>
                        <td className="px-4 py-3 text-sm text-slate-300">{key.algorithm}</td>
                        <td className="px-4 py-3 text-sm">
                          <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                            key.status === "active"
                              ? "bg-green-600/20 text-green-400"
                              : "bg-slate-600/40 text-slate-400"
                          }`}>
                            {key.status}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-sm text-slate-300 text-right font-mono">{key.rotation_number}</td>
                        <td className="px-4 py-3 text-sm text-slate-400">
                          {new Date(key.activated_at).toLocaleDateString()}
                        </td>
                        <td className="px-4 py-3 text-sm">
                          <span className={`font-medium ${overdue ? "text-red-400" : "text-slate-300"}`}>
                            {days}d ago {overdue && "(overdue)"}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-right">
                          <button
                            onClick={() => rotateKey(key.id, key.key_alias)}
                            disabled={rotatingKeyId === key.id}
                            className="px-3 py-1.5 text-xs font-medium rounded-md bg-blue-600 hover:bg-blue-700 text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            {rotatingKeyId === key.id ? "Rotating..." : "Rotate Key"}
                          </button>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </section>

        {/* 2. Sensitive Data */}
        <section className="mb-6">
          <h2 className="text-lg font-semibold text-slate-200 mb-4">Sensitive Data</h2>
          <div className="bg-slate-800 rounded-xl border border-slate-700 overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700">
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Label</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Data Type</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Masked Value</th>
                  <th className="text-right px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700">
                {sensitiveData.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="px-4 py-8 text-center text-slate-500">No sensitive data entries</td>
                  </tr>
                ) : (
                  sensitiveData.map((d) => (
                    <tr key={d.id} className="bg-slate-800 hover:bg-slate-750 transition-colors">
                      <td className="px-4 py-3 text-sm text-slate-200 font-medium">{d.label}</td>
                      <td className="px-4 py-3 text-sm text-slate-300">{d.data_type}</td>
                      <td className="px-4 py-3 text-sm font-mono">
                        {d.revealed_value ? (
                          <span className="text-yellow-400">{d.revealed_value}</span>
                        ) : (
                          <span className="text-slate-500">{d.masked_value}</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-right">
                        {!d.revealed_value && (
                          <button
                            onClick={() => setRevealModal({ id: d.id, label: d.label })}
                            className="px-3 py-1.5 text-xs font-medium rounded-md bg-yellow-600 hover:bg-yellow-700 text-white transition-colors"
                          >
                            Reveal
                          </button>
                        )}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>

        {/* 3. Audit Ledger */}
        <section className="mb-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-slate-200">Audit Ledger</h2>
            <div className="flex items-center gap-3">
              {chainStatus && (
                <span className={`px-3 py-1 rounded-full text-xs font-medium ${
                  chainStatus.valid
                    ? "bg-green-600/20 text-green-400 border border-green-600/40"
                    : "bg-red-600/20 text-red-400 border border-red-600/40"
                }`}>
                  Chain: {chainStatus.valid ? "Valid" : `Broken at entry ${chainStatus.broken_at}`}
                  {" "}({chainStatus.entries_checked} checked)
                </span>
              )}
              <button
                onClick={verifyChain}
                disabled={verifyingChain}
                className="px-4 py-2 text-sm font-medium rounded-lg bg-blue-600 hover:bg-blue-700 text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {verifyingChain ? "Verifying..." : "Verify Chain"}
              </button>
            </div>
          </div>
          <div className="bg-slate-800 rounded-xl border border-slate-700 overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700">
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Timestamp</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Action</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Actor</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Resource</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Details</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700">
                {auditEntries.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-4 py-8 text-center text-slate-500">No audit entries</td>
                  </tr>
                ) : (
                  auditEntries.map((entry) => (
                    <tr key={entry.id} className="bg-slate-800 hover:bg-slate-750 transition-colors">
                      <td className="px-4 py-3 text-sm text-slate-400">
                        {new Date(entry.created_at).toLocaleString()}
                      </td>
                      <td className="px-4 py-3 text-sm text-slate-200 font-medium">{entry.action}</td>
                      <td className="px-4 py-3 text-sm text-slate-300 font-mono">{entry.user_id?.slice(0, 8) || "-"}</td>
                      <td className="px-4 py-3 text-sm text-slate-300 font-mono">{entry.resource_type}/{entry.resource_id}</td>
                      <td className="px-4 py-3 text-sm text-slate-400 max-w-[300px] truncate">{typeof entry.details === 'object' ? JSON.stringify(entry.details) : entry.details}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>

        {/* 4. Rate Limit Status */}
        <section className="mb-6">
          <h2 className="text-lg font-semibold text-slate-200 mb-4">Rate Limit Status</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {rateLimits.length === 0 ? (
              <div className="col-span-full bg-slate-800 rounded-xl border border-slate-700 p-6 text-center text-slate-500">
                No rate limits configured
              </div>
            ) : (
              rateLimits.map((rl, idx) => {
                const isLow = rl.requests_remaining < 10;
                return (
                  <div key={idx} className="bg-slate-800 rounded-xl border border-slate-700 p-6">
                    <p className="text-sm font-medium text-slate-200 font-mono mb-2">{rl.identifier}</p>
                    <div className="flex justify-between text-xs text-slate-400 mb-2">
                      <span>Type: {rl.type}</span>
                      <span>Resets: {new Date(rl.window_reset).toLocaleTimeString()}</span>
                    </div>
                    <p className={`text-sm font-medium ${isLow ? "text-red-400" : "text-slate-300"}`}>
                      {rl.requests_remaining} requests remaining
                    </p>
                  </div>
                );
              })
            )}
          </div>
        </section>

        {/* 5. Retention Policies */}
        <section className="mb-6">
          <h2 className="text-lg font-semibold text-slate-200 mb-4">Retention Policies</h2>
          <div className="bg-slate-800 rounded-xl border border-slate-700 overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700">
                  <SortableHeader label="Table" sortKey="table_name" currentSort={retentionSortConfig} onSort={requestRetentionSort} />
                  <SortableHeader label="Retention (years)" sortKey="retention_years" currentSort={retentionSortConfig} onSort={requestRetentionSort} align="right" />
                  <SortableHeader label="Last Purge" sortKey="last_purge_at" currentSort={retentionSortConfig} onSort={requestRetentionSort} />
                  <SortableHeader label="Next Purge" sortKey="next_purge_at" currentSort={retentionSortConfig} onSort={requestRetentionSort} />
                  <SortableHeader label="Active" sortKey="is_active" currentSort={retentionSortConfig} onSort={requestRetentionSort} />
                  <th className="text-right px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700">
                {sortedRetentionPolicies.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-4 py-8 text-center text-slate-500">No retention policies</td>
                  </tr>
                ) : (
                  sortedRetentionPolicies.map((p) => (
                    <tr key={p.id} className="bg-slate-800 hover:bg-slate-750 transition-colors">
                      <td className="px-4 py-3 text-sm text-slate-300 font-mono">{p.table_name}</td>
                      <td className="px-4 py-3 text-sm text-slate-300 text-right">{p.retention_years}</td>
                      <td className="px-4 py-3 text-sm text-slate-400">
                        {p.last_purge_at ? new Date(p.last_purge_at).toLocaleString() : "Never"}
                      </td>
                      <td className="px-4 py-3 text-sm text-slate-400">
                        {p.next_purge_at ? new Date(p.next_purge_at).toLocaleString() : "N/A"}
                      </td>
                      <td className="px-4 py-3 text-sm">
                        <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${p.is_active ? "bg-green-600/20 text-green-400" : "bg-slate-600/40 text-slate-400"}`}>
                          {p.is_active ? "Active" : "Inactive"}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-right">
                        <button
                          onClick={() => runCleanup(p.id)}
                          disabled={cleaningPolicyId === p.id}
                          className="px-3 py-1.5 text-xs font-medium rounded-md bg-red-600 hover:bg-red-700 text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                          {cleaningPolicyId === p.id ? "Cleaning..." : "Run Cleanup"}
                        </button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>

        {/* Reveal Confirmation Modal */}
        {revealModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
            <div className="bg-slate-800 rounded-xl border border-slate-700 p-6 w-full max-w-md mx-4">
              <h2 className="text-lg font-semibold text-slate-100 mb-2">Confirm Reveal</h2>
              <p className="text-sm text-slate-400 mb-6">
                You are about to reveal sensitive data for <span className="text-slate-200 font-medium">"{revealModal.label}"</span>.
                This action will be logged in the audit ledger.
              </p>
              <div className="flex justify-end gap-3">
                <button
                  onClick={() => setRevealModal(null)}
                  className="px-4 py-2 text-sm font-medium rounded-lg bg-slate-700 hover:bg-slate-600 text-slate-300 transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={() => revealSensitive(revealModal.id)}
                  disabled={revealing}
                  className="px-4 py-2 text-sm font-medium rounded-lg bg-yellow-600 hover:bg-yellow-700 text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {revealing ? "Revealing..." : "Reveal"}
                </button>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
