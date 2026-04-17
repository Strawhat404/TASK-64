import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import DashboardCard from "./DashboardCard";

describe("DashboardCard", () => {
  it("renders title and value", () => {
    render(<DashboardCard title="Total Users" value="42" />);

    expect(screen.getByText("Total Users")).toBeInTheDocument();
    expect(screen.getByText("42")).toBeInTheDocument();
  });

  it("renders with different values", () => {
    render(<DashboardCard title="Open Exceptions" value="7" />);

    expect(screen.getByText("Open Exceptions")).toBeInTheDocument();
    expect(screen.getByText("7")).toBeInTheDocument();
  });

  it("renders zero value", () => {
    render(<DashboardCard title="Errors" value="0" />);

    expect(screen.getByText("Errors")).toBeInTheDocument();
    expect(screen.getByText("0")).toBeInTheDocument();
  });

  it("renders with subtitle when provided", () => {
    render(
      <DashboardCard title="Match Rate" value="95%" subtitle="Last 30 days" />
    );

    expect(screen.getByText("Match Rate")).toBeInTheDocument();
    expect(screen.getByText("95%")).toBeInTheDocument();
  });
});
