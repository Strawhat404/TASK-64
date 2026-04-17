import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import axios from "axios";
import OpenExceptions from "./OpenExceptions";

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

const mockExceptions = [
  { id: "e1", exception_type: "unmatched", severity: "high", amount: 500, status: "open", description: "No match found", created_at: "2026-04-16", assigned_to: null },
  { id: "e2", exception_type: "variance_over_threshold", severity: "critical", amount: 1200, variance_amount: 50, status: "open", description: "Variance exceeds threshold", created_at: "2026-04-15", assigned_to: "u1" },
];

const mockSummary = { total_open: 2, unmatched_items: 1, suspected_duplicates: 0, variance_alerts: 1, match_rate: 85.5 };

describe("OpenExceptions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedAxios.get.mockImplementation((url: string) => {
      if (url.includes("/exceptions")) return Promise.resolve({ data: mockExceptions });
      if (url.includes("/summary")) return Promise.resolve({ data: mockSummary });
      return Promise.resolve({ data: [] });
    });
  });

  it("renders exceptions heading", async () => {
    render(<MemoryRouter><OpenExceptions /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("Open Exceptions")).toBeInTheDocument();
    });
  });

  it("shows summary cards", async () => {
    render(<MemoryRouter><OpenExceptions /></MemoryRouter>);
    await waitFor(() => {
      const text = document.body.textContent || "";
      expect(text).toMatch(/Open|Unmatched|Variance/i);
    });
  });

  it("renders exception rows", async () => {
    render(<MemoryRouter><OpenExceptions /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/unmatched/i)).toBeInTheDocument();
    });
  });

  it("shows severity badges", async () => {
    render(<MemoryRouter><OpenExceptions /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("high")).toBeInTheDocument();
      expect(screen.getByText("critical")).toBeInTheDocument();
    });
  });

  it("has filter controls", async () => {
    render(<MemoryRouter><OpenExceptions /></MemoryRouter>);
    await waitFor(() => {
      const selects = document.querySelectorAll("select");
      expect(selects.length).toBeGreaterThanOrEqual(1);
    });
  });

  it("shows empty state when no exceptions", async () => {
    mockedAxios.get.mockImplementation((url: string) => {
      if (url.includes("/exceptions")) return Promise.resolve({ data: [] });
      if (url.includes("/summary")) return Promise.resolve({ data: { total_open: 0, unmatched_items: 0, suspected_duplicates: 0, variance_alerts: 0, match_rate: 100 } });
      return Promise.resolve({ data: [] });
    });
    render(<MemoryRouter><OpenExceptions /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/No exceptions/i)).toBeInTheDocument();
    });
  });
});
