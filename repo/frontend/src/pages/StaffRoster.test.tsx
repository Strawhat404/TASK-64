import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import axios from "axios";
import StaffRoster from "./StaffRoster";

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

const mockStaff = [
  { id: "st1", full_name: "Alice Smith", specialization: "plumbing", is_available: true, user_id: "u2", created_at: "2026-01-01", updated_at: "2026-01-01" },
  { id: "st2", full_name: "Bob Jones", specialization: "electrical", is_available: false, user_id: "u3", created_at: "2026-01-01", updated_at: "2026-01-01" },
];

describe("StaffRoster", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedAxios.get.mockImplementation((url: string) => {
      if (url.includes("/credentials")) return Promise.resolve({ data: [] });
      if (url.includes("/availability")) return Promise.resolve({ data: [] });
      return Promise.resolve({ data: mockStaff });
    });
    mockedAxios.put.mockResolvedValue({ data: {} });
  });

  it("renders staff list", async () => {
    render(<MemoryRouter><StaffRoster /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("Alice Smith")).toBeInTheDocument();
      expect(screen.getByText("Bob Jones")).toBeInTheDocument();
    });
  });

  it("shows specialization", async () => {
    render(<MemoryRouter><StaffRoster /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("plumbing")).toBeInTheDocument();
      expect(screen.getByText("electrical")).toBeInTheDocument();
    });
  });

  it("shows availability status badges", async () => {
    render(<MemoryRouter><StaffRoster /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("Available")).toBeInTheDocument();
      expect(screen.getByText("Unavailable")).toBeInTheDocument();
    });
  });

  it("has add staff button", async () => {
    render(<MemoryRouter><StaffRoster /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/Add Staff/i)).toBeInTheDocument();
    });
  });

  it("shows empty state when no staff", async () => {
    mockedAxios.get.mockResolvedValue({ data: [] });
    render(<MemoryRouter><StaffRoster /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/No staff/i)).toBeInTheDocument();
    });
  });

  it("has details button for each staff", async () => {
    render(<MemoryRouter><StaffRoster /></MemoryRouter>);
    await waitFor(() => {
      const detailBtns = screen.getAllByText("Details");
      expect(detailBtns.length).toBe(2);
    });
  });
});
