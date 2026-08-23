import React, { useCallback, useEffect, useState } from 'react';
import {
  Table,
  Tag,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Space,
  message,
  Popconfirm,
  Tabs,
  Descriptions,
  Progress,
  InputNumber,
  Typography,
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  EyeOutlined,
  PlayCircleOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  getQualityBatches,
  createQualityBatch,
  startQualityBatch,
  completeQualityBatch,
  submitQARecord,
  getQualityBatchStats,
  getQualityBatchRecords,
  type QualityAuditBatch,
  type QualityAuditRecord,
  type QualityAuditStats,
  type ContentElement,
  getPendingElements,
} from '@/services/content-api';
import AppLayout, { Content } from '@/components/Layout';
import {
  COLORS,
  SPACING,
  FONT,
  TABLE,
  RADIUS,
  riskLabel,
  ANIMATION as ANIM,
} from '@/utils/constants';

const { Title, Text } = Typography;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const BATCH_STATUS_LABELS: Record<string, { color: string; label: string }> = {
  draft: { color: 'default', label: '草稿' },
  in_progress: { color: 'processing', label: '进行中' },
  completed: { color: 'success', label: '已完成' },
};

function renderBatchStatus(status: string) {
  const cfg = BATCH_STATUS_LABELS[status] || { color: 'default', label: status };
  return <Tag color={cfg.color}>{cfg.label}</Tag>;
}

function formatDate(iso?: string) {
  if (!iso) return '-';
  return new Date(iso).toLocaleString('zh-CN');
}

// ---------------------------------------------------------------------------
// Batch Detail Drawer
// ---------------------------------------------------------------------------

const BatchDetailDrawer: React.FC<{
  visible: boolean;
  batch: QualityAuditBatch | null;
  onClose: () => void;
}> = ({ visible, batch, onClose }) => {
  const [stats, setStats] = useState<QualityAuditStats | null>(null);
  const [records, setRecords] = useState<QualityAuditRecord[]>([]);
  const [samples, setSamples] = useState<ContentElement[]>([]);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<'samples' | 'records' | 'stats'>('samples');
  const [selectedElement, setSelectedElement] = useState<ContentElement | null>(null);
  const [qaScore, setQaScore] = useState(50);
  const [qaLevel, setQaLevel] = useState<string>('pass');
  const [disagree, setDisagree] = useState(false);
  const [comment, setComment] = useState('');

  const fetchData = useCallback(async () => {
    if (!batch) return;
    setLoading(true);
    try {
      const [s, r] = await Promise.all([
        getQualityBatchStats(batch.id),
        getQualityBatchRecords(batch.id),
      ]);
      setStats(s);
      setRecords(r);
    } catch {
      message.error('获取抽检数据失败');
    } finally {
      setLoading(false);
    }
  }, [batch]);

  const fetchSamples = useCallback(async () => {
    if (!batch) return;
    try {
      const data = await getPendingElements({
        ai_status: batch.filter_status,
        page_size: batch.sample_size,
      });
      setSamples(data.items ?? []);
    } catch {
      setSamples([]);
    }
  }, [batch]);

  useEffect(() => {
    if (visible && batch) {
      fetchData();
      fetchSamples();
    }
  }, [visible, batch, fetchData, fetchSamples]);

  // Refetch samples when switching to samples tab
  useEffect(() => {
    if (visible && activeTab === 'samples' && batch) {
      fetchSamples();
    }
  }, [visible, activeTab, batch, fetchSamples]);

  const handleQA = async () => {
    if (!selectedElement || !batch) return;
    try {
      await submitQARecord(batch.id, selectedElement.id, {
        qa_score: qaScore,
        qa_level: qaLevel,
        disagree,
        comment: comment || undefined,
      });
      message.success('抽检记录已提交');
      setSelectedElement(null);
      setQaScore(50);
      setQaLevel('pass');
      setDisagree(false);
      setComment('');
      fetchData();
      fetchSamples();
    } catch {
      message.error('提交失败');
    }
  };

  const progressPercent = batch && batch.sample_size > 0
    ? Math.round((batch.reviewed_count / batch.sample_size) * 100)
    : 0;

  return (
    <Modal
      title="抽检详情"
      open={visible}
      onCancel={onClose}
      width={900}
      footer={
        batch && batch.status === 'draft' ? (
          <Popconfirm title="确认开始抽检？" onConfirm={async () => { await startQualityBatch(batch.id); onClose(); message.success('抽检已开始'); }}>
            <Button type="primary" icon={<PlayCircleOutlined />}>开始抽检</Button>
          </Popconfirm>
        ) : batch && batch.status === 'in_progress' ? (
          <Popconfirm title="确认完成抽检？" onConfirm={async () => { await completeQualityBatch(batch.id); onClose(); message.success('抽检已完成'); }}>
            <Button type="primary" icon={<CheckCircleOutlined />}>完成抽检</Button>
          </Popconfirm>
        ) : null
      }
    >
      {batch && (
        <>
          <Descriptions bordered column={2} size="small" style={{ marginBottom: SPACING.base }}>
            <Descriptions.Item label="批次名称">{batch.name}</Descriptions.Item>
            <Descriptions.Item label="状态">{renderBatchStatus(batch.status)}</Descriptions.Item>
            <Descriptions.Item label="模式">{batch.mode === 'local_correction' ? '仅本地修正' : '连带用户判罚'}</Descriptions.Item>
            <Descriptions.Item label="样本量">{batch.sample_size}</Descriptions.Item>
            <Descriptions.Item label="已审核">{batch.reviewed_count}</Descriptions.Item>
            <Descriptions.Item label="筛选条件">{batch.filter_status}</Descriptions.Item>
          </Descriptions>

          {batch.status === 'in_progress' && (
            <div style={{ marginBottom: SPACING.base }}>
              <Progress percent={progressPercent} status={progressPercent === 100 ? 'success' : 'active'} />
            </div>
          )}

          <Tabs
            activeKey={activeTab}
            onChange={(key) => setActiveTab(key as typeof activeTab)}
            items={[
              {
                key: 'samples',
                label: '待抽检样本',
                children: (
                  <div style={{ maxHeight: 400, overflow: 'auto' }}>
                    {samples.map((elem) => {
                      const isReviewed = records.some((r) => r.element_id === elem.id);
                      const isReviewing = selectedElement?.id === elem.id;
                      return (
                        <div
                          key={elem.id}
                          style={{
                            padding: SPACING.base,
                            marginBottom: SPACING.sm,
                            background: isReviewing ? COLORS.bgContainerHover : isReviewed ? COLORS.success + '18' : COLORS.bgContainer,
                            border: isReviewing ? `1px solid ${COLORS.brandBlue}` : `1px solid ${COLORS.borderDefault}`,
                            borderRadius: RADIUS.md,
                            cursor: isReviewed ? 'default' : 'pointer',
                            transition: ANIM.fast,
                          }}
                          onClick={() => !isReviewed && setSelectedElement(elem)}
                          onMouseEnter={(e) => {
                            if (!isReviewed) (e.currentTarget as HTMLDivElement).style.borderColor = COLORS.borderHover;
                          }}
                          onMouseLeave={(e) => {
                            (e.currentTarget as HTMLDivElement).style.borderColor = isReviewing
                              ? COLORS.brandBlue
                              : COLORS.borderDefault;
                          }}
                        >
                          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                            <Space size={SPACING.sm}>
                              <Tag>{elem.element_kind}</Tag>
                              <span style={{ color: COLORS.textTertiary, fontSize: FONT.small }}>
                                风险分: {elem.ai_risk_score} ({riskLabel(elem.ai_risk_score)})
                              </span>
                            </Space>
                            {isReviewed ? (
                              <Tag color="success">已抽检</Tag>
                            ) : isReviewing ? (
                              <Tag color="processing">抽检中</Tag>
                            ) : (
                              <Tag color="default">待抽检</Tag>
                            )}
                          </div>
                        </div>
                      );
                    })}
                    {samples.length === 0 && (
                      <div style={{ textAlign: 'center', color: COLORS.textMuted, padding: SPACING.xxl }}>
                        暂无样本
                      </div>
                    )}
                  </div>
                ),
              },
              {
                key: 'records',
                label: '抽检记录',
                children: (
                  <Table<QualityAuditRecord>
                    columns={[
                      { title: '元素ID', dataIndex: 'element_id', width: TABLE.idColumnWidth, render: (v: string) => <Text copyable={{ text: v }} style={{ fontSize: FONT.caption }}>{v.slice(0, 8)}...</Text> },
                      { title: '原始分', dataIndex: 'original_score' },
                      { title: '抽检分', dataIndex: 'qa_score' },
                      { title: '等级', dataIndex: 'qa_level', render: (v: string) => <Tag color={v === 'pass' ? 'success' : 'error'}>{v}</Tag> },
                      { title: '分歧', dataIndex: 'disagree', render: (v: boolean) => <Tag color={v ? 'warning' : 'success'}>{v ? '是' : '否'}</Tag> },
                      { title: '备注', dataIndex: 'comment', ellipsis: true },
                      { title: '时间', dataIndex: 'created_at', render: formatDate },
                    ]}
                    dataSource={records}
                    rowKey="id"
                    loading={loading}
                    pagination={{ pageSize: 20, showTotal: (total) => `共 ${total} 条` }}
                  />
                ),
              },
              {
                key: 'stats',
                label: '统计',
                children: stats ? (
                  <Descriptions bordered column={2} size="small">
                    <Descriptions.Item label="总样本">{stats.total_samples}</Descriptions.Item>
                    <Descriptions.Item label="已审核">{stats.reviewed_count}</Descriptions.Item>
                    <Descriptions.Item label="分歧数">{stats.disagree_count}</Descriptions.Item>
                    <Descriptions.Item label="分歧率">{(stats.disagree_rate * 100).toFixed(1)}%</Descriptions.Item>
                    <Descriptions.Item label="平均抽检分">{stats.avg_qa_score.toFixed(1)}</Descriptions.Item>
                    <Descriptions.Item label="等级分布">
                      {Object.entries(stats.level_counts).map(([k, v]) => (
                        <Tag key={k} style={{ margin: 2 }}>{`${k}: ${v}`}</Tag>
                      ))}
                    </Descriptions.Item>
                  </Descriptions>
                ) : (
                  <div style={{ textAlign: 'center', color: COLORS.textMuted, padding: SPACING.xxl }}>
                    暂无统计数据
                  </div>
                ),
              },
            ]}
          />

          {/* QA Form */}
          {selectedElement && (
            <div style={{ marginTop: SPACING.base, padding: SPACING.base, background: COLORS.bgContainer, border: `1px solid ${COLORS.borderDefault}`, borderRadius: RADIUS.md }}>
              <Title level={5} style={{ color: COLORS.textPrimary, marginTop: 0 }}>
                抽检评分 — {selectedElement.element_kind}
              </Title>
              <Space direction="vertical" style={{ width: '100%' }} size={SPACING.base}>
                <div style={{ display: 'flex', gap: SPACING.base }}>
                  <div style={{ flex: 1 }}>
                    <span style={{ color: COLORS.textSecondary, fontSize: FONT.small }}>抽检分数</span>
                    <InputNumber
                      min={0}
                      max={100}
                      value={qaScore}
                      onChange={(v) => setQaScore(v ?? 50)}
                      style={{ width: '100%', marginTop: SPACING.xs }}
                      aria-label="抽检分数"
                    />
                  </div>
                  <div style={{ flex: 1 }}>
                    <span style={{ color: COLORS.textSecondary, fontSize: FONT.small }}>抽检等级</span>
                    <Select
                      value={qaLevel}
                      onChange={setQaLevel}
                      style={{ width: '100%', marginTop: SPACING.xs }}
                      options={[
                        { label: '通过', value: 'pass' },
                        { label: '轻微问题', value: 'minor_issue' },
                        { label: '严重问题', value: 'major_issue' },
                        { label: '致命问题', value: 'critical' },
                      ]}
                      aria-label="抽检等级"
                    />
                  </div>
                </div>
                <div style={{ display: 'flex', gap: SPACING.sm, alignItems: 'center' }}>
                  <label style={{ color: COLORS.textSecondary, fontSize: FONT.small }}>
                    <input type="checkbox" checked={disagree} onChange={(e) => setDisagree(e.target.checked)} style={{ marginRight: SPACING.xs }} />
                    与原审核结论存在分歧
                  </label>
                </div>
                <Input.TextArea
                  placeholder="抽检备注（可选）"
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                  rows={2}
                  aria-label="抽检备注"
                />
                <Space>
                  <Button onClick={() => { setSelectedElement(null); setQaScore(50); setQaLevel('pass'); setDisagree(false); setComment(''); }}>取消</Button>
                  <Button type="primary" onClick={handleQA}>提交抽检</Button>
                </Space>
              </Space>
            </div>
          )}
        </>
      )}
    </Modal>
  );
};

// ---------------------------------------------------------------------------
// QualityAudit Page
// ---------------------------------------------------------------------------

const QualityAuditPage: React.FC = () => {
  const [batches, setBatches] = useState<QualityAuditBatch[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [selectedBatch, setSelectedBatch] = useState<QualityAuditBatch | null>(null);
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [form] = Form.useForm<{ name: string; mode: string; filter_status: string; sample_size: number }>();

  const fetchBatches = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getQualityBatches({ page: 1, page_size: 50 });
      setBatches(data.items ?? []);
      setTotal(data.total ?? 0);
    } catch {
      message.error('获取抽检批次失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchBatches(); }, [fetchBatches]);

  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      await createQualityBatch(values);
      message.success('批次创建成功');
      setCreateModalVisible(false);
      fetchBatches();
    } catch {
      message.error('创建失败');
    }
  };

  const columns: ColumnsType<QualityAuditBatch> = [
    { title: '批次名称', dataIndex: 'name', key: 'name', ellipsis: true },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => renderBatchStatus(status),
    },
    { title: '模式', dataIndex: 'mode', key: 'mode', width: 140, render: (v: string) => v === 'local_correction' ? '仅本地修正' : '连带用户判罚' },
    { title: '样本量', dataIndex: 'sample_size', key: 'sample_size', width: 80 },
    { title: '已审核', dataIndex: 'reviewed_count', key: 'reviewed_count', width: 80 },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180, render: formatDate },
    {
      title: '操作',
      key: 'actions',
      width: TABLE.actionColumnWidth,
      render: (_: unknown, record: QualityAuditBatch) => (
        <Tooltip title="查看详情">
          <Button
            type="link"
            icon={<EyeOutlined />}
            onClick={() => { setSelectedBatch(record); setDetailVisible(true); }}
          />
        </Tooltip>
      ),
    },
  ];

  return (
    <AppLayout>
      <Content style={{ padding: SPACING.lg, background: COLORS.bgBase }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: SPACING.lg, flexWrap: 'wrap', gap: SPACING.sm }}>
          <Title level={4} style={{ color: COLORS.textPrimary, margin: 0 }}>
            质量抽检
          </Title>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => { form.resetFields(); setCreateModalVisible(true); }}>
            新建批次
          </Button>
        </div>

        <div style={{ marginBottom: SPACING.base, color: COLORS.textSecondary, fontSize: FONT.small }}>
          共 <Text style={{ color: COLORS.textPrimary }}>{total}</Text> 个抽检批次
        </div>

        <Table<QualityAuditBatch>
          columns={columns}
          dataSource={batches}
          rowKey="id"
          loading={loading}
          scroll={{ x: TABLE.scrollX }}
          pagination={TABLE.pagination}
        />

        <BatchDetailDrawer
          visible={detailVisible}
          batch={selectedBatch}
          onClose={() => { setDetailVisible(false); setSelectedBatch(null); }}
        />

        <Modal
          title="新建抽检批次"
          open={createModalVisible}
          onCancel={() => setCreateModalVisible(false)}
          onOk={handleCreate}
        >
          <Form form={form} layout="vertical" initialValues={{ mode: 'local_correction', sample_size: 20, filter_status: 'pending_human' }}>
            <Form.Item name="name" label="批次名称" rules={[{ required: true, message: '请输入批次名称' }]}>
              <Input placeholder="例如：2026年6月首批抽检" aria-label="批次名称" />
            </Form.Item>
            <Form.Item name="mode" label="修正模式" rules={[{ required: true }]}>
              <Select options={[
                { label: '仅本地修正', value: 'local_correction' },
                { label: '连带用户判罚', value: 'full_correction' },
              ]} aria-label="修正模式" />
            </Form.Item>
            <Form.Item name="filter_status" label="筛选条件" rules={[{ required: true }]}>
              <Select options={[
                { label: '待人审', value: 'pending_human' },
                { label: '机审通过', value: 'ai_passed' },
                { label: '机审拒绝', value: 'ai_rejected' },
              ]} aria-label="筛选条件" />
            </Form.Item>
            <Form.Item name="sample_size" label="样本量" rules={[{ required: true, message: '请输入样本量' }]}>
              <InputNumber min={1} max={1000} style={{ width: '100%' }} aria-label="样本量" />
            </Form.Item>
          </Form>
        </Modal>
      </Content>
    </AppLayout>
  );
};

export default QualityAuditPage;
