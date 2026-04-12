import { useState, useEffect, useCallback } from "react";
import axios from "axios";
import Sidebar from "../components/Sidebar";
import ExportButton from "../components/ExportButton";
import { SortableHeader, useSortableData } from "../components/SortableHeader";

interface Exception {
  id: string;
  exception_type: string;
  severity: string;
  amount: number;
  variance_amount: number;
  description: string;
  assigned_to: string;
  status: string;
  disposition?: string;
  resolution_notes?: string;
  created_at: string;
  transaction_id?: string;
  transaction_details?: {
    reference: string;
    amount: number;
    date: string;
    source: string;
  };
}

interface ReconciliationSummary {
  total_open: number;
  unmatched_items: number;
  suspected_duplicates: number;
  variance_alerts: number;
  match_rate: number;
}

const SEVERITY_COLORS: Record<string, string> = {
  low: "bg-slate-600/40 text-slate-300",
  medium: "bg-yellow-600/20 text-yellow-400",
  high: "bg-orange-600/20 text-orange-400",
  critical: "bg-red-600/20 text-red-400",
};

const STATUS_COLORS: Record<string, string> = {
  open: "bg-red-600/20 text-red-400",
  in_progress: "bg-blue-600/20 text-blue-400",
  resolved: "bg-green-600/20 text-green-400",
  dismissed: "bg-slate-600/40 text-slate-400",
};

const DISPOSITIONS = [
  { value: "matched_manually", label: "Matched Manually" },
  { value: "written_off", label: "Written Off" },
  { value: "corrected", label: "Corrected" },
  { value: "duplicate_confirmed", label: "Duplicate Confirmed" },
  { value: "false_positive", label: "False Positive" },
];

export default function OpenExceptions() {
  const [exceptions, setExceptions] = useState<Exception[]>([]);
  const [summary, setSummary] = useState<ReconciliationSummary>({
    total_open: 0,
    unmatched_items: 0,
    suspected_duplicates: 0,
    variance_alerts: 0,
    match_rate: 0,
  });
  const [loading, setLoading] = useState(true);

  // Filters
  const [typeFilter, setTypeFilter] = useState("");
  const [severityFilter, setSeverityFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [assignedFilter, setAssignedFilter] = useState("");

  // Detail panel
  const [selectedExc, setSelectedExc] = useState<Exception | null>(null);
  const [assignTo, setAssignTo] = useState("");
  const [disposition, setDisposition] = useState("");
  const [resolveNotes, setResolveNotes] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [excRes, sumRes] = await Promise.all([
        axios.get("/api/reconciliation/exceptions").catch(() => ({ data: [] })),
        axios.get("/api/reconciliation/summary").catch(() => ({
          data: { total_open: 0, unmatched_items: 0, suspected_duplicates: 0, variance_alerts: 0, match_rate: 0 },
        })),
      ]);
      const excData = excRes.data?.data ?? excRes.data;
      setExceptions(Array.isArray(excData) ? excData : []);
      setSummary(sumRes.data);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const { sortedItems: sortedExceptions, sortConfig, requestSort } = useSortableData(exceptions);

  const filteredExceptions = sortedExceptions.filter((e) => {
    if (typeFilter && e.exception_type !== typeFilter) return false;
    if (severityFilter && e.severity !== severityFilter) return false;
    if (statusFilter && e.status !== statusFilter) return false;
    if (assignedFilter && e.assigned_to !== assignedFilter) return false;
    return true;
  });

  const exceptionTypes = [...new Set(exceptions.map((e) => e.exception_type))];
  const severities = ["low", "medium", "high", "critical"];
  const statuses = ["open", "in_progress", "resolved", "dismissed"];
  const assignees = [...new Set(exceptions.map((e) => e.assigned_to).filter(Boolean))];

  const handleAssign = async () => {
    if (!selectedExc || !assignTo) return;
    setSubmitting(true);
    try {
      await axios.put(`/api/reconciliation/exceptions/${selectedExc.id}/assign`, {
        assigned_to: assignTo,
      });
      fetchData();
    } catch {
      alert("Failed to assign exception.");
    } finally {
      setSubmitting(false);
    }
  };

  const handleResolve = async () => {
    if (!selectedExc || !disposition) return;
    setSubmitting(true);
    try {
      await axios.put(`/api/reconciliation/exceptions/${selectedExc.id}/resolve`, {
        disposition,
        resolution_notes: resolveNotes,
      });
      setSelectedExc(null);
      fetchData();
    } catch {
      alert("Failed to resolve exception.");
    } finally {
      setSubmitting(false);
    }
  };

  const truncateId = (id: string) => id.length > 8 ? id.slice(0, 8) + "..." : id;

  return (
    <div className="flex h-screen bg-slate-950">
      <Sidebar />
      <main className="flex-1 overflow-auto p-6">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-slate-100">Open Exceptions</h1>
            <p className="text-slate-400 mt-1">Financial reconciliation exception management</p>
          </div>
          <div className="flex gap-2">
            <ExportButton
              url="/api/reconciliation/exceptions/export?format=csv"
              filename="exceptions.csv"
              format="csv"
              label="CSV Export"
            />
            <ExportButton
              url="/api/reconciliation/exceptions/export?format=xlsx"
              filename="exceptions.xlsx"
              format="excel"
              label="Excel Export"
            />
          </div>
        </div>

        {/* Summary Stats */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
          <div className="bg-slate-800 rounded-xl border border-slate-700 p-6">
            <p className="text-xs font-medium text-slate-400 uppercase tracking-wider">Total Open</p>
            <p className={`text-2xl font-bold mt-1 ${summary.total_open > 0 ? "text-red-400" : "text-slate-100"}`}>
              {summary.total_open}
            </p>
          </div>
          <div className="bg-slate-800 rounded-xl border border-slate-700 p-6">
            <p className="text-xs font-medium text-slate-400 uppercase tracking-wider">Unmatched Items</p>
            <p className="text-2xl font-bold mt-1 text-slate-100">{summary.unmatched_items}</p>
          </div>
          <div className="bg-slate-800 rounded-xl border border-slate-700 p-6">
            <p className="text-xs font-medium text-slate-400 uppercase tracking-wider">Suspected Duplicates</p>
            <p className="text-2xl font-bold mt-1 text-slate-100">{summary.suspected_duplicates}</p>
          </div>
          <div className="bg-slate-800 rounded-xl border border-slate-700 p-6">
            <p className="text-xs font-medium text-slate-400 uppercase tracking-wider">Variance Alerts</p>
            <p className={`text-2xl font-bold mt-1 ${summary.variance_alerts > 0 ? "text-red-400" : "text-slate-100"}`}>
              {summary.variance_alerts}
            </p>
            <p className="text-xs text-slate-500 mt-1">{">"} $1.00 variance</p>
          </div>
        </div>

        {/* Filters */}
        <div className="flex flex-wrap gap-3 mb-4">
          <select
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value)}
            className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300 focus:outline-none focus:border-blue-500"
          >
            <option value="">All Types</option>
            {exceptionTypes.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </select>
          <select
            value={severityFilter}
            onChange={(e) => setSeverityFilter(e.target.value)}
            className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300 focus:outline-none focus:border-blue-500"
          >
            <option value="">All Severities</option>
            {severities.map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300 focus:outline-none focus:border-blue-500"
          >
            <option value="">All Statuses</option>
            {statuses.map((s) => (
              <option key={s} value={s}>{s.replace(/_/g, " ")}</option>
            ))}
          </select>
          <select
            value={assignedFilter}
            onChange={(e) => setAssignedFilter(e.target.value)}
            className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300 focus:outline-none focus:border-blue-500"
          >
            <option value="">All Assignees</option>
            {assignees.map((a) => (
              <option key={a} value={a}>{a}</option>
            ))}
          </select>
        </div>

        {/* Exception Table */}
        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin w-8 h-8 border-2 border-slate-600 border-t-blue-500 rounded-full" />
          </div>
        ) : (
          <div className="bg-slate-800 rounded-xl border border-slate-700 overflow-hidden mb-6">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-slate-700">
                    <SortableHeader label="ID" sortKey="id" currentSort={sortConfig} onSort={requestSort} />
                    <SortableHeader label="Type" sortKey="exception_type" currentSort={sortConfig} onSort={requestSort} />
                    <SortableHeader label="Severity" sortKey="severity" currentSort={sortConfig} onSort={requestSort} />
                    <SortableHeader label="Amount" sortKey="amount" currentSort={sortConfig} onSort={requestSort} align="right" />
                    <SortableHeader label="Variance" sortKey="variance_amount" currentSort={sortConfig} onSort={requestSort} align="right" />
                    <SortableHeader label="Description" sortKey="description" currentSort={sortConfig} onSort={requestSort} />
                    <SortableHeader label="Assigned To" sortKey="assigned_to" currentSort={sortConfig} onSort={requestSort} />
                    <SortableHeader label="Status" sortKey="status" currentSort={sortConfig} onSort={requestSort} />
                    <SortableHeader label="Created" sortKey="created_at" currentSort={sortConfig} onSort={requestSort} />
                    <th className="text-right px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-700">
                  {filteredExceptions.length === 0 ? (
                    <tr>
                      <td colSpan={10} className="px-4 py-12 text-center text-slate-500">
                        No exceptions found
                      </td>
                    </tr>
                  ) : (
                    filteredExceptions.map((exc) => (
                      <tr
                        key={exc.id}
                        onClick={() => {
                          setSelectedExc(exc);
                          setAssignTo(exc.assigned_to || "");
                          setDisposition("");
                          setResolveNotes("");
                        }}
                        className="bg-slate-800 hover:bg-slate-750 transition-colors cursor-pointer"
                      >
                        <td className="px-4 py-3 text-sm text-slate-400 font-mono">{truncateId(exc.id)}</td>
                        <td className="px-4 py-3 text-sm text-slate-300">{exc.exception_type}</td>
                        <td className="px-4 py-3 text-sm">
                          <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${SEVERITY_COLORS[exc.severity] || ""}`}>
                            {exc.severity}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-sm text-slate-200 text-right font-mono">
                          ${exc.amount?.toFixed(2)}
                        </td>
                        <td className="px-4 py-3 text-sm text-right font-mono">
                          <span className={exc.variance_amount > 1 ? "text-red-400" : "text-slate-300"}>
                            ${exc.variance_amount?.toFixed(2)}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-sm text-slate-300 max-w-[200px] truncate">{exc.description}</td>
                        <td className="px-4 py-3 text-sm text-slate-300">{exc.assigned_to || "-"}</td>
                        <td className="px-4 py-3 text-sm">
                          <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[exc.status] || ""}`}>
                            {exc.status?.replace(/_/g, " ")}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-sm text-slate-400">
                          {new Date(exc.created_at).toLocaleDateString()}
                        </td>
                        <td className="px-4 py-3 text-right">
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              setSelectedExc(exc);
                              setAssignTo(exc.assigned_to || "");
                              setDisposition("");
                              setResolveNotes("");
                            }}
                            className="px-3 py-1.5 text-xs font-medium rounded-md bg-slate-700 hover:bg-slate-600 text-slate-300 transition-colors"
                          >
                            View
                          </button>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Reconciliation Summary */}
        <div className="bg-slate-800 rounded-xl border border-slate-700 p-6">
          <h2 className="text-lg font-semibold text-slate-100 mb-4">Reconciliation Summary</h2>
          <div className="flex items-center gap-4 mb-4">
            <p className="text-sm text-slate-300">
              Match Rate: <span className="text-xl font-bold text-emerald-400">{summary.match_rate?.toFixed(1)}%</span>
            </p>
          </div>
          <div className="w-full bg-slate-700 rounded-full h-3">
            <div
              className="h-3 rounded-full bg-emerald-500 transition-all"
              style={{ width: `${summary.match_rate || 0}%` }}
            />
          </div>
          <p className="text-xs text-slate-500 mt-2">Bar chart visualization for detailed breakdown coming soon</p>
        </div>

        {/* Detail Slide-over Panel */}
        {selectedExc && (
          <div className="fixed inset-0 z-50 flex justify-end bg-black/40 backdrop-blur-sm">
            <div
              className="absolute inset-0"
              onClick={() => setSelectedExc(null)}
            />
            <div className="relative w-full max-w-lg bg-slate-900 border-l border-slate-700 overflow-auto">
              <div className="p-6">
                <div className="flex items-start justify-between mb-6">
                  <div>
                    <h2 className="text-lg font-semibold text-slate-100">Exception Details</h2>
                    <p className="text-xs text-slate-400 font-mono mt-1">{selectedExc.id}</p>
                  </div>
                  <button
                    onClick={() => setSelectedExc(null)}
                    className="text-slate-400 hover:text-slate-200 text-xl leading-none"
                  >
                    x
                  </button>
                </div>

                {/* Exception Info */}
                <div className="grid grid-cols-2 gap-4 mb-6">
                  <div>
                    <p className="text-xs text-slate-500">Type</p>
                    <p className="text-sm text-slate-200">{selectedExc.exception_type}</p>
                  </div>
                  <div>
                    <p className="text-xs text-slate-500">Severity</p>
                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${SEVERITY_COLORS[selectedExc.severity] || ""}`}>
                      {selectedExc.severity}
                    </span>
                  </div>
                  <div>
                    <p className="text-xs text-slate-500">Amount</p>
                    <p className="text-sm text-slate-200 font-mono">${selectedExc.amount?.toFixed(2)}</p>
                  </div>
                  <div>
                    <p className="text-xs text-slate-500">Variance</p>
                    <p className={`text-sm font-mono ${selectedExc.variance_amount > 1 ? "text-red-400" : "text-slate-200"}`}>
                      ${selectedExc.variance_amount?.toFixed(2)}
                    </p>
                  </div>
                  <div className="col-span-2">
                    <p className="text-xs text-slate-500">Description</p>
                    <p className="text-sm text-slate-200">{selectedExc.description}</p>
                  </div>
                  <div>
                    <p className="text-xs text-slate-500">Status</p>
                    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[selectedExc.status] || ""}`}>
                      {selectedExc.status?.replace(/_/g, " ")}
                    </span>
                  </div>
                  <div>
                    <p className="text-xs text-slate-500">Created</p>
                    <p className="text-sm text-slate-200">{new Date(selectedExc.created_at).toLocaleString()}</p>
                  </div>
                </div>

                {/* Transaction Details */}
                {selectedExc.transaction_details && (
                  <div className="mb-6">
                    <h3 className="text-sm font-semibold text-slate-200 mb-3">Linked Transaction</h3>
                    <div className="bg-slate-800 rounded-lg border border-slate-700 p-4 space-y-2">
                      <div className="flex justify-between">
                        <span className="text-xs text-slate-500">Reference</span>
                        <span className="text-sm text-slate-200 font-mono">{selectedExc.transaction_details.reference}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-xs text-slate-500">Amount</span>
                        <span className="text-sm text-slate-200 font-mono">${selectedExc.transaction_details.amount?.toFixed(2)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-xs text-slate-500">Date</span>
                        <span className="text-sm text-slate-200">{selectedExc.transaction_details.date}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-xs text-slate-500">Source</span>
                        <span className="text-sm text-slate-200">{selectedExc.transaction_details.source}</span>
                      </div>
                    </div>
                  </div>
                )}

                {/* Assignment */}
                <div className="mb-6">
                  <h3 className="text-sm font-semibold text-slate-200 mb-3">Assignment</h3>
                  <div className="flex gap-2">
                    <select
                      value={assignTo}
                      onChange={(e) => setAssignTo(e.target.value)}
                      className="flex-1 px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300 focus:outline-none focus:border-blue-500"
                    >
                      <option value="">Select assignee...</option>
                      {assignees.map((a) => (
                        <option key={a} value={a}>{a}</option>
                      ))}
                    </select>
                    <button
                      onClick={handleAssign}
                      disabled={submitting || !assignTo}
                      className="px-4 py-2 text-sm font-medium rounded-lg bg-blue-600 hover:bg-blue-700 text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Assign
                    </button>
                  </div>
                </div>

                {/* Resolution */}
                <div>
                  <h3 className="text-sm font-semibold text-slate-200 mb-3">Resolution</h3>
                  <select
                    value={disposition}
                    onChange={(e) => setDisposition(e.target.value)}
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300 focus:outline-none focus:border-blue-500 mb-3"
                  >
                    <option value="">Select disposition...</option>
                    {DISPOSITIONS.map((d) => (
                      <option key={d.value} value={d.value}>{d.label}</option>
                    ))}
                  </select>
                  <textarea
                    value={resolveNotes}
                    onChange={(e) => setResolveNotes(e.target.value)}
                    rows={3}
                    placeholder="Resolution notes..."
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:border-blue-500 resize-none mb-3"
                  />
                  <button
                    onClick={handleResolve}
                    disabled={submitting || !disposition}
                    className="w-full px-4 py-2 text-sm font-medium rounded-lg bg-green-600 hover:bg-green-700 text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {submitting ? "Resolving..." : "Resolve Exception"}
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
