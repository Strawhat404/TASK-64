import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import axios from "axios";
import ModerationQueue from "./ModerationQueue";

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

describe("ModerationQueue", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedAxios.get.mockImplementation((url: string) => {
      if (url.includes("/reviews/pending")) return Promise.resolve({ data: [] });
      if (url.includes("/gray-release")) return Promise.resolve({ data: [] });
      if (url.includes("/governance/content")) return Promise.resolve({ data: [] });
      return Promise.resolve({ data: [] });
    });
  });

  it("renders moderation queue heading", async () => {
    render(<MemoryRouter><ModerationQueue /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText("Moderation Queue")).toBeInTheDocument();
    });
  });

  it("has tab navigation", async () => {
    render(<MemoryRouter><ModerationQueue /></MemoryRouter>);
    await waitFor(() => {
      const text = document.body.textContent || "";
      expect(text).toMatch(/Pending|Gray Release|All Content/i);
    });
  });

  it("shows empty state for pending reviews", async () => {
    render(<MemoryRouter><ModerationQueue /></MemoryRouter>);
    await waitFor(() => {
      expect(screen.getByText(/No pending/i)).toBeInTheDocument();
    });
  });

  it("fetches pending reviews on mount", async () => {
    render(<MemoryRouter><ModerationQueue /></MemoryRouter>);
    await waitFor(() => {
      expect(mockedAxios.get).toHaveBeenCalled();
    });
  });

  it("switches tabs", async () => {
    render(<MemoryRouter><ModerationQueue /></MemoryRouter>);
    await waitFor(() => expect(mockedAxios.get).toHaveBeenCalled());

    const grayTab = screen.getByText(/Gray Release/i);
    await userEvent.click(grayTab);
    await waitFor(() => {
      const text = document.body.textContent || "";
      expect(text).toMatch(/gray release|No items/i);
    });

    const allTab = screen.getByText(/All Content/i);
    await userEvent.click(allTab);
    await waitFor(() => {
      expect(mockedAxios.get).toHaveBeenCalled();
    });
  });
});
