import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import App from "./App";

// Mock all page components to lightweight stubs
vi.mock("./pages/LoginPage", () => ({ default: () => <div data-testid="login-page">LoginPage</div> }));
vi.mock("./pages/Dashboard", () => ({ default: () => <div data-testid="dashboard-page">Dashboard</div> }));
vi.mock("./pages/ServiceCatalog", () => ({ default: () => <div data-testid="services-page">ServiceCatalog</div> }));
vi.mock("./pages/Schedules", () => ({ default: () => <div data-testid="schedules-page">Schedules</div> }));
vi.mock("./pages/StaffRoster", () => ({ default: () => <div data-testid="staff-page">StaffRoster</div> }));
vi.mock("./pages/UserManagement", () => ({ default: () => <div data-testid="users-page">UserManagement</div> }));
vi.mock("./pages/AuditLogs", () => ({ default: () => <div data-testid="audit-page">AuditLogs</div> }));
vi.mock("./pages/ModerationQueue", () => ({ default: () => <div data-testid="moderation-page">ModerationQueue</div> }));
vi.mock("./pages/OpenExceptions", () => ({ default: () => <div data-testid="exceptions-page">OpenExceptions</div> }));
vi.mock("./pages/ReconciliationImport", () => ({ default: () => <div data-testid="reconciliation-page">ReconciliationImport</div> }));
vi.mock("./pages/SecurityDashboard", () => ({ default: () => <div data-testid="security-page">SecurityDashboard</div> }));

const mockUseAuth = vi.fn();
vi.mock("./context/AuthContext", () => ({
  useAuth: () => mockUseAuth(),
}));

describe("App routing", () => {
  it("renders login page at /login without auth", () => {
    mockUseAuth.mockReturnValue({ user: null, loading: false });
    render(<MemoryRouter initialEntries={["/login"]}><App /></MemoryRouter>);
    expect(screen.getByTestId("login-page")).toBeInTheDocument();
  });

  it("redirects unauthenticated user to login from /", async () => {
    mockUseAuth.mockReturnValue({ user: null, loading: false });
    render(<MemoryRouter initialEntries={["/"]}><App /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByTestId("login-page")).toBeInTheDocument();
    });
  });

  it("renders dashboard for authenticated user at /", async () => {
    mockUseAuth.mockReturnValue({
      user: { id: "u1", username: "admin", role_name: "Administrator", full_name: "Admin" },
      loading: false,
    });
    render(<MemoryRouter initialEntries={["/"]}><App /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByTestId("dashboard-page")).toBeInTheDocument();
    });
  });

  it("renders services page for authenticated user", async () => {
    mockUseAuth.mockReturnValue({
      user: { id: "u1", username: "admin", role_name: "Administrator", full_name: "Admin" },
      loading: false,
    });
    render(<MemoryRouter initialEntries={["/services"]}><App /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByTestId("services-page")).toBeInTheDocument();
    });
  });

  it("renders schedules page for authenticated user", async () => {
    mockUseAuth.mockReturnValue({
      user: { id: "u1", username: "admin", role_name: "Scheduler", full_name: "S" },
      loading: false,
    });
    render(<MemoryRouter initialEntries={["/schedules"]}><App /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByTestId("schedules-page")).toBeInTheDocument();
    });
  });

  it("blocks scheduler from /security (admin-only)", async () => {
    mockUseAuth.mockReturnValue({
      user: { id: "u1", username: "sched", role_name: "Scheduler", full_name: "S" },
      loading: false,
    });
    render(<MemoryRouter initialEntries={["/security"]}><App /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.queryByTestId("security-page")).not.toBeInTheDocument();
    });
  });

  it("blocks scheduler from /users (admin-only)", async () => {
    mockUseAuth.mockReturnValue({
      user: { id: "u1", username: "sched", role_name: "Scheduler", full_name: "S" },
      loading: false,
    });
    render(<MemoryRouter initialEntries={["/users"]}><App /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.queryByTestId("users-page")).not.toBeInTheDocument();
    });
  });

  it("allows admin to access /moderation", async () => {
    mockUseAuth.mockReturnValue({
      user: { id: "u1", username: "admin", role_name: "Administrator", full_name: "A" },
      loading: false,
    });
    render(<MemoryRouter initialEntries={["/moderation"]}><App /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByTestId("moderation-page")).toBeInTheDocument();
    });
  });

  it("allows reviewer to access /moderation", async () => {
    mockUseAuth.mockReturnValue({
      user: { id: "u1", username: "rev", role_name: "Reviewer", full_name: "R" },
      loading: false,
    });
    render(<MemoryRouter initialEntries={["/moderation"]}><App /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByTestId("moderation-page")).toBeInTheDocument();
    });
  });

  it("blocks reviewer from /reconciliation", async () => {
    mockUseAuth.mockReturnValue({
      user: { id: "u1", username: "rev", role_name: "Reviewer", full_name: "R" },
      loading: false,
    });
    render(<MemoryRouter initialEntries={["/reconciliation"]}><App /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.queryByTestId("reconciliation-page")).not.toBeInTheDocument();
    });
  });

  it("allows auditor to access /audit", async () => {
    mockUseAuth.mockReturnValue({
      user: { id: "u1", username: "aud", role_name: "Auditor", full_name: "A" },
      loading: false,
    });
    render(<MemoryRouter initialEntries={["/audit"]}><App /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByTestId("audit-page")).toBeInTheDocument();
    });
  });

  it("blocks scheduler from /audit", async () => {
    mockUseAuth.mockReturnValue({
      user: { id: "u1", username: "sched", role_name: "Scheduler", full_name: "S" },
      loading: false,
    });
    render(<MemoryRouter initialEntries={["/audit"]}><App /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.queryByTestId("audit-page")).not.toBeInTheDocument();
    });
  });
});
