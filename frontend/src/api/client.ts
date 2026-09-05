import type {
  ApiErrorBody,
  AuditEntry,
  BatchRunResponse,
  BatchSummary,
  CaseDetail,
  CaseSummary,
} from './types'

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

/** Thrown for any non-2xx response; message is the backend's ErrorDTO.error when present. */
export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, init)
  if (!res.ok) {
    let message = res.statusText
    try {
      const body = (await res.json()) as ApiErrorBody
      if (body.error) message = body.error
    } catch {
      // response wasn't JSON; fall back to statusText
    }
    throw new ApiError(res.status, message)
  }
  return res.json() as Promise<T>
}

/** POST /api/batch/run — triggers a full pipeline run over the fixed demo dataset. */
export function runBatch(): Promise<BatchRunResponse> {
  return request<BatchRunResponse>('/api/batch/run', { method: 'POST' })
}

/** GET /api/batch/summary — always freshly queried from the DB. */
export function getBatchSummary(): Promise<BatchSummary> {
  return request<BatchSummary>('/api/batch/summary')
}

/** GET /api/cases — the list view. */
export function listCases(): Promise<CaseSummary[]> {
  return request<CaseSummary[]>('/api/cases')
}

/** GET /api/cases/:id — full reasoning chain for one case. */
export function getCase(id: string): Promise<CaseDetail> {
  return request<CaseDetail>(`/api/cases/${encodeURIComponent(id)}`)
}

/** GET /api/cases/:id/audit — raw audit log for one case. */
export function getCaseAudit(id: string): Promise<AuditEntry[]> {
  return request<AuditEntry[]>(`/api/cases/${encodeURIComponent(id)}/audit`)
}

/** GET /api/guardrail/spotlight — the disputed-case lockout, pre-filtered server-side. */
export function getGuardrailSpotlight(): Promise<CaseDetail> {
  return request<CaseDetail>('/api/guardrail/spotlight')
}
