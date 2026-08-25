import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  Card,
  Row,
  Col,
  Select,
  Slider,
  Tag,
  Button,
  Space,
  message,
  Tooltip,
  Spin,
  Popconfirm,
  Image,
  Checkbox,
  Upload,
  Pagination,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  EyeOutlined,
  InboxOutlined,
  BellOutlined,
} from '@ant-design/icons';
import { CardSkeleton } from '@/components/Skeleton';
import {
  getPendingElements,
  humanReview,
  getElementStats,
  batchReview,
  uploadFile,
  uploadContent,
  type ContentElement,
  type ElementStats,
} from '@/services/content-api';
import useAuthStore from '@/stores/auth';
import AppLayout, { Content } from '@/components/Layout';
import {
  COLORS,
  SPACING,
  FONT,
  TABLE,
  RADIUS,
  riskTagColor,
  riskLabel,
  ELEMENT_KIND_LABELS,
  AI_STATUS_COLORS,
  AI_STATUS_LABELS,
  ANIMATION as ANIM,
} from '@/utils/constants';

// --- Constants already imported from '@/utils/constants' ---

const RiskBadge: React.FC<{ score: number }> = ({ score }) => (
  <Tag color={riskTagColor(score)} style={{ margin: 0, fontWeight: 600 }}>
    {riskLabel(score)} {score}分
  </Tag>
);

const ElementCard: React.FC<{
  element: ContentElement;
  onApprove: (id: string) => void;
  onReject: (id: string, reason: string) => void;
  selected?: boolean;
  onSelect?: (id: string) => void;
  focused?: boolean;
  onFocus?: () => void;
  consecutiveApproves?: number;
}> = ({ element, onApprove, onReject, selected, onSelect, focused, onFocus, consecutiveApproves = 0 }) => {
  const [rejectOpen, setRejectOpen] = useState(false);

  const rejectOptions = [
    { label: '版权争议', value: 'copyright' },
    { label: '画面模糊', value: 'blur' },
    { label: '政治敏感', value: 'politics' },
    { label: '色情低俗', value: 'porn' },
    { label: '暴力恐怖', value: 'violence' },
    { label: '垃圾广告', value: 'spam' },
  ];

  const isImageKind = element.element_kind.includes('image') ||
    element.element_kind.includes('frame') ||
    element.element_kind.includes('snapshot');

  return (
    <Card
      hoverable
      onFocus={onFocus}
      tabIndex={0}
      role="button"
      aria-label={`${ELEMENT_KIND_LABELS[element.element_kind] || element.element_kind} - AI风险分 ${element.ai_risk_score}`}
      style={{
        border: focused
          ? `2px solid ${COLORS.brandBlue}`
          : element.is_conflict
            ? `2px solid ${COLORS.conflictOrange || '#fa8c16'}`
            : `1px solid ${COLORS.borderDefault}`,
        borderRadius: RADIUS.md,
        transition: `opacity ${ANIM.fast}, transform ${ANIM.fast}, border-color ${ANIM.normal}`,
        cursor: 'pointer',
      }}
      onMouseEnter={(e) => {
        (e.currentTarget as HTMLDivElement).style.opacity = '1';
        (e.currentTarget as HTMLDivElement).style.transform = 'translateY(-2px)';
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLDivElement).style.opacity = '0.95';
        (e.currentTarget as HTMLDivElement).style.transform = 'translateY(0)';
      }}
    >
      {/* Header: kind + conflict + checkbox */}
      <div style={{ marginBottom: SPACING.sm, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Space size={SPACING.xs}>
          <Tag>{ELEMENT_KIND_LABELS[element.element_kind] || element.element_kind}</Tag>
          {element.is_conflict && (
            <Tooltip title="AI 结论存在分歧，请重点审核">
              <Tag color="warning">
                <WarningOutlined /> 分歧
              </Tag>
            </Tooltip>
          )}
        </Space>
        {onSelect && (
          <Checkbox
            checked={selected}
            onChange={(e) => {
              e.stopPropagation();
              onSelect(element.id);
            }}
            onClick={(e) => e.stopPropagation()}
          />
        )}
      </div>

      {/* Content preview */}
      {isImageKind && element.element_content ? (
        <Image
          src={element.element_content}
          alt={`${element.element_kind} 预览`}
          style={{
            width: '100%',
            height: 120,
            objectFit: 'cover',
            borderRadius: RADIUS.sm,
            cursor: 'zoom-in',
            marginBottom: SPACING.sm,
          }}
          preview={{
            mask: (
              <Tooltip title="点击查看大图">
                <Button size="small" type="primary" icon={<EyeOutlined />} />
              </Tooltip>
            ),
          }}
          fallback="data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTAwJSIgaGVpZ2h0PSIxMDAlIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjxyZWN0IHdpZHRoPSIxMDAlIiBoZWlnaHQ9IjEwMCUiIGZpbGw9IiMxYTFhMWEiLz48dGV4dCB4PSI1MCUiIHk9IjUwJSIgZG9taW5hbnQtYmFzZWxpbmU9Im1pZGRsZSIgdGV4dC1hbmNob3I9Im1pZGRsZSIgZmlsbD0iIzY2NiIgZm9udC1zaXplPSIxMiI+5rWL6Ki/5LusPC90ZXh0Pjwvc3ZnPg=="
          onError={(e: React.SyntheticEvent<HTMLImageElement>) => {
            (e.target as HTMLImageElement).style.display = 'none';
            const parent = (e.target as HTMLImageElement).parentElement!;
            if (parent.querySelector('span') === null) {
              const span = document.createElement('span');
              span.style.color = COLORS.textMuted;
              span.textContent = '预览不可用';
              parent.appendChild(span);
            }
          }}
        />
      ) : (
        <div style={{
          padding: SPACING.sm,
          color: COLORS.textTertiary,
          fontSize: FONT.caption,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          maxWidth: '100%',
        }}>
          {element.element_content}
        </div>
      )}

      {/* AI Score + Status */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: SPACING.sm }}>
        <RiskBadge score={element.ai_risk_score} />
        <Tag color={AI_STATUS_COLORS[element.ai_status] || 'default'}>
          {AI_STATUS_LABELS[element.ai_status] || element.ai_status}
        </Tag>
      </div>

      {/* Risk types */}
      {element.ai_risk_types.length > 0 && (
        <div style={{ marginBottom: SPACING.sm }}>
          <Space size={SPACING.xs}>
            {element.ai_risk_types.map((t) => (
              <Tag key={t} color="error">{t}</Tag>
            ))}
          </Space>
        </div>
      )}

      {/* Confidence */}
      <div style={{ fontSize: FONT.caption, color: COLORS.textSecondary, marginBottom: SPACING.sm }}>
        置信度: {(element.ai_confidence * 100).toFixed(1)}%
      </div>

      {/* Actions */}
      <Space.Compact style={{ width: '100%' }}>
        <Popconfirm
          title={consecutiveApproves >= 5 ? `已连续通过 ${consecutiveApproves} 次，请确认` : '确认通过？'}
          description={consecutiveApproves >= 5 ? '连续快速通过可能增加误审风险' : '审核通过后该元素将标记为已处理'}
          okText="确认通过"
          cancelText="取消"
          onConfirm={() => onApprove(element.id)}
        >
          <Button
            type="primary"
            icon={<CheckCircleOutlined />}
            style={{ flex: 1 }}
            aria-label="通过"
          >
            通过
          </Button>
        </Popconfirm>
        <Popconfirm
          title="确认打回"
          description="确定要对此元素进行打回操作吗？此操作不可撤销。"
          okText="确认"
          cancelText="取消"
          onConfirm={() => setRejectOpen(true)}
        >
          <Button
            danger
            icon={<CloseCircleOutlined />}
            style={{ flex: 1 }}
            aria-label="打回"
          >
            打回
          </Button>
        </Popconfirm>
      </Space.Compact>

      {/* Reject reason selector */}
      {rejectOpen && (
        <Select
          mode="tags"
          placeholder="选择打回原因"
          options={rejectOptions}
          style={{ width: '100%', marginTop: SPACING.sm }}
          onChange={(values) => {
            if (values.length > 0) {
              onReject(element.id, values.join(','));
              setRejectOpen(false);
            }
          }}
        />
      )}
    </Card>
  );
};

// ---------------------------------------------------------------------------
// Review Page
// ---------------------------------------------------------------------------

const ReviewPage: React.FC = () => {
  const user = useAuthStore((s) => s.user);
  const token = useAuthStore((s) => s.token);
  const [elements, setElements] = useState<ContentElement[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [filterType, setFilterType] = useState<string>('');
  const [filterStatus, setFilterStatus] = useState<string>('');
  const [riskRange, setRiskRange] = useState<[number, number]>([0, 100]);
  const [sortBy, setSortBy] = useState<string>('ai_risk_score');
  const [sortOrder, setSortOrder] = useState<string>('desc');
  const [stats, setStats] = useState<ElementStats>({ pending_human: 0, human_passed: 0, human_rejected: 0, conflict: 0 });
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [focusedIndex, setFocusedIndex] = useState<number>(-1);
  const [wsConnected, setWsConnected] = useState(false);
  const [newTaskCount, setNewTaskCount] = useState(0);
  const [consecutiveApproves, setConsecutiveApproves] = useState(0);
  // P1 fix: Esc no longer rejects instantly. First press "arms" the reject
  // (with a hint), second press within 3s confirms it.
  const [escArmed, setEscArmed] = useState(false);
  const reviewerId = user?.id || '';
  const wsRef = useRef<WebSocket | null>(null);

  const toggleSelect = useCallback((id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const toggleAll = useCallback(() => {
    setSelectedIds((prev) => {
      if (prev.size === elements.length) return new Set();
      return new Set(elements.map((e) => e.id));
    });
  }, [elements]);

  const handleBatchApprove = async () => {
    if (selectedIds.size === 0) return;
    try {
      await batchReview([...selectedIds], 'approve', reviewerId);
      message.success(`已通过 ${selectedIds.size} 项`);
      setSelectedIds(new Set());
      fetchElements();
    } catch {
      message.error('批量操作失败');
    }
  };

  const handleBatchReject = async () => {
    if (selectedIds.size === 0) return;
    try {
      await batchReview([...selectedIds], 'reject', reviewerId, '批量打回');
      message.success(`已打回 ${selectedIds.size} 项`);
      setSelectedIds(new Set());
      fetchElements();
    } catch {
      message.error('批量操作失败');
    }
  };

  const fetchStats = useCallback(async () => {
    try {
      const data = await getElementStats({
        ai_status: filterStatus || undefined,
        element_kind: filterType || undefined,
        risk_min: riskRange[0],
        risk_max: riskRange[1],
      });
      setStats(data);
    } catch {
      // non-critical
    }
  }, [filterStatus, filterType, riskRange]);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  const fetchElements = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getPendingElements({
        ai_status: filterStatus || undefined,
        element_kind: filterType || undefined,
        risk_min: riskRange[0],
        risk_max: riskRange[1],
        sort_by: sortBy,
        sort_order: sortOrder,
        page,
        page_size: pageSize,
      });
      const items = data.items ?? [];
      setElements(items);
      setTotal(data.total ?? 0);
      // P1 fix: drop selections that are no longer on the current page
      // (page switch / filter change / item resolved elsewhere) so stale
      // IDs can't leak into a later batch operation invisibly.
      const visible = new Set(items.map((e) => e.id));
      setSelectedIds((prev) => {
        if (prev.size === 0) return prev;
        const next = new Set([...prev].filter((id) => visible.has(id)));
        return next.size === prev.size ? prev : next;
      });
    } catch {
      message.error('获取待审元素失败');
      setElements([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, filterStatus, filterType, riskRange, sortBy, sortOrder]);

  useEffect(() => {
    fetchElements();
  }, [fetchElements]);

  const handleApprove = async (elementId: string) => {
    try {
      await humanReview(elementId, 'approve', undefined, undefined, reviewerId);
      setConsecutiveApproves((prev) => prev + 1);
      message.success(`审核通过 — ${riskLabel(elements.find((e) => e.id === elementId)?.ai_risk_score ?? 0)}`);
      fetchElements();
    } catch {
      message.error('操作失败');
    }
  };

  // Track consecutive approves and reset after 3s of inactivity
  useEffect(() => {
    if (consecutiveApproves > 0) {
      const timer = setTimeout(() => setConsecutiveApproves(0), 3000);
      return () => clearTimeout(timer);
    }
  }, [consecutiveApproves]);

  const handleReject = async (elementId: string, reason: string) => {
    try {
      await humanReview(elementId, 'reject', reason, undefined, reviewerId);
      message.success('已打回');
      setEscArmed(false);
      fetchElements();
    } catch {
      message.error('操作失败');
    }
  };

  // Keyboard shortcuts
  const handleKeyboardShortcut = useCallback(
    (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement)?.tagName;
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;

      if (focusedIndex >= 0 && focusedIndex < elements.length) {
        if (e.key === 'Enter') {
          e.preventDefault();
          if (consecutiveApproves >= 5) {
            message.warning(`已连续通过 ${consecutiveApproves} 次，请确认操作`);
            return;
          }
          handleApprove(elements[focusedIndex].id);
        } else if (e.key === ' ') {
          e.preventDefault();
          if (consecutiveApproves >= 5) {
            message.warning(`已连续通过 ${consecutiveApproves} 次，请确认操作`);
            return;
          }
          handleApprove(elements[focusedIndex].id);
        } else if (e.key === 'Escape') {
          e.preventDefault();
          // P1 fix: two-stage confirm — first Esc arms, second Esc within
          // the arm window rejects. Any other key disarms.
          if (!escArmed) {
            setEscArmed(true);
            message.info('再按一次 Esc 确认打回，按其他键取消');
          } else {
            handleReject(elements[focusedIndex].id, '快捷键打回');
          }
          setFocusedIndex(-1);
        } else if (e.key === 'ArrowLeft') {
          e.preventDefault();
          setEscArmed(false);
          setFocusedIndex((i) => Math.max(0, i - 1));
        } else if (e.key === 'ArrowRight') {
          e.preventDefault();
          setEscArmed(false);
          setFocusedIndex((i) => Math.min(elements.length - 1, i + 1));
        }
      }
    },
    [focusedIndex, elements, handleApprove, handleReject, consecutiveApproves, escArmed],
  );

  useEffect(() => {
    window.addEventListener('keydown', handleKeyboardShortcut);
    return () => window.removeEventListener('keydown', handleKeyboardShortcut);
  }, [handleKeyboardShortcut]);

  // Esc arm auto-disarms after 3s
  useEffect(() => {
    if (escArmed) {
      const timer = setTimeout(() => setEscArmed(false), 3000);
      return () => clearTimeout(timer);
    }
  }, [escArmed]);

  // WebSocket
  useEffect(() => {
    if (!token || !user?.tenantId) return;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/review/ws?token=${token}`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => setWsConnected(true);

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === 'new_task') {
          setNewTaskCount((prev) => prev + 1);
          setTimeout(() => {
            fetchElements();
            fetchStats();
          }, 500);
        }
      } catch {
        // ignore malformed
      }
    };

    ws.onclose = () => {
      setWsConnected(false);
      setNewTaskCount(0);
    };

    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, [token, user?.tenantId]);

  // Shortcut hint bar
  const shortcutHint = focusedIndex >= 0 && focusedIndex < elements.length ? (
    <div style={{
      marginBottom: SPACING.base,
      padding: `${SPACING.xs} ${SPACING.sm}`,
      background: COLORS.bgContainerHover,
      borderRadius: RADIUS.sm,
      fontSize: FONT.caption,
      color: COLORS.textSecondary,
      display: 'flex',
      gap: SPACING.base,
    }}>
      <span>← → 切换元素</span>
      <span>Space/Enter 通过</span>
      <span>Esc 打回（连按两次确认）</span>
      {escArmed && (
        <span style={{ color: COLORS.conflictOrange, fontWeight: 600 }}>
          已武装打回 — 再按 Esc 确认
        </span>
      )}
      <span style={{ color: COLORS.brandBlue }}>
        当前: 第 {focusedIndex + 1}/{elements.length} 项
      </span>
    </div>
  ) : null;

  // File upload
  const handleFileUpload = useCallback(async (file: File) => {
    setUploading(true);
    try {
      const uploadResult = await uploadFile(file, user?.tenantId);
      const isVideo = uploadResult.is_video;
      const tenantId = user?.tenantId || '';
      const creatorId = user?.id || '';

      const contentResult = await uploadContent({
        content_type: isVideo ? 'short_video' : 'photo',
        title: file.name.replace(/\.[^.]+$/, ''),
        description: '',
        review_policy: 'post_then_review',
        file_urls: [uploadResult.url],
        tenant_id: tenantId,
        creator_id: creatorId,
      });

      message.success(
        isVideo
          ? `视频上传成功！正在抽帧和转写，内容ID: ${contentResult.content.id.slice(0, 8)}...`
          : `图片上传成功！内容ID: ${contentResult.content.id.slice(0, 8)}...`,
      );
      fetchElements();
      fetchStats();
    } catch {
      message.error('上传失败，请重试');
    } finally {
      setUploading(false);
    }
  }, [user, fetchElements, fetchStats]);

  const beforeUpload = useCallback((file: File) => {
    const isValidType = file.type.startsWith('image/') || file.type.startsWith('video/');
    if (!isValidType) {
      message.error('仅支持图片和视频文件！');
      return Upload.LIST_IGNORE;
    }
    const isLt100M = file.size / 1024 / 1024 < 100;
    if (!isLt100M) {
      message.error('文件大小不能超过 100MB！');
      return Upload.LIST_IGNORE;
    }
    handleFileUpload(file);
    return false;
  }, [handleFileUpload]);

  const isEmpty = !loading && elements.length === 0;

  return (
    <AppLayout>
      <Content style={{ padding: SPACING.lg, background: COLORS.bgBase }}>
        {/* Upload Area */}
        <div style={{ marginBottom: SPACING.lg }}>
          <Upload.Dragger
            name="file"
            customRequest={({ file }) => {
              const f = file as File;
              beforeUpload(f);
            }}
            accept="image/*,video/*"
            multiple={false}
            showUploadList={false}
            style={{ borderRadius: RADIUS.md }}
          >
            <div style={{ padding: `${SPACING.xl} 0` }}>
              <InboxOutlined style={{ fontSize: 48, color: COLORS.textMuted, marginBottom: SPACING.base }} />
              <p style={{ color: COLORS.textSecondary, marginBottom: SPACING.xs, fontSize: FONT.body }}>
                点击或拖拽文件到此区域上传
              </p>
              <p style={{ color: COLORS.textMuted, fontSize: FONT.caption }}>
                支持图片 (JPEG/PNG/GIF) 和视频 (MP4/MOV/WebM)，最大 100MB
              </p>
            </div>
          </Upload.Dragger>
          {uploading && (
            <div style={{ marginTop: SPACING.sm, textAlign: 'center' }}>
              <Spin size="small" />
              <span style={{ color: COLORS.textSecondary, marginLeft: SPACING.sm, fontSize: FONT.caption }}>上传中...</span>
            </div>
          )}
        </div>

        {/* Keyboard shortcut hint */}
        {shortcutHint}

        {/* Filters */}
        <div style={{ marginBottom: SPACING.lg }}>
          <Space wrap size={SPACING.sm}>
            <Select
              placeholder="元素类型"
              allowClear
              style={{ minWidth: 120 }}
              value={filterType || undefined}
              onChange={(val) => setFilterType(val || '')}
              options={[
                { label: '封面图', value: 'cover_image' },
                { label: '视频帧', value: 'video_frame' },
                { label: '标题', value: 'title' },
                { label: '评论', value: 'comment' },
                { label: '转写文本', value: 'asr_text' },
                { label: '直播截帧', value: 'live_snapshot' },
              ]}
            />
            <Select
              placeholder="AI 状态"
              allowClear
              style={{ minWidth: 120 }}
              value={filterStatus || undefined}
              onChange={(val) => setFilterStatus(val || '')}
              options={[
                { label: '待机审', value: 'pending_ai' },
                { label: '机审中', value: 'ai_processing' },
                { label: '机审通过', value: 'ai_passed' },
                { label: '机审拒绝', value: 'ai_rejected' },
                { label: '待人审', value: 'pending_human' },
                { label: '人审中', value: 'in_human_review' },
              ]}
            />
            <Select
              value={`${sortBy}-${sortOrder}`}
              onChange={(val) => {
                const [field, order] = val.split('-');
                setSortBy(field);
                setSortOrder(order);
              }}
              style={{ minWidth: 130 }}
              options={[
                { label: '风险分降序', value: 'ai_risk_score-desc' },
                { label: '风险分升序', value: 'ai_risk_score-asc' },
                { label: '最新创建', value: 'created_at-desc' },
                { label: '最早创建', value: 'created_at-asc' },
              ]}
            />
            <Slider
              range
              value={riskRange}
              onChange={(val) => setRiskRange(val as [number, number])}
              marks={{ 0: '0', 50: '50', 100: '100' }}
              style={{ width: 200 }}
            />
            <Button type="primary" onClick={fetchElements}>
              刷新
            </Button>
          </Space>
        </div>

        {/* Stats bar */}
        <div style={{ marginBottom: SPACING.lg, display: 'flex', gap: SPACING.sm, alignItems: 'center', flexWrap: 'wrap' }}>
          <Tag color={wsConnected ? 'success' : 'default'}>
            {wsConnected ? '● 实时连接' : '○ 离线'}
          </Tag>
          {newTaskCount > 0 && (
            <Tag color="warning" style={{ cursor: 'pointer' }} onClick={fetchElements}>
              <BellOutlined /> 新任务: {newTaskCount} (点击刷新)
            </Tag>
          )}
          <Tag color="info">待审: {stats.pending_human}</Tag>
          <Tag color="success">已通过: {stats.human_passed}</Tag>
          <Tag color="error">已打回: {stats.human_rejected}</Tag>
          <Tag color="warning">分歧: {stats.conflict}</Tag>
        </div>

        {/* Batch action bar */}
        {selectedIds.size > 0 && (
          <div style={{
            marginBottom: SPACING.base,
            padding: `${SPACING.sm} ${SPACING.base}`,
            background: COLORS.bgContainerHover,
            borderRadius: RADIUS.sm,
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}>
            <Space>
              <Checkbox
                checked={selectedIds.size === elements.length && elements.length > 0}
                indeterminate={selectedIds.size > 0 && selectedIds.size < elements.length}
                onChange={toggleAll}
              />
              <span style={{ color: COLORS.textTertiary, fontSize: FONT.small }}>
                已选择 <span style={{ color: COLORS.textPrimary, fontWeight: 600 }}>{selectedIds.size}</span> 项
              </span>
            </Space>
            <Space size={SPACING.sm}>
              <Button size="small" onClick={toggleAll}>
                {selectedIds.size === elements.length ? '取消全选' : '全选'}
              </Button>
              <Popconfirm title={`确定批量通过 ${selectedIds.size} 项？`} okText="确认" cancelText="取消" onConfirm={handleBatchApprove}>
                <Button size="small" type="primary" icon={<CheckCircleOutlined />}>
                  批量通过
                </Button>
              </Popconfirm>
              <Popconfirm title={`确定批量打回 ${selectedIds.size} 项？`} okText="确认" cancelText="取消" onConfirm={handleBatchReject}>
                <Button size="small" danger icon={<CloseCircleOutlined />}>
                  批量打回
                </Button>
              </Popconfirm>
              <Button size="small" onClick={() => setSelectedIds(new Set())}>
                清空
              </Button>
            </Space>
          </div>
        )}

        {/* Card Grid */}
        {loading ? (
          <Row gutter={[SPACING.base, SPACING.base]}>
            <CardSkeleton count={6} />
          </Row>
        ) : isEmpty ? (
            <div style={{ textAlign: 'center', padding: `${SPACING.xxl} 0`, color: COLORS.textMuted }}>
              <InboxOutlined style={{ fontSize: 48, marginBottom: SPACING.base, display: 'block' }} />
              <p style={{ fontSize: FONT.body }}>暂无待审元素</p>
              <p style={{ fontSize: FONT.caption, marginTop: SPACING.xs }}>
                {filterType || filterStatus || riskRange[0] !== 0 || riskRange[1] !== 100
                  ? '尝试调整筛选条件'
                  : '上传文件开始审核'}
              </p>
            </div>
          ) : (
            <Row gutter={[SPACING.base, SPACING.base]}>
              {elements.map((element, idx) => (
                <Col xs={24} sm={12} md={8} lg={6} key={element.id}>
                  <ElementCard
                    element={element}
                    onApprove={handleApprove}
                    onReject={handleReject}
                    selected={selectedIds.has(element.id)}
                    onSelect={toggleSelect}
                    focused={idx === focusedIndex}
                    onFocus={() => setFocusedIndex(idx)}
                    consecutiveApproves={consecutiveApproves}
                  />
                </Col>
              ))}
            </Row>
          )}

        {/* Pagination */}
        {!isEmpty && (
          <div style={{ marginTop: SPACING.lg, textAlign: 'right' }}>
            <Pagination
              {...TABLE.pagination}
              current={page}
              total={total}
              pageSize={pageSize}
              onChange={(p) => setPage(p)}
            />
          </div>
        )}
      </Content>
    </AppLayout>
  );
};

export default ReviewPage;
