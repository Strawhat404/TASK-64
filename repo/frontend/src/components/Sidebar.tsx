import { NavLink } from "react-router-dom";
import { useAuth } from "../context/AuthContext";

const navItems = [
  { to: "/", label: "Dashboard", icon: "\u25A0", roles: null },
  { to: "/services", label: "Services", icon: "\u2630", roles: null },
  { to: "/schedules", label: "Schedules", icon: "\u25D2", roles: null },
  { to: "/staff", label: "Staff", icon: "\u263A", roles: null },
  { to: "/moderation", label: "Moderation", icon: "\u2691", roles: ["Administrator", "Reviewer"] },
  { to: "/reconciliation", label: "Reconciliation", icon: "\u2696", roles: ["Administrator", "Auditor"] },
  { to: "/exceptions", label: "Exceptions", icon: "\u26A0", roles: ["Administrator", "Auditor"] },
  { to: "/security", label: "Security", icon: "\u2620", roles: ["Administrator"] },
  { to: "/users", label: "Users", icon: "\u2726", roles: ["Administrator"] },
  { to: "/audit", label: "Audit Logs", icon: "\u2637", roles: ["Auditor", "Administrator"] },
];

export default function Sidebar() {
  const { user, logout } = useAuth();

  const visibleItems = navItems.filter(
    (item) => !item.roles || (user && item.roles.includes(user.role_name))
  );

  return (
    <aside className="w-64 min-h-screen bg-slate-900 border-r border-slate-800 flex flex-col">
      <div className="p-6 border-b border-slate-800">
        <h1 className="text-xl font-bold text-slate-100 tracking-tight">
          Compliance Console
        </h1>
        <p className="text-xs text-slate-500 mt-1">Local Operations</p>
      </div>

      <nav className="flex-1 p-4 space-y-1">
        {visibleItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === "/"}
            className={({ isActive }) =>
              `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                isActive
                  ? "bg-slate-700 text-slate-100"
                  : "text-slate-400 hover:bg-slate-800 hover:text-slate-200"
              }`
            }
          >
            <span className="text-lg leading-none">{item.icon}</span>
            {item.label}
          </NavLink>
        ))}
      </nav>

      <div className="p-4 border-t border-slate-800">
        <div className="flex items-center gap-3 mb-3 px-3">
          <div className="w-8 h-8 rounded-full bg-slate-700 flex items-center justify-center text-sm font-bold text-slate-300">
            {user?.full_name?.charAt(0) || user?.username?.charAt(0) || "U"}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-slate-200 truncate">
              {user?.full_name || user?.username}
            </p>
            <p className="text-xs text-slate-500 capitalize">{user?.role_name}</p>
          </div>
        </div>
        <button
          onClick={logout}
          className="w-full px-3 py-2 text-sm text-slate-400 hover:text-slate-200 hover:bg-slate-800 rounded-lg transition-colors text-left"
        >
          Sign out
        </button>
      </div>
    </aside>
  );
}
