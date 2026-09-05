import { ROOT_CAUSE_COLORS, ROOT_CAUSE_ORDER, rootCauseLabel } from '../lib/format'

interface RootCauseChartProps {
  counts: Record<string, number>
}

const CHART_HEIGHT = 160
const BAR_WIDTH = 56
const BAR_GAP = 20
const LABEL_HEIGHT = 52

/** Dependency-free inline SVG bar chart — same rationale as TierBreakdownChart: no charting library for 6 bars. */
export function RootCauseChart({ counts }: RootCauseChartProps) {
  const causes = ROOT_CAUSE_ORDER.map((cause) => ({ cause, count: counts[cause] ?? 0 }))
  const maxCount = Math.max(1, ...causes.map((c) => c.count))
  const width = causes.length * (BAR_WIDTH + BAR_GAP)

  return (
    <svg
      viewBox={`0 0 ${width} ${CHART_HEIGHT + LABEL_HEIGHT}`}
      className="h-60 w-full max-w-2xl"
      role="img"
      aria-label="Root cause distribution chart"
    >
      {causes.map((c, i) => {
        const barHeight = (c.count / maxCount) * (CHART_HEIGHT - 24)
        const x = i * (BAR_WIDTH + BAR_GAP) + BAR_GAP / 2
        const y = CHART_HEIGHT - barHeight
        const label = rootCauseLabel(c.cause)
        const words = label.split(' ')
        return (
          <g key={c.cause}>
            <text x={x + BAR_WIDTH / 2} y={y - 8} textAnchor="middle" className="fill-ink text-sm font-semibold">
              {c.count}
            </text>
            <rect x={x} y={y} width={BAR_WIDTH} height={Math.max(barHeight, 2)} rx={4} fill={ROOT_CAUSE_COLORS[c.cause]} />
            {words.map((word, wi) => (
              <text
                key={wi}
                x={x + BAR_WIDTH / 2}
                y={CHART_HEIGHT + 16 + wi * 14}
                textAnchor="middle"
                className="fill-ink-muted text-[11px]"
              >
                {word}
              </text>
            ))}
          </g>
        )
      })}
    </svg>
  )
}
