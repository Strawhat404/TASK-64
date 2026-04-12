import { useState } from "react";
import axios from "axios";

interface ExportButtonProps {
  url: string;
  filename: string;
  format: "csv" | "excel";
  label?: string;
}

export default function ExportButton({ url, filename, format, label }: ExportButtonProps) {
  const [loading, setLoading] = useState(false);

  const handleExport = async () => {
    setLoading(true);
    try {
      const isExcel = format === "excel";
      const res = await axios.get(url, { responseType: isExcel ? "blob" : "text" });

      let blob: Blob;
      if (isExcel) {
        blob = new Blob([res.data], {
          type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        });
      } else {
        blob = new Blob([res.data], { type: "text/csv" });
      }

      const downloadUrl = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = downloadUrl;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(downloadUrl);
    } catch {
      alert("Failed to export data.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <button
      onClick={handleExport}
      disabled={loading}
      className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg bg-slate-700 hover:bg-slate-600 text-slate-300 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
    >
      {loading ? (
        <span className="inline-block w-4 h-4 border-2 border-slate-500 border-t-slate-200 rounded-full animate-spin" />
      ) : (
        <span>{"\u2193"}</span>
      )}
      {label || (format === "excel" ? "Excel Export" : "CSV Export")}
    </button>
  );
}
