import { tierLabel } from '../lib/format'

interface TierBreakdownChartProps {
  tierBreakdown: Record<string, number>
}

const TIER_ORDER = ['silent_retry', 'nudge', 'incentivized_nudge', 'escalate']
const TIER_COLORS: Record<string, string> = {
  silent_retry: '#0ea5e9', // sky
  nudge: '#8b5cf6', // violet
  incentivized_nudge: '#f59e0b', // amber
  escalate: '#ef4444', // red
}

const CHART_HEIGHT = 160
const BAR_WIDTH = 64
const BAR_GAP = 32
const LABEL_HEIGHT = 40

/**
 * Dependency-free inline SVG bar chart — deliberately not a charting
 * library, for 4 bars that's an unnecessary dependency and a possible
 * failure point during a live demo.
 */
export function TierBreakdownChart({ tierBreakdown }: TierBreakdownChartProps) {
  const tiers = TIER_ORDER.map((tier) => ({ tier, count: tierBreakdown[tier] ?? 0 }))
  const maxCount = Math.max(1, ...tiers.map((t) => t.count))
  const width = tiers.length * (BAR_WIDTH + BAR_GAP)

  return (
    <svg
      viewBox={`0 0 ${width} ${CHART_HEIGHT + LABEL_HEIGHT}`}
      className="h-56 w-full max-w-xl"
      role="img"
      aria-label="Tier breakdown chart"
    >
      {tiers.map((t, i) => {
        const barHeight = (t.count / maxCount) * (CHART_HEIGHT - 24)
        const x = i * (BAR_WIDTH + BAR_GAP) + BAR_GAP / 2
        const y = CHART_HEIGHT - barHeight
        return (
          <g key={t.tier}>
            <text
              x={x + BAR_WIDTH / 2}
              y={y - 8}
              textAnchor="middle"
              className="fill-slate-700 text-sm font-semibold"
            >
              {t.count}
            </text>
            <rect
              x={x}
              y={y}
              width={BAR_WIDTH}
              height={Math.max(barHeight, 2)}
              rx={4}
              fill={TIER_COLORS[t.tier]}
            />
            <text
              x={x + BAR_WIDTH / 2}
              y={CHART_HEIGHT + 20}
              textAnchor="middle"
              className="fill-slate-600 text-xs"
            >
              {tierLabel(t.tier)}
            </text>
          </g>
        )
      })}
    </svg>
  )
}
