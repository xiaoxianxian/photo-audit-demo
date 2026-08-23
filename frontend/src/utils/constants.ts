/**
 * Design system constants — single source of truth for colors, spacing, typography, and risk helpers.
 * Aligned with .design_library/photo-audit/ design tokens.
 * All pages should import from here instead of using magic values.
 */

// ── Dark theme palette (GitHub Dark Dimmed inspired) ──────────────────
export const COLORS = {
  // Background layers
  bgBase: '#0d1117',
  bgContainer: '#161b22',
  bgContainerHover: '#1c2128',
  bgOffline: '#0d1117',
  bgCard: '#161b22',

  // Borders
  borderDefault: '#30363d',
  borderHover: '#484f58',

  // Text
  textPrimary: '#e6edf3',
  textSecondary: '#8b949e',
  textTertiary: '#c9d1d9',
  textMuted: '#8b949e',

  // Brand (accent)
  brandBlue: '#58a6ff',
  brandGreen: '#3fb950',

  // Semantic
  danger: '#f85149',
  warning: '#d29922',
  success: '#3fb950',
  error: '#f85149',
  info: '#58a6ff',
  conflictOrange: '#fa8c16',
} as const

// ── Light theme palette ──────────────────────────────────────────────
export const LIGHT_COLORS = {
  bgBase: '#f8fafc',
  bgContainer: '#ffffff',
  bgContainerHover: '#f1f5f9',
  bgOffline: '#f1f5f9',
  bgCard: '#ffffff',
  borderDefault: '#e2e8f0',
  borderHover: '#cbd5e1',
  textPrimary: '#0f172a',
  textSecondary: '#64748b',
  textTertiary: '#475569',
  textMuted: '#94a3b8',
  brandBlue: '#3b82f6',
  brandGreen: '#10b981',
  danger: '#ef4444',
  warning: '#f59e0b',
  success: '#10b981',
  error: '#ef4444',
  info: '#0ea5e9',
  conflictOrange: '#f59e0b',
} as const

// ── Spacing scale (matches design library --space-*) ─────────────────
export const SPACING = {
  xs: 4,   // --space-1
  sm: 8,   // --space-2
  md: 12,  // --space-3
  base: 16, // --space-4
  lg: 24,  // --space-5
  xl: 32,  // --space-6
  xxl: 48, // --space-7
} as const

// ── Typography scale (matches design library --font-size-*) ──────────
export const FONT = {
  display: 56,  // --font-size-display
  h1: 40,   // --font-size-h1
  h2: 32,   // --font-size-h2
  h3: 24,   // --font-size-h3
  h4: 20,   // --font-size-h4
  body: 16, // --font-size-body
  small: 14,// --font-size-mono
  caption: 12,
  eyebrow: 11,
} as const

// ── Animation / Motion (matches design library --transition-*) ───────
export const ANIMATION = {
  fast: '150ms cubic-bezier(0.25, 0.1, 0.25, 1)',
  normal: '250ms cubic-bezier(0.25, 0.1, 0.25, 1)',
  slow: '400ms cubic-bezier(0.25, 0.1, 0.25, 1)',
} as const

// ── Border radius (matches design library --radius-*) ────────────────
export const RADIUS = {
  xs: 2,
  sm: 4,
  md: 8,
  lg: 12,
  xl: 16,
  full: 9999,
} as const

// ── Shadows (matches design library --shadow-*) ──────────────────────
export const SHADOW = {
  card: '0 1px 2px rgba(0,0,0,.2), 0 1px 1px rgba(0,0,0,.15)',
  cardHover: '0 4px 8px -2px rgba(0,0,0,.3)',
  float: '0 8px 24px -8px rgba(0,0,0,.4)',
  modal: '0 16px 40px -12px rgba(0,0,0,.5)',
} as const

// ── Risk score helpers (shared across all pages) ─────────────────────
export const RISK_THRESHOLDS = [20, 60, 80] as const

export function riskColor(score: number): string {
  if (score <= RISK_THRESHOLDS[0]) return COLORS.success
  if (score <= RISK_THRESHOLDS[1]) return COLORS.warning
  if (score <= RISK_THRESHOLDS[2]) return COLORS.conflictOrange
  return COLORS.danger
}

export function riskLabel(score: number): string {
  if (score <= RISK_THRESHOLDS[0]) return '低风险'
  if (score <= RISK_THRESHOLDS[1]) return '中等风险'
  if (score <= RISK_THRESHOLDS[2]) return '高风险'
  return '极高风险'
}

export function riskTagColor(score: number): string {
  if (score <= RISK_THRESHOLDS[0]) return 'success'
  if (score <= RISK_THRESHOLDS[1]) return 'warning'
  if (score <= RISK_THRESHOLDS[2]) return 'orange'
  return 'error'
}

// ── Element kind labels ──────────────────────────────────────────────
export const ELEMENT_KIND_LABELS: Record<string, string> = {
  cover_image: '封面',
  video_frame: '视频帧',
  title: '标题',
  comment: '评论',
  asr_text: 'ASR 转写',
  live_snapshot: '直播截帧',
  description: '描述',
}

// ── AI status labels ────────────────────────────────────────────────
export const AI_STATUS_COLORS: Record<string, string> = {
  pending_ai: 'default',
  ai_processing: 'processing',
  ai_passed: 'success',
  ai_rejected: 'error',
  ai_failed: 'warning',
  pending_human: 'warning',
  in_human_review: 'processing',
  human_passed: 'success',
  human_rejected: 'error',
}

export const AI_STATUS_LABELS: Record<string, string> = {
  pending_ai: '待 AI 审核',
  ai_processing: 'AI 审核中',
  ai_passed: '机审通过',
  ai_rejected: '机审驳回',
  ai_failed: '审核失败',
  pending_human: '待人审',
  in_human_review: '人审中',
  human_passed: '人工通过',
  human_rejected: '人工驳回',
}

// ── Action labels ───────────────────────────────────────────────────
export const ACTION_LABELS: Record<string, string> = {
  approve: '通过',
  reject: '驳回',
  flag: '标记',
}

// ── Table defaults ──────────────────────────────────────────────────
export const TABLE = {
  actionColumnWidth: 140,
  idColumnWidth: 100,
  scrollX: 1200,
  pagination: { pageSize: 20, showSizeChanger: true, showTotal: (total: number) => `共 ${total} 条` },
} as const

// ── Component dimensions ────────────────────────────────────────────
export const SIZE = {
  buttonSm: 32,
  buttonMd: 40,
  buttonLg: 48,
  inputHeight: 40,
  iconSm: 16,
  iconMd: 20,
  sidebarWidth: 240,
  navHeight: 56,
} as const

// ── Layout gutters ──────────────────────────────────────────────────
export const GUTTER = {
  sm: 16,
  md: 24,
  lg: 32,
} as const
