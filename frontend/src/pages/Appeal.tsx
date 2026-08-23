import React, { useCallback, useEffect, useState } from 'react';
import {
  Table,
  Tag,
  Button,
  Modal,
  Form,
  Input,
  Space,
  message,
  Popconfirm,
  Tabs,
  Typography,
  Empty,
  Tooltip,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import {
  getAppeals,
  resolveAppeal,
  getElementsByContent,
  type AppealItem,
  type ContentElement,
} from '@/services/content-api';
import useAuthStore from '@/stores/auth';
import AppLayout, { Content } from '@/components/Layout';
import {
  COLORS,
  SPACING,
  FONT,
  ANIMATION,
  TABLE,
  RADIUS,
  riskLabel,
  riskTagColor,
} from '@/utils/constants';

const { Text } = Typography;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const APPEAL_STATUS_LABELS: Record<string, { color: string; label: string }> = {
  submitted: { color: 'warning', label: '待审核' },
  under_review: { color: 'processing', label: '审核中' },
  resolved_approved: { color: 'success', label: '已改判' },
  resolved_maintained: { color: 'error', label: '维持原判' },
};

function formatDate(iso?: string) {
  if (!iso) return '-';
  return new Date(iso).toLocaleString('zh-CN');
}

// ---------------------------------------------------------------------------
// Appeal Detail Modal
// ---------------------------------------------------------------------------

const AppealDetailModal: React.FC<{
  visible: boolean;
  appeal: AppealItem | null;
  onClose: () => void;
  onResolve: (id: string, decision: 'approved' | 'maintained', comment: string) => void;
}> = ({ visible, appeal, onClose, onResolve }) => {
  const [form] = Form.useForm<{ comment: string }>();
  const [elements, setElements] = useState<ContentElement[]>([]);
  const [elementsLoading, setElementsLoading] = useState(false);

  useEffect(() => {
    if (visible && appeal) {
      setElementsLoading(true);
      getElementsByContent(appeal.content_id)
        .then((data) => setElements(Array.isArray(data) ? data : []))
        .catch(() => setElements([]))
        .finally(() => setElementsLoading(false));
    } else {
      setElements([]);
    }
  }, [visible, appeal]);

  if (!appeal) return null;

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      await onResolve(appeal.id, 'maintained', values.comment);
      message.success('申诉已处理');
      onClose();
    } catch {
      message.error('操作失败');
    }
  };

  return (
    <Modal
      title="申诉详情"
      open={visible}
      onCancel={onClose}
      onOk={handleSubmit}
      width={640}
      footer={[
        <Button key="cancel" onClick={onClose}>
          取消
        </Button>,
        <Popconfirm
          key="maintain-confirm"
          title="确认维持原判？"
          description="维持原判后申诉将不可撤销。"
          okText="确认维持"
          cancelText="取消"
          onConfirm={async () => {
            await onResolve(appeal.id, 'maintained', '维持原判');
            onClose();
          }}
        >
          <Button type="default" icon={<CloseCircleOutlined />}>
            维持原判
          </Button>
        </Popconfirm>,
        <Popconfirm
          key="reverse-confirm"
          title="确认改判？"
          description="改判后将恢复原内容，此操作不可撤销。"
          okText="确认改判"
          cancelText="取消"
          onConfirm={async () => {
            await onResolve(appeal.id, 'approved', '改判通过');
            onClose();
          }}
        >
          <Button type="primary" icon={<CheckCircleOutlined />}>
            改判通过
          </Button>
        </Popconfirm>,
      ]}
    >
      <Form form={form} layout="vertical">
        <Form.Item label="申诉内容">
          <Input.TextArea rows={4} value={appeal.reason} readOnly />
        </Form.Item>
        {appeal.evidence_urls && appeal.evidence_urls.length > 0 && (
          <Form.Item label="证明材料">
            <Space direction="vertical" style={{ width: '100%' }}>
              {appeal.evidence_urls.map((url, i) => (
                <a key={i} href={url} target="_blank" rel="noopener noreferrer">
                  查看附件 {i + 1}
                </a>
              ))}
            </Space>
          </Form.Item>
        )}
        <Form.Item label="处理备注" name="comment">
          <Input.TextArea rows={3} placeholder="可选" />
        </Form.Item>

        {/* Original AI review results */}
        <Form.Item label="原始 AI 审核结果">
          {elementsLoading ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: SPACING.xs }}>
              {[1, 2, 3].map((i) => (
                <div key={i} style={{
                  padding: SPACING.sm,
                  background: COLORS.bgContainer,
                  borderRadius: SPACING.xs,
                  border: `1px solid ${COLORS.borderDefault}`,
                }}>
                  <div style={{
                    height: 14,
                    borderRadius: 4,
                    background: `linear-gradient(90deg, ${COLORS.bgContainer} 25%, ${COLORS.bgBase} 50%, ${COLORS.bgContainer} 75%)`,
                    backgroundSize: '200% 100%',
                    animation: `skeleton-loading ${ANIMATION.normal} ease-in-out infinite`,
                    marginBottom: 8,
                  }} />
                  <div style={{
                    height: 12,
                    width: '60%',
                    borderRadius: 4,
                    background: `linear-gradient(90deg, ${COLORS.bgContainer} 25%, ${COLORS.bgBase} 50%, ${COLORS.bgContainer} 75%)`,
                    backgroundSize: '200% 100%',
                    animation: `skeleton-loading ${ANIMATION.normal} ease-in-out infinite`,
                  }} />
                </div>
              ))}
            </div>
          ) : (
            elements.length > 0 ? (
              <Space direction="vertical" style={{ width: '100%' }} size={SPACING.xs}>
                {elements.map((elem) => (
                  <div key={elem.id} style={{
                    padding: `${SPACING.sm} ${SPACING.base}`,
                    background: COLORS.bgBase,
                    borderRadius: RADIUS.sm,
                    fontSize: FONT.small,
                  }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: SPACING.xs }}>
                      <span style={{ color: COLORS.textSecondary }}>{elem.element_kind}</span>
                      <Tag color={riskTagColor(elem.ai_risk_score)}>
                        风险分: {elem.ai_risk_score} ({riskLabel(elem.ai_risk_score)})
                      </Tag>
                    </div>
                    {elem.ai_risk_types.length > 0 && (
                      <div style={{ display: 'flex', gap: SPACING.xs, flexWrap: 'wrap' }}>
                        {elem.ai_risk_types.map((t) => (
                          <Tag key={t} color="error">{t}</Tag>
                        ))}
                      </div>
                    )}
                    <div style={{ color: COLORS.textMuted, fontSize: FONT.caption, marginTop: SPACING.xs }}>
                      置信度: {(elem.ai_confidence * 100).toFixed(1)}%
                      {elem.is_conflict && ' | AI意见不一致'}
                    </div>
                  </div>
                ))}
              </Space>
            ) : (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无元素数据" />
            )
          )}
        </Form.Item>
      </Form>
    </Modal>
  );
};

// ---------------------------------------------------------------------------
// Appeal Management Page
// ---------------------------------------------------------------------------

const AppealPage: React.FC = () => {
  const user = useAuthStore((s) => s.user);
  const [appeals, setAppeals] = useState<AppealItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedAppeal, setSelectedAppeal] = useState<AppealItem | null>(null);
  const [activeTab, setActiveTab] = useState<'submitted' | 'resolved'>('submitted');

  const fetchAppeals = useCallback(async () => {
    setLoading(true);
    try {
      const status = activeTab === 'submitted' ? 'submitted' : undefined;
      const data = await getAppeals({ status, page: 1, page_size: 50 });
      setAppeals(data.items ?? []);
      setTotal(data.total ?? 0);
    } catch {
      message.error('获取申诉列表失败');
      setAppeals([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [activeTab]);

  useEffect(() => {
    fetchAppeals();
  }, [fetchAppeals]);

  const handleResolve = async (
    appealId: string,
    decision: 'approved' | 'maintained',
    comment: string,
  ) => {
    try {
      await resolveAppeal(appealId, decision, comment, user?.id ?? '');
      message.success('申诉已处理');
      fetchAppeals();
    } catch {
      message.error('处理申诉失败');
    }
  };

  const columns = [
    {
      title: '申诉ID',
      dataIndex: 'id',
      key: 'id',
      width: TABLE.idColumnWidth,
      render: (v: string) => <Text copyable={{ text: v }} style={{ fontSize: FONT.caption }}>{v.slice(0, 8)}...</Text>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const cfg = APPEAL_STATUS_LABELS[status] || { color: 'default', label: status };
        return <Tag color={cfg.color}>{cfg.label}</Tag>;
      },
    },
    {
      title: '申诉理由',
      dataIndex: 'reason',
      key: 'reason',
      ellipsis: true,
    },
    {
      title: '提交时间',
      dataIndex: 'submitted_at',
      key: 'submitted_at',
      render: formatDate,
    },
    {
      title: '操作',
      key: 'actions',
      width: TABLE.actionColumnWidth,
      render: (_: unknown, record: AppealItem) => (
        <Space size={SPACING.xs}>
          <Tooltip title="查看详情">
            <Button
              type="link"
              icon={<EyeOutlined />}
              onClick={() => {
                setSelectedAppeal(record);
                setDetailVisible(true);
              }}
            />
          </Tooltip>
          {record.status === 'submitted' && (
            <Popconfirm
              title="确认维持原判？"
              onConfirm={() => handleResolve(record.id, 'maintained', '维持原判')}
            >
              <Tooltip title="维持原判">
                <Button type="link" danger icon={<CloseCircleOutlined />} />
              </Tooltip>
            </Popconfirm>
          )}
          {record.status === 'submitted' && (
            <Popconfirm
              title="确认改判？"
              onConfirm={() => handleResolve(record.id, 'approved', '改判通过')}
            >
              <Tooltip title="改判通过">
                <Button type="link" icon={<CheckCircleOutlined />} style={{ color: COLORS.success }} />
              </Tooltip>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <AppLayout>
      <Content style={{ padding: SPACING.lg, background: COLORS.bgBase }}>
        <div style={{ marginBottom: SPACING.lg, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography.Title level={4} style={{ color: COLORS.textPrimary, margin: 0 }}>
            申诉管理
          </Typography.Title>
        </div>

        <Tabs
          activeKey={activeTab}
          onChange={(key) => { setActiveTab(key as 'submitted' | 'resolved'); setDetailVisible(false); setSelectedAppeal(null); }}
          items={[
            {
              key: 'submitted',
              label: <span>待处理 ({total})</span>,
            },
            {
              key: 'resolved',
              label: <span>已处理</span>,
            },
          ]}
        />

        <div style={{ marginBottom: SPACING.base, color: COLORS.textSecondary, fontSize: FONT.small }}>
          共 <Text style={{ color: COLORS.textPrimary }}>{total}</Text> 条申诉记录
        </div>

        <Table
          columns={columns}
          dataSource={appeals}
          rowKey="id"
          loading={loading}
          scroll={{ x: TABLE.scrollX }}
          pagination={TABLE.pagination}
        />

        <AppealDetailModal
          visible={detailVisible}
          appeal={selectedAppeal}
          onClose={() => {
            setDetailVisible(false);
            setSelectedAppeal(null);
          }}
          onResolve={handleResolve}
        />
      </Content>
    </AppLayout>
  );
};

export default AppealPage;
