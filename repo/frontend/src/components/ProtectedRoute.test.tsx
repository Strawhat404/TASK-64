import { render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import ProtectedRoute from "./ProtectedRoute";

const mockUseAuth = vi.fn();

vi.mock("../context/AuthContext", () => ({
  useAuth: () => mockUseAuth(),
}));

describe("ProtectedRoute", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state when auth is loading", () => {
    mockUseAuth.mockReturnValue({ user: null, loading: true });

    render(
      <MemoryRouter>
        <ProtectedRoute>
          <div>Protected Content</div>
        </ProtectedRoute>
      </MemoryRouter>
    );

    expect(screen.getByText("Loading...")).toBeInTheDocument();
    expect(screen.queryByText("Protected Content")).not.toBeInTheDocument();
  });

  it("redirects to /login when not authenticated", async () => {
    mockUseAuth.mockReturnValue({ user: null, loading: false });

    render(
      <MemoryRouter initialEntries={["/dashboard"]}>
        <Routes>
          <Route
            path="/dashboard"
            element={
              <ProtectedRoute>
                <div>Protected Content</div>
              </ProtectedRoute>
            }
          />
          <Route path="/login" element={<div>Login Page</div>} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText("Login Page")).toBeInTheDocument();
    });
    expect(screen.queryByText("Protected Content")).not.toBeInTheDocument();
  });

  it("renders children when authenticated", async () => {
    mockUseAuth.mockReturnValue({
      user: {
        id: "user-1",
        username: "admin",
        role_name: "Administrator",
        email: "admin@test.com",
        full_name: "Admin User",
      },
      loading: false,
    });

    render(
      <MemoryRouter>
        <ProtectedRoute>
          <div>Protected Content</div>
        </ProtectedRoute>
      </MemoryRouter>
    );

    expect(screen.getByText("Protected Content")).toBeInTheDocument();
  });

  it("renders children when user has required role", () => {
    mockUseAuth.mockReturnValue({
      user: {
        id: "user-1",
        username: "admin",
        role_name: "Administrator",
        email: "admin@test.com",
        full_name: "Admin User",
      },
      loading: false,
    });

    render(
      <MemoryRouter>
        <ProtectedRoute roles={["Administrator", "Auditor"]}>
          <div>Admin Content</div>
        </ProtectedRoute>
      </MemoryRouter>
    );

    expect(screen.getByText("Admin Content")).toBeInTheDocument();
  });

  it("redirects to / when user lacks required role", async () => {
    mockUseAuth.mockReturnValue({
      user: {
        id: "user-1",
        username: "reviewer",
        role_name: "Reviewer",
        email: "reviewer@test.com",
        full_name: "Reviewer User",
      },
      loading: false,
    });

    render(
      <MemoryRouter initialEntries={["/admin"]}>
        <Routes>
          <Route
            path="/admin"
            element={
              <ProtectedRoute roles={["Administrator"]}>
                <div>Admin Content</div>
              </ProtectedRoute>
            }
          />
          <Route path="/" element={<div>Dashboard</div>} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText("Dashboard")).toBeInTheDocument();
    });
    expect(screen.queryByText("Admin Content")).not.toBeInTheDocument();
  });

  it("renders when no roles restriction is specified", () => {
    mockUseAuth.mockReturnValue({
      user: {
        id: "user-1",
        username: "reviewer",
        role_name: "Reviewer",
        email: "reviewer@test.com",
        full_name: "Reviewer User",
      },
      loading: false,
    });

    render(
      <MemoryRouter>
        <ProtectedRoute>
          <div>Any User Content</div>
        </ProtectedRoute>
      </MemoryRouter>
    );

    expect(screen.getByText("Any User Content")).toBeInTheDocument();
  });
});
