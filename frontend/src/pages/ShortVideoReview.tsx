import React, { useCallback, useEffect, useState } from 'react';
import {
  Table,
  Tag,
  Button,
  Modal,
  Select,
  Space,
  message,
  Popconfirm,
  Divider,
  Row,
  Col,
  Badge,
  Typography,
  Card,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  PlayCircleOutlined,
  CommentOutlined,
  FileTextOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  getContents,
  getPendingElements,
  humanReview,
  type ContentItem,
  type ContentElement,
} from '@/services/content-api';
import useAuthStore from '@/stores/auth';
import AppLayout, { Content } from '@/components/Layout';
import {
  COLORS,
  SPACING,
  FONT,
  TABLE,
  RADIUS,
  riskColor,
  riskLabel,
  riskTagColor,
  ELEMENT_KIND_LABELS,
  AI_STATUS_COLORS,
  AI_STATUS_LABELS,
} from '@/utils/constants';

const { Text } = Typography;

// ---------------------------------------------------------------------------
// ShortVideo Review Panel
// ---------------------------------------------------------------------------

const ShortVideoPanel: React.FC<{
  content: ContentItem;
  elements: ContentElement[];
  onApprove: (elementId: string) => void;
  onReject: (elementId: string, reason: string) => void;
}> = ({ content, elements, onApprove, onReject }) => {
  const [rejectingId, setRejectingId] = useState<string | null>(null);

  const videoUrl = elements.find((e) => e.element_kind === 'cover_image')?.element_content;
  const title = elements.find((e) => e.element_kind === 'title')?.element_content;
  const description = elements.find((e) => e.element_kind === 'description')?.element_content;
  const asrElements = elements.filter((e) => e.element_kind === 'asr_text');
  const commentElements = elements.filter((e) => e.element_kind === 'comment');
  const frameElements = elements.filter((e) => e.element_kind === 'video_frame');
  const conflictElements = elements.filter((e) => e.is_conflict);

  const hasConflict = conflictElements.length > 0;
  const overallRisk = content.ai_risk_score;

  return (
    <Row gutter={[SPACING.base, SPACING.base]}>
      {/* Left: Video Player */}
      <Col xs={24} md={12}>
        <Card
          title="视频预览"
          size="small"
          style={{ background: COLORS.bgContainer, borderColor: COLORS.borderDefault }}
          styles={{ body: { padding: `0 ${SPACING.base} ${SPACING.base}` } }}
        >
          {videoUrl ? (
            <video
              controls
              preload="metadata"
              style={{ width: '100%', borderRadius: RADIUS.md, background: COLORS.bgBase }}
              src={videoUrl}
            >
              您的浏览器不支持视频播放
            </video>
          ) : (
            <div style={{
              height: 200,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: COLORS.bgBase,
              borderRadius: RADIUS.md,
            }}>
              <PlayCircleOutlined style={{ fontSize: 48, color: COLORS.textMuted }} />
            </div>
          )}

          {title && (
            <div style={{ marginTop: SPACING.sm }}>
              <Text strong style={{ color: COLORS.textPrimary }}>标题：</Text>
              <Text style={{ color: COLORS.textTertiary }}>{title}</Text>
            </div>
          )}
          {description && (
            <div style={{ marginTop: SPACING.xs }}>
              <Text strong style={{ color: COLORS.textPrimary }}>描述：</Text>
              <Text style={{ color: COLORS.textTertiary }}>{description}</Text>
            </div>
          )}

          {/* AI Risk Summary */}
          <div style={{ marginTop: SPACING.base, display: 'flex', gap: SPACING.sm, flexWrap: 'wrap' }}>
            <Badge
              count={overallRisk}
              style={{
                backgroundColor: riskColor(overallRisk),
                fontSize: FONT.small,
                fontWeight: 600,
                cursor: 'pointer',
              }}
              title={`风险分: ${overallRisk} (${riskLabel(overallRisk)})`}
            />
            {content.ai_risk_score > 60 && (
              <Tag color="error">高风险</Tag>
            )}
            {hasConflict && (
              <Tag color="warning">
                <WarningOutlined /> AI意见不一致
              </Tag>
            )}
          </div>
        </Card>

        {/* Video Frames */}
        {frameElements.length > 0 && (
          <Card
            title="视频帧"
            size="small"
            style={{ marginTop: SPACING.base, background: COLORS.bgContainer, borderColor: COLORS.borderDefault }}
          >
            <div style={{ display: 'flex', gap: SPACING.sm, flexWrap: 'wrap' }}>
              {frameElements.map((frame) => (
                <img
                  key={frame.id}
                  src={frame.element_content}
                  alt="视频帧"
                  style={{
                    width: 120,
                    height: 68,
                    objectFit: 'cover',
                    borderRadius: RADIUS.sm,
                    cursor: 'zoom-in',
                  }}
                />
              ))}
            </div>
          </Card>
        )}
      </Col>

      {/* Right: ASR Transcript + Comments + Actions */}
      <Col xs={24} md={12}>
        {/* ASR Transcript */}
        {asrElements.length > 0 && (
          <Card
            title={<><FileTextOutlined /> ASR 转写</>}
            size="small"
            style={{ background: COLORS.bgContainer, borderColor: COLORS.borderDefault, marginBottom: SPACING.sm }}
          >
            {asrElements.map((asr) => (
              <div key={asr.id} style={{
                padding: SPACING.sm,
                marginBottom: SPACING.sm,
                background: COLORS.bgBase,
                borderRadius: RADIUS.sm,
              }}>
                <Text style={{ color: COLORS.textTertiary, lineHeight: 1.8 }}>{asr.element_content}</Text>
                {asr.ai_risk_types.length > 0 && (
                  <div style={{ marginTop: SPACING.xs }}>
                    {asr.ai_risk_types.map((t) => (
                      <Tag key={t} color="error" style={{ marginRight: SPACING.xs, fontSize: FONT.caption }}>{t}</Tag>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </Card>
        )}

        {/* Comments */}
        {commentElements.length > 0 && (
          <Card
            title={<><CommentOutlined /> 评论</>}
            size="small"
            style={{ background: COLORS.bgContainer, borderColor: COLORS.borderDefault, marginBottom: SPACING.sm }}
          >
            {commentElements.map((comment) => (
              <div key={comment.id} style={{
                padding: SPACING.sm,
                marginBottom: SPACING.sm,
                background: COLORS.bgBase,
                borderRadius: RADIUS.sm,
              }}>
                <Text style={{ color: COLORS.textTertiary }}>{comment.element_content}</Text>
                {comment.ai_risk_types.length > 0 && (
                  <div style={{ marginTop: SPACING.xs }}>
                    {comment.ai_risk_types.map((t) => (
                      <Tag key={t} color="error" style={{ marginRight: SPACING.xs, fontSize: FONT.caption }}>{t}</Tag>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </Card>
        )}

        <Divider style={{ borderColor: COLORS.borderDefault }} />

        {/* Review Actions */}
        <Card size="small" style={{ background: COLORS.bgContainer, borderColor: COLORS.borderDefault }}>
          <Typography.Title level={5} style={{ color: COLORS.textPrimary, marginTop: 0, marginBottom: 0 }}>人工审核</Typography.Title>
          <Space direction="vertical" style={{ width: '100%' }} size={SPACING.base}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Text style={{ color: COLORS.textSecondary }}>AI 风险分：</Text>
              <Text style={{ color: riskColor(overallRisk), fontWeight: 600 }}>{overallRisk} / 100</Text>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Text style={{ color: COLORS.textSecondary }}>AI 状态：</Text>
              <Tag color={AI_STATUS_COLORS[content.status] || 'default'}>
                {AI_STATUS_LABELS[content.status] || content.status}
              </Tag>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Text style={{ color: COLORS.textSecondary }}>元素数量：</Text>
              <Text style={{ color: COLORS.textPrimary }}>{elements.length}</Text>
            </div>
            <Space.Compact style={{ width: '100%' }}>
              <Button
                type="primary"
                icon={<CheckCircleOutlined />}
                onClick={() => onApprove(elements[0]?.id)}
                disabled={elements.length === 0}
                style={{ flex: 1 }}
                aria-label="全部通过"
              >
                全部通过
              </Button>
              <Popconfirm
                title="确认打回"
                description="确定要对此视频进行打回操作吗？此操作不可撤销。"
                okText="确认"
                cancelText="取消"
                onConfirm={() => setRejectingId(elements[0]?.id || null)}
              >
                <Button
                  danger
                  icon={<CloseCircleOutlined />}
                  style={{ flex: 1 }}
                  aria-label="全部打回"
                >
                  全部打回
                </Button>
              </Popconfirm>
            </Space.Compact>
          </Space>
        </Card>

        {/* Per-element review */}
        {elements.length > 1 && (
          <Card
            title="逐元素审核"
            size="small"
            style={{ marginTop: SPACING.base, background: COLORS.bgContainer, borderColor: COLORS.borderDefault }}
          >
            <Table<ContentElement>
              columns={[
                {
                  title: '类型',
                  dataIndex: 'element_kind',
                  width: TABLE.idColumnWidth,
                  render: (v: string) => ELEMENT_KIND_LABELS[v] || v,
                },
                {
                  title: '风险分',
                  dataIndex: 'ai_risk_score',
                  width: 80,
                  render: (v: number) => (
                    <Text style={{ color: riskColor(v) }}>{v}</Text>
                  ),
                },
                {
                  title: 'AI 状态',
                  dataIndex: 'ai_status',
                  width: 120,
                  render: (v: string) => (
                    <Tag color={AI_STATUS_COLORS[v] || 'default'}>
                      {AI_STATUS_LABELS[v] || v}
                    </Tag>
                  ),
                },
                {
                  title: '操作',
                  key: 'actions',
                  width: TABLE.actionColumnWidth,
                  render: (_: unknown, record: ContentElement) => (
                    <Space size={SPACING.xs}>
                      <Button
                        size="small"
                        type="link"
                        icon={<CheckCircleOutlined />}
                        onClick={() => onApprove(record.id)}
                      >
                        通过
                      </Button>
                      <Popconfirm
                        title="选择打回原因"
                        onConfirm={() => setRejectingId(record.id)}
                      >
                        <Button size="small" type="link" danger icon={<CloseCircleOutlined />}>
                          打回
                        </Button>
                      </Popconfirm>
                    </Space>
                  ),
                },
              ]}
              dataSource={elements}
              rowKey="id"
              pagination={TABLE.pagination}
              size="small"
            />
          </Card>
        )}

        {/* Reject reason modal */}
        <Modal
          title="选择打回原因"
          open={!!rejectingId}
          onCancel={() => setRejectingId(null)}
          onOk={() => {
            if (rejectingId) {
              onReject(rejectingId, '批量打回');
              setRejectingId(null);
            }
          }}
        >
          <Select
            mode="tags"
            placeholder="选择打回原因"
            options={[
              { label: '版权争议', value: 'copyright' },
              { label: '内容低质', value: 'low_quality' },
              { label: '政治敏感', value: 'politics' },
              { label: '色情低俗', value: 'porn' },
              { label: '暴力恐怖', value: 'violence' },
              { label: '垃圾广告', value: 'spam' },
            ]}
            style={{ width: '100%' }}
            onChange={(values) => {
              if (values.length > 0 && rejectingId) {
                onReject(rejectingId, values.join(','));
                setRejectingId(null);
              }
            }}
          />
        </Modal>
      </Col>
    </Row>
  );
};

// ---------------------------------------------------------------------------
// ShortVideo Review Page
// ---------------------------------------------------------------------------

const ShortVideoReviewPage: React.FC = () => {
  const user = useAuthStore((s) => s.user);
  const reviewerId = user?.id || '';

  const [contents, setContents] = useState<ContentItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, _setPage] = useState(1);
  const [pageSize] = useState(10);
  const [selectedContent, setSelectedContent] = useState<ContentItem | null>(null);
  const [elements, setElements] = useState<ContentElement[]>([]);
  const [elementsLoading, setElementsLoading] = useState(false);

  const fetchContents = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getContents({ content_type: 'short_video', page, page_size: pageSize });
      setContents(data.items ?? []);
      setTotal(data.total ?? 0);
    } catch {
      message.error('获取短视频列表失败');
      setContents([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [page, pageSize]);

  const fetchElements = useCallback(async (contentId: string) => {
    setElementsLoading(true);
    try {
      const data = await getPendingElements({ page_size: 100 });
      setElements(data.items?.filter((e) => e.content_id === contentId) ?? []);
    } catch {
      setElements([]);
    } finally {
      setElementsLoading(false);
    }
  }, []);

  const handleSelectContent = useCallback((content: ContentItem) => {
    setSelectedContent(content);
    fetchElements(content.id);
  }, [fetchElements]);

  useEffect(() => { fetchContents(); }, [fetchContents]);

  const handleApprove = async (elementId: string) => {
    try {
      await humanReview(elementId, 'approve', undefined, undefined, reviewerId);
      message.success('审核通过');
      if (selectedContent) fetchElements(selectedContent.id);
      fetchContents();
    } catch {
      message.error('操作失败');
    }
  };

  const handleReject = async (elementId: string, reason: string) => {
    try {
      await humanReview(elementId, 'reject', reason, undefined, reviewerId);
      message.success('已打回');
      if (selectedContent) fetchElements(selectedContent.id);
      fetchContents();
    } catch {
      message.error('操作失败');
    }
  };

  const columns: ColumnsType<ContentItem> = [
    {
      title: '内容ID',
      dataIndex: 'id',
      key: 'id',
      width: TABLE.idColumnWidth,
      render: (v: string) => (
        <Text copyable={{ text: v }} style={{ fontSize: FONT.caption }}>{v.slice(0, 8)}...</Text>
      ),
    },
    {
      title: 'AI 风险分',
      dataIndex: 'ai_risk_score',
      key: 'ai_risk_score',
      width: 100,
      render: (v: number) => (
        <Badge count={v} style={{ backgroundColor: riskColor(v) }} />
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (v: string) => (
        <Tag color={v === 'approved' ? 'success' : v === 'rejected' ? 'error' : 'warning'}>
          {v}
        </Tag>
      ),
    },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180, render: (v: string) => new Date(v).toLocaleString('zh-CN') },
    {
      title: '操作',
      key: 'actions',
      width: TABLE.actionColumnWidth,
      render: (_: unknown, record: ContentItem) => (
        <Button type="link" onClick={() => handleSelectContent(record)}>
          审核
        </Button>
      ),
    },
  ];

  return (
    <AppLayout>
      <Content style={{ padding: SPACING.lg, background: COLORS.bgBase }}>
        <div style={{ marginBottom: SPACING.lg }}>
          <Typography.Title level={4} style={{ color: COLORS.textPrimary, margin: 0 }}>
            短视频审核
          </Typography.Title>
        </div>

        {selectedContent ? (
          <>
            <div style={{ marginBottom: SPACING.base, display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: SPACING.sm }}>
              <Space>
                <Button onClick={() => setSelectedContent(null)}>← 返回列表</Button>
                <Text style={{ color: COLORS.textSecondary }}>
                  内容ID: <Text copyable={{ text: selectedContent.id }}>{selectedContent.id.slice(0, 8)}...</Text>
                </Text>
              </Space>
              <Space>
                <Tag color={riskTagColor(selectedContent.ai_risk_score)}>
                  风险分: {selectedContent.ai_risk_score}
                </Tag>
              </Space>
            </div>

            {elementsLoading ? (
              <div style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                minHeight: 400,
                background: COLORS.bgContainer,
                borderRadius: RADIUS.md,
                border: `1px solid ${COLORS.borderDefault}`,
              }}>
                <div style={{
                  width: 24,
                  height: 24,
                  border: `3px solid ${COLORS.borderDefault}`,
                  borderTopColor: COLORS.brandBlue,
                  borderRadius: '50%',
                  animation: `spin 0.8s linear infinite`,
                }} />
              </div>
            ) : (
              <ShortVideoPanel
                content={selectedContent}
                elements={elements}
                onApprove={handleApprove}
                onReject={handleReject}
              />
            )}
          </>
        ) : (
          <>
            <div style={{ marginBottom: SPACING.lg, display: 'flex', gap: SPACING.sm }}>
              <Tag color="info">待审短视频: {total}</Tag>
            </div>

            <Table<ContentItem>
              columns={columns}
              dataSource={contents}
              rowKey="id"
              loading={loading}
              scroll={{ x: TABLE.scrollX }}
              pagination={TABLE.pagination}
            />
          </>
        )}
      </Content>
    </AppLayout>
  );
};

export default ShortVideoReviewPage;
