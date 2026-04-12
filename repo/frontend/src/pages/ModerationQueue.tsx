import { useState, useEffect, useCallback } from "react";
import axios from "axios";
import Sidebar from "../components/Sidebar";
import { SortableHeader, useSortableData } from "../components/SortableHeader";

interface ContentItem {
  id: string;
  title: string;
  content_type: string;
  status: string;
  submitted_by: string;
  submitted_at: string;
  review_level: string;
  severity: string;
  gray_release_at?: string;
  versions?: ContentVersion[];
}

interface ContentVersion {
  version: number;
  updated_at: string;
  updated_by: string;
  changes_summary: string;
}

interface PendingReview {
  id: string;
  content_id: string;
  title: string;
  content_type: string;
  submitted_by: string;
  submitted_at: string;
  review_level: string;
  severity: string;
}

interface GrayReleaseItem {
  id: string;
  title: string;
  content_type: string;
  gray_release_at: string;
  status: string;
}

const STATUS_COLORS: Record<string, string> = {
  draft: "bg-slate-600 text-slate-200",
  pending_review: "bg-yellow-600/20 text-yellow-400 border border-yellow-600/40",
  in_review: "bg-blue-600/20 text-blue-400 border border-blue-600/40",
  approved: "bg-green-600/20 text-green-400 border border-green-600/40",
  gray_release: "bg-purple-600/20 text-purple-400 border border-purple-600/40",
  published: "bg-emerald-600/20 text-emerald-400 border border-emerald-600/40",
  rejected: "bg-red-600/20 text-red-400 border border-red-600/40",
  archived: "bg-slate-600/40 text-slate-400",
};

function StatusBadge({ status }: { status: string }) {
  const cls = STATUS_COLORS[status] || "bg-slate-600 text-slate-300";
  return (
    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${cls}`}>
      {status.replace(/_/g, " ")}
    </span>
  );
}

export default function ModerationQueue() {
  const [activeTab, setActiveTab] = useState<"pending" | "gray" | "all">("pending");
  const [pendingReviews, setPendingReviews] = useState<PendingReview[]>([]);
  const [grayReleaseItems, setGrayReleaseItems] = useState<GrayReleaseItem[]>([]);
  const [allContent, setAllContent] = useState<ContentItem[]>([]);
  const [loading, setLoading] = useState(true);

  // Filters for All Content tab
  const [statusFilter, setStatusFilter] = useState("");
  const [typeFilter, setTypeFilter] = useState("");

  // Review decision modal
  const [reviewModal, setReviewModal] = useState<{
    open: boolean;
    reviewId: string;
    contentTitle: string;
    decision: "approved" | "rejected" | "escalated";
  } | null>(null);
  const [reviewNotes, setReviewNotes] = useState("");
  const [submitting, setSubmitting] = useState(false);

  // Content detail modal
  const [detailModal, setDetailModal] = useState<ContentItem | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [pendingRes, grayRes, allRes] = await Promise.all([
        axios.get("/api/governance/reviews/pending").catch(() => ({ data: [] })),
        axios.get("/api/governance/gray-release").catch(() => ({ data: [] })),
        axios.get("/api/governance/content").catch(() => ({ data: [] })),
      ]);
      setPendingReviews(Array.isArray(pendingRes.data) ? pendingRes.data : []);
      setGrayReleaseItems(Array.isArray(grayRes.data) ? grayRes.data : []);
      setAllContent(Array.isArray(allRes.data) ? allRes.data : []);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const openReviewModal = (
    reviewId: string,
    contentTitle: string,
    decision: "approved" | "rejected" | "escalated"
  ) => {
    setReviewModal({ open: true, reviewId, contentTitle, decision });
    setReviewNotes("");
  };

  const submitReviewDecision = async () => {
    if (!reviewModal) return;
    setSubmitting(true);
    try {
      await axios.post(`/api/governance/reviews/${reviewModal.reviewId}/decide`, {
        decision: reviewModal.decision,
        decision_notes: reviewNotes,
      });
      setReviewModal(null);
      fetchData();
    } catch {
      alert("Failed to submit review decision.");
    } finally {
      setSubmitting(false);
    }
  };

  const promoteContent = async (id: string) => {
    try {
      await axios.post(`/api/governance/content/${id}/promote`);
      fetchData();
    } catch {
      alert("Failed to promote content.");
    }
  };

  const rollbackContent = async (id: string, targetVersion: number) => {
    try {
      await axios.post(`/api/governance/content/${id}/rollback`, {
        target_version: targetVersion,
      });
      fetchData();
      setDetailModal(null);
    } catch {
      alert("Failed to rollback content.");
    }
  };

  const getHoursRemaining = (grayReleaseAt: string): number => {
    const releaseTime = new Date(grayReleaseAt).getTime();
    const now = Date.now();
    const elapsed = (now - releaseTime) / (1000 * 60 * 60);
    return Math.max(0, 24 - elapsed);
  };

  const getProgressPercent = (grayReleaseAt: string): number => {
    const releaseTime = new Date(grayReleaseAt).getTime();
    const now = Date.now();
    const elapsed = (now - releaseTime) / (1000 * 60 * 60);
    return Math.min(100, (elapsed / 24) * 100);
  };

  const filteredContent = allContent.filter((c) => {
    if (statusFilter && c.status !== statusFilter) return false;
    if (typeFilter && c.content_type !== typeFilter) return false;
    return true;
  });

  const { sortedItems: sortedPendingReviews, sortConfig: pendingSortConfig, requestSort: requestPendingSort } = useSortableData(pendingReviews);
  const { sortedItems: sortedFilteredContent, sortConfig: contentSortConfig, requestSort: requestContentSort } = useSortableData(filteredContent);

  const contentTypes = [...new Set(allContent.map((c) => c.content_type))];
  const statuses = [...new Set(allContent.map((c) => c.status))];

  const tabs = [
    { key: "pending" as const, label: "Pending Review" },
    { key: "gray" as const, label: "Gray Release" },
    { key: "all" as const, label: "All Content" },
  ];

  return (
    <div className="flex h-screen bg-slate-950">
      <Sidebar />
      <main className="flex-1 overflow-auto p-6">
        {/* Header */}
        <div className="mb-6">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-slate-100">Moderation Queue</h1>
            <span className="px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-600/20 text-blue-400 border border-blue-600/40">
              {pendingReviews.length} pending
            </span>
          </div>
          <p className="text-slate-400 mt-1">Review, approve, and manage content lifecycle</p>
        </div>

        {/* Tabs */}
        <div className="flex gap-1 mb-6 bg-slate-900 rounded-lg p-1 w-fit">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                activeTab === tab.key
                  ? "bg-slate-700 text-slate-100"
                  : "text-slate-400 hover:text-slate-200"
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin w-8 h-8 border-2 border-slate-600 border-t-blue-500 rounded-full" />
          </div>
        ) : (
          <>
            {/* Pending Review Tab */}
            {activeTab === "pending" && (
              <div className="bg-slate-800 rounded-xl border border-slate-700 overflow-hidden">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-slate-700">
                      <SortableHeader label="Title" sortKey="title" currentSort={pendingSortConfig} onSort={requestPendingSort} />
                      <SortableHeader label="Type" sortKey="content_type" currentSort={pendingSortConfig} onSort={requestPendingSort} />
                      <SortableHeader label="Submitted By" sortKey="submitted_by" currentSort={pendingSortConfig} onSort={requestPendingSort} />
                      <SortableHeader label="Date" sortKey="submitted_at" currentSort={pendingSortConfig} onSort={requestPendingSort} />
                      <SortableHeader label="Review Level" sortKey="review_level" currentSort={pendingSortConfig} onSort={requestPendingSort} />
                      <SortableHeader label="Severity" sortKey="severity" currentSort={pendingSortConfig} onSort={requestPendingSort} />
                      <th className="text-right px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-700">
                    {sortedPendingReviews.length === 0 ? (
                      <tr>
                        <td colSpan={7} className="px-4 py-12 text-center text-slate-500">
                          No pending reviews
                        </td>
                      </tr>
                    ) : (
                      sortedPendingReviews.map((review) => (
                        <tr key={review.id} className="bg-slate-800 hover:bg-slate-750 transition-colors">
                          <td className="px-4 py-3 text-sm text-slate-200 font-medium">{review.title}</td>
                          <td className="px-4 py-3 text-sm text-slate-300">{review.content_type}</td>
                          <td className="px-4 py-3 text-sm text-slate-300">{review.submitted_by}</td>
                          <td className="px-4 py-3 text-sm text-slate-400">
                            {new Date(review.submitted_at).toLocaleDateString()}
                          </td>
                          <td className="px-4 py-3 text-sm text-slate-300 capitalize">{review.review_level}</td>
                          <td className="px-4 py-3 text-sm">
                            <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                              review.severity === "critical" ? "bg-red-600/20 text-red-400" :
                              review.severity === "high" ? "bg-orange-600/20 text-orange-400" :
                              review.severity === "medium" ? "bg-yellow-600/20 text-yellow-400" :
                              "bg-slate-600/20 text-slate-400"
                            }`}>
                              {review.severity}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-right">
                            <div className="flex gap-2 justify-end">
                              <button
                                onClick={() => openReviewModal(review.id, review.title, "approved")}
                                className="px-3 py-1.5 text-xs font-medium rounded-md bg-green-600 hover:bg-green-700 text-white transition-colors"
                              >
                                Approve
                              </button>
                              <button
                                onClick={() => openReviewModal(review.id, review.title, "rejected")}
                                className="px-3 py-1.5 text-xs font-medium rounded-md bg-red-600 hover:bg-red-700 text-white transition-colors"
                              >
                                Reject
                              </button>
                              <button
                                onClick={() => openReviewModal(review.id, review.title, "escalated")}
                                className="px-3 py-1.5 text-xs font-medium rounded-md bg-yellow-600 hover:bg-yellow-700 text-white transition-colors"
                              >
                                Escalate
                              </button>
                            </div>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            )}

            {/* Gray Release Tab */}
            {activeTab === "gray" && (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {grayReleaseItems.length === 0 ? (
                  <div className="col-span-full text-center py-12 text-slate-500">
                    No items in gray release
                  </div>
                ) : (
                  grayReleaseItems.map((item) => {
                    const hoursRemaining = getHoursRemaining(item.gray_release_at);
                    const progress = getProgressPercent(item.gray_release_at);
                    const canPromote = hoursRemaining <= 0;

                    return (
                      <div key={item.id} className="bg-slate-800 rounded-xl border border-slate-700 p-6">
                        <div className="flex items-start justify-between mb-3">
                          <h3 className="text-sm font-semibold text-slate-100">{item.title}</h3>
                          <StatusBadge status="gray_release" />
                        </div>
                        <p className="text-xs text-slate-400 mb-1">
                          Released: {new Date(item.gray_release_at).toLocaleString()}
                        </p>
                        <p className="text-xs text-slate-400 mb-4">
                          {canPromote
                            ? "Ready to promote"
                            : `${hoursRemaining.toFixed(1)}h remaining`}
                        </p>

                        {/* Progress bar */}
                        <div className="w-full bg-slate-700 rounded-full h-2 mb-4">
                          <div
                            className={`h-2 rounded-full transition-all ${
                              canPromote ? "bg-green-500" : "bg-purple-500"
                            }`}
                            style={{ width: `${progress}%` }}
                          />
                        </div>

                        <button
                          onClick={() => promoteContent(item.id)}
                          disabled={!canPromote}
                          className={`w-full px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
                            canPromote
                              ? "bg-emerald-600 hover:bg-emerald-700 text-white"
                              : "bg-slate-700 text-slate-500 cursor-not-allowed"
                          }`}
                        >
                          Promote to Published
                        </button>
                      </div>
                    );
                  })
                )}
              </div>
            )}

            {/* All Content Tab */}
            {activeTab === "all" && (
              <>
                {/* Filters */}
                <div className="flex gap-3 mb-4">
                  <select
                    value={statusFilter}
                    onChange={(e) => setStatusFilter(e.target.value)}
                    className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300 focus:outline-none focus:border-blue-500"
                  >
                    <option value="">All Statuses</option>
                    {statuses.map((s) => (
                      <option key={s} value={s}>{s.replace(/_/g, " ")}</option>
                    ))}
                  </select>
                  <select
                    value={typeFilter}
                    onChange={(e) => setTypeFilter(e.target.value)}
                    className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300 focus:outline-none focus:border-blue-500"
                  >
                    <option value="">All Types</option>
                    {contentTypes.map((t) => (
                      <option key={t} value={t}>{t}</option>
                    ))}
                  </select>
                </div>

                <div className="bg-slate-800 rounded-xl border border-slate-700 overflow-hidden">
                  <table className="w-full">
                    <thead>
                      <tr className="border-b border-slate-700">
                        <SortableHeader label="Title" sortKey="title" currentSort={contentSortConfig} onSort={requestContentSort} />
                        <SortableHeader label="Type" sortKey="content_type" currentSort={contentSortConfig} onSort={requestContentSort} />
                        <SortableHeader label="Status" sortKey="status" currentSort={contentSortConfig} onSort={requestContentSort} />
                        <SortableHeader label="Submitted By" sortKey="submitted_by" currentSort={contentSortConfig} onSort={requestContentSort} />
                        <SortableHeader label="Date" sortKey="submitted_at" currentSort={contentSortConfig} onSort={requestContentSort} />
                        <th className="text-right px-4 py-3 text-xs font-medium text-slate-400 uppercase tracking-wider">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-700">
                      {sortedFilteredContent.length === 0 ? (
                        <tr>
                          <td colSpan={6} className="px-4 py-12 text-center text-slate-500">
                            No content found
                          </td>
                        </tr>
                      ) : (
                        sortedFilteredContent.map((item) => (
                          <tr
                            key={item.id}
                            className="bg-slate-800 hover:bg-slate-750 transition-colors cursor-pointer"
                            onClick={() => setDetailModal(item)}
                          >
                            <td className="px-4 py-3 text-sm text-slate-200 font-medium">{item.title}</td>
                            <td className="px-4 py-3 text-sm text-slate-300">{item.content_type}</td>
                            <td className="px-4 py-3 text-sm"><StatusBadge status={item.status} /></td>
                            <td className="px-4 py-3 text-sm text-slate-300">{item.submitted_by}</td>
                            <td className="px-4 py-3 text-sm text-slate-400">
                              {item.submitted_at ? new Date(item.submitted_at).toLocaleDateString() : "-"}
                            </td>
                            <td className="px-4 py-3 text-right">
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  setDetailModal(item);
                                }}
                                className="px-3 py-1.5 text-xs font-medium rounded-md bg-slate-700 hover:bg-slate-600 text-slate-300 transition-colors"
                              >
                                View
                              </button>
                            </td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>
              </>
            )}
          </>
        )}

        {/* Review Decision Modal */}
        {reviewModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
            <div className="bg-slate-800 rounded-xl border border-slate-700 p-6 w-full max-w-lg mx-4">
              <h2 className="text-lg font-semibold text-slate-100 mb-1">
                {reviewModal.decision === "approved"
                  ? "Approve Content"
                  : reviewModal.decision === "rejected"
                  ? "Reject Content"
                  : "Escalate Content"}
              </h2>
              <p className="text-sm text-slate-400 mb-4">
                {reviewModal.contentTitle}
              </p>

              <label className="block text-sm font-medium text-slate-300 mb-2">
                Review Notes
              </label>
              <textarea
                value={reviewNotes}
                onChange={(e) => setReviewNotes(e.target.value)}
                rows={4}
                placeholder="Add notes for this decision..."
                className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:border-blue-500 resize-none"
              />

              <div className="flex justify-end gap-3 mt-4">
                <button
                  onClick={() => setReviewModal(null)}
                  className="px-4 py-2 text-sm font-medium rounded-lg bg-slate-700 hover:bg-slate-600 text-slate-300 transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={submitReviewDecision}
                  disabled={submitting}
                  className={`px-4 py-2 text-sm font-medium rounded-lg text-white transition-colors ${
                    reviewModal.decision === "approved"
                      ? "bg-green-600 hover:bg-green-700"
                      : reviewModal.decision === "rejected"
                      ? "bg-red-600 hover:bg-red-700"
                      : "bg-yellow-600 hover:bg-yellow-700"
                  } ${submitting ? "opacity-50 cursor-not-allowed" : ""}`}
                >
                  {submitting ? "Submitting..." : "Confirm"}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Content Detail Modal */}
        {detailModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
            <div className="bg-slate-800 rounded-xl border border-slate-700 p-6 w-full max-w-2xl mx-4 max-h-[80vh] overflow-auto">
              <div className="flex items-start justify-between mb-4">
                <div>
                  <h2 className="text-lg font-semibold text-slate-100">{detailModal.title}</h2>
                  <p className="text-sm text-slate-400 mt-1">
                    {detailModal.content_type} &middot; <StatusBadge status={detailModal.status} />
                  </p>
                </div>
                <button
                  onClick={() => setDetailModal(null)}
                  className="text-slate-400 hover:text-slate-200 text-xl leading-none"
                >
                  x
                </button>
              </div>

              <div className="grid grid-cols-2 gap-4 mb-6">
                <div>
                  <p className="text-xs text-slate-500">Submitted By</p>
                  <p className="text-sm text-slate-200">{detailModal.submitted_by || "-"}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500">Submitted Date</p>
                  <p className="text-sm text-slate-200">
                    {detailModal.submitted_at ? new Date(detailModal.submitted_at).toLocaleString() : "-"}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-slate-500">Review Level</p>
                  <p className="text-sm text-slate-200 capitalize">{detailModal.review_level || "-"}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500">Severity</p>
                  <p className="text-sm text-slate-200 capitalize">{detailModal.severity || "-"}</p>
                </div>
              </div>

              {/* Version History */}
              <h3 className="text-sm font-semibold text-slate-200 mb-3">Version History</h3>
              {detailModal.versions && detailModal.versions.length > 0 ? (
                <div className="space-y-2 mb-4">
                  {detailModal.versions.map((v) => (
                    <div
                      key={v.version}
                      className="flex items-center justify-between p-3 bg-slate-900 rounded-lg border border-slate-700"
                    >
                      <div>
                        <p className="text-sm text-slate-200">Version {v.version}</p>
                        <p className="text-xs text-slate-400">
                          {v.updated_by} &middot; {new Date(v.updated_at).toLocaleString()}
                        </p>
                        <p className="text-xs text-slate-500 mt-1">{v.changes_summary}</p>
                      </div>
                      <button
                        onClick={() => rollbackContent(detailModal.id, v.version)}
                        className="px-3 py-1.5 text-xs font-medium rounded-md bg-orange-600 hover:bg-orange-700 text-white transition-colors"
                      >
                        Rollback
                      </button>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-slate-500 mb-4">No version history available</p>
              )}

              <div className="flex justify-end">
                <button
                  onClick={() => setDetailModal(null)}
                  className="px-4 py-2 text-sm font-medium rounded-lg bg-slate-700 hover:bg-slate-600 text-slate-300 transition-colors"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
