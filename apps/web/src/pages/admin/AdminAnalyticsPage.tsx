import { useState, type ReactNode } from "react";
import Alert from "@mui/material/Alert";
import Box from "@mui/material/Box";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import ToggleButton from "@mui/material/ToggleButton";
import ToggleButtonGroup from "@mui/material/ToggleButtonGroup";
import Typography from "@mui/material/Typography";
import InsightsOutlined from "@mui/icons-material/InsightsOutlined";
import BarChartOutlined from "@mui/icons-material/BarChartOutlined";
import TableRowsOutlined from "@mui/icons-material/TableRowsOutlined";
import { PageBanner } from "@/components/PageBanner";
import {
  amber,
  brass,
  clay,
  displayFamily,
  monoFamily,
  moss,
  stone,
} from "@/theme";
import { formatPesewas } from "@/features/catalog/money";
import {
  errorMessage,
  formatBps,
  statusLabel,
  typeLabel,
  type CollectionStat,
  type DesignStat,
  type RecentOrder,
  type StoreAnalytics,
  type TimeBucket,
} from "@/features/analytics/api";
import { useStoreAnalytics } from "@/features/analytics/hooks";

const cardSx = {
  p: { xs: 2, md: 3 },
  border: "1px solid",
  borderColor: "divider",
  borderRadius: 0,
  position: "relative",
  overflow: "hidden",
} as const;

/** Faint tailor-tick brand motif tucked in a card corner. */
function CardDecoration() {
  return (
    <Box
      aria-hidden
      sx={{
        position: "absolute",
        right: -6,
        bottom: -10,
        display: "flex",
        alignItems: "flex-end",
        gap: 0.75,
        opacity: 0.07,
        pointerEvents: "none",
      }}
    >
      <Box sx={{ width: 11, height: 56, bgcolor: amber }} />
      <Box sx={{ width: 11, height: 38, bgcolor: amber }} />
      <Box sx={{ width: 11, height: 48, bgcolor: amber }} />
    </Box>
  );
}

function MonoLabel({ children }: { children: ReactNode }) {
  return (
    <Typography
      variant="overline"
      component="p"
      sx={{ color: stone, fontFamily: monoFamily }}
    >
      {children}
    </Typography>
  );
}

type PanelView = "chart" | "table";

/** Chart / table switch shown in a panel header. */
function ViewToggle({
  value,
  onChange,
  label,
}: {
  value: PanelView;
  onChange: (next: PanelView) => void;
  label: string;
}) {
  return (
    <ToggleButtonGroup
      size="small"
      exclusive
      value={value}
      onChange={(_event, next: PanelView | null) => next && onChange(next)}
      aria-label={`${label} view`}
      sx={{
        "& .MuiToggleButton-root": { px: 1, py: 0.25, color: stone, border: "1px solid", borderColor: "divider" },
        "& .Mui-selected": { color: "text.primary" },
      }}
    >
      <ToggleButton value="chart" aria-label="Chart view">
        <BarChartOutlined sx={{ fontSize: 18 }} />
      </ToggleButton>
      <ToggleButton value="table" aria-label="Table view">
        <TableRowsOutlined sx={{ fontSize: 18 }} />
      </ToggleButton>
    </ToggleButtonGroup>
  );
}

/** Panel header: mono caption on the left, a chart/table toggle on the right. */
function PanelHeader({
  caption,
  view,
  onView,
}: {
  caption: string;
  view: PanelView;
  onView: (next: PanelView) => void;
}) {
  return (
    <Stack direction="row" sx={{ justifyContent: "space-between", alignItems: "center", gap: 1, mb: 0.5 }}>
      <MonoLabel>{caption}</MonoLabel>
      <ViewToggle value={view} onChange={onView} label={caption} />
    </Stack>
  );
}

interface RankedItem {
  id: string;
  label: string;
  /** Numeric magnitude that drives the bar length. */
  value: number;
  /** Primary value text shown at the end of the row. */
  valueText: string;
  /** Optional muted secondary text. */
  sub?: string;
}

/** Horizontal bar chart for a small ranked list — labels left, bars below. */
function RankedBars({ items }: { items: RankedItem[] }) {
  const peak = items.reduce((max, item) => Math.max(max, item.value), 0);

  return (
    <Stack spacing={1.75} sx={{ mt: 2 }}>
      {items.map((item) => {
        const ratio = peak > 0 ? item.value / peak : 0;
        const width = item.value > 0 ? `${Math.max(ratio * 100, 3)}%` : "0%";

        return (
          <Box key={item.id}>
            <Stack
              direction="row"
              sx={{ justifyContent: "space-between", alignItems: "baseline", gap: 2, mb: 0.75 }}
            >
              <Typography
                sx={{ fontWeight: 500, fontSize: "0.85rem", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}
              >
                {item.label}
              </Typography>
              <Stack direction="row" spacing={1} sx={{ flexShrink: 0, alignItems: "baseline" }}>
                <Typography sx={{ fontFamily: monoFamily, fontSize: "0.8rem", fontWeight: 600 }}>
                  {item.valueText}
                </Typography>
                {item.sub && (
                  <Typography sx={{ color: stone, fontFamily: monoFamily, fontSize: "0.7rem" }}>
                    {item.sub}
                  </Typography>
                )}
              </Stack>
            </Stack>
            <Box sx={{ height: 8, bgcolor: "divider", overflow: "hidden" }}>
              <Box
                aria-hidden="true"
                sx={{ width, height: "100%", bgcolor: amber, transition: "width 240ms ease", "&:hover": { bgcolor: brass } }}
              />
            </Box>
          </Box>
        );
      })}
    </Stack>
  );
}

interface DeltaProps {
  bps: number;
}

function Delta({ bps }: DeltaProps) {
  const positive = bps >= 0;

  return (
    <Typography
      component="span"
      sx={{
        fontFamily: monoFamily,
        fontSize: "0.75rem",
        letterSpacing: "0.04em",
        color: positive ? amber : clay,
      }}
      data-testid="stat-delta"
    >
      {formatBps(bps)}
    </Typography>
  );
}

interface StatCardProps {
  label: string;
  value: string;
  deltaBps?: number;
  caption?: string;
  testId?: string;
}

function StatCard({ label, value, deltaBps, caption, testId }: StatCardProps) {
  return (
    <Paper
      elevation={0}
      sx={{ ...cardSx, height: "100%", display: "flex", flexDirection: "column" }}
      data-testid={testId}
    >
      <CardDecoration />
      <Box sx={{ position: "relative" }}>
        <MonoLabel>{label}</MonoLabel>
        <Typography
          component="p"
          sx={{
            fontFamily: displayFamily,
            fontWeight: 700,
            letterSpacing: "-0.02em",
            lineHeight: 1,
            fontSize: { xs: "2rem", md: "2.4rem" },
            mt: 1.25,
          }}
        >
          {value}
        </Typography>
      </Box>
      <Stack
        direction="row"
        spacing={1.5}
        sx={{ mt: "auto", pt: 1.5, alignItems: "baseline", position: "relative" }}
      >
        {deltaBps !== undefined && <Delta bps={deltaBps} />}
        {caption && (
          <Typography
            component="span"
            sx={{ fontFamily: monoFamily, fontSize: "0.7rem", color: "text.secondary", letterSpacing: "0.04em" }}
          >
            {caption}
          </Typography>
        )}
      </Stack>
    </Paper>
  );
}

interface RevenueChartProps {
  series: TimeBucket[];
}

function maxRevenue(series: TimeBucket[]): number {
  return series.reduce((peak, bucket) => Math.max(peak, bucket.revenuePesewas), 0);
}

const CHART_W = 1000;
const CHART_H = 180;
const CHART_PAD_TOP = 12;

function RevenueChart({ series }: RevenueChartProps) {
  const peak = maxRevenue(series);
  const count = series.length;

  const points = series.map((bucket, index) => {
    const x = count > 1 ? (index / (count - 1)) * CHART_W : 0;
    const ratio = peak > 0 ? bucket.revenuePesewas / peak : 0;
    const y = CHART_H - ratio * (CHART_H - CHART_PAD_TOP);
    return { x, y };
  });

  const linePath = points
    .map((p, i) => `${i === 0 ? "M" : "L"} ${p.x.toFixed(1)} ${p.y.toFixed(1)}`)
    .join(" ");
  const areaPath =
    points.length > 0
      ? `M 0 ${CHART_H} ${points.map((p) => `L ${p.x.toFixed(1)} ${p.y.toFixed(1)}`).join(" ")} L ${CHART_W} ${CHART_H} Z`
      : "";

  return (
    <Paper elevation={0} sx={cardSx} data-testid="revenue-chart">
      <MonoLabel>booked revenue · last 12 weeks</MonoLabel>
      <Box
        component="figure"
        aria-label="Booked revenue per week"
        sx={{ m: 0, mt: 2.5 }}
      >
        <Box sx={{ position: "relative", height: 180, borderBottom: "1px solid", borderColor: "divider" }}>
          <Box
            component="svg"
            viewBox={`0 0 ${CHART_W} ${CHART_H}`}
            preserveAspectRatio="none"
            sx={{ width: "100%", height: "100%", display: "block" }}
          >
            <defs>
              <linearGradient id="revenueFill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={amber} stopOpacity={0.32} />
                <stop offset="100%" stopColor={amber} stopOpacity={0} />
              </linearGradient>
            </defs>
            {peak > 0 && <path d={areaPath} fill="url(#revenueFill)" />}
            <path
              d={linePath}
              fill="none"
              stroke={amber}
              strokeWidth={2}
              vectorEffect="non-scaling-stroke"
              strokeLinejoin="round"
              strokeLinecap="round"
            />
          </Box>
          {/* One hover target per week — keeps the data points discoverable. */}
          <Box sx={{ position: "absolute", inset: 0, display: "flex" }}>
            {series.map((bucket, index) => (
              <Box
                key={`${bucket.startAt}-${index}`}
                data-testid="revenue-bar"
                title={`${bucket.label}: ${formatPesewas(bucket.revenuePesewas)} · ${bucket.orderCount} orders`}
                sx={{ flex: 1, minWidth: 6, "&:hover": { bgcolor: "action.hover" } }}
              />
            ))}
          </Box>
        </Box>
        <Box
          sx={{
            display: "flex",
            gap: { xs: 0.5, md: 1 },
            mt: 1,
            px: 0.5,
          }}
        >
          {series.map((bucket, index) => (
            <Typography
              key={`label-${bucket.startAt}-${index}`}
              component="span"
              sx={{
                flex: 1,
                minWidth: 6,
                textAlign: "center",
                fontFamily: monoFamily,
                fontSize: "0.55rem",
                color: stone,
                letterSpacing: "0.02em",
                whiteSpace: "nowrap",
                overflow: "hidden",
                textOverflow: "ellipsis",
                // Thin the axis: show every other tick on small charts.
                visibility: index % 2 === 0 ? "visible" : "hidden",
              }}
            >
              {bucket.label}
            </Typography>
          ))}
        </Box>
      </Box>
    </Paper>
  );
}

interface DesignTableProps {
  rows: DesignStat[];
}

function TopDesignsTable({ rows }: DesignTableProps) {
  const [view, setView] = useState<PanelView>("chart");
  // Bar length tracks revenue (the metric that actually varies), so rank by it
  // too — the table and chart then read top-to-bottom in the same order.
  const ranked = [...rows].sort((a, b) => b.revenuePesewas - a.revenuePesewas);

  return (
    <Paper elevation={0} sx={{ ...cardSx, height: "100%" }} data-testid="top-designs">
      <PanelHeader caption="top designs · by revenue" view={view} onView={setView} />
      {view === "chart" ? (
        <RankedBars
          items={ranked.map((row) => ({
            id: row.designId,
            label: row.name,
            value: row.revenuePesewas,
            valueText: formatPesewas(row.revenuePesewas),
            sub: `${row.orderCount.toLocaleString("en-GH")} orders`,
          }))}
        />
      ) : (
        <Table size="small" sx={{ mt: 1.5 }}>
          <TableHead>
            <TableRow>
              <TableCell>Design</TableCell>
              <TableCell align="right">Orders</TableCell>
              <TableCell align="right">Revenue</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {ranked.map((row) => (
              <TableRow key={row.designId}>
                <TableCell sx={{ fontWeight: 500 }}>{row.name}</TableCell>
                <TableCell align="right">{row.orderCount.toLocaleString("en-GH")}</TableCell>
                <TableCell align="right">{formatPesewas(row.revenuePesewas)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </Paper>
  );
}

interface CollectionTableProps {
  rows: CollectionStat[];
}

function TopCollectionsTable({ rows }: CollectionTableProps) {
  const [view, setView] = useState<PanelView>("chart");

  return (
    <Paper elevation={0} sx={{ ...cardSx, height: "100%" }} data-testid="top-collections">
      <PanelHeader caption="top collections · by revenue" view={view} onView={setView} />
      {view === "chart" ? (
        <RankedBars
          items={rows.map((row) => ({
            id: row.collectionId,
            label: row.name,
            value: row.revenuePesewas,
            valueText: formatPesewas(row.revenuePesewas),
            sub: `${row.orderCount.toLocaleString("en-GH")} orders`,
          }))}
        />
      ) : (
        <Table size="small" sx={{ mt: 1.5 }}>
          <TableHead>
            <TableRow>
              <TableCell>Collection</TableCell>
              <TableCell align="right">Orders</TableCell>
              <TableCell align="right">Revenue</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.map((row) => (
              <TableRow key={row.collectionId}>
                <TableCell sx={{ fontWeight: 500 }}>{row.name}</TableCell>
                <TableCell align="right">{row.orderCount.toLocaleString("en-GH")}</TableCell>
                <TableCell align="right">{formatPesewas(row.revenuePesewas)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </Paper>
  );
}

function formatRelative(iso: string): string {
  const date = new Date(iso);

  if (Number.isNaN(date.getTime())) return "—";

  return date.toLocaleDateString("en-GH", { day: "numeric", month: "short" });
}

interface RecentOrdersProps {
  rows: RecentOrder[];
}

function RecentOrdersList({ rows }: RecentOrdersProps) {
  // A recent-activity feed is chronological event data, not a magnitude
  // comparison — the correct visualization is a list, so this panel stays a list.
  return (
    <Paper elevation={0} sx={cardSx} data-testid="recent-orders">
      <MonoLabel>recent activity</MonoLabel>
      <Stack divider={<Box sx={{ borderBottom: "1px solid", borderColor: "divider" }} />} sx={{ mt: 1.5 }}>
        {rows.map((row) => (
          <Stack
            key={row.ref}
            direction="row"
            sx={{ py: 1.25, alignItems: "baseline", justifyContent: "space-between", gap: 2 }}
          >
            <Box sx={{ minWidth: 0 }}>
              <Typography component="span" sx={{ fontWeight: 600, fontFamily: monoFamily, fontSize: "0.8rem" }}>
                {row.ref}
              </Typography>
              <Typography component="span" sx={{ ml: 1.5, color: stone, fontSize: "0.8rem" }}>
                {typeLabel(row.type)} · {statusLabel(row.status)}
              </Typography>
            </Box>
            <Stack direction="row" spacing={1.5} sx={{ alignItems: "baseline", flexShrink: 0 }}>
              <Typography component="span" sx={{ fontWeight: 500, fontSize: "0.85rem" }}>
                {formatPesewas(row.totalPesewas)}
              </Typography>
              <Typography
                component="span"
                sx={{ color: stone, fontFamily: monoFamily, fontSize: "0.7rem" }}
              >
                {formatRelative(row.createdAt)}
              </Typography>
            </Stack>
          </Stack>
        ))}
      </Stack>
    </Paper>
  );
}

function PageHeading() {
  return (
    <Box sx={{ mb: 4 }}>
      <PageBanner
        tone="ink"
        icon={<InsightsOutlined />}
        breadcrumbs={[{ label: "Admin", to: "/admin" }, { label: "Analytics" }]}
        title="Analytics"
        description="How the store is performing — booked revenue, orders, and the designs and collections doing the most work."
      />
    </Box>
  );
}

function LoadingSkeletons() {
  return (
    <Stack spacing={{ xs: 3, md: 4 }} data-testid="analytics-loading">
      <Grid container spacing={{ xs: 2, md: 3 }}>
        {[0, 1, 2, 3, 4].map((key) => (
          <Grid key={key} size={{ xs: 12, sm: 6, md: 2.4 }}>
            <Skeleton variant="rectangular" height={132} />
          </Grid>
        ))}
      </Grid>
      <Skeleton variant="rectangular" height={260} />
      <Grid container spacing={{ xs: 3, md: 4 }}>
        <Grid size={{ xs: 12, md: 6 }}>
          <Skeleton variant="rectangular" height={240} />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <Skeleton variant="rectangular" height={240} />
        </Grid>
      </Grid>
    </Stack>
  );
}

function EmptyState() {
  return (
    <Paper elevation={0} sx={{ ...cardSx, p: { xs: 4, md: 6 }, textAlign: "center" }} data-testid="analytics-empty">
      <MonoLabel>no orders yet</MonoLabel>
      <Typography sx={{ mt: 1.5, color: stone, maxWidth: 420, mx: "auto" }}>
        No orders yet — analytics will populate as the store sells.
      </Typography>
    </Paper>
  );
}

function Headline({ analytics }: { analytics: StoreAnalytics }) {
  return (
    <Grid container spacing={{ xs: 2, md: 3 }}>
      <Grid size={{ xs: 12, sm: 6, md: 2.4 }}>
        <StatCard
          label="booked revenue"
          value={formatPesewas(analytics.bookedRevenuePesewas)}
          deltaBps={analytics.comparison.revenueChangeBps}
          caption="30d vs prior"
          testId="revenue-card"
        />
      </Grid>
      <Grid size={{ xs: 12, sm: 6, md: 2.4 }}>
        <StatCard
          label="orders"
          value={analytics.orderCount.toLocaleString("en-GH")}
          deltaBps={analytics.comparison.orderCountChangeBps}
          caption="30d vs prior"
          testId="orders-card"
        />
      </Grid>
      <Grid size={{ xs: 12, sm: 6, md: 2.4 }}>
        <StatCard
          label="avg order value"
          value={formatPesewas(analytics.averageOrderValuePesewas)}
          testId="aov-card"
        />
      </Grid>
      <Grid size={{ xs: 12, sm: 6, md: 2.4 }}>
        <StatCard
          label="waitlist"
          value={analytics.waitlistCount.toLocaleString("en-GH")}
          testId="waitlist-card"
        />
      </Grid>
      <Grid size={{ xs: 12, sm: 6, md: 2.4 }}>
        <StatCard
          label="customers"
          value={analytics.customerCount.toLocaleString("en-GH")}
          testId="customers-card"
        />
      </Grid>
    </Grid>
  );
}

function Dashboard({ analytics }: { analytics: StoreAnalytics }) {
  return (
    <Stack spacing={{ xs: 3, md: 4 }} data-testid="analytics-dashboard">
      <Headline analytics={analytics} />

      <RevenueChart series={analytics.revenueSeries} />

      <Grid container spacing={{ xs: 3, md: 4 }}>
        <Grid size={{ xs: 12, md: 6 }}>
          <TopDesignsTable rows={analytics.topDesigns} />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <TopCollectionsTable rows={analytics.topCollections} />
        </Grid>
      </Grid>

      <RecentOrdersList rows={analytics.recentOrders} />

      <Box sx={{ color: stone }}>
        <Typography sx={{ fontFamily: monoFamily, fontSize: "0.7rem", letterSpacing: "0.04em", color: moss }}>
          Booked revenue counts garment + delivery for booked, in production, ready, and fulfilled orders.
        </Typography>
      </Box>
    </Stack>
  );
}

export function AdminAnalyticsPage() {
  const query = useStoreAnalytics();
  const analytics = query.data;
  const hasOrders = analytics ? analytics.orderCount > 0 || analytics.recentOrders.length > 0 : false;

  return (
    <Box>
      <PageHeading />

      {query.isPending && <LoadingSkeletons />}

      {query.isError && (
        <Alert severity="error" sx={{ borderColor: clay }}>
          {errorMessage(query.error)}
        </Alert>
      )}

      {query.isSuccess && analytics && !hasOrders && <EmptyState />}

      {query.isSuccess && analytics && hasOrders && <Dashboard analytics={analytics} />}
    </Box>
  );
}
