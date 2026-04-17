import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import axios from "axios";
import UserManagement from "./UserManagement";

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

const mockUsers = [
  { id: "u1", username: "admin", email: "admin@test.com", full_name: "Admin User", role_name: "Administrator", is_active: true, last_login_at: "2026-04-16T10:00:00Z", created_at: "2026-01-01" },
  { id: "u2", username: "reviewer", email: "rev@test.com", full_name: "Reviewer", role_name: "Reviewer", is_active: false, last_login_at: null, created_at: "2026-01-02" },
];

describe("UserManagement", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedAxios.get.mockResolvedValue({ data: mockUsers });
  });

  it("renders user list", async () => {
    render(<MemoryRouter><UserManagement /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("admin")).toBeInTheDocument();
      expect(screen.getByText("reviewer")).toBeInTheDocument();
    });
  });

  it("shows role names", async () => {
    render(<MemoryRouter><UserManagement /></MemoryRouter>);
    await waitFor(() => {
      const body = document.body.textContent || "";
      expect(body).toContain("Administrator");
      expect(body).toContain("Reviewer");
    });
  });

  it("shows active/inactive badges", async () => {
    render(<MemoryRouter><UserManagement /></MemoryRouter>);
    await waitFor(() => {
      const badges = screen.getAllByText(/Active|Inactive/);
      expect(badges.length).toBeGreaterThanOrEqual(2);
    });
  });

  it("has create user button", async () => {
    render(<MemoryRouter><UserManagement /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("+ Create User")).toBeInTheDocument();
    });
  });

  it("opens create form on click", async () => {
    render(<MemoryRouter><UserManagement /></MemoryRouter>);
    await waitFor(() => expect(screen.getByText("+ Create User")).toBeInTheDocument());
    await userEvent.click(screen.getByText("+ Create User"));
    await waitFor(() => {
      const inputs = document.querySelectorAll('input[type="text"], input[type="email"], input[type="password"]');
      expect(inputs.length).toBeGreaterThan(0);
    });
  });

  it("shows empty state with no users", async () => {
    mockedAxios.get.mockResolvedValue({ data: [] });
    render(<MemoryRouter><UserManagement /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/No users/i)).toBeInTheDocument();
    });
  });

  it("shows Never for null last_login_at", async () => {
    render(<MemoryRouter><UserManagement /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("Never")).toBeInTheDocument();
    });
  });
});
