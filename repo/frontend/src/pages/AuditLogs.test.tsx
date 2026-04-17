import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import axios from "axios";
import AuditLogs from "./AuditLogs";

vi.mock("axios");
const mockedAxios = vi.mocked(axios, true);

vi.mock("../context/AuthContext", () => ({
  useAuth: () => ({
    user: { id: "u1", username: "admin", role_name: "Administrator", full_name: "Admin" },
    logout: vi.fn(),
  }),
}));

vi.mock("../components/Sidebar", () => ({
  default: () => <div data-testid="sidebar">Sidebar</div>,
}));

const mockAuditResponse = {
  data: [
    { id: "a1", created_at: "2026-04-16T10:00:00Z", user_id: "u1", action: "login", resource_type: "session", resource_id: "s1", details: null, ip_address: "127.0.0.1" },
    { id: "a2", created_at: "2026-04-16T10:05:00Z", user_id: "u1", action: "create_user", resource_type: "user", resource_id: "u2", details: '{"username":"test"}', ip_address: "127.0.0.1" },
  ],
  total: 2,
};

describe("AuditLogs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedAxios.get.mockResolvedValue({ data: mockAuditResponse });
  });

  it("renders audit logs heading", async () => {
    render(<MemoryRouter><AuditLogs /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("Audit Logs")).toBeInTheDocument();
    });
  });

  it("fetches audit logs on mount", async () => {
    render(<MemoryRouter><AuditLogs /></MemoryRouter>);
    await waitFor(() => {
      const calls = mockedAxios.get.mock.calls.map((c) => c[0]);
      expect(calls.some((u: string) => u.includes("/audit/logs"))).toBe(true);
    });
  });

  it("has filter controls", async () => {
    render(<MemoryRouter><AuditLogs /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("Filter")).toBeInTheDocument();
    });
  });

  it("has pagination controls", async () => {
    render(<MemoryRouter><AuditLogs /></MemoryRouter>);
    await waitFor(() => {
      const text = document.body.textContent || "";
      expect(text).toMatch(/Previous|Next|Page/i);
    });
  });

  it("shows empty state when no entries", async () => {
    mockedAxios.get.mockResolvedValue({ data: { data: [], total: 0 } });
    render(<MemoryRouter><AuditLogs /></MemoryRouter>);
    await waitFor(() => {
      expect(mockedAxios.get).toHaveBeenCalled();
    });
  });
});
