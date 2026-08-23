import axiosInstance from './api';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ContentItem {
  id: string;
  tenant_id: string;
  content_type: 'photo' | 'short_video' | 'live_stream';
  review_policy: 'post_then_review' | 'review_before_post';
  ai_risk_score: number;
  status: string;
  creator_id: string;
  created_at: string;
  updated_at: string;
}

export interface ContentElement {
  id: string;
  content_id: string;
  element_kind: string;
  element_content: string;
  ai_risk_score: number;
  ai_risk_types: string[];
  ai_confidence: number;
  ai_status: string;
  human_status: string;
  is_conflict: boolean;
  thumbnail_url?: string;
  created_at: string;
}

export interface DashboardStats {
  total_reviewed: number;
  today_reviewed: number;
  approval_rate: number;
  rejection_rate: number;
  avg_risk_score: number;
  appeal_count: number;
  active_streams: number;
  pending_elements: number;
  conflict_count: number;
  accuracy_rate: number;
}

export interface ReviewerPerformance {
  reviewer_id: string;
  reviewer_name: string;
  total_reviews: number;
  approved: number;
  rejected: number;
  appeals: number;
  accuracy: number;
  avg_time_sec: number;
}

export interface AppealItem {
  id: string;
  tenant_id: string;
  content_id: string;
  applicant_id: string;
  reason: string;
  evidence_urls: string[];
  status: string;
  submitted_at: string;
  resolved_at?: string;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

interface Paginated<T> {
  data: T;
  total: number;
  page: number;
  page_size: number;
}

// The interceptor unwraps AxiosResponse to { code, message, data }.
// So axiosInstance.get(...) returns the backend wrapper (ApiResponse) directly.
// unwrap extracts the `data` field from the response.
// Note: The function signature uses `any` because the TypeScript generic on
// axiosInstance.get<ResponseType> declares AxiosResponse but the interceptor
// transforms it to ApiResponse. The runtime shape is always { data: ..., code: ..., message: ... }.
function unwrap<T>(res: any): T {
  return res.data;
}

// ---------------------------------------------------------------------------
// Content helpers
// ---------------------------------------------------------------------------

export async function getContents(params?: Record<string, unknown>): Promise<{ items: ContentItem[]; total: number }> {
  const res = await axiosInstance.get<Paginated<ContentItem[]>>('/contents', { params });
  const d = unwrap<Paginated<ContentItem[]>>(res);
  return { items: d.data, total: d.total };
}

export async function uploadFile(file: File, tenantId?: string): Promise<{ url: string; filename: string; size: number; content_type: string; is_video: boolean }> {
  const formData = new FormData();
  formData.append('file', file);
  if (tenantId) {
    formData.append('tenant_id', tenantId);
  }
  const res = await axiosInstance.post<{ data: { url: string; filename: string; size: number; content_type: string; is_video: boolean } }>('/contents/upload/file', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
  return unwrap<{ data: { url: string; filename: string; size: number; content_type: string; is_video: boolean } }>(res).data;
}

export async function uploadContent(data: {
  content_type: string;
  title?: string;
  description?: string;
  review_policy: string;
  file_urls: string[];
  tenant_id: string;
  creator_id: string;
}): Promise<{ content: ContentItem; elements: ContentElement[] }> {
  const res = await axiosInstance.post<{ data: { content: ContentItem; elements: ContentElement[] } }>('/contents', data);
  return unwrap<{ data: { content: ContentItem; elements: ContentElement[] } }>(res).data;
}

export async function getContentById(id: string): Promise<ContentItem> {
  const res = await axiosInstance.get<{ data: ContentItem }>(`/contents/${id}`);
  return unwrap<{ data: ContentItem }>(res).data;
}

export async function updateContentStatus(id: string, status: string): Promise<void> {
  return axiosInstance.put(`/contents/${id}/status`, { status });
}

// ---------------------------------------------------------------------------
// Element helpers (core audit workflow)
// ---------------------------------------------------------------------------

export async function getPendingElements(params?: Record<string, unknown>): Promise<{ items: ContentElement[]; total: number }> {
  const res = await axiosInstance.get<Paginated<ContentElement[]>>('/review/pending', { params });
  const d = unwrap<Paginated<ContentElement[]>>(res);
  return { items: d.data, total: d.total };
}

export async function getElementsByContent(contentId: string): Promise<ContentElement[]> {
  const res = await axiosInstance.get<{ data: ContentElement[] }>(`/review/content/${contentId}`);
  return unwrap<{ data: ContentElement[] }>(res).data;
}

export async function humanReview(
  elementId: string,
  action: 'approve' | 'reject',
  reason?: string,
  comment?: string,
  reviewerId?: string,
): Promise<void> {
  const body: Record<string, string> = { element_id: elementId, action };
  if (reason) body.reason = reason;
  if (comment) body.comment = comment;
  if (reviewerId) body.reviewer_id = reviewerId;
  return axiosInstance.post('/review/human', body);
}

export async function batchReview(
  elementIds: string[],
  action: 'approve' | 'reject',
  reviewerId?: string,
  reason?: string,
  comment?: string,
): Promise<void> {
  const body: Record<string, unknown> = { element_ids: elementIds, action };
  if (reviewerId) body.reviewer_id = reviewerId;
  if (reason) body.reason = reason;
  if (comment) body.comment = comment;
  return axiosInstance.post('/review/batch', body);
}

export interface ElementStats {
  pending_human: number;
  human_passed: number;
  human_rejected: number;
  conflict: number;
}

export async function getElementStats(params?: Record<string, unknown>): Promise<ElementStats> {
  const res = await axiosInstance.get<{ data: ElementStats }>('/review/stats', { params });
  return unwrap<{ data: ElementStats }>(res).data;
}

export async function resolveAppeal(
  appealId: string,
  decision: 'approved' | 'maintained',
  comment: string,
  reviewerId: string,
): Promise<void> {
  const body: Record<string, string> = { decision, reviewer_id: reviewerId };
  if (comment) body.comment = comment;
  return axiosInstance.put(`/review/appeal/${appealId}`, body);
}

// ---------------------------------------------------------------------------
// Dashboard & analytics helpers
// ---------------------------------------------------------------------------

export async function getDashboardStats(): Promise<DashboardStats> {
  const res = await axiosInstance.get<{ data: DashboardStats }>('/dashboard/stats');
  return unwrap<{ data: DashboardStats }>(res).data;
}

export async function getReviewerPerformance(
  page = 1,
  pageSize = 20,
): Promise<{ items: ReviewerPerformance[]; total: number }> {
  const res = await axiosInstance.get<Paginated<ReviewerPerformance[]>>('/dashboard/reviewers', {
    params: { page, page_size: pageSize },
  });
  const d = unwrap<Paginated<ReviewerPerformance[]>>(res);
  return { items: d.data, total: d.total };
}

export interface DailyTrendPoint {
  date: string;
  total_reviewed: number;
  approval_rate: number;
  rejection_rate: number;
}

export async function getDailyTrend(): Promise<DailyTrendPoint[]> {
  const res = await axiosInstance.get<{ data: DailyTrendPoint[] }>('/dashboard/trend');
  return unwrap<{ data: DailyTrendPoint[] }>(res).data;
}

export async function getAuditLogs(page = 1, pageSize = 20, params?: Record<string, string>): Promise<{ items: AuditLogItem[]; total: number }> {
  const res = await axiosInstance.get<Paginated<AuditLogItem[]>>('/review/logs', {
    params: { page, page_size: pageSize, ...params },
  });
  const d = unwrap<Paginated<AuditLogItem[]>>(res);
  return { items: d.data, total: d.total };
}

export interface AuditLogItem {
  id: string;
  task_id: string;
  element_id: string;
  reviewer_id: string | null;
  review_type: string;
  action: string;
  penalty_level_code: string | null;
  reason: string | null;
  comment: string | null;
  ai_score_before: number | null;
  ai_score_after: number | null;
  is_conflict: boolean;
  created_at: string;
}

export async function getAppeals(params?: Record<string, unknown>): Promise<{ items: AppealItem[]; total: number }> {
  const res = await axiosInstance.get<Paginated<AppealItem[]>>('/appeals', { params });
  const d = unwrap<Paginated<AppealItem[]>>(res);
  return { items: d.data, total: d.total };
}

export async function submitAppeal(data: { content_id: string; reason: string; evidence_urls?: string[]; applicant_id?: string }): Promise<AppealItem> {
  const res = await axiosInstance.post<{ data: AppealItem }>('/appeals', data);
  return unwrap<{ data: AppealItem }>(res).data;
}

// ---------------------------------------------------------------------------
// Quality Audit helpers
// ---------------------------------------------------------------------------

export interface QualityAuditBatch {
  id: string;
  tenant_id: string;
  created_by: string;
  name: string;
  mode: 'local_correction' | 'full_correction';
  filter_status: string;
  sample_size: number;
  status: 'draft' | 'in_progress' | 'completed';
  reviewed_count: number;
  created_at: string;
  completed_at?: string;
}

export interface QualityAuditRecord {
  id: string;
  batch_id: string;
  element_id: string;
  original_score: number;
  qa_score: number;
  qa_level: 'pass' | 'minor_issue' | 'major_issue' | 'critical';
  disagree: boolean;
  comment?: string;
  created_by: string;
  created_at: string;
}

export interface QualityAuditStats {
  batch_id: string;
  batch_name: string;
  total_samples: number;
  reviewed_count: number;
  disagree_count: number;
  disagree_rate: number;
  level_counts: Record<string, number>;
  avg_qa_score: number;
}

export async function getQualityBatches(params?: Record<string, unknown>): Promise<{ items: QualityAuditBatch[]; total: number }> {
  const res = await axiosInstance.get<Paginated<QualityAuditBatch[]>>('/quality/batches', { params });
  const d = unwrap<Paginated<QualityAuditBatch[]>>(res);
  return { items: d.data, total: d.total };
}

export async function createQualityBatch(data: { name: string; mode: string; filter_status: string; sample_size: number }): Promise<QualityAuditBatch> {
  const res = await axiosInstance.post<{ data: QualityAuditBatch }>('/quality/batches', data);
  return unwrap<{ data: QualityAuditBatch }>(res).data;
}

export async function getQualityBatchById(id: string): Promise<QualityAuditBatch> {
  const res = await axiosInstance.get<{ data: QualityAuditBatch }>(`/quality/batches/${id}`);
  return unwrap<{ data: QualityAuditBatch }>(res).data;
}

export async function startQualityBatch(id: string): Promise<void> {
  return axiosInstance.post(`/quality/batches/${id}/start`);
}

export async function completeQualityBatch(id: string): Promise<void> {
  return axiosInstance.post(`/quality/batches/${id}/complete`);
}

export async function submitQARecord(
  batchId: string,
  elementId: string,
  data: { qa_score: number; qa_level: string; disagree: boolean; comment?: string },
): Promise<QualityAuditRecord> {
  const res = await axiosInstance.post<{ data: QualityAuditRecord }>(`/quality/batches/${batchId}/records`, { ...data, element_id: elementId });
  return unwrap<{ data: QualityAuditRecord }>(res).data;
}

export async function getQualityBatchStats(id: string): Promise<QualityAuditStats> {
  const res = await axiosInstance.get<{ data: QualityAuditStats }>(`/quality/batches/${id}/stats`);
  return unwrap<{ data: QualityAuditStats }>(res).data;
}

export async function getQualityBatchRecords(id: string): Promise<QualityAuditRecord[]> {
  const res = await axiosInstance.get<{ data: QualityAuditRecord[] }>(`/quality/batches/${id}/records`);
  return unwrap<{ data: QualityAuditRecord[] }>(res).data;
}

// ---------------------------------------------------------------------------
// Live Wall helpers
// ---------------------------------------------------------------------------

export interface LiveFrameSnapshot {
  snapshot_url: string;
  ai_risk_score: number;
  ai_risk_types: string[];
  ai_confidence: number;
}

export interface LiveStreamWithSnapshot {
  id: string;
  tenant_id: string;
  content_id: string;
  stream_key: string;
  stream_url: string;
  play_url: string;
  status: string;
  viewer_count: number;
  started_at: string;
  created_at: string;
  updated_at: string;
  latest_snapshot?: LiveFrameSnapshot;
}

export async function getLiveWall(): Promise<LiveStreamWithSnapshot[]> {
  const res = await axiosInstance.get<{ data: LiveStreamWithSnapshot[] }>('/live/wall');
  return unwrap<{ data: LiveStreamWithSnapshot[] }>(res).data;
}

export async function startLiveStream(data: { content_id: string; stream_key: string; stream_url?: string; play_url?: string }): Promise<{ stream: LiveStreamWithSnapshot; rtmp_push_url: string; stream_key: string }> {
  const res = await axiosInstance.post<{ data: { stream: LiveStreamWithSnapshot; rtmp_push_url: string; stream_key: string } }>('/live/streams', data);
  return unwrap<{ data: { stream: LiveStreamWithSnapshot; rtmp_push_url: string; stream_key: string } }>(res).data;
}

export async function stopLiveStream(id: string): Promise<void> {
  return axiosInstance.delete(`/live/streams/${id}`);
}

export async function createSnapshot(
  streamId: string,
  data: { snapshot_url: string; snapshot_time: string; ai_risk_score: number; ai_risk_types: string[]; ai_confidence: number },
): Promise<void> {
  return axiosInstance.post(`/live/streams/${streamId}/snapshot`, data);
}

export async function getActiveStreamCount(): Promise<{ active_streams: number }> {
  const res = await axiosInstance.get<{ data: { active_streams: number } }>('/live/wall/count');
  return unwrap<{ data: { active_streams: number } }>(res).data;
}

// ---------------------------------------------------------------------------
// Tenant Config helpers (audit rules, levels, custom words)
// ---------------------------------------------------------------------------

export interface TenantAuditRule {
  id: string;
  tenant_id: string;
  rule_name: string;
  rule_expression?: string;
  action: string;
  priority: number;
  status: number;
  created_at: string;
}

export interface TenantAuditLevel {
  id: string;
  tenant_id: string;
  level_code: string;
  level_name: string;
  description?: string;
  status: number;
  created_at: string;
}

export interface TenantCustomWord {
  id: string;
  tenant_id: string;
  word: string;
  category?: string;
  status: number;
  created_at: string;
}

// --- Rules ---

export async function getAuditRules(page = 1, pageSize = 50): Promise<{ items: TenantAuditRule[]; total: number }> {
  const res = await axiosInstance.get<{ data: TenantAuditRule[]; total: number; page: number; page_size: number }>('/audit-rules', { params: { page, page_size: pageSize } });
  const d = unwrap<{ data: TenantAuditRule[]; total: number; page: number; page_size: number }>(res);
  return { items: d.data, total: d.total };
}

export async function createAuditRule(data: { rule_name: string; rule_expression?: string; action: string; priority: number }): Promise<TenantAuditRule> {
  const res = await axiosInstance.post<{ data: TenantAuditRule }>('/audit-rules', data);
  return unwrap<{ data: TenantAuditRule }>(res).data;
}

export async function updateAuditRule(id: string, data: Partial<{ rule_name: string; rule_expression: string; action: string; priority: number; status: number }>): Promise<void> {
  return axiosInstance.put(`/audit-rules/${id}`, data);
}

export async function deleteAuditRule(id: string): Promise<void> {
  return axiosInstance.delete(`/audit-rules/${id}`);
}

// --- Levels ---

export async function getAuditLevels(page = 1, pageSize = 50): Promise<{ items: TenantAuditLevel[]; total: number }> {
  const res = await axiosInstance.get<{ data: TenantAuditLevel[]; total: number; page: number; page_size: number }>('/audit-levels', { params: { page, page_size: pageSize } });
  const d = unwrap<{ data: TenantAuditLevel[]; total: number; page: number; page_size: number }>(res);
  return { items: d.data, total: d.total };
}

export async function createAuditLevel(data: { level_code: string; level_name: string; description?: string }): Promise<TenantAuditLevel> {
  const res = await axiosInstance.post<{ data: TenantAuditLevel }>('/audit-levels', data);
  return unwrap<{ data: TenantAuditLevel }>(res).data;
}

export async function updateAuditLevel(id: string, data: Partial<{ level_code: string; level_name: string; description: string; status: number }>): Promise<void> {
  return axiosInstance.put(`/audit-levels/${id}`, data);
}

export async function deleteAuditLevel(id: string): Promise<void> {
  return axiosInstance.delete(`/audit-levels/${id}`);
}

// --- Custom Words ---

export async function getCustomWords(page = 1, pageSize = 50): Promise<{ items: TenantCustomWord[]; total: number }> {
  const res = await axiosInstance.get<{ data: TenantCustomWord[]; total: number; page: number; page_size: number }>('/custom-words', { params: { page, page_size: pageSize } });
  const d = unwrap<{ data: TenantCustomWord[]; total: number; page: number; page_size: number }>(res);
  return { items: d.data, total: d.total };
}

export async function createCustomWord(data: { word: string; category?: string }): Promise<TenantCustomWord> {
  const res = await axiosInstance.post<{ data: TenantCustomWord }>('/custom-words', data);
  return unwrap<{ data: TenantCustomWord }>(res).data;
}

export async function updateCustomWord(id: string, data: Partial<{ word: string; category: string; status: number }>): Promise<void> {
  return axiosInstance.put(`/custom-words/${id}`, data);
}

export async function deleteCustomWord(id: string): Promise<void> {
  return axiosInstance.delete(`/custom-words/${id}`);
}

// ---------------------------------------------------------------------------
// AI Config helpers
// ---------------------------------------------------------------------------

export interface AIConfigItem {
  id: string;
  tenant_id: string;
  agnes_api_key: string;
  agnes_endpoint: string;
  agnes_concurrency: number;
  deepseek_api_key: string;
  deepseek_model: string;
  fallback_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export async function getAIConfig(): Promise<AIConfigItem | null> {
  const res = await axiosInstance.get<{ data: AIConfigItem } | { data: null }>('/ai-config');
  const d = unwrap<{ data: AIConfigItem } | { data: null }>(res);
  return d.data;
}

export async function saveAIConfig(data: {
  agnes_api_key?: string;
  agnes_endpoint?: string;
  agnes_concurrency?: number;
  deepseek_api_key?: string;
  deepseek_model?: string;
  fallback_enabled?: boolean;
}): Promise<AIConfigItem> {
  const res = await axiosInstance.put<{ data: AIConfigItem }>('/ai-config', data);
  return unwrap<{ data: AIConfigItem }>(res).data;
}
