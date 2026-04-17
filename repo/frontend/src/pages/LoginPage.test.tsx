import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import LoginPage from "./LoginPage";

const mockLogin = vi.fn();
const mockNavigate = vi.fn();

vi.mock("../context/AuthContext", () => ({
  useAuth: () => ({
    user: null,
    login: mockLogin,
  }),
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

describe("LoginPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders login form", () => {
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>
    );

    expect(screen.getByText("Compliance Console")).toBeInTheDocument();
    expect(screen.getByText("Local Operations Management")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Enter your username")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Enter your password")).toBeInTheDocument();
    expect(screen.getByText("Sign in")).toBeInTheDocument();
  });

  it("has required fields", () => {
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>
    );

    const usernameInput = screen.getByPlaceholderText("Enter your username");
    const passwordInput = screen.getByPlaceholderText("Enter your password");

    expect(usernameInput).toBeRequired();
    expect(passwordInput).toBeRequired();
  });

  it("password field is type password", () => {
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>
    );

    const passwordInput = screen.getByPlaceholderText("Enter your password");
    expect(passwordInput).toHaveAttribute("type", "password");
  });

  it("calls login on form submit", async () => {
    mockLogin.mockResolvedValueOnce(undefined);

    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>
    );

    await userEvent.type(screen.getByPlaceholderText("Enter your username"), "admin");
    await userEvent.type(screen.getByPlaceholderText("Enter your password"), "password12345");
    await userEvent.click(screen.getByText("Sign in"));

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith("admin", "password12345", undefined);
    });
  });

  it("displays error message on login failure", async () => {
    mockLogin.mockRejectedValueOnce({
      response: {
        data: {
          error: "Invalid credentials",
        },
      },
    });

    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>
    );

    await userEvent.type(screen.getByPlaceholderText("Enter your username"), "admin");
    await userEvent.type(screen.getByPlaceholderText("Enter your password"), "wrongpassword1");
    await userEvent.click(screen.getByText("Sign in"));

    await waitFor(() => {
      expect(screen.getByText("Invalid credentials")).toBeInTheDocument();
    });
  });

  it("shows captcha challenge when required", async () => {
    mockLogin.mockRejectedValueOnce({
      response: {
        data: {
          error: "CAPTCHA verification required",
          captcha_required: "true",
          captcha_challenge: "What is 5 + 3?",
        },
      },
    });

    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>
    );

    await userEvent.type(screen.getByPlaceholderText("Enter your username"), "admin");
    await userEvent.type(screen.getByPlaceholderText("Enter your password"), "wrongpassword1");
    await userEvent.click(screen.getByText("Sign in"));

    await waitFor(() => {
      expect(screen.getByText(/What is 5 \+ 3\?/)).toBeInTheDocument();
      expect(screen.getByPlaceholderText("Enter your answer")).toBeInTheDocument();
    });
  });

  it("shows loading state during login", async () => {
    let resolveLogin: () => void;
    mockLogin.mockImplementationOnce(
      () => new Promise<void>((resolve) => { resolveLogin = resolve; })
    );

    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>
    );

    await userEvent.type(screen.getByPlaceholderText("Enter your username"), "admin");
    await userEvent.type(screen.getByPlaceholderText("Enter your password"), "password12345");
    await userEvent.click(screen.getByText("Sign in"));

    await waitFor(() => {
      expect(screen.getByText("Signing in...")).toBeInTheDocument();
    });

    // Button should be disabled
    expect(screen.getByText("Signing in...").closest("button")).toBeDisabled();

    // Resolve login
    resolveLogin!();
  });

  it("displays form labels", () => {
    render(
      <MemoryRouter>
        <LoginPage />
      </MemoryRouter>
    );

    expect(screen.getByText("Username")).toBeInTheDocument();
    expect(screen.getByText("Password")).toBeInTheDocument();
  });
});
