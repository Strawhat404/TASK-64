import { useState, useEffect, useCallback } from "react";
import axios from "axios";
import Sidebar from "../components/Sidebar";
import { SortableHeader, useSortableData } from "../components/SortableHeader";

interface AuditEntry {
  id: string;
  created_at: string;
  user_id: number;
  action: string;
  resource_type: string;
  resource_id: string | null;
  details: string;
  ip_address: string;
}

export default function AuditLogs() {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const pageSize = 20;

  const [filters, setFilters] = useState({
    user: "",
    action: "",
    dateFrom: "",
    dateTo: "",
  });

  const fetchLogs = useCallback(async () => {
    try {
      const params: Record<string, string | number> = {
        page,
        per_page: pageSize,
      };
      if (filters.user) params.user_id = filters.user;
      if (filters.action) params.action = filters.action;
      if (filters.dateFrom) params.start_date = filters.dateFrom;
      if (filters.dateTo) params.end_date = filters.dateTo;

      const res = await axios.get("/api/audit/logs", { params });
      if (res.data && Array.isArray(res.data.data)) {
        setEntries(res.data.data);
        setTotal(res.data.total || 0);
      } else if (Array.isArray(res.data)) {
        setEntries(res.data);
        setTotal(res.data.length);
      } else {
        setEntries([]);
        setTotal(0);
      }
    } catch {
      setEntries([]);
      setTotal(0);
    }
  }, [page, filters]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const { sortedItems: sortedEntries, sortConfig: auditSortConfig, requestSort: requestAuditSort } = useSortableData(entries);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  const handleFilterSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    fetchLogs();
  };

  const actionColors: Record<string, string> = {
    login: "text-green-400",
    logout: "text-slate-400",
    create: "text-blue-400",
    update: "text-yellow-400",
    delete: "text-red-400",
    login_failed: "text-red-400",
  };

  return (
    <div className="flex min-h-screen bg-slate-950">
      <Sidebar />
      <main className="flex-1 p-6 overflow-auto">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-slate-100">Audit Logs</h1>
          <p className="text-slate-400 mt-1">Review system activity and compliance trail</p>
        </div>

        {/* Filters */}
        <form
          onSubmit={handleFilterSubmit}
          className="bg-slate-900 border border-slate-800 rounded-xl p-4 mb-6"
        >
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4 items-end">
            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1">User</label>
              <input
                type="text"
                value={filters.user}
                onChange={(e) => setFilters({ ...filters, user: e.target.value })}
                placeholder="Filter by username"
                className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-sm text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1">Action</label>
              <select
                value={filters.action}
                onChange={(e) => setFilters({ ...filters, action: e.target.value })}
                className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-sm text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">All Actions</option>
                <option value="login">Login</option>
                <option value="logout">Logout</option>
                <option value="create">Create</option>
                <option value="update">Update</option>
                <option value="delete">Delete</option>
                <option value="login_failed">Login Failed</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1">Date From</label>
              <input
                type="date"
                value={filters.dateFrom}
                onChange={(e) => setFilters({ ...filters, dateFrom: e.target.value })}
                className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-sm text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-400 mb-1">Date To</label>
              <input
                type="date"
                value={filters.dateTo}
                onChange={(e) => setFilters({ ...filters, dateTo: e.target.value })}
                className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-sm text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div className="flex gap-2">
              <button
                type="submit"
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
              >
                Filter
              </button>
              <button
                type="button"
                onClick={() => {
                  setFilters({ user: "", action: "", dateFrom: "", dateTo: "" });
                  setPage(1);
                }}
                className="px-4 py-2 text-sm text-slate-400 hover:text-slate-200 transition-colors"
              >
                Clear
              </button>
            </div>
          </div>
        </form>

        {/* Table */}
        <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-slate-700">
                <SortableHeader label="Timestamp" sortKey="created_at" currentSort={auditSortConfig} onSort={requestAuditSort} />
                <SortableHeader label="User" sortKey="user_id" currentSort={auditSortConfig} onSort={requestAuditSort} />
                <SortableHeader label="Action" sortKey="action" currentSort={auditSortConfig} onSort={requestAuditSort} />
                <SortableHeader label="Resource" sortKey="resource_type" currentSort={auditSortConfig} onSort={requestAuditSort} />
                <SortableHeader label="Details" sortKey="details" currentSort={auditSortConfig} onSort={requestAuditSort} />
                <SortableHeader label="IP Address" sortKey="ip_address" currentSort={auditSortConfig} onSort={requestAuditSort} />
              </tr>
            </thead>
            <tbody>
              {sortedEntries.length === 0 && (
                <tr>
                  <td colSpan={6} className="p-8 text-center text-slate-500">
                    No audit log entries found.
                  </td>
                </tr>
              )}
              {sortedEntries.map((entry) => (
                <tr
                  key={entry.id}
                  className="border-b border-slate-800 bg-slate-800/50 hover:bg-slate-800 transition-colors"
                >
                  <td className="p-4 text-sm text-slate-300">
                    {new Date(entry.created_at).toLocaleString()}
                  </td>
                  <td className="p-4 text-sm text-slate-200 font-medium">{entry.user_id}</td>
                  <td className="p-4">
                    <span
                      className={`text-sm font-medium ${actionColors[entry.action] || "text-slate-300"}`}
                    >
                      {entry.action.replace("_", " ")}
                    </span>
                  </td>
                  <td className="p-4 text-sm text-slate-300">
                    {entry.resource_type}
                    {entry.resource_id ? ` #${entry.resource_id}` : ""}
                  </td>
                  <td className="p-4 text-sm text-slate-400 max-w-xs truncate">
                    {entry.details}
                  </td>
                  <td className="p-4 text-sm text-slate-500 font-mono">{entry.ip_address}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <div className="flex items-center justify-between mt-4">
          <p className="text-sm text-slate-500">
            Showing {entries.length} of {total} entries
          </p>
          <div className="flex gap-2">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="px-3 py-1.5 text-sm bg-slate-800 border border-slate-700 rounded-lg text-slate-300 hover:bg-slate-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            >
              Previous
            </button>
            <span className="px-3 py-1.5 text-sm text-slate-400">
              Page {page} of {totalPages}
            </span>
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className="px-3 py-1.5 text-sm bg-slate-800 border border-slate-700 rounded-lg text-slate-300 hover:bg-slate-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            >
              Next
            </button>
          </div>
        </div>
      </main>
    </div>
  );
}
