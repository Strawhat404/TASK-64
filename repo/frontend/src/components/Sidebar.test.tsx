import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import Sidebar from "./Sidebar";

const mockLogout = vi.fn();
const mockUseAuth = vi.fn();

vi.mock("../context/AuthContext", () => ({
  useAuth: () => mockUseAuth(),
}));

describe("Sidebar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders common nav items for all roles", () => {
    mockUseAuth.mockReturnValue({
      user: {
        id: "user-1",
        username: "scheduler",
        role_name: "Scheduler",
        full_name: "Scheduler User",
      },
      logout: mockLogout,
    });

    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>
    );

    expect(screen.getByText("Dashboard")).toBeInTheDocument();
    expect(screen.getByText("Services")).toBeInTheDocument();
    expect(screen.getByText("Schedules")).toBeInTheDocument();
    expect(screen.getByText("Staff")).toBeInTheDocument();
  });

  it("shows admin-only items for Administrator", () => {
    mockUseAuth.mockReturnValue({
      user: {
        id: "user-1",
        username: "admin",
        role_name: "Administrator",
        full_name: "Admin User",
      },
      logout: mockLogout,
    });

    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>
    );

    expect(screen.getByText("Users")).toBeInTheDocument();
    expect(screen.getByText("Security")).toBeInTheDocument();
    expect(screen.getByText("Audit Logs")).toBeInTheDocument();
    expect(screen.getByText("Moderation")).toBeInTheDocument();
    expect(screen.getByText("Reconciliation")).toBeInTheDocument();
    expect(screen.getByText("Exceptions")).toBeInTheDocument();
  });

  it("hides admin-only items for Scheduler role", () => {
    mockUseAuth.mockReturnValue({
      user: {
        id: "user-1",
        username: "scheduler",
        role_name: "Scheduler",
        full_name: "Scheduler User",
      },
      logout: mockLogout,
    });

    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>
    );

    expect(screen.queryByText("Users")).not.toBeInTheDocument();
    expect(screen.queryByText("Security")).not.toBeInTheDocument();
    expect(screen.queryByText("Audit Logs")).not.toBeInTheDocument();
    expect(screen.queryByText("Moderation")).not.toBeInTheDocument();
    expect(screen.queryByText("Reconciliation")).not.toBeInTheDocument();
  });

  it("shows Reviewer-specific items", () => {
    mockUseAuth.mockReturnValue({
      user: {
        id: "user-1",
        username: "reviewer",
        role_name: "Reviewer",
        full_name: "Reviewer User",
      },
      logout: mockLogout,
    });

    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>
    );

    expect(screen.getByText("Moderation")).toBeInTheDocument();
    expect(screen.queryByText("Users")).not.toBeInTheDocument();
    expect(screen.queryByText("Security")).not.toBeInTheDocument();
  });

  it("shows Auditor-specific items", () => {
    mockUseAuth.mockReturnValue({
      user: {
        id: "user-1",
        username: "auditor",
        role_name: "Auditor",
        full_name: "Auditor User",
      },
      logout: mockLogout,
    });

    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>
    );

    expect(screen.getByText("Audit Logs")).toBeInTheDocument();
    expect(screen.getByText("Reconciliation")).toBeInTheDocument();
    expect(screen.getByText("Exceptions")).toBeInTheDocument();
    expect(screen.queryByText("Users")).not.toBeInTheDocument();
    expect(screen.queryByText("Security")).not.toBeInTheDocument();
  });

  it("displays user info in sidebar footer", () => {
    mockUseAuth.mockReturnValue({
      user: {
        id: "user-1",
        username: "admin",
        role_name: "Administrator",
        full_name: "Admin User",
      },
      logout: mockLogout,
    });

    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>
    );

    expect(screen.getByText("Admin User")).toBeInTheDocument();
    expect(screen.getByText("Administrator")).toBeInTheDocument();
  });

  it("shows user initial avatar", () => {
    mockUseAuth.mockReturnValue({
      user: {
        id: "user-1",
        username: "admin",
        role_name: "Administrator",
        full_name: "Admin User",
      },
      logout: mockLogout,
    });

    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>
    );

    expect(screen.getByText("A")).toBeInTheDocument();
  });

  it("calls logout when sign out button is clicked", async () => {
    mockUseAuth.mockReturnValue({
      user: {
        id: "user-1",
        username: "admin",
        role_name: "Administrator",
        full_name: "Admin User",
      },
      logout: mockLogout,
    });

    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>
    );

    await userEvent.click(screen.getByText("Sign out"));
    expect(mockLogout).toHaveBeenCalledTimes(1);
  });

  it("renders application title", () => {
    mockUseAuth.mockReturnValue({
      user: {
        id: "user-1",
        username: "admin",
        role_name: "Administrator",
        full_name: "Admin User",
      },
      logout: mockLogout,
    });

    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>
    );

    expect(screen.getByText("Compliance Console")).toBeInTheDocument();
    expect(screen.getByText("Local Operations")).toBeInTheDocument();
  });
});
