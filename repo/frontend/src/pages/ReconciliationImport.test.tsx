import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import axios from "axios";
import ReconciliationImport from "./ReconciliationImport";

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

const mockFeeds = [
  { id: "f1", filename: "test.csv", feed_type: "internal", record_count: 10, status: "completed", imported_by: "u1", imported_at: "2026-04-16" },
];

describe("ReconciliationImport", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedAxios.get.mockImplementation((url: string) => {
      if (url.includes("/feeds")) return Promise.resolve({ data: mockFeeds });
      if (url.includes("/matches")) return Promise.resolve({ data: [] });
      return Promise.resolve({ data: [] });
    });
  });

  it("renders reconciliation heading", async () => {
    render(<MemoryRouter><ReconciliationImport /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("Financial Reconciliation")).toBeInTheDocument();
    });
  });

  it("has file upload area", async () => {
    render(<MemoryRouter><ReconciliationImport /></MemoryRouter>);
    await waitFor(() => {
      const text = document.body.textContent || "";
      expect(text).toMatch(/drag|drop|upload|csv|file/i);
    });
  });

  it("has feed type selector", async () => {
    render(<MemoryRouter><ReconciliationImport /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("Internal")).toBeInTheDocument();
      expect(screen.getByText("External")).toBeInTheDocument();
    });
  });

  it("shows imported feeds table", async () => {
    render(<MemoryRouter><ReconciliationImport /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("test.csv")).toBeInTheDocument();
    });
  });

  it("shows empty state when no feeds", async () => {
    mockedAxios.get.mockResolvedValue({ data: [] });
    render(<MemoryRouter><ReconciliationImport /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/No feeds/i)).toBeInTheDocument();
    });
  });

  it("has run matching button for feeds", async () => {
    render(<MemoryRouter><ReconciliationImport /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/Run Matching/i)).toBeInTheDocument();
    });
  });
});
