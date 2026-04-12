interface DashboardCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  trend?: "up" | "down" | "neutral";
  trendValue?: string;
}

export default function DashboardCard({
  title,
  value,
  subtitle,
  trend,
  trendValue,
}: DashboardCardProps) {
  const trendColor =
    trend === "up"
      ? "text-green-400"
      : trend === "down"
        ? "text-red-400"
        : "text-slate-400";

  const trendIcon =
    trend === "up" ? "\u2191" : trend === "down" ? "\u2193" : "\u2192";

  return (
    <div className="bg-slate-800 rounded-xl border border-slate-700 p-6 hover:border-slate-600 transition-colors">
      <p className="text-sm text-slate-400 font-medium mb-1">{title}</p>
      <p className="text-3xl font-bold text-slate-100 mb-1">{value}</p>
      <div className="flex items-center gap-2">
        {subtitle && <p className="text-sm text-slate-500">{subtitle}</p>}
        {trend && trendValue && (
          <span className={`text-sm font-medium ${trendColor}`}>
            {trendIcon} {trendValue}
          </span>
        )}
      </div>
    </div>
  );
}
