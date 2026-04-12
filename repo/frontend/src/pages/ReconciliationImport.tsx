import { useState, useEffect, useCallback, useRef } from "react";
import axios from "axios";
import Sidebar from "../components/Sidebar";

interface Feed {
  id: string;
  filename: string;
  feed_type: string;
  record_count: number;
  status: string;
  imported_by: string;
  imported_at: string;
}

interface MatchResult {
  id: string;
  internal_tx_id: string;
  external_tx_id: string;
  match_confidence: number;
  match_method: string;
  amount_variance: number;
  status: string;
}

interface MatchSummary {
  matched_count: number;
  exception_count: number;
  total_internal: number;
  total_external: number;
}

const FEED_STATUS_COLORS: Record<string, string> = {
  pending: "bg-yellow-600/20 text-yellow-400 border border-yellow-600/40",
  processing: "bg-blue-600/20 text-blue-400 border border-blue-600/40",
  completed: "bg-green-600/20 text-green-400 border border-green-600/40",
  failed: "bg-red-600/20 text-red-400 border border-red-600/40",
};

export default function ReconciliationImport() {
  const [feeds, setFeeds] = useState<Feed[]>([]);
  const [matches, setMatches] = useState<MatchResult[]>([]);
  const [matchSummary, setMatchSummary] = useState<MatchSummary | null>(null);
  const [loading, setLoading] = useState(true);

  // Upload state
  const [feedType, setFeedType] = useState<"internal" | "external">("internal");
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Matching state
  const [matchingFeedId, setMatchingFeedId] = useState<string | null>(null);
  const [showMatches, setShowMatches] = useState(false);

  const fetchFeeds = useCallback(async () => {
    setLoading(true);
    try {
      const res = await axios.get("/api/reconciliation/feeds").catch(() => ({ data: [] }));
      const feedsData = res.data?.data ?? res.data;
      setFeeds(Array.isArray(feedsData) ? feedsData : []);
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchMatches = useCallback(async () => {
    try {
      const res = await axios.get("/api/reconciliation/matches").catch(() => ({ data: [] }));
      const matchData = res.data?.data ?? res.data;
      setMatches(Array.isArray(matchData) ? matchData : []);
      setShowMatches(true);
    } catch {
      // ignore
    }
  }, []);

  useEffect(() => {
    fetchFeeds();
  }, [fetchFeeds]);

  const handleFileDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer.files?.[0];
    if (file) setSelectedFile(file);
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) setSelectedFile(file);
  };

  const handleUpload = async () => {
    if (!selectedFile) return;
    setUploading(true);
    setUploadProgress(0);

    const formData = new FormData();
    formData.append("file", selectedFile);
    formData.append("feed_type", feedType);

    try {
      await axios.post("/api/reconciliation/import", formData, {
        headers: { "Content-Type": "multipart/form-data" },
        onUploadProgress: (e) => {
          if (e.total) {
            setUploadProgress(Math.round((e.loaded / e.total) * 100));
          }
        },
      });
      setSelectedFile(null);
      setUploadProgress(0);
      if (fileInputRef.current) fileInputRef.current.value = "";
      fetchFeeds();
    } catch {
      alert("Failed to upload file.");
    } finally {
      setUploading(false);
    }
  };

  const runMatching = async (feedId: string) => {
    setMatchingFeedId(feedId);
    try {
      const res = await axios.post(`/api/reconciliation/feeds/${feedId}/match`);
      setMatchSummary(res.data);
      await fetchFeeds();
      await fetchMatches();
    } catch {
      alert("Failed to run matching.");
    } finally {
      setMatchingFeedId(null);
    }
  };

  const confidenceColor = (c: number) => {
    if (c >= 90) return "text-green-400";
    if (c >= 70) return "text-yellow-400";
    return "text-red-400";
  };

  return (
    <div className="flex h-screen bg-slate-950">
      <Sidebar />
      <main className="flex-1 overflow-auto p-6">
        {/* Header */}
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-slate-100">Financial Reconciliation</h1>
          <p className="text-slate-400 mt-1">Import feeds and run transaction matching</p>
        </div>

        {/* Import Section */}
        <div className="bg-slate-800 rounded-xl border border-slate-700 p-6 mb-6">
          <h2 className="text-lg font-semibold text-slate-100 mb-4">Import Feed</h2>

          <div className="flex gap-4 mb-4">
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-2">Feed Type</label>
              <div className="flex gap-2">
                <button
                  onClick={() => setFeedType("internal")}
                  className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
                    feedType === "internal"
                      ? "bg-blue-600 text-white"
                      : "bg-slate-700 text-slate-300 hover:bg-slate-600"
                  }`}
                >
                  Internal
                </button>
                <button
                  onClick={() => setFeedType("external")}
                  className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
                    feedType === "external"
                      ? "bg-blue-600 text-white"
                      : "bg-slate-700 text-slate-300 hover:bg-slate-600"
                  }`}
                >
                  External
                </button>
              </div>
            </div>
          </div>

          {/* Dropzone */}
          <div
            onDragOver={(e) => {
              e.preventDefault();
              setDragOver(true);
            }}
            onDragLeave={() => setDragOver(false)}
            onDrop={handleFileDrop}
            onClick={() => fileInputRef.current?.click()}
            className={`border-2 border-dashed rounded-xl p-8 text-center cursor-pointer transition-colors ${
              dragOver
                ? "border-blue-500 bg-blue-600/10"
                : "border-slate-700 hover:border-slate-600"
            }`}
          >
            <input
              ref={fileInputRef}
              type="file"
              accept=".csv,.xlsx"
              onChange={handleFileSelect}
              className="hidden"
            />
            <p className="text-3xl text-slate-500 mb-2">{"\u2191"}</p>
            <p className="text-sm text-slate-300">
              {selectedFile
                ? selectedFile.name
                : "Drag and drop a file here, or click to browse"}
            </p>
            <p className="text-xs text-slate-500 mt-1">CSV, XLSX supported</p>
          </div>

          {/* Upload Progress */}
          {uploading && (
            <div className="mt-4">
              <div className="w-full bg-slate-700 rounded-full h-2">
                <div
                  className="h-2 rounded-full bg-blue-500 transition-all"
                  style={{ width: `${uploadProgress}%` }}
                />
              </div>
              <p className="text-xs text-slate-400 mt-1">{uploadProgress}% uploaded</p>
            </div>
          )}

          {selectedFile && !uploading && (
            <div className="mt-4 flex items-center justify-between">
              <p className="text-sm text-slate-300">
                Selected: <span className="font-medium text-slate-100">{selectedFile.name}</span>
              </p>
              <button
                onClick={handleUpload}
                className="px-4 py-2 text-sm font-medium rounded-lg bg-blue-600 hover:bg-blue-700 text-white transition-colors"
              >
                Upload
              </button>
            </div>
          )}
        </div>

        {/* Feeds Table */}
        <div className="bg-slate-800 rounded-xl border border-slate-700 overflow-hidden mb-6">
          <div className="px-4 py-3 border-b border-slate-700">
            <h2 className="text-sm font-semibold text-slate-200">Imported Feeds</h2>
          </div>
          {loading ? (
            <div className="flex items-center justify-center h-32">
              <div className="animate-spin w-6 h-6 border-2 border-slate-600 border-t-blue-500 rounded-full" />
            </div>
          ) : (
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700">
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Filename</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Type</th>
                  <th className="text-right px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Records</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Status</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Imported By</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Date</th>
                  <th className="text-right px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700">
                {feeds.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="px-4 py-12 text-center text-slate-500">
                      No feeds imported yet
                    </td>
                  </tr>
                ) : (
                  feeds.map((feed) => (
                    <tr key={feed.id} className="bg-slate-800 hover:bg-slate-750 transition-colors">
                      <td className="px-4 py-3 text-sm text-slate-200 font-medium">{feed.filename}</td>
                      <td className="px-4 py-3 text-sm text-slate-300 capitalize">{feed.feed_type}</td>
                      <td className="px-4 py-3 text-sm text-slate-300 text-right font-mono">{feed.record_count}</td>
                      <td className="px-4 py-3 text-sm">
                        <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${FEED_STATUS_COLORS[feed.status] || ""}`}>
                          {feed.status}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-sm text-slate-300">{feed.imported_by}</td>
                      <td className="px-4 py-3 text-sm text-slate-400">
                        {new Date(feed.imported_at).toLocaleDateString()}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <button
                          onClick={() => runMatching(feed.id)}
                          disabled={matchingFeedId === feed.id || feed.status === "processing"}
                          className="px-3 py-1.5 text-xs font-medium rounded-md bg-blue-600 hover:bg-blue-700 text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                          {matchingFeedId === feed.id ? "Matching..." : "Run Matching"}
                        </button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          )}
        </div>

        {/* Match Results */}
        {showMatches && (
          <div className="bg-slate-800 rounded-xl border border-slate-700 overflow-hidden">
            <div className="px-4 py-3 border-b border-slate-700">
              <h2 className="text-sm font-semibold text-slate-200">Match Results</h2>
            </div>

            {/* Summary Cards */}
            {matchSummary && (
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 p-4 border-b border-slate-700">
                <div className="bg-slate-900 rounded-lg border border-slate-700 p-4">
                  <p className="text-xs font-medium text-slate-400 uppercase tracking-wider">Total Matched</p>
                  <p className="text-xl font-bold text-slate-100 mt-1">{matchSummary.matched_count}</p>
                </div>
                <div className="bg-slate-900 rounded-lg border border-slate-700 p-4">
                  <p className="text-xs font-medium text-slate-400 uppercase tracking-wider">Exceptions</p>
                  <p className="text-xl font-bold text-red-400 mt-1">{matchSummary.exception_count}</p>
                </div>
                <div className="bg-slate-900 rounded-lg border border-slate-700 p-4">
                  <p className="text-xs font-medium text-slate-400 uppercase tracking-wider">Total Internal / External</p>
                  <p className="text-xl font-bold text-slate-100 mt-1">{matchSummary.total_internal} / {matchSummary.total_external}</p>
                </div>
              </div>
            )}

            {/* Matches Table */}
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700">
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Internal Tx</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">External Tx</th>
                  <th className="text-right px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Confidence</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Method</th>
                  <th className="text-right px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Variance</th>
                  <th className="text-left px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700">
                {matches.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-4 py-12 text-center text-slate-500">
                      No match results yet
                    </td>
                  </tr>
                ) : (
                  matches.map((m) => (
                    <tr key={m.id} className="bg-slate-800 hover:bg-slate-750 transition-colors">
                      <td className="px-4 py-3 text-sm text-slate-200 font-mono">{m.internal_tx_id}</td>
                      <td className="px-4 py-3 text-sm text-slate-200 font-mono">{m.external_tx_id}</td>
                      <td className={`px-4 py-3 text-sm text-right font-bold ${confidenceColor(m.match_confidence)}`}>
                        {m.match_confidence}%
                      </td>
                      <td className="px-4 py-3 text-sm text-slate-300">{m.match_method}</td>
                      <td className="px-4 py-3 text-sm text-right font-mono text-slate-300">
                        ${m.amount_variance?.toFixed(2)}
                      </td>
                      <td className="px-4 py-3 text-sm text-slate-300 capitalize">{m.status}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        )}
      </main>
    </div>
  );
}
