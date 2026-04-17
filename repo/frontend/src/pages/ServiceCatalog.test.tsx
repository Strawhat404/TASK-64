import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import axios from "axios";
import ServiceCatalog from "./ServiceCatalog";

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

const mockServices = [
  { id: "s1", name: "Basic Inspection", tier: "standard", base_price_usd: 100, duration_minutes: 60, headcount: 1, is_active: true, description: "Basic service" },
  { id: "s2", name: "Premium Audit", tier: "premium", base_price_usd: 200, duration_minutes: 120, headcount: 2, is_active: false, description: "Premium service" },
];

describe("ServiceCatalog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedAxios.get.mockResolvedValue({ data: mockServices });
  });

  it("renders service list", async () => {
    render(<MemoryRouter><ServiceCatalog /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("Basic Inspection")).toBeInTheDocument();
      expect(screen.getByText("Premium Audit")).toBeInTheDocument();
    });
  });

  it("shows tier badges", async () => {
    render(<MemoryRouter><ServiceCatalog /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("standard")).toBeInTheDocument();
      expect(screen.getByText("premium")).toBeInTheDocument();
    });
  });

  it("shows active/inactive status", async () => {
    render(<MemoryRouter><ServiceCatalog /></MemoryRouter>);
    await waitFor(() => {
      const statuses = screen.getAllByText(/active|inactive/i);
      expect(statuses.length).toBeGreaterThanOrEqual(2);
    });
  });

  it("shows add service button", async () => {
    render(<MemoryRouter><ServiceCatalog /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("+ Add Service")).toBeInTheDocument();
    });
  });

  it("opens add modal on button click", async () => {
    render(<MemoryRouter><ServiceCatalog /></MemoryRouter>);
    await waitFor(() => expect(screen.getByText("+ Add Service")).toBeInTheDocument());
    await userEvent.click(screen.getByText("+ Add Service"));
    await waitFor(() => {
      // Modal opens with form inputs visible
      const inputs = document.querySelectorAll("input, select");
      expect(inputs.length).toBeGreaterThan(0);
    });
  });

  it("shows empty state when no services", async () => {
    mockedAxios.get.mockResolvedValue({ data: [] });
    render(<MemoryRouter><ServiceCatalog /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/No services/i)).toBeInTheDocument();
    });
  });
});
