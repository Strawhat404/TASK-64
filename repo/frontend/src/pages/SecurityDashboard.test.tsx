import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import axios from "axios";
import SecurityDashboard from "./SecurityDashboard";

vi.mock("axios");
const mockedAxios = vi.mocked(axios, true);

vi.mock("../context/AuthContext", () => ({
  useAuth: () => ({
    user: { id: "u1", username: "admin", role_name: "Administrator", full_name: "Admin" },
  }),
}));

vi.mock("../components/Sidebar", () => ({
  default: () => <div data-testid="sidebar">Sidebar</div>,
}));

describe("SecurityDashboard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedAxios.get.mockImplementation((url: string) => {
      if (url.includes("/keys")) return Promise.resolve({ data: [{ id: "k1", key_alias: "default-key", algorithm: "AES-256-GCM", status: "active", rotation_number: 0, activated_at: "2026-01-01" }] });
      if (url.includes("/sensitive")) return Promise.resolve({ data: [{ id: "sd1", data_type: "ssn", label: "Test SSN", masked_value: "****6789", created_at: "2026-04-16" }] });
      if (url.includes("/audit-ledger")) return Promise.resolve({ data: [] });
      if (url.includes("/rate-limits")) return Promise.resolve({ data: [] });
      if (url.includes("/retention")) return Promise.resolve({ data: [] });
      return Promise.resolve({ data: [] });
    });
  });

  it("renders security dashboard heading", async () => {
    render(<MemoryRouter><SecurityDashboard /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/Security/i)).toBeInTheDocument();
    });
  });

  it("shows encryption keys section", async () => {
    render(<MemoryRouter><SecurityDashboard /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/Encryption Key/i)).toBeInTheDocument();
      expect(screen.getByText("default-key")).toBeInTheDocument();
    });
  });

  it("shows sensitive data section", async () => {
    render(<MemoryRouter><SecurityDashboard /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/Sensitive Data/i)).toBeInTheDocument();
      expect(screen.getByText("Test SSN")).toBeInTheDocument();
    });
  });

  it("shows masked values", async () => {
    render(<MemoryRouter><SecurityDashboard /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("****6789")).toBeInTheDocument();
    });
  });

  it("has reveal button for sensitive data", async () => {
    render(<MemoryRouter><SecurityDashboard /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("Reveal")).toBeInTheDocument();
    });
  });

  it("has verify chain button", async () => {
    render(<MemoryRouter><SecurityDashboard /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/Verify Chain/i)).toBeInTheDocument();
    });
  });

  it("has rotate key button", async () => {
    render(<MemoryRouter><SecurityDashboard /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/Rotate Key/i)).toBeInTheDocument();
    });
  });

  it("fetches all security data on mount", async () => {
    render(<MemoryRouter><SecurityDashboard /></MemoryRouter>);
    await waitFor(() => {
      const calls = mockedAxios.get.mock.calls.map((c) => c[0]);
      expect(calls.some((u: string) => u.includes("/keys"))).toBe(true);
      expect(calls.some((u: string) => u.includes("/sensitive"))).toBe(true);
    });
  });
});
