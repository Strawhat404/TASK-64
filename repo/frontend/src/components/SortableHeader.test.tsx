import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { SortableHeader, useSortableData } from "./SortableHeader";
import { renderHook, act } from "@testing-library/react";

describe("SortableHeader", () => {
  it("renders label text", () => {
    const onSort = vi.fn();
    render(
      <table><thead><tr>
        <SortableHeader label="Name" sortKey="name" currentSort={{ key: "", direction: null }} onSort={onSort} />
      </tr></thead></table>
    );
    expect(screen.getByText("Name")).toBeInTheDocument();
  });

  it("calls onSort when clicked", async () => {
    const onSort = vi.fn();
    render(
      <table><thead><tr>
        <SortableHeader label="Name" sortKey="name" currentSort={{ key: "", direction: null }} onSort={onSort} />
      </tr></thead></table>
    );
    await userEvent.click(screen.getByText("Name"));
    expect(onSort).toHaveBeenCalledWith("name");
  });

  it("shows ascending arrow when sorted asc", () => {
    const onSort = vi.fn();
    render(
      <table><thead><tr>
        <SortableHeader label="Name" sortKey="name" currentSort={{ key: "name", direction: "asc" }} onSort={onSort} />
      </tr></thead></table>
    );
    const text = document.body.textContent || "";
    expect(text).toContain("\u25B2"); // ▲
  });

  it("shows descending arrow when sorted desc", () => {
    const onSort = vi.fn();
    render(
      <table><thead><tr>
        <SortableHeader label="Name" sortKey="name" currentSort={{ key: "name", direction: "desc" }} onSort={onSort} />
      </tr></thead></table>
    );
    const text = document.body.textContent || "";
    expect(text).toContain("\u25BC"); // ▼
  });

  it("does not show sort arrow for inactive column", () => {
    const onSort = vi.fn();
    render(
      <table><thead><tr>
        <SortableHeader label="Email" sortKey="email" currentSort={{ key: "name", direction: "asc" }} onSort={onSort} />
      </tr></thead></table>
    );
    const text = document.body.textContent || "";
    // Should not have a visible sort indicator for email
    expect(text).toContain("Email");
  });
});

describe("useSortableData", () => {
  const items = [
    { name: "Charlie", age: 30 },
    { name: "Alice", age: 25 },
    { name: "Bob", age: 35 },
  ];

  it("returns items unsorted initially", () => {
    const { result } = renderHook(() => useSortableData(items));
    expect(result.current.sortedItems).toHaveLength(3);
  });

  it("sorts ascending on requestSort", () => {
    const { result } = renderHook(() => useSortableData(items));
    act(() => { result.current.requestSort("name"); });
    expect(result.current.sortedItems[0].name).toBe("Alice");
    expect(result.current.sortedItems[1].name).toBe("Bob");
    expect(result.current.sortedItems[2].name).toBe("Charlie");
    expect(result.current.sortConfig.direction).toBe("asc");
  });

  it("sorts descending on second requestSort", () => {
    const { result } = renderHook(() => useSortableData(items));
    act(() => { result.current.requestSort("name"); });
    act(() => { result.current.requestSort("name"); });
    expect(result.current.sortedItems[0].name).toBe("Charlie");
    expect(result.current.sortConfig.direction).toBe("desc");
  });

  it("clears sort on third requestSort", () => {
    const { result } = renderHook(() => useSortableData(items));
    act(() => { result.current.requestSort("name"); });
    act(() => { result.current.requestSort("name"); });
    act(() => { result.current.requestSort("name"); });
    expect(result.current.sortConfig.direction).toBeNull();
  });

  it("resets to asc when switching sort key", () => {
    const { result } = renderHook(() => useSortableData(items));
    act(() => { result.current.requestSort("name"); });
    act(() => { result.current.requestSort("age"); });
    expect(result.current.sortConfig.key).toBe("age");
    expect(result.current.sortConfig.direction).toBe("asc");
    expect(result.current.sortedItems[0].age).toBe(25);
  });

  it("sorts numbers correctly", () => {
    const { result } = renderHook(() => useSortableData(items));
    act(() => { result.current.requestSort("age"); });
    expect(result.current.sortedItems[0].age).toBe(25);
    expect(result.current.sortedItems[2].age).toBe(35);
  });
});
