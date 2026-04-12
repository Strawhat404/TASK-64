import { useState, useEffect } from "react";
import axios from "axios";
import Sidebar from "../components/Sidebar";
import { SortableHeader, useSortableData } from "../components/SortableHeader";

interface StaffMember {
  id: string;
  tenant_id: string;
  user_id: string;
  full_name: string;
  specialization: string;
  is_available: boolean;
  created_at: string;
  updated_at: string;
}

const emptyStaff = {
  full_name: "",
  user_id: "",
  specialization: "",
  is_available: true,
};

export default function StaffRoster() {
  const [staff, setStaff] = useState<StaffMember[]>([]);
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<StaffMember | null>(null);
  const [form, setForm] = useState(emptyStaff);
  const [error, setError] = useState("");
  const [detailStaff, setDetailStaff] = useState<StaffMember | null>(null);
  const [credentials, setCredentials] = useState<any[]>([]);
  const [availability, setAvailability] = useState<any[]>([]);
  const [credForm, setCredForm] = useState({ credential_name: "", issuing_authority: "", expiry_date: "" });
  const [availForm, setAvailForm] = useState({ day_of_week: 0, start_time: "09:00", end_time: "17:00", is_recurring: true });
  const { sortedItems: sortedStaff, sortConfig, requestSort } = useSortableData(staff);

  const fetchStaff = async () => {
    try {
      const res = await axios.get("/api/staff");
      setStaff(Array.isArray(res.data) ? res.data : []);
    } catch {
      setStaff([]);
    }
  };

  useEffect(() => {
    fetchStaff();
  }, []);

  const openAdd = () => {
    setEditing(null);
    setForm(emptyStaff);
    setError("");
    setShowModal(true);
  };

  const openEdit = (member: StaffMember) => {
    setEditing(member);
    setForm({
      full_name: member.full_name,
      user_id: member.user_id,
      specialization: member.specialization,
      is_available: member.is_available,
    });
    setError("");
    setShowModal(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    try {
      if (editing) {
        await axios.put(`/api/staff/${editing.id}`, form);
      } else {
        await axios.post("/api/staff", form);
      }
      setShowModal(false);
      fetchStaff();
    } catch (err: unknown) {
      setError(
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ||
          "Failed to save staff member"
      );
    }
  };

  const toggleAvailability = async (member: StaffMember) => {
    try {
      await axios.put(`/api/staff/${member.id}`, {
        is_available: !member.is_available,
      });
      fetchStaff();
    } catch {
      /* ignore */
    }
  };

  const fetchStaffDetails = async (member: StaffMember) => {
    setDetailStaff(member);
    const [credRes, availRes] = await Promise.all([
      axios.get(`/api/staff/${member.id}/credentials`).catch(() => ({ data: [] })),
      axios.get(`/api/staff/${member.id}/availability`).catch(() => ({ data: [] })),
    ]);
    setCredentials(Array.isArray(credRes.data) ? credRes.data : []);
    setAvailability(Array.isArray(availRes.data) ? availRes.data : []);
  };

  const addCredential = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!detailStaff) return;
    try {
      await axios.post(`/api/staff/${detailStaff.id}/credentials`, credForm);
      setCredForm({ credential_name: "", issuing_authority: "", expiry_date: "" });
      fetchStaffDetails(detailStaff);
    } catch {
      /* ignore */
    }
  };

  const addAvailability = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!detailStaff) return;
    try {
      await axios.post(`/api/staff/${detailStaff.id}/availability`, availForm);
      setAvailForm({ day_of_week: 0, start_time: "09:00", end_time: "17:00", is_recurring: true });
      fetchStaffDetails(detailStaff);
    } catch {
      /* ignore */
    }
  };

  const dayNames = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

  const statusColor = (isAvailable: boolean) =>
    isAvailable
      ? "bg-green-900/50 text-green-400"
      : "bg-red-900/50 text-red-400";

  return (
    <div className="flex min-h-screen bg-slate-950">
      <Sidebar />
      <main className="flex-1 p-6 overflow-auto">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-slate-100">Staff Roster</h1>
            <p className="text-slate-400 mt-1">Manage staff members and availability</p>
          </div>
          <button
            onClick={openAdd}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
          >
            + Add Staff
          </button>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-slate-700">
                <SortableHeader label="Name" sortKey="full_name" currentSort={sortConfig} onSort={requestSort} />
                <SortableHeader label="Specialization" sortKey="specialization" currentSort={sortConfig} onSort={requestSort} />
                <SortableHeader label="Availability" sortKey="is_available" currentSort={sortConfig} onSort={requestSort} />
                <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody>
              {staff.length === 0 && (
                <tr>
                  <td colSpan={4} className="p-8 text-center text-slate-500">
                    No staff members found. Add your first team member.
                  </td>
                </tr>
              )}
              {sortedStaff.map((member) => (
                <tr
                  key={member.id}
                  className="border-b border-slate-800 bg-slate-800/50 hover:bg-slate-800 transition-colors"
                >
                  <td className="p-4 text-sm text-slate-200 font-medium">{member.full_name}</td>
                  <td className="p-4 text-sm text-slate-300 capitalize">{member.specialization}</td>
                  <td className="p-4">
                    <span
                      className={`text-xs px-2 py-1 rounded-full ${statusColor(member.is_available)}`}
                    >
                      {member.is_available ? "Available" : "Unavailable"}
                    </span>
                  </td>
                  <td className="p-4">
                    <div className="flex gap-3">
                      <button
                        onClick={() => openEdit(member)}
                        className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
                      >
                        Edit
                      </button>
                      <button
                        onClick={() => toggleAvailability(member)}
                        className="text-sm text-yellow-400 hover:text-yellow-300 transition-colors"
                      >
                        {member.is_available
                          ? "Set Unavailable"
                          : "Set Available"}
                      </button>
                      <button
                        onClick={() => fetchStaffDetails(member)}
                        className="text-sm text-purple-400 hover:text-purple-300 transition-colors"
                      >
                        Details
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Modal */}
        {showModal && (
          <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
            <div className="bg-slate-900 border border-slate-700 rounded-2xl p-6 w-full max-w-lg">
              <h2 className="text-lg font-bold text-slate-100 mb-4">
                {editing ? "Edit Staff Member" : "Add Staff Member"}
              </h2>
              {error && (
                <div className="mb-4 p-3 bg-red-900/30 border border-red-800 rounded-lg text-sm text-red-300">
                  {error}
                </div>
              )}
              <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-1">Full Name</label>
                  <input
                    type="text"
                    value={form.full_name}
                    onChange={(e) => setForm({ ...form, full_name: e.target.value })}
                    required
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-1">User ID</label>
                  <input
                    type="text"
                    value={form.user_id}
                    onChange={(e) => setForm({ ...form, user_id: e.target.value })}
                    required
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">Specialization</label>
                    <input
                      type="text"
                      value={form.specialization}
                      onChange={(e) => setForm({ ...form, specialization: e.target.value })}
                      required
                      className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">Availability</label>
                    <label className="flex items-center gap-2 mt-2">
                      <input
                        type="checkbox"
                        checked={form.is_available}
                        onChange={(e) => setForm({ ...form, is_available: e.target.checked })}
                        className="w-4 h-4 rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-2 focus:ring-blue-500"
                      />
                      <span className="text-sm text-slate-300">{form.is_available ? "Available" : "Unavailable"}</span>
                    </label>
                  </div>
                </div>
                <div className="flex gap-3 justify-end pt-2">
                  <button
                    type="button"
                    onClick={() => setShowModal(false)}
                    className="px-4 py-2 text-sm text-slate-400 hover:text-slate-200 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
                  >
                    {editing ? "Save Changes" : "Add Staff"}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Detail Modal */}
        {detailStaff && (
          <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
            <div className="bg-slate-900 border border-slate-700 rounded-2xl p-6 w-full max-w-3xl max-h-[90vh] overflow-y-auto">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-bold text-slate-100">
                  {detailStaff.full_name} - Details
                </h2>
                <button
                  onClick={() => setDetailStaff(null)}
                  className="text-slate-400 hover:text-slate-200 transition-colors"
                >
                  Close
                </button>
              </div>

              <div className="mb-4 p-3 bg-slate-800 rounded-lg text-sm text-slate-300 space-y-1">
                <p><span className="text-slate-400">Specialization:</span> {detailStaff.specialization}</p>
                <p><span className="text-slate-400">Status:</span> {detailStaff.is_available ? "Available" : "Unavailable"}</p>
                <p><span className="text-slate-400">User ID:</span> {detailStaff.user_id}</p>
              </div>

              {/* Credentials */}
              <h3 className="text-md font-semibold text-slate-200 mb-2">Credentials</h3>
              <div className="bg-slate-800 rounded-lg overflow-hidden mb-3">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-slate-700">
                      <th className="text-left px-3 py-2 text-xs font-medium text-slate-400 uppercase">Name</th>
                      <th className="text-left px-3 py-2 text-xs font-medium text-slate-400 uppercase">Authority</th>
                      <th className="text-left px-3 py-2 text-xs font-medium text-slate-400 uppercase">Expiry</th>
                    </tr>
                  </thead>
                  <tbody>
                    {credentials.length === 0 && (
                      <tr><td colSpan={3} className="p-3 text-center text-slate-500 text-sm">No credentials found.</td></tr>
                    )}
                    {credentials.map((c, i) => (
                      <tr key={i} className="border-b border-slate-700">
                        <td className="px-3 py-2 text-sm text-slate-200">{c.credential_name}</td>
                        <td className="px-3 py-2 text-sm text-slate-300">{c.issuing_authority}</td>
                        <td className="px-3 py-2 text-sm text-slate-300">{c.expiry_date ? new Date(c.expiry_date).toLocaleDateString() : "N/A"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <form onSubmit={addCredential} className="flex gap-2 mb-6 items-end">
                <div className="flex-1">
                  <label className="block text-xs text-slate-400 mb-1">Credential Name</label>
                  <input type="text" value={credForm.credential_name} onChange={(e) => setCredForm({ ...credForm, credential_name: e.target.value })} required className="w-full px-2 py-1 bg-slate-800 border border-slate-600 rounded text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500" />
                </div>
                <div className="flex-1">
                  <label className="block text-xs text-slate-400 mb-1">Authority</label>
                  <input type="text" value={credForm.issuing_authority} onChange={(e) => setCredForm({ ...credForm, issuing_authority: e.target.value })} required className="w-full px-2 py-1 bg-slate-800 border border-slate-600 rounded text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500" />
                </div>
                <div className="flex-1">
                  <label className="block text-xs text-slate-400 mb-1">Expiry Date</label>
                  <input type="date" value={credForm.expiry_date} onChange={(e) => setCredForm({ ...credForm, expiry_date: e.target.value })} className="w-full px-2 py-1 bg-slate-800 border border-slate-600 rounded text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500" />
                </div>
                <button type="submit" className="px-3 py-1 bg-blue-600 hover:bg-blue-700 text-white rounded text-sm font-medium transition-colors">Add</button>
              </form>

              {/* Availability */}
              <h3 className="text-md font-semibold text-slate-200 mb-2">Availability Windows</h3>
              <div className="bg-slate-800 rounded-lg overflow-hidden mb-3">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-slate-700">
                      <th className="text-left px-3 py-2 text-xs font-medium text-slate-400 uppercase">Day</th>
                      <th className="text-left px-3 py-2 text-xs font-medium text-slate-400 uppercase">Start</th>
                      <th className="text-left px-3 py-2 text-xs font-medium text-slate-400 uppercase">End</th>
                      <th className="text-left px-3 py-2 text-xs font-medium text-slate-400 uppercase">Recurring</th>
                    </tr>
                  </thead>
                  <tbody>
                    {availability.length === 0 && (
                      <tr><td colSpan={4} className="p-3 text-center text-slate-500 text-sm">No availability windows found.</td></tr>
                    )}
                    {availability.map((a, i) => (
                      <tr key={i} className="border-b border-slate-700">
                        <td className="px-3 py-2 text-sm text-slate-200">{dayNames[a.day_of_week] || a.day_of_week}</td>
                        <td className="px-3 py-2 text-sm text-slate-300">{a.start_time}</td>
                        <td className="px-3 py-2 text-sm text-slate-300">{a.end_time}</td>
                        <td className="px-3 py-2 text-sm text-slate-300">{a.is_recurring ? "Yes" : "No"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <form onSubmit={addAvailability} className="flex gap-2 items-end">
                <div>
                  <label className="block text-xs text-slate-400 mb-1">Day</label>
                  <select value={availForm.day_of_week} onChange={(e) => setAvailForm({ ...availForm, day_of_week: parseInt(e.target.value) })} className="px-2 py-1 bg-slate-800 border border-slate-600 rounded text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500">
                    {dayNames.map((d, i) => <option key={i} value={i}>{d}</option>)}
                  </select>
                </div>
                <div>
                  <label className="block text-xs text-slate-400 mb-1">Start</label>
                  <input type="time" value={availForm.start_time} onChange={(e) => setAvailForm({ ...availForm, start_time: e.target.value })} className="px-2 py-1 bg-slate-800 border border-slate-600 rounded text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500" />
                </div>
                <div>
                  <label className="block text-xs text-slate-400 mb-1">End</label>
                  <input type="time" value={availForm.end_time} onChange={(e) => setAvailForm({ ...availForm, end_time: e.target.value })} className="px-2 py-1 bg-slate-800 border border-slate-600 rounded text-sm text-slate-100 focus:outline-none focus:ring-1 focus:ring-blue-500" />
                </div>
                <label className="flex items-center gap-1 text-sm text-slate-300">
                  <input type="checkbox" checked={availForm.is_recurring} onChange={(e) => setAvailForm({ ...availForm, is_recurring: e.target.checked })} className="w-3 h-3" />
                  Recurring
                </label>
                <button type="submit" className="px-3 py-1 bg-blue-600 hover:bg-blue-700 text-white rounded text-sm font-medium transition-colors">Add</button>
              </form>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
