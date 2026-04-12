import { useState, useEffect } from "react";
import axios from "axios";
import Sidebar from "../components/Sidebar";

interface UserRecord {
  id: string;
  username: string;
  email: string;
  full_name: string;
  role_name: string;
  is_active: boolean;
  last_login_at: string | null;
  created_at: string;
}

export default function UserManagement() {
  const [users, setUsers] = useState<UserRecord[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    username: "",
    email: "",
    full_name: "",
    password: "",
    role_id: "2",
  });

  const fetchUsers = async () => {
    try {
      const res = await axios.get("/api/users");
      setUsers(Array.isArray(res.data) ? res.data : []);
    } catch {
      setUsers([]);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (form.password.length < 12) {
      setError("Password must be at least 12 characters long.");
      return;
    }

    try {
      await axios.post("/api/users", { ...form, role_id: parseInt(form.role_id, 10) });
      setShowForm(false);
      setForm({ username: "", email: "", full_name: "", password: "", role_id: "2" });
      fetchUsers();
    } catch (err: unknown) {
      setError(
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ||
          "Failed to create user"
      );
    }
  };

  const toggleActive = async (user: UserRecord) => {
    try {
      await axios.put(`/api/users/${user.id}`, {
        is_active: !user.is_active,
      });
      fetchUsers();
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
            <h1 className="text-2xl font-bold text-slate-100">User Management</h1>
            <p className="text-slate-400 mt-1">Manage system users and access control</p>
          </div>
          <button
            onClick={() => {
              setShowForm(true);
              setError("");
            }}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
          >
            + Create User
          </button>
        </div>

        {/* Create User Form */}
        {showForm && (
          <div className="bg-slate-900 border border-slate-700 rounded-xl p-6 mb-6">
            <h2 className="text-lg font-semibold text-slate-100 mb-4">Create New User</h2>
            {error && (
              <div className="mb-4 p-3 bg-red-900/30 border border-red-800 rounded-lg text-sm text-red-300">
                {error}
              </div>
            )}
            <form onSubmit={handleCreate} className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Username</label>
                <input
                  type="text"
                  value={form.username}
                  onChange={(e) => setForm({ ...form, username: e.target.value })}
                  required
                  className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
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
                <label className="block text-sm font-medium text-slate-300 mb-1">Email</label>
                <input
                  type="email"
                  value={form.email}
                  onChange={(e) => setForm({ ...form, email: e.target.value })}
                  required
                  className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Role</label>
                <select
                  value={form.role_id}
                  onChange={(e) => setForm({ ...form, role_id: e.target.value })}
                  className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="1">Administrator</option>
                  <option value="2">Scheduler</option>
                  <option value="3">Reviewer</option>
                  <option value="4">Auditor</option>
                </select>
              </div>
              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-slate-300 mb-1">Password</label>
                <input
                  type="password"
                  value={form.password}
                  onChange={(e) => setForm({ ...form, password: e.target.value })}
                  required
                  minLength={12}
                  className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="Minimum 12 characters"
                />
                <p className="text-xs text-slate-500 mt-1">
                  Password must be at least 12 characters long with a mix of letters, numbers, and symbols.
                </p>
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
                  Create User
                </button>
              </div>
            </form>
          </div>
        )}

        {/* Users Table */}
        <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-slate-700">
                <th className="text-left p-4 text-sm font-medium text-slate-400">Username</th>
                <th className="text-left p-4 text-sm font-medium text-slate-400">Full Name</th>
                <th className="text-left p-4 text-sm font-medium text-slate-400">Email</th>
                <th className="text-left p-4 text-sm font-medium text-slate-400">Role</th>
                <th className="text-left p-4 text-sm font-medium text-slate-400">Status</th>
                <th className="text-left p-4 text-sm font-medium text-slate-400">Last Login</th>
                <th className="text-left p-4 text-sm font-medium text-slate-400">Actions</th>
              </tr>
            </thead>
            <tbody>
              {users.length === 0 && (
                <tr>
                  <td colSpan={7} className="p-8 text-center text-slate-500">
                    No users found.
                  </td>
                </tr>
              )}
              {users.map((u) => (
                <tr
                  key={u.id}
                  className="border-b border-slate-800 bg-slate-800/50 hover:bg-slate-800 transition-colors"
                >
                  <td className="p-4 text-sm text-slate-200 font-medium">{u.username}</td>
                  <td className="p-4 text-sm text-slate-300">{u.full_name}</td>
                  <td className="p-4 text-sm text-slate-300">{u.email}</td>
                  <td className="p-4">
                    <span className="text-xs px-2 py-1 rounded-full bg-slate-700 text-slate-300 capitalize">
                      {u.role_name}
                    </span>
                  </td>
                  <td className="p-4">
                    <span
                      className={`text-xs px-2 py-1 rounded-full ${
                        u.is_active
                          ? "bg-green-900/50 text-green-400"
                          : "bg-red-900/50 text-red-400"
                      }`}
                    >
                      {u.is_active ? "Active" : "Inactive"}
                    </span>
                  </td>
                  <td className="p-4 text-sm text-slate-400">
                    {u.last_login_at
                      ? new Date(u.last_login_at).toLocaleString()
                      : "Never"}
                  </td>
                  <td className="p-4">
                    <button
                      onClick={() => toggleActive(u)}
                      className={`text-sm transition-colors ${
                        u.is_active
                          ? "text-red-400 hover:text-red-300"
                          : "text-green-400 hover:text-green-300"
                      }`}
                    >
                      {u.is_active ? "Deactivate" : "Reactivate"}
                    </button>
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
