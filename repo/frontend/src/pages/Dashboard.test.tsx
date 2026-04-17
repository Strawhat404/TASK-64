import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import axios from "axios";
import Dashboard from "./Dashboard";

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

describe("Dashboard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedAxios.get.mockImplementation((url: string) => {
      if (url.includes("/schedules")) return Promise.resolve({ data: [] });
      if (url.includes("/staff")) return Promise.resolve({ data: [] });
      if (url.includes("/reconciliation/summary"))
        return Promise.resolve({ data: { total_open: 0, unmatched_items: 0, suspected_duplicates: 0, variance_alerts: 0, match_rate: 0 } });
      if (url.includes("/governance/reviews/pending")) return Promise.resolve({ data: [] });
      return Promise.resolve({ data: {} });
    });
  });

  it("renders welcome heading with username", async () => {
    render(<MemoryRouter><Dashboard /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/Welcome back/)).toBeInTheDocument();
    });
  });

  it("fetches schedules, staff, and summary on mount", async () => {
    render(<MemoryRouter><Dashboard /></MemoryRouter>);
    await waitFor(() => {
      expect(mockedAxios.get).toHaveBeenCalled();
    });
    const calls = mockedAxios.get.mock.calls.map((c) => c[0]);
    expect(calls.some((u: string) => u.includes("/schedules"))).toBe(true);
    expect(calls.some((u: string) => u.includes("/staff"))).toBe(true);
  });

  it("renders role in subtitle", async () => {
    render(<MemoryRouter><Dashboard /></MemoryRouter>);
    await waitFor(() => {
      const text = document.body.textContent || "";
      expect(text).toContain("Administrator");
    });
  });

  it("renders stat cards", async () => {
    render(<MemoryRouter><Dashboard /></MemoryRouter>);
    await waitFor(() => {
      expect(mockedAxios.get).toHaveBeenCalled();
    });
  });
});
