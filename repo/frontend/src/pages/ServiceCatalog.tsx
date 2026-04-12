import { useState, useEffect } from "react";
import axios from "axios";
import Sidebar from "../components/Sidebar";
import { SortableHeader, useSortableData } from "../components/SortableHeader";

interface Service {
  id: string;
  name: string;
  tier: string;
  base_price_usd: number;
  duration_minutes: number;
  is_active: boolean;
  description: string;
  headcount: number;
  required_tools: string[];
  add_ons: any;
  daily_cap: number | null;
}

const emptyService: Omit<Service, "id"> = {
  name: "",
  tier: "standard",
  base_price_usd: 0,
  duration_minutes: 60,
  is_active: true,
  description: "",
  headcount: 1,
  required_tools: [],
  add_ons: null,
  daily_cap: null,
};

const surcharges: Record<string, number> = {
  standard: 0,
  premium: 0.25,
  enterprise: 0.5,
};

export default function ServiceCatalog() {
  const [services, setServices] = useState<Service[]>([]);
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<Service | null>(null);
  const [form, setForm] = useState(emptyService);
  const [error, setError] = useState("");
  const { sortedItems: sortedServices, sortConfig, requestSort } = useSortableData(services);

  const fetchServices = async () => {
    try {
      const res = await axios.get("/api/services");
      setServices(Array.isArray(res.data) ? res.data : []);
    } catch {
      setServices([]);
    }
  };

  useEffect(() => {
    fetchServices();
  }, []);

  const openAdd = () => {
    setEditing(null);
    setForm(emptyService);
    setError("");
    setShowModal(true);
  };

  const openEdit = (svc: Service) => {
    setEditing(svc);
    setForm({
      name: svc.name,
      tier: svc.tier,
      base_price_usd: svc.base_price_usd,
      duration_minutes: svc.duration_minutes,
      is_active: svc.is_active,
      description: svc.description,
      headcount: svc.headcount,
      required_tools: svc.required_tools,
      add_ons: svc.add_ons,
      daily_cap: svc.daily_cap,
    });
    setError("");
    setShowModal(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    try {
      if (editing) {
        await axios.put(`/api/services/${editing.id}`, form);
      } else {
        await axios.post("/api/services", form);
      }
      setShowModal(false);
      fetchServices();
    } catch (err: unknown) {
      setError(
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ||
          "Failed to save service"
      );
    }
  };

  const calcTotal = (base: number, tier: string) => {
    const surcharge = surcharges[tier] || 0;
    return (base * (1 + surcharge)).toFixed(2);
  };

  return (
    <div className="flex min-h-screen bg-slate-950">
      <Sidebar />
      <main className="flex-1 p-6 overflow-auto">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-slate-100">Service Catalog</h1>
            <p className="text-slate-400 mt-1">Manage service offerings and pricing</p>
          </div>
          <button
            onClick={openAdd}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
          >
            + Add Service
          </button>
        </div>

        {/* Pricing Calculator */}
        <div className="bg-slate-800 border border-slate-700 rounded-xl p-4 mb-6">
          <h3 className="text-sm font-semibold text-slate-300 mb-2">Pricing Tiers &amp; Surcharges</h3>
          <div className="flex gap-6 text-sm text-slate-400">
            <span>Standard: +0%</span>
            <span>Premium: +25%</span>
            <span>Enterprise: +50%</span>
          </div>
        </div>

        {/* Table */}
        <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-slate-700">
                <SortableHeader label="Name" sortKey="name" currentSort={sortConfig} onSort={requestSort} />
                <SortableHeader label="Tier" sortKey="tier" currentSort={sortConfig} onSort={requestSort} />
                <SortableHeader label="Base Price" sortKey="base_price_usd" currentSort={sortConfig} onSort={requestSort} />
                <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Total w/ Surcharge</th>
                <SortableHeader label="Duration" sortKey="duration_minutes" currentSort={sortConfig} onSort={requestSort} />
                <SortableHeader label="Headcount" sortKey="headcount" currentSort={sortConfig} onSort={requestSort} />
                <SortableHeader label="Daily Cap" sortKey="daily_cap" currentSort={sortConfig} onSort={requestSort} />
                <SortableHeader label="Status" sortKey="is_active" currentSort={sortConfig} onSort={requestSort} />
                <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody>
              {services.length === 0 && (
                <tr>
                  <td colSpan={9} className="p-8 text-center text-slate-500">
                    No services found. Add your first service to get started.
                  </td>
                </tr>
              )}
              {sortedServices.map((svc) => (
                <tr
                  key={svc.id}
                  className="border-b border-slate-800 bg-slate-800/50 hover:bg-slate-800 transition-colors"
                >
                  <td className="p-4 text-sm text-slate-200 font-medium">{svc.name}</td>
                  <td className="p-4">
                    <span className="text-xs px-2 py-1 rounded-full bg-slate-700 text-slate-300 capitalize">
                      {svc.tier}
                    </span>
                  </td>
                  <td className="p-4 text-sm text-slate-300">${svc.base_price_usd.toFixed(2)}</td>
                  <td className="p-4 text-sm text-slate-100 font-medium">
                    ${calcTotal(svc.base_price_usd, svc.tier)}
                  </td>
                  <td className="p-4 text-sm text-slate-300">{svc.duration_minutes} min</td>
                  <td className="p-4 text-sm text-slate-300">{svc.headcount}</td>
                  <td className="p-4 text-sm text-slate-300">{svc.daily_cap ?? "No limit"}</td>
                  <td className="p-4">
                    <span
                      className={`text-xs px-2 py-1 rounded-full ${
                        svc.is_active
                          ? "bg-green-900/50 text-green-400"
                          : "bg-slate-700 text-slate-400"
                      }`}
                    >
                      {svc.is_active ? "active" : "inactive"}
                    </span>
                  </td>
                  <td className="p-4">
                    <button
                      onClick={() => openEdit(svc)}
                      className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
                    >
                      Edit
                    </button>
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
                {editing ? "Edit Service" : "Add Service"}
              </h2>
              {error && (
                <div className="mb-4 p-3 bg-red-900/30 border border-red-800 rounded-lg text-sm text-red-300">
                  {error}
                </div>
              )}
              <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-1">Name</label>
                  <input
                    type="text"
                    value={form.name}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    required
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">Tier</label>
                    <select
                      value={form.tier}
                      onChange={(e) => setForm({ ...form, tier: e.target.value })}
                      className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="standard">Standard</option>
                      <option value="premium">Premium</option>
                      <option value="enterprise">Enterprise</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">Base Price ($)</label>
                    <input
                      type="number"
                      step="0.01"
                      min="0"
                      value={form.base_price_usd}
                      onChange={(e) => setForm({ ...form, base_price_usd: parseFloat(e.target.value) || 0 })}
                      required
                      className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">Duration (min)</label>
                    <input
                      type="number"
                      min="1"
                      value={form.duration_minutes}
                      onChange={(e) =>
                        setForm({ ...form, duration_minutes: parseInt(e.target.value) || 60 })
                      }
                      required
                      className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">Headcount</label>
                    <input type="number" min={1} max={10} value={form.headcount}
                      onChange={(e) => setForm({ ...form, headcount: parseInt(e.target.value) || 1 })}
                      className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500" />
                    <p className="text-xs text-slate-500 mt-1">Staff required (1-10)</p>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">Daily Cap</label>
                    <input type="number" min={0} value={form.daily_cap ?? ""}
                      onChange={(e) => setForm({ ...form, daily_cap: e.target.value ? parseInt(e.target.value) : null })}
                      className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="No limit" />
                    <p className="text-xs text-slate-500 mt-1">Maximum daily bookings (leave empty for no limit)</p>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">Active</label>
                    <label className="flex items-center gap-2 mt-2">
                      <input
                        type="checkbox"
                        checked={form.is_active}
                        onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
                        className="w-4 h-4 rounded border-slate-600 bg-slate-800 text-blue-500 focus:ring-2 focus:ring-blue-500"
                      />
                      <span className="text-sm text-slate-300">{form.is_active ? "Active" : "Inactive"}</span>
                    </label>
                  </div>
                </div>
                <div className="md:col-span-2">
                  <label className="block text-sm font-medium text-slate-300 mb-1">Required Tools</label>
                  <input type="text" value={(form.required_tools || []).join(", ")}
                    onChange={(e) => setForm({ ...form, required_tools: e.target.value ? e.target.value.split(",").map(s => s.trim()) : [] })}
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="Comma-separated list of tools" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-1">Description</label>
                  <textarea
                    value={form.description}
                    onChange={(e) => setForm({ ...form, description: e.target.value })}
                    rows={3}
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-slate-100 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
                  />
                </div>
                {form.base_price_usd > 0 && (
                  <div className="text-sm text-slate-400">
                    Final price with surcharge: <strong className="text-slate-100">${calcTotal(form.base_price_usd, form.tier)}</strong>
                  </div>
                )}
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
                    {editing ? "Save Changes" : "Add Service"}
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
