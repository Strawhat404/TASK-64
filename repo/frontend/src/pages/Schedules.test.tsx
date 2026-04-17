import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import axios from "axios";
import Schedules from "./Schedules";

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

describe("Schedules", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedAxios.get.mockImplementation((url: string) => {
      if (url.includes("/schedules")) return Promise.resolve({ data: [] });
      if (url.includes("/services")) return Promise.resolve({ data: [{ id: "s1", name: "Test Service" }] });
      if (url.includes("/staff")) return Promise.resolve({ data: [{ id: "st1", full_name: "Staff 1" }] });
      return Promise.resolve({ data: [] });
    });
  });

  it("renders schedule page heading", async () => {
    render(<MemoryRouter><Schedules /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("Schedules")).toBeInTheDocument();
    });
  });

  it("has date picker", async () => {
    render(<MemoryRouter><Schedules /></MemoryRouter>);
    await waitFor(() => {
      const dateInput = document.querySelector('input[type="date"]');
      expect(dateInput).toBeTruthy();
    });
  });

  it("has new schedule button", async () => {
    render(<MemoryRouter><Schedules /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/New Schedule/i)).toBeInTheDocument();
    });
  });

  it("shows empty state when no schedules", async () => {
    render(<MemoryRouter><Schedules /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/No schedules/i)).toBeInTheDocument();
    });
  });

  it("fetches data on mount", async () => {
    render(<MemoryRouter><Schedules /></MemoryRouter>);
    await waitFor(() => {
      expect(mockedAxios.get).toHaveBeenCalled();
    });
  });
});
