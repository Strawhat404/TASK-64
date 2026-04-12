import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import axios from "axios";
import Sidebar from "../components/Sidebar";
import DashboardCard from "../components/DashboardCard";
import { useAuth } from "../context/AuthContext";

interface DashboardStats {
  totalScheduled: number;
  pendingConfirmations: number;
  availableStaff: number;
  completedToday: number;
  openExceptions: number;
  pendingModeration: number;
}

export default function Dashboard() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const [stats, setStats] = useState<DashboardStats>({
    totalScheduled: 0,
    pendingConfirmations: 0,
    availableStaff: 0,
    completedToday: 0,
    openExceptions: 0,
    pendingModeration: 0,
  });
  const [todaySchedules, setTodaySchedules] = useState<any[]>([]);
  const [staffList, setStaffList] = useState<any[]>([]);
  const [filterStatus, setFilterStatus] = useState("");
  const [filterStaff, setFilterStaff] = useState("");
  const [filterDateFrom, setFilterDateFrom] = useState("");
  const [filterDateTo, setFilterDateTo] = useState("");

  const [filterCategory, setFilterCategory] = useState("");
  const [filterTag, setFilterTag] = useState("");
  const [sortKey, setSortKey] = useState<string>("");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("asc");

  const filteredSchedules = todaySchedules.filter((s: any) => {
    if (filterStatus && s.status !== filterStatus) return false;
    if (filterStaff && s.staff_id !== filterStaff) return false;
    if (filterDateFrom) {
      const schedDate = new Date(s.scheduled_start).toISOString().split("T")[0];
      if (schedDate < filterDateFrom) return false;
    }
    if (filterDateTo) {
      const schedDate = new Date(s.scheduled_start).toISOString().split("T")[0];
      if (schedDate > filterDateTo) return false;
    }
    if (filterCategory && s.service_id !== filterCategory) return false;
    if (filterTag) {
      const tags: string[] = s.tags || [];
      if (!tags.some((t: string) => t.toLowerCase().includes(filterTag.toLowerCase()))) return false;
    }
    return true;
  });

  const sortedSchedules = [...filteredSchedules].sort((a: any, b: any) => {
    if (!sortKey) return 0;
    let aVal = sortKey === "client_name" ? a.client_name :
               sortKey === "staff" ? (staffList.find((st: any) => st.id === a.staff_id)?.full_name || a.staff_id) :
               sortKey === "time" ? a.scheduled_start :
               sortKey === "status" ? a.status : "";
    let bVal = sortKey === "client_name" ? b.client_name :
               sortKey === "staff" ? (staffList.find((st: any) => st.id === b.staff_id)?.full_name || b.staff_id) :
               sortKey === "time" ? b.scheduled_start :
               sortKey === "status" ? b.status : "";
    if (aVal < bVal) return sortDir === "asc" ? -1 : 1;
    if (aVal > bVal) return sortDir === "asc" ? 1 : -1;
    return 0;
  });

  const handleSort = (key: string) => {
    if (sortKey === key) {
      setSortDir(sortDir === "asc" ? "desc" : "asc");
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  };

  const getExportData = () => {
    const headers = ["Client", "Staff", "Start", "End", "Status"];
    const rows = sortedSchedules.map((s: any) => [
      s.client_name,
      staffList.find((st: any) => st.id === s.staff_id)?.full_name || s.staff_id,
      new Date(s.scheduled_start).toLocaleString(),
      new Date(s.scheduled_end).toLocaleString(),
      s.status,
    ]);
    return { headers, rows };
  };

  const exportSchedulesCSV = () => {
    const { headers, rows } = getExportData();
    const csv = [headers, ...rows].map(r => r.map((c: string) => `"${c}"`).join(",")).join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "schedules.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  const exportSchedulesExcel = () => {
    const { headers, rows } = getExportData();
    // Generate XML Spreadsheet (Excel-compatible) format
    const escXml = (s: string) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
    let xml = '<?xml version="1.0"?><?mso-application progid="Excel.Sheet"?>';
    xml += '<Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet" xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet">';
    xml += '<Worksheet ss:Name="Schedules"><Table>';
    xml += '<Row>' + headers.map(h => `<Cell><Data ss:Type="String">${escXml(h)}</Data></Cell>`).join('') + '</Row>';
    for (const row of rows) {
      xml += '<Row>' + row.map((c: string) => `<Cell><Data ss:Type="String">${escXml(c)}</Data></Cell>`).join('') + '</Row>';
    }
    xml += '</Table></Worksheet></Workbook>';
    const blob = new Blob([xml], { type: "application/vnd.ms-excel" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "schedules.xls";
    a.click();
    URL.revokeObjectURL(url);
  };

  useEffect(() => {
    const today = new Date().toISOString().split("T")[0];

    Promise.all([
      axios.get(`/api/schedules?start_date=${today}&end_date=${today}`).catch(() => ({ data: [] })),
      axios.get("/api/staff").catch(() => ({ data: [] })),
      axios.get("/api/reconciliation/summary").catch(() => ({ data: { total_open: 0 } })),
      axios.get("/api/governance/reviews/pending").catch(() => ({ data: [] })),
    ]).then(([schedulesRes, staffRes, exceptionsSumRes, moderationRes]) => {
      const schedules = Array.isArray(schedulesRes.data) ? schedulesRes.data : [];
      const staff = Array.isArray(staffRes.data) ? staffRes.data : [];
      const moderationData = Array.isArray(moderationRes.data) ? moderationRes.data : [];

      setStats({
        totalScheduled: schedules.length,
        pendingConfirmations: schedules.filter(
          (s: { status: string }) => s.status === "pending"
        ).length,
        availableStaff: staff.filter(
          (s: { is_available: boolean }) => s.is_available === true
        ).length,
        completedToday: schedules.filter(
          (s: { status: string }) => s.status === "completed"
        ).length,
        openExceptions: exceptionsSumRes.data?.total_open ?? 0,
        pendingModeration: moderationData.length,
      });
      setTodaySchedules(schedules);
      setStaffList(staff);
    });
  }, []);

  return (
    <div className="flex min-h-screen bg-slate-950">
      <Sidebar />
      <main className="flex-1 p-6 overflow-auto">
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-slate-100">
            Welcome back, {user?.full_name || user?.username}
          </h1>
          <p className="text-slate-400 mt-1">
            Role: <span className="capitalize">{user?.role_name}</span> &middot;{" "}
            {new Date().toLocaleDateString("en-US", {
              weekday: "long",
              year: "numeric",
              month: "long",
              day: "numeric",
            })}
          </p>
        </div>

        <section className="mb-8">
          <h2 className="text-lg font-semibold text-slate-200 mb-4">
            Today's Operations
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            <DashboardCard
              title="Total Scheduled"
              value={stats.totalScheduled}
              subtitle="For today"
            />
            <DashboardCard
              title="Pending Confirmations"
              value={stats.pendingConfirmations}
              subtitle="Awaiting response"
              trend={stats.pendingConfirmations > 0 ? "up" : "neutral"}
              trendValue={stats.pendingConfirmations > 0 ? "Needs attention" : "Clear"}
            />
            <DashboardCard
              title="Available Staff"
              value={stats.availableStaff}
              subtitle="Ready to assign"
            />
            <DashboardCard
              title="Completed Today"
              value={stats.completedToday}
              subtitle="Finished operations"
              trend="up"
              trendValue={`${stats.totalScheduled > 0 ? Math.round((stats.completedToday / stats.totalScheduled) * 100) : 0}%`}
            />
            <DashboardCard
              title="Open Exceptions"
              value={stats.openExceptions}
              subtitle="Reconciliation exceptions"
              trend={stats.openExceptions > 0 ? "up" : "neutral"}
              trendValue={stats.openExceptions > 0 ? "Needs attention" : "Clear"}
            />
            <DashboardCard
              title="Moderation Queue"
              value={stats.pendingModeration}
              subtitle="Pending reviews"
              trend={stats.pendingModeration > 0 ? "up" : "neutral"}
              trendValue={stats.pendingModeration > 0 ? "Needs attention" : "Clear"}
            />
          </div>
        </section>

        <section>
          <h2 className="text-lg font-semibold text-slate-200 mb-4">
            Quick Actions
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <button
              onClick={() => navigate("/schedules")}
              className="p-4 bg-slate-800 border border-slate-700 rounded-xl text-left hover:bg-slate-750 hover:border-slate-600 transition-colors"
            >
              <span className="text-2xl mb-2 block">+</span>
              <span className="text-sm font-medium text-slate-200">
                New Schedule
              </span>
              <p className="text-xs text-slate-500 mt-1">
                Create a new service schedule
              </p>
            </button>
            <button
              onClick={() => navigate("/services")}
              className="p-4 bg-slate-800 border border-slate-700 rounded-xl text-left hover:bg-slate-750 hover:border-slate-600 transition-colors"
            >
              <span className="text-2xl mb-2 block">\u2630</span>
              <span className="text-sm font-medium text-slate-200">
                Service Catalog
              </span>
              <p className="text-xs text-slate-500 mt-1">
                View and manage services
              </p>
            </button>
            <button
              onClick={() => navigate("/staff")}
              className="p-4 bg-slate-800 border border-slate-700 rounded-xl text-left hover:bg-slate-750 hover:border-slate-600 transition-colors"
            >
              <span className="text-2xl mb-2 block">\u263A</span>
              <span className="text-sm font-medium text-slate-200">
                Staff Roster
              </span>
              <p className="text-xs text-slate-500 mt-1">
                Manage staff and availability
              </p>
            </button>
            <button
              onClick={() => navigate("/schedules")}
              className="p-4 bg-slate-800 border border-slate-700 rounded-xl text-left hover:bg-slate-750 hover:border-slate-600 transition-colors"
            >
              <span className="text-2xl mb-2 block">\u25D2</span>
              <span className="text-sm font-medium text-slate-200">
                View Schedule
              </span>
              <p className="text-xs text-slate-500 mt-1">
                Check today's schedule
              </p>
            </button>
          </div>
        </section>

        {/* Today's Schedule Overview with Multi-Condition Filtering */}
        <section className="mt-8">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-slate-200">Today's Schedule Overview</h2>
            <div className="flex gap-2">
              <button onClick={exportSchedulesCSV} className="px-4 py-2 text-sm font-medium rounded-lg bg-slate-700 hover:bg-slate-600 text-slate-300 transition-colors">Export CSV</button>
              <button onClick={exportSchedulesExcel} className="px-4 py-2 text-sm font-medium rounded-lg bg-slate-700 hover:bg-slate-600 text-slate-300 transition-colors">Export Excel</button>
            </div>
          </div>
          <div className="flex flex-wrap gap-3 mb-4">
            <select
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value)}
              className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300"
            >
              <option value="">All Statuses</option>
              <option value="pending">Pending</option>
              <option value="confirmed">Confirmed</option>
              <option value="in_progress">In Progress</option>
              <option value="completed">Completed</option>
              <option value="cancelled">Cancelled</option>
            </select>
            <select
              value={filterStaff}
              onChange={(e) => setFilterStaff(e.target.value)}
              className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300"
            >
              <option value="">All Staff</option>
              {staffList.map((s: any) => (
                <option key={s.id} value={s.id}>{s.full_name}</option>
              ))}
            </select>
            <select
              value={filterCategory}
              onChange={(e) => setFilterCategory(e.target.value)}
              className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300"
            >
              <option value="">All Services</option>
              {[...new Set(todaySchedules.map((s: any) => s.service_id))].map((sid: string) => (
                <option key={sid} value={sid}>{sid.slice(0, 8)}...</option>
              ))}
            </select>
            <input
              type="date"
              value={filterDateFrom}
              onChange={(e) => setFilterDateFrom(e.target.value)}
              className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300"
              placeholder="Date from"
            />
            <input
              type="date"
              value={filterDateTo}
              onChange={(e) => setFilterDateTo(e.target.value)}
              className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300"
              placeholder="Date to"
            />
            <input
              type="text"
              value={filterTag}
              onChange={(e) => setFilterTag(e.target.value)}
              className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300"
              placeholder="Filter by tag..."
            />
          </div>
          <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700">
                  {[
                    { label: "Client", key: "client_name" },
                    { label: "Staff", key: "staff" },
                    { label: "Time", key: "time" },
                    { label: "Status", key: "status" },
                  ].map(col => (
                    <th
                      key={col.key}
                      onClick={() => handleSort(col.key)}
                      className="text-left p-4 text-sm font-medium text-slate-400 cursor-pointer hover:text-slate-200 select-none"
                    >
                      {col.label}
                      {sortKey === col.key ? (sortDir === "asc" ? " \u25B2" : " \u25BC") : ""}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {sortedSchedules.length === 0 ? (
                  <tr><td colSpan={4} className="p-8 text-center text-slate-500">No schedules match filters</td></tr>
                ) : sortedSchedules.map((sch: any) => (
                  <tr key={sch.id} className="border-b border-slate-800 bg-slate-800/50 hover:bg-slate-800 transition-colors">
                    <td className="p-4 text-sm text-slate-200">{sch.client_name}</td>
                    <td className="p-4 text-sm text-slate-300">{staffList.find((s: any) => s.id === sch.staff_id)?.full_name || sch.staff_id}</td>
                    <td className="p-4 text-sm text-slate-300">{new Date(sch.scheduled_start).toLocaleTimeString()} - {new Date(sch.scheduled_end).toLocaleTimeString()}</td>
                    <td className="p-4"><span className={`text-xs px-2 py-1 rounded-full ${
                      sch.status === "completed" ? "bg-slate-700 text-slate-300" :
                      sch.status === "cancelled" ? "bg-red-900/50 text-red-400" :
                      sch.status === "confirmed" ? "bg-green-900/50 text-green-400" :
                      sch.status === "in_progress" ? "bg-blue-900/50 text-blue-400" :
                      "bg-yellow-900/50 text-yellow-400"
                    }`}>{sch.status?.replace(/_/g, " ")}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      </main>
    </div>
  );
}
