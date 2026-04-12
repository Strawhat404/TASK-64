import { Routes, Route } from "react-router-dom";
import ProtectedRoute from "./components/ProtectedRoute";
import LoginPage from "./pages/LoginPage";
import Dashboard from "./pages/Dashboard";
import ServiceCatalog from "./pages/ServiceCatalog";
import Schedules from "./pages/Schedules";
import StaffRoster from "./pages/StaffRoster";
import UserManagement from "./pages/UserManagement";
import AuditLogs from "./pages/AuditLogs";
import ModerationQueue from "./pages/ModerationQueue";
import OpenExceptions from "./pages/OpenExceptions";
import ReconciliationImport from "./pages/ReconciliationImport";
import SecurityDashboard from "./pages/SecurityDashboard";

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <Dashboard />
          </ProtectedRoute>
        }
      />
      <Route
        path="/services"
        element={
          <ProtectedRoute>
            <ServiceCatalog />
          </ProtectedRoute>
        }
      />
      <Route
        path="/schedules"
        element={
          <ProtectedRoute>
            <Schedules />
          </ProtectedRoute>
        }
      />
      <Route
        path="/staff"
        element={
          <ProtectedRoute>
            <StaffRoster />
          </ProtectedRoute>
        }
      />
      <Route
        path="/moderation"
        element={
          <ProtectedRoute roles={["Administrator", "Reviewer"]}>
            <ModerationQueue />
          </ProtectedRoute>
        }
      />
      <Route
        path="/reconciliation"
        element={
          <ProtectedRoute roles={["Administrator", "Auditor"]}>
            <ReconciliationImport />
          </ProtectedRoute>
        }
      />
      <Route
        path="/exceptions"
        element={
          <ProtectedRoute roles={["Administrator", "Auditor"]}>
            <OpenExceptions />
          </ProtectedRoute>
        }
      />
      <Route
        path="/security"
        element={
          <ProtectedRoute roles={["Administrator"]}>
            <SecurityDashboard />
          </ProtectedRoute>
        }
      />
      <Route
        path="/users"
        element={
          <ProtectedRoute roles={["Administrator"]}>
            <UserManagement />
          </ProtectedRoute>
        }
      />
      <Route
        path="/audit"
        element={
          <ProtectedRoute roles={["Auditor", "Administrator"]}>
            <AuditLogs />
          </ProtectedRoute>
        }
      />
    </Routes>
  );
}
