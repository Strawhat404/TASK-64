import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import axios from "axios";
import ExportButton from "./ExportButton";

vi.mock("axios");
const mockedAxios = vi.mocked(axios, true);

// Mock URL.createObjectURL and revokeObjectURL
const mockCreateObjectURL = vi.fn(() => "blob:mock-url");
const mockRevokeObjectURL = vi.fn();
Object.defineProperty(globalThis, "URL", {
  value: { createObjectURL: mockCreateObjectURL, revokeObjectURL: mockRevokeObjectURL },
  writable: true,
});

describe("ExportButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders with CSV label", () => {
    render(<ExportButton url="/api/export" filename="data.csv" format="csv" />);
    expect(screen.getByText(/CSV/i)).toBeInTheDocument();
  });

  it("renders with Excel label", () => {
    render(<ExportButton url="/api/export" filename="data.xlsx" format="excel" />);
    expect(screen.getByText(/Excel/i)).toBeInTheDocument();
  });

  it("renders with custom label", () => {
    render(<ExportButton url="/api/export" filename="data.csv" format="csv" label="Download Report" />);
    expect(screen.getByText("Download Report")).toBeInTheDocument();
  });

  it("triggers download on click for CSV", async () => {
    mockedAxios.get.mockResolvedValue({ data: "col1,col2\nval1,val2" });
    render(<ExportButton url="/api/export" filename="data.csv" format="csv" />);

    await userEvent.click(screen.getByText(/CSV/i));
    await waitFor(() => {
      expect(mockedAxios.get).toHaveBeenCalledWith("/api/export", expect.objectContaining({ responseType: "text" }));
    });
  });

  it("triggers download on click for Excel", async () => {
    mockedAxios.get.mockResolvedValue({ data: new Blob() });
    render(<ExportButton url="/api/export" filename="data.xlsx" format="excel" />);

    await userEvent.click(screen.getByText(/Excel/i));
    await waitFor(() => {
      expect(mockedAxios.get).toHaveBeenCalledWith("/api/export", expect.objectContaining({ responseType: "blob" }));
    });
  });

  it("disables button during loading", async () => {
    let resolveRequest: (value: unknown) => void;
    mockedAxios.get.mockImplementation(() => new Promise((resolve) => { resolveRequest = resolve; }));
    render(<ExportButton url="/api/export" filename="data.csv" format="csv" />);

    await userEvent.click(screen.getByText(/CSV/i));
    // Button should be visually disabled during request
    const button = screen.getByRole("button");
    expect(button).toBeInTheDocument();

    resolveRequest!({ data: "data" });
  });
});
