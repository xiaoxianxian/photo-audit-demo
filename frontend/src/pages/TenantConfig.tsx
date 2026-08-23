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
  Typography,
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  getAuditRules,
  createAuditRule,
  updateAuditRule,
  deleteAuditRule,
  getAuditLevels,
  createAuditLevel,
  updateAuditLevel,
  deleteAuditLevel,
  getCustomWords,
  createCustomWord,
  updateCustomWord,
  deleteCustomWord,
  type TenantAuditRule,
  type TenantAuditLevel,
  type TenantCustomWord,
} from '@/services/content-api';
import AppLayout, { Content } from '@/components/Layout';
import {
  COLORS,
  SPACING,
  TABLE,
} from '@/utils/constants';

const { Title } = Typography;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const ACTION_LABELS: Record<string, { color: string; label: string }> = {
  approve: { color: 'success', label: '自动通过' },
  reject: { color: 'error', label: '自动驳回' },
  flag: { color: 'warning', label: '标记审查' },
};

function renderAction(action: string) {
  const cfg = ACTION_LABELS[action] || { color: 'default', label: action };
  return <Tag color={cfg.color}>{cfg.label}</Tag>;
}

function formatDate(iso?: string) {
  if (!iso) return '-';
  return new Date(iso).toLocaleString('zh-CN');
}

// ---------------------------------------------------------------------------
// Rule Manager
// ---------------------------------------------------------------------------

const RuleManager: React.FC = () => {
  const [rules, setRules] = useState<TenantAuditRule[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [mode, setMode] = useState<'create' | 'edit'>('create');
  const [editingRule, setEditingRule] = useState<TenantAuditRule | undefined>();
  const [form] = Form.useForm<{ rule_name: string; rule_expression?: string; action: string; priority: number }>();

  const fetchRules = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getAuditRules(1, 100);
      setRules(data.items ?? []);
      setTotal(data.total ?? 0);
    } catch {
      message.error('获取审核规则失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchRules(); }, [fetchRules]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (mode === 'create') {
        await createAuditRule(values);
        message.success('规则创建成功');
      } else {
        await updateAuditRule(editingRule!.id, values);
        message.success('规则更新成功');
      }
      setModalVisible(false);
      fetchRules();
    } catch {
      message.error('操作失败');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteAuditRule(id);
      message.success('规则已删除');
      fetchRules();
    } catch {
      message.error('删除失败');
    }
  };

  const columns: ColumnsType<TenantAuditRule> = [
    { title: '规则名称', dataIndex: 'rule_name', key: 'rule_name', ellipsis: true },
    {
      title: '动作',
      dataIndex: 'action',
      key: 'action',
      width: 100,
      render: (action: string) => renderAction(action),
    },
    { title: '优先级', dataIndex: 'priority', key: 'priority', width: 80 },
    { title: '表达式', dataIndex: 'rule_expression', key: 'rule_expression', ellipsis: true, render: (v: string) => v || '-' },
    { title: '状态', dataIndex: 'status', key: 'status', width: 80, render: (v: number) => <Tag color={v === 1 ? 'success' : 'error'}>{v === 1 ? '启用' : '停用'}</Tag> },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180, render: formatDate },
    {
      title: '操作',
      key: 'actions',
      width: TABLE.actionColumnWidth,
      render: (_: unknown, record: TenantAuditRule) => (
        <Space size={SPACING.xs}>
          <Tooltip title="编辑规则">
            <Button type="text" icon={<EditOutlined />} onClick={() => { setMode('edit'); setEditingRule(record); form.setFieldsValue(record); setModalVisible(true); }} />
          </Tooltip>
          <Popconfirm title="确定删除？" onConfirm={() => handleDelete(record.id)}>
            <Tooltip title="删除规则">
              <Button type="text" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      <div style={{ marginBottom: SPACING.base, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span style={{ color: COLORS.textSecondary }}>共 {total} 条规则</span>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => { setMode('create'); setEditingRule(undefined); form.resetFields(); setModalVisible(true); }}>
          新建规则
        </Button>
      </div>
      <Table<TenantAuditRule> columns={columns} dataSource={rules} rowKey="id" loading={loading} scroll={{ x: TABLE.scrollX }} pagination={TABLE.pagination} />
      <Modal
        title={mode === 'create' ? '新建审核规则' : '编辑审核规则'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
      >
        <Form form={form} layout="vertical" initialValues={{ priority: 0 }}>
          <Form.Item name="rule_name" label="规则名称" rules={[{ required: true, message: '请输入规则名称' }]}>
            <Input placeholder="例如：禁止色情内容" aria-label="规则名称" />
          </Form.Item>
          <Form.Item name="rule_expression" label="规则表达式（可选）">
            <Input placeholder="例如：risk_score > 80 && risk_types contains porn" aria-label="规则表达式" />
          </Form.Item>
          <Form.Item name="action" label="动作" rules={[{ required: true, message: '请选择动作' }]}>
            <Select options={[
              { label: '自动通过', value: 'approve' },
              { label: '自动驳回', value: 'reject' },
              { label: '标记审查', value: 'flag' },
            ]} aria-label="动作" />
          </Form.Item>
          <Form.Item name="priority" label="优先级" rules={[{ required: true, message: '请输入优先级' }]}>
            <Input type="number" placeholder="数字越小优先级越高" aria-label="优先级" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

// ---------------------------------------------------------------------------
// Level Manager
// ---------------------------------------------------------------------------

const LevelManager: React.FC = () => {
  const [levels, setLevels] = useState<TenantAuditLevel[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [mode, setMode] = useState<'create' | 'edit'>('create');
  const [editingLevel, setEditingLevel] = useState<TenantAuditLevel | undefined>();
  const [form] = Form.useForm<{ level_code: string; level_name: string; description?: string }>();

  const fetchLevels = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getAuditLevels(1, 100);
      setLevels(data.items ?? []);
      setTotal(data.total ?? 0);
    } catch {
      message.error('获取判罚等级失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchLevels(); }, [fetchLevels]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (mode === 'create') {
        await createAuditLevel(values);
        message.success('等级创建成功');
      } else {
        await updateAuditLevel(editingLevel!.id, values);
        message.success('等级更新成功');
      }
      setModalVisible(false);
      fetchLevels();
    } catch {
      message.error('操作失败');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteAuditLevel(id);
      message.success('等级已删除');
      fetchLevels();
    } catch {
      message.error('删除失败');
    }
  };

  const columns: ColumnsType<TenantAuditLevel> = [
    { title: '等级代码', dataIndex: 'level_code', key: 'level_code' },
    { title: '等级名称', dataIndex: 'level_name', key: 'level_name' },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true, render: (v: string) => v || '-' },
    { title: '状态', dataIndex: 'status', key: 'status', width: 80, render: (v: number) => <Tag color={v === 1 ? 'success' : 'error'}>{v === 1 ? '启用' : '停用'}</Tag> },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180, render: formatDate },
    {
      title: '操作',
      key: 'actions',
      width: TABLE.actionColumnWidth,
      render: (_: unknown, record: TenantAuditLevel) => (
        <Space size={SPACING.xs}>
          <Tooltip title="编辑等级">
            <Button type="text" icon={<EditOutlined />} onClick={() => { setMode('edit'); setEditingLevel(record); form.setFieldsValue(record); setModalVisible(true); }} />
          </Tooltip>
          <Popconfirm title="确定删除？" onConfirm={() => handleDelete(record.id)}>
            <Tooltip title="删除等级">
              <Button type="text" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      <div style={{ marginBottom: SPACING.base, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span style={{ color: COLORS.textSecondary }}>共 {total} 个等级</span>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => { setMode('create'); setEditingLevel(undefined); form.resetFields(); setModalVisible(true); }}>
          新建等级
        </Button>
      </div>
      <Table<TenantAuditLevel> columns={columns} dataSource={levels} rowKey="id" loading={loading} scroll={{ x: TABLE.scrollX }} pagination={TABLE.pagination} />
      <Modal
        title={mode === 'create' ? '新建判罚等级' : '编辑判罚等级'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="level_code" label="等级代码" rules={[{ required: true, message: '请输入等级代码' }]}>
            <Input placeholder="例如：warning, freeze, ban" aria-label="等级代码" />
          </Form.Item>
          <Form.Item name="level_name" label="等级名称" rules={[{ required: true, message: '请输入等级名称' }]}>
            <Input placeholder="例如：警告、冻结、封禁" aria-label="等级名称" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea placeholder="可选" rows={2} aria-label="描述" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

// ---------------------------------------------------------------------------
// Word Manager
// ---------------------------------------------------------------------------

const WordManager: React.FC = () => {
  const [words, setWords] = useState<TenantCustomWord[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [mode, setMode] = useState<'create' | 'edit'>('create');
  const [editingWord, setEditingWord] = useState<TenantCustomWord | undefined>();
  const [form] = Form.useForm<{ word: string; category?: string }>();

  const fetchWords = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getCustomWords(1, 100);
      setWords(data.items ?? []);
      setTotal(data.total ?? 0);
    } catch {
      message.error('获取敏感词失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchWords(); }, [fetchWords]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (mode === 'create') {
        await createCustomWord(values);
        message.success('词添加成功');
      } else {
        await updateCustomWord(editingWord!.id, values);
        message.success('词更新成功');
      }
      setModalVisible(false);
      fetchWords();
    } catch {
      message.error('操作失败');
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteCustomWord(id);
      message.success('词已删除');
      fetchWords();
    } catch {
      message.error('删除失败');
    }
  };

  const columns: ColumnsType<TenantCustomWord> = [
    { title: '词汇', dataIndex: 'word', key: 'word', ellipsis: true },
    { title: '分类', dataIndex: 'category', key: 'category', width: 120, render: (v: string) => v || '-' },
    { title: '状态', dataIndex: 'status', key: 'status', width: 80, render: (v: number) => <Tag color={v === 1 ? 'success' : 'error'}>{v === 1 ? '启用' : '停用'}</Tag> },
    { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180, render: formatDate },
    {
      title: '操作',
      key: 'actions',
      width: TABLE.actionColumnWidth,
      render: (_: unknown, record: TenantCustomWord) => (
        <Space size={SPACING.xs}>
          <Tooltip title="编辑词汇">
            <Button type="text" icon={<EditOutlined />} onClick={() => { setMode('edit'); setEditingWord(record); form.setFieldsValue(record); setModalVisible(true); }} />
          </Tooltip>
          <Popconfirm title="确定删除？" onConfirm={() => handleDelete(record.id)}>
            <Tooltip title="删除词汇">
              <Button type="text" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      <div style={{ marginBottom: SPACING.base, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <span style={{ color: COLORS.textSecondary }}>共 {total} 个敏感词</span>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => { setMode('create'); setEditingWord(undefined); form.resetFields(); setModalVisible(true); }}>
          添加词汇
        </Button>
      </div>
      <Table<TenantCustomWord> columns={columns} dataSource={words} rowKey="id" loading={loading} scroll={{ x: TABLE.scrollX }} pagination={TABLE.pagination} />
      <Modal
        title={mode === 'create' ? '添加敏感词' : '编辑敏感词'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={handleSubmit}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="word" label="词汇" rules={[{ required: true, message: '请输入词汇' }]}>
            <Input placeholder="例如：违规关键词" aria-label="词汇" />
          </Form.Item>
          <Form.Item name="category" label="分类">
            <Input placeholder="例如：政治、色情、广告" aria-label="分类" />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

// ---------------------------------------------------------------------------
// TenantConfig Page
// ---------------------------------------------------------------------------

const TenantConfigPage: React.FC = () => (
  <AppLayout>
    <Content style={{ padding: SPACING.lg, background: COLORS.bgBase }}>
      <div style={{ marginBottom: SPACING.lg }}>
        <Title level={4} style={{ color: COLORS.textPrimary, margin: 0 }}>
          租户配置
        </Title>
      </div>
      <Tabs
        items={[
          { key: 'rules', label: '审核规则', children: <RuleManager /> },
          { key: 'levels', label: '判罚等级', children: <LevelManager /> },
          { key: 'words', label: '敏感词库', children: <WordManager /> },
        ]}
      />
    </Content>
  </AppLayout>
);

export default TenantConfigPage;
