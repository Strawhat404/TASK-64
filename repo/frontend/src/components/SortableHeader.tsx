import { useState, useCallback } from "react";

export type SortDirection = "asc" | "desc" | null;
export interface SortConfig {
  key: string;
  direction: SortDirection;
}

interface SortableHeaderProps {
  label: string;
  sortKey: string;
  currentSort: SortConfig;
  onSort: (key: string) => void;
  className?: string;
  align?: "left" | "right";
}

export function SortableHeader({ label, sortKey, currentSort, onSort, className = "", align = "left" }: SortableHeaderProps) {
  const isActive = currentSort.key === sortKey;
  const arrow = isActive ? (currentSort.direction === "asc" ? " \u25B2" : " \u25BC") : "";

  return (
    <th
      className={`${align === "right" ? "text-right" : "text-left"} px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider cursor-pointer hover:text-slate-200 select-none transition-colors ${className}`}
      onClick={() => onSort(sortKey)}
    >
      {label}{arrow}
    </th>
  );
}

export function useSortableData<T>(items: T[], defaultKey = "", defaultDir: SortDirection = null) {
  const [sortConfig, setSortConfig] = useState<SortConfig>({ key: defaultKey, direction: defaultDir });

  const requestSort = useCallback((key: string) => {
    setSortConfig((prev) => {
      if (prev.key === key) {
        if (prev.direction === "asc") return { key, direction: "desc" };
        if (prev.direction === "desc") return { key: "", direction: null };
        return { key, direction: "asc" };
      }
      return { key, direction: "asc" };
    });
  }, []);

  const sortedItems = [...items].sort((a, b) => {
    if (!sortConfig.key || !sortConfig.direction) return 0;
    const aVal = (a as Record<string, unknown>)[sortConfig.key];
    const bVal = (b as Record<string, unknown>)[sortConfig.key];
    if (aVal == null && bVal == null) return 0;
    if (aVal == null) return 1;
    if (bVal == null) return -1;
    if (typeof aVal === "number" && typeof bVal === "number") {
      return sortConfig.direction === "asc" ? aVal - bVal : bVal - aVal;
    }
    const aStr = String(aVal).toLowerCase();
    const bStr = String(bVal).toLowerCase();
    if (aStr < bStr) return sortConfig.direction === "asc" ? -1 : 1;
    if (aStr > bStr) return sortConfig.direction === "asc" ? 1 : -1;
    return 0;
  });

  return { sortedItems, sortConfig, requestSort };
}
