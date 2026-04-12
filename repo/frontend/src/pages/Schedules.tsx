import { useState, useEffect } from "react";
import axios from "axios";
import Sidebar from "../components/Sidebar";
import { SortableHeader, useSortableData } from "../components/SortableHeader";

interface Schedule {
  id: string;
  service_id: string;
  staff_id: string;
  client_name: string;
  scheduled_start: string;
  scheduled_end: string;
  status: string;
  requires_confirmation: boolean;
  confirmed_at?: string;
  reassignment_reason?: string;
}

interface Service {
  id: string;
  name: string;
  duration_minutes: number;
}

interface Staff {
  id: string;
  full_name: string;
  is_available: boolean;
}

const statusColors: Record<string, string> = {
  pending: "bg-yellow-900/50 text-yellow-400",
  confirmed: "bg-green-900/50 text-green-400",
  in_progress: "bg-blue-900/50 text-blue-400",
  completed: "bg-slate-700 text-slate-300",
  cancelled: "bg-red-900/50 text-red-400",
};

export default function Schedules() {
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [services, setServices] = useState<Service[]>([]);
  const [staff, setStaff] = useState<Staff[]>([]);
  const [selectedDate, setSelectedDate] = useState(
    new Date().toISOString().split("T")[0]
  );
  const [showForm, setShowForm] = useState(false);
  const [conflict, setConflict] = useState("");
  const [error, setError] = useState("");
  const { sortedItems: sortedSchedules, sortConfig, requestSort } = useSortableData(schedules);

  const [backupForm, setBackupForm] = useState<{ scheduleId: string; backup_staff_id: string; reason_code: string; notes: string } | null>(null);

  const [form, setForm] = useState({
    service_id: "",
    staff_id: "",
    client_name: "",
    scheduled_start: new Date().toISOString().split("T")[0] + "T09:00",
    scheduled_end: new Date().toISOString().split("T")[0] + "T10:00",
  });

  const fetchSchedules = async () => {
    try {
      const res = await axios.get(`/api/schedules?start_date=${selectedDate}&end_date=${selectedDate}`);
      setSchedules(Array.isArray(res.data) ? res.data : []);
    } catch {
      setSchedules([]);
    }
  };

  const fetchMeta = async () => {
    try {
      const [svcRes, staffRes] = await Promise.all([
        axios.get("/api/services"),
        axios.get("/api/staff"),
      ]);
      setServices(Array.isArray(svcRes.data) ? svcRes.data : []);
      setStaff(Array.isArray(staffRes.data) ? staffRes.data : []);
    } catch {
      /* ignore */
    }
  };

  useEffect(() => {
    fetchSchedules();
  }, [selectedDate]);

  useEffect(() => {
    fetchMeta();
  }, []);

  // Check for conflicts when staff or time changes
  useEffect(() => {
    if (!form.staff_id || !form.scheduled_start) {
      setConflict("");
      return;
    }
    const formDate = form.scheduled_start.split("T")[0];
    const existing = schedules.filter(
      (s) =>
        s.staff_id === form.staff_id &&
        s.scheduled_start.split("T")[0] === formDate &&
        s.status !== "cancelled"
    );
    if (existing.length > 0) {
      setConflict(
        `Warning: This staff member already has ${existing.length} schedule(s) on this date.`
      );
    } else {
      setConflict("");
    }
  }, [form.staff_id, form.scheduled_start, schedules]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    try {
      await axios.post("/api/schedules", {
        ...form,
        scheduled_start: new Date(form.scheduled_start).toISOString(),
        scheduled_end: new Date(form.scheduled_end).toISOString(),
      });
      setShowForm(false);
      setForm({
        service_id: "",
        staff_id: "",
        client_name: "",
        scheduled_start: selectedDate + "T09:00",
        scheduled_end: selectedDate + "T10:00",
      });
      fetchSchedules();
    } catch (err: unknown) {
      setError(
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ||
          "Failed to create schedule"
      );
    }
  };

  const handleBackup = async () => {
    if (!backupForm) return;
    try {
      await axios.post(`/api/schedules/${backupForm.scheduleId}/backup`, {
        backup_staff_id: backupForm.backup_staff_id,
        reason_code: backupForm.reason_code,
        notes: backupForm.notes,
      });
      setBackupForm(null);
      fetchSchedules();
    } catch (err: any) {
      setError(err?.response?.data?.error || "Failed to request backup");
    }
  };

  const reasonCodes = ["sick_leave", "emergency", "no_show", "schedule_conflict", "requested", "other"];

  const updateStatus = async (id: string, status: string) => {
    try {
      if (status === "confirmed") {
        await axios.post(`/api/schedules/${id}/confirm`);
      } else {
        await axios.put(`/api/schedules/${id}`, { status });
      }
      fetchSchedules();
    } catch {
      /* ignore */
    }
  };

  return (
    <div className="flex min-h-screen bg-slate-950">
      <Sidebar />
      <main className="flex-1 p-6 overflow-auto">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-slate-100">Schedules</h1>
            <p className="text-slate-400 mt-1">Manage service schedules and assignments</p>
          </div>
          <div className="flex items-center gap-3">
            <input
              type="date"
              value={selectedDate}
              onChange={(e) => setSelectedDate(e.target.value)}
              className="px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              onClick={() => {
                setShowForm(true);
                setForm((f) => ({ ...f, scheduled_start: selectedDate + "T09:00", scheduled_end: selectedDate + "T10:00" }));
              }}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
            >
              + New Schedule
            </button>
          </div>
        </div>

        {/* Create Form */}
        {showForm && (
          <div className="bg-slate-900 border border-slate-700 rounded-xl p-6 mb-6">
            <h2 className="text-lg font-semibold text-slate-100 mb-4">Create Schedule</h2>
            {error && (
              <div className="mb-4 p-3 bg-red-900/30 border border-red-800 rounded-lg text-sm text-red-300">
                {error}
              </div>
            )}
            {conflict && (
              <div className="mb-4 p-3 bg-yellow-900/30 border border-yellow-800 rounded-lg text-sm text-yellow-300">
                {conflict}
              </div>
            )}
            <form onSubmit={handleCreate} className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Service</label>
                <select
                  value={form.service_id}
                  onChange={(e) => setForm({ ...form, service_id: e.target.value })}
                  required
                  className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="" disabled>Select a service</option>
                  {services.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name} ({s.duration_minutes}min)
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Staff Member</label>
                <select
                  value={form.staff_id}
                  onChange={(e) => setForm({ ...form, staff_id: e.target.value })}
                  required
                  className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="" disabled>Select staff</option>
                  {staff.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.full_name}{" "}
                      {!s.is_available ? "(unavailable)" : ""}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Client Name</label>
                <input
                  type="text"
                  value={form.client_name}
                  onChange={(e) => setForm({ ...form, client_name: e.target.value })}
                  required
                  className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Scheduled Start</label>
                <input
                  type="datetime-local"
                  value={form.scheduled_start}
                  onChange={(e) => setForm({ ...form, scheduled_start: e.target.value })}
                  required
                  className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Scheduled End</label>
                <input
                  type="datetime-local"
                  value={form.scheduled_end}
                  onChange={(e) => setForm({ ...form, scheduled_end: e.target.value })}
                  required
                  className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div className="md:col-span-2 flex gap-3 justify-end">
                <button
                  type="button"
                  onClick={() => setShowForm(false)}
                  className="px-4 py-2 text-sm text-slate-400 hover:text-slate-200 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
                >
                  Create Schedule
                </button>
              </div>
            </form>
          </div>
        )}

        {/* Schedule List */}
        <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-slate-700">
                <SortableHeader label="Service" sortKey="service_id" currentSort={sortConfig} onSort={requestSort} />
                <SortableHeader label="Staff" sortKey="staff_id" currentSort={sortConfig} onSort={requestSort} />
                <SortableHeader label="Date" sortKey="scheduled_start" currentSort={sortConfig} onSort={requestSort} />
                <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Time</th>
                <SortableHeader label="Status" sortKey="status" currentSort={sortConfig} onSort={requestSort} />
                <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody>
              {schedules.length === 0 && (
                <tr>
                  <td colSpan={6} className="p-8 text-center text-slate-500">
                    No schedules for this date.
                  </td>
                </tr>
              )}
              {sortedSchedules.map((sch) => (
                <tr
                  key={sch.id}
                  className="border-b border-slate-800 bg-slate-800/50 hover:bg-slate-800 transition-colors"
                >
                  <td className="p-4 text-sm text-slate-200 font-medium">
                    {`Service #${sch.service_id}`}
                  </td>
                  <td className="p-4 text-sm text-slate-300">
                    {`Staff #${sch.staff_id}`}
                  </td>
                  <td className="p-4 text-sm text-slate-300">{new Date(sch.scheduled_start).toLocaleDateString()}</td>
                  <td className="p-4 text-sm text-slate-300">
                    {new Date(sch.scheduled_start).toLocaleTimeString()} - {new Date(sch.scheduled_end).toLocaleTimeString()}
                  </td>
                  <td className="p-4">
                    <span
                      className={`text-xs px-2 py-1 rounded-full ${statusColors[sch.status] || "bg-slate-700 text-slate-400"}`}
                    >
                      {sch.status.replace("_", " ")}
                    </span>
                  </td>
                  <td className="p-4">
                    <div className="flex gap-2">
                      {sch.status === "pending" && (
                        <button
                          onClick={() => updateStatus(sch.id, "confirmed")}
                          className="text-xs text-green-400 hover:text-green-300 transition-colors"
                        >
                          Confirm
                        </button>
                      )}
                      {sch.status === "confirmed" && (
                        <button
                          onClick={() => updateStatus(sch.id, "in_progress")}
                          className="text-xs text-blue-400 hover:text-blue-300 transition-colors"
                        >
                          Start
                        </button>
                      )}
                      {sch.status === "in_progress" && (
                        <button
                          onClick={() => updateStatus(sch.id, "completed")}
                          className="text-xs text-slate-400 hover:text-slate-300 transition-colors"
                        >
                          Complete
                        </button>
                      )}
                      {sch.status !== "cancelled" && sch.status !== "completed" && (
                        <button
                          onClick={() => updateStatus(sch.id, "cancelled")}
                          className="text-xs text-red-400 hover:text-red-300 transition-colors"
                        >
                          Cancel
                        </button>
                      )}
                      {sch.status !== "cancelled" && sch.status !== "completed" && (
                        <button
                          onClick={() => setBackupForm({ scheduleId: sch.id, backup_staff_id: "", reason_code: "sick_leave", notes: "" })}
                          className="text-xs text-purple-400 hover:text-purple-300 transition-colors"
                        >
                          Backup
                        </button>
                      )}
                    </div>
                    {backupForm && backupForm.scheduleId === sch.id && (
                      <div className="mt-2 p-3 bg-slate-900 border border-slate-600 rounded-lg space-y-2">
                        <div className="flex gap-2 items-end flex-wrap">
                          <div>
                            <label className="block text-xs text-slate-400 mb-1">Backup Staff</label>
                            <select
                              value={backupForm.backup_staff_id}
                              onChange={(e) => setBackupForm({ ...backupForm, backup_staff_id: e.target.value })}
                              required
                              className="px-2 py-1 bg-slate-800 border border-slate-600 rounded text-xs text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500"
                            >
                              <option value="" disabled>Select staff</option>
                              {staff.filter(s => s.id !== sch.staff_id).map((s) => (
                                <option key={s.id} value={s.id}>{s.full_name}{!s.is_available ? " (unavailable)" : ""}</option>
                              ))}
                            </select>
                          </div>
                          <div>
                            <label className="block text-xs text-slate-400 mb-1">Reason</label>
                            <select
                              value={backupForm.reason_code}
                              onChange={(e) => setBackupForm({ ...backupForm, reason_code: e.target.value })}
                              className="px-2 py-1 bg-slate-800 border border-slate-600 rounded text-xs text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500"
                            >
                              {reasonCodes.map((r) => (
                                <option key={r} value={r}>{r.replace(/_/g, " ")}</option>
                              ))}
                            </select>
                          </div>
                          <div className="flex-1">
                            <label className="block text-xs text-slate-400 mb-1">Notes</label>
                            <input
                              type="text"
                              value={backupForm.notes}
                              onChange={(e) => setBackupForm({ ...backupForm, notes: e.target.value })}
                              placeholder="Optional notes..."
                              className="w-full px-2 py-1 bg-slate-800 border border-slate-600 rounded text-xs text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500"
                            />
                          </div>
                          <button
                            onClick={handleBackup}
                            disabled={!backupForm.backup_staff_id}
                            className="px-2 py-1 bg-purple-600 hover:bg-purple-700 disabled:opacity-50 text-white rounded text-xs font-medium transition-colors"
                          >
                            Submit
                          </button>
                          <button
                            onClick={() => setBackupForm(null)}
                            className="px-2 py-1 text-xs text-slate-400 hover:text-slate-200 transition-colors"
                          >
                            Cancel
                          </button>
                        </div>
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </main>
    </div>
  );
}
