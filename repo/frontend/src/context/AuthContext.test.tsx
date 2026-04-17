import { render, screen, waitFor, act } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import axios from "axios";
import { AuthProvider, useAuth } from "./AuthContext";

vi.mock("axios");
const mockedAxios = vi.mocked(axios, true);

function TestConsumer() {
  const { user, loading, login, logout } = useAuth();
  return (
    <div>
      {loading && <span data-testid="loading">Loading</span>}
      {user && <span data-testid="username">{user.username}</span>}
      {user && <span data-testid="role">{user.role_name}</span>}
      {!user && !loading && <span data-testid="no-user">No user</span>}
      <button data-testid="login-btn" onClick={() => login("admin", "password12345")}>
        Login
      </button>
      <button data-testid="logout-btn" onClick={() => logout()}>
        Logout
      </button>
    </div>
  );
}

describe("AuthContext", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state initially then resolves to no user", async () => {
    mockedAxios.get.mockRejectedValueOnce(new Error("Not authenticated"));

    render(
      <AuthProvider>
        <TestConsumer />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId("no-user")).toBeInTheDocument();
    });
  });

  it("loads existing session on mount", async () => {
    mockedAxios.get.mockResolvedValueOnce({
      data: {
        user: {
          id: "user-1",
          username: "admin",
          email: "admin@test.com",
          role_name: "Administrator",
          full_name: "Admin User",
        },
      },
    });

    render(
      <AuthProvider>
        <TestConsumer />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId("username")).toHaveTextContent("admin");
      expect(screen.getByTestId("role")).toHaveTextContent("Administrator");
    });
  });

  it("login sets user on success", async () => {
    mockedAxios.get.mockRejectedValueOnce(new Error("Not authenticated"));
    mockedAxios.post.mockResolvedValueOnce({
      data: {
        user: {
          id: "user-1",
          username: "admin",
          email: "admin@test.com",
          role_name: "Administrator",
          full_name: "Admin User",
        },
      },
    });

    render(
      <AuthProvider>
        <TestConsumer />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId("no-user")).toBeInTheDocument();
    });

    await act(async () => {
      await userEvent.click(screen.getByTestId("login-btn"));
    });

    await waitFor(() => {
      expect(screen.getByTestId("username")).toHaveTextContent("admin");
    });

    expect(mockedAxios.post).toHaveBeenCalledWith(
      "/api/auth/login",
      { username: "admin", password: "password12345" },
      { headers: {} }
    );
  });

  it("logout clears user", async () => {
    mockedAxios.get.mockResolvedValueOnce({
      data: {
        user: {
          id: "user-1",
          username: "admin",
          email: "admin@test.com",
          role_name: "Administrator",
          full_name: "Admin User",
        },
      },
    });
    mockedAxios.post.mockResolvedValueOnce({});

    render(
      <AuthProvider>
        <TestConsumer />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId("username")).toHaveTextContent("admin");
    });

    await act(async () => {
      await userEvent.click(screen.getByTestId("logout-btn"));
    });

    await waitFor(() => {
      expect(screen.getByTestId("no-user")).toBeInTheDocument();
    });

    expect(mockedAxios.post).toHaveBeenCalledWith("/api/auth/logout");
  });

  it("throws error when useAuth is used outside provider", () => {
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});

    expect(() => {
      render(<TestConsumer />);
    }).toThrow("useAuth must be used within AuthProvider");

    spy.mockRestore();
  });

  it("login sends captcha header when provided", async () => {
    mockedAxios.get.mockRejectedValueOnce(new Error("Not authenticated"));
    mockedAxios.post.mockResolvedValueOnce({
      data: {
        user: {
          id: "user-1",
          username: "admin",
          email: "admin@test.com",
          role_name: "Administrator",
          full_name: "Admin User",
        },
      },
    });

    function CaptchaConsumer() {
      const { user, loading, login } = useAuth();
      return (
        <div>
          {loading && <span data-testid="loading">Loading</span>}
          {user && <span data-testid="username">{user.username}</span>}
          {!user && !loading && <span data-testid="no-user">No user</span>}
          <button
            data-testid="captcha-login"
            onClick={() => login("admin", "password12345", "42")}
          >
            Login with Captcha
          </button>
        </div>
      );
    }

    render(
      <AuthProvider>
        <CaptchaConsumer />
      </AuthProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId("no-user")).toBeInTheDocument();
    });

    await act(async () => {
      await userEvent.click(screen.getByTestId("captcha-login"));
    });

    expect(mockedAxios.post).toHaveBeenCalledWith(
      "/api/auth/login",
      { username: "admin", password: "password12345" },
      { headers: { "X-Captcha-Token": "42" } }
    );
  });
});
