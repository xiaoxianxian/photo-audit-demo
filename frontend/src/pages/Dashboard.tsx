import React, { useCallback, useEffect, useMemo, useState } from 'react';
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
  Empty,
  Row,
  Col,
  Statistic,
  Card,
  Tabs,
  Tooltip,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  UserAddOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import {
  getTenants,
  createTenant,
  updateTenant,
  deleteTenant,
  getTeams,
  createTeam,
  addTeamMember,
} from '@/services/api';
import {
  getDashboardStats,
  getReviewerPerformance,
  getDailyTrend,
  type DashboardStats,
  type ReviewerPerformance,
  type DailyTrendPoint,
} from '@/services/content-api';
import useAuthStore from '@/stores/auth';
import type { UserInfo } from '@/stores/auth';
import AppLayout, { Content } from '@/components/Layout';
import {
  COLORS,
  SPACING,
  FONT,
  TABLE,
  RADIUS,
  ANIMATION as ANIM,
} from '@/utils/constants';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface Tenant {
  id: string;
  name: string;
  country_code: string;
  status: string;
  created_at: string;
}

interface Team {
  id: string;
  name: string;
  leader_id: string;
  member_count: number;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const STATUS_MAP: Record<string, { color: string; label: string }> = {
  active: { color: 'success', label: '活跃' },
  inactive: { color: 'error', label: '停用' },
  pending: { color: 'warning', label: '待激活' },
};

function renderStatus(status: string) {
  const cfg = STATUS_MAP[status] ?? STATUS_MAP.pending;
  return <Tag color={cfg.color}>{cfg.label}</Tag>;
}

function formatDate(iso?: string) {
  if (!iso) return '-';
  return new Date(iso).toLocaleString('zh-CN');
}

// ---------------------------------------------------------------------------
// TenantTable
// ---------------------------------------------------------------------------

const TenantTable: React.FC<{
  tenants: Tenant[];
  loading: boolean;
  onCreateModal: () => void;
  onEditModal: (item: Tenant) => void;
  onDelete: (id: string) => void;
}> = ({ tenants, loading, onCreateModal, onEditModal, onDelete }) => {
  const columns: ColumnsType<Tenant> = useMemo(
    () => [
      { title: 'ID', dataIndex: 'id', key: 'id', width: TABLE.idColumnWidth },
      { title: '名称', dataIndex: 'name', key: 'name', sorter: true },
      { title: '国家/地区', dataIndex: 'country_code', key: 'country_code' },
      { title: '状态', dataIndex: 'status', key: 'status', render: renderStatus },
      { title: '创建时间', dataIndex: 'created_at', key: 'created_at', render: formatDate },
      {
        title: '操作',
        key: 'actions',
        width: TABLE.actionColumnWidth,
        render: (_, record) => (
          <Space size={SPACING.sm}>
            <Tooltip title="编辑租户">
              <Button
                type="text"
                icon={<EditOutlined />}
                onClick={() => onEditModal(record)}
              />
            </Tooltip>
            <Popconfirm title="确定删除该租户？" onConfirm={() => onDelete(record.id)}>
              <Tooltip title="删除租户">
                <Button type="text" danger icon={<DeleteOutlined />} />
              </Tooltip>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [onEditModal, onDelete]
  );

  return (
    <>
      <div style={{ marginBottom: SPACING.base }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={onCreateModal}>
          新建租户
        </Button>
      </div>
      <Table<Tenant>
        columns={columns}
        dataSource={tenants}
        rowKey="id"
        loading={loading}
        scroll={{ x: TABLE.scrollX }}
        pagination={TABLE.pagination}
        locale={{
          emptyText: (
            <Empty description="暂无租户数据">
              <Button type="primary" icon={<PlusOutlined />} onClick={onCreateModal}>
                新建租户
              </Button>
            </Empty>
          ),
        }}
      />
    </>
  );
};

// ---------------------------------------------------------------------------
// TeamTable
// ---------------------------------------------------------------------------

const TeamTable: React.FC<{
  teams: Team[];
  leaders: UserInfo[];
  loading: boolean;
  onCreateModal: () => void;
  onAddMember: (teamId: string) => void;
}> = ({ teams, leaders, loading, onCreateModal, onAddMember }) => {
  const columns: ColumnsType<Team> = useMemo(
    () => [
      { title: 'ID', dataIndex: 'id', key: 'id', width: TABLE.idColumnWidth },
      { title: '名称', dataIndex: 'name', key: 'name', sorter: true },
      {
        title: '负责人',
        dataIndex: 'leader_id',
        key: 'leader_id',
        render: (leaderId: string) => {
          const leader = leaders.find((l) => l.id === leaderId);
          return leader?.username ?? leaderId;
        },
      },
      { title: '成员数', dataIndex: 'member_count', key: 'member_count' },
      {
        title: '操作',
        key: 'actions',
        width: TABLE.actionColumnWidth,
        render: (_, record) => (
          <Space size={SPACING.sm}>
            <Tooltip title="添加成员">
              <Button
                type="text"
                icon={<UserAddOutlined />}
                onClick={() => onAddMember(record.id)}
              />
            </Tooltip>
          </Space>
        ),
      },
    ],
    [leaders, onAddMember]
  );

  return (
    <>
      <div style={{ marginBottom: SPACING.base }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={onCreateModal}>
          新建团队
        </Button>
      </div>
      <Table<Team>
        columns={columns}
        dataSource={teams}
        rowKey="id"
        loading={loading}
        scroll={{ x: TABLE.scrollX }}
        pagination={TABLE.pagination}
        locale={{
          emptyText: (
            <Empty description="暂无团队数据">
              <Button type="primary" icon={<PlusOutlined />} onClick={onCreateModal}>
                新建团队
              </Button>
            </Empty>
          ),
        }}
      />
    </>
  );
};

// ---------------------------------------------------------------------------
// TenantModal
// ---------------------------------------------------------------------------

const TenantModal: React.FC<{
  visible: boolean;
  mode: 'create' | 'edit';
  initial?: Tenant;
  onCancel: () => void;
  onSubmit: (payload: { name: string; country_code: string }) => void;
}> = ({ visible, mode, initial, onCancel, onSubmit }) => {
  const [form] = Form.useForm<Tenant>();

  useEffect(() => {
    if (visible) {
      form.setFieldsValue(initial ?? {});
    } else {
      form.resetFields();
    }
  }, [visible, initial, form]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      onSubmit(values);
    } catch {
      // validation failed
    }
  };

  return (
    <Modal
      title={mode === 'create' ? '新建租户' : '编辑租户'}
      open={visible}
      onCancel={onCancel}
      onOk={handleSubmit}
    >
      <Form<Tenant> form={form} layout="vertical">
        <Form.Item
          name="name"
          label="租户名称"
          rules={[{ required: true, message: '请输入租户名称' }]}
        >
          <Input placeholder="例如：ACME Corp" aria-label="租户名称" />
        </Form.Item>
        <Form.Item
          name="country_code"
          label="国家/地区代码"
          rules={[{ required: true, message: '请输入国家/地区代码' }]}
        >
          <Input placeholder="例如：CN" maxLength={2} aria-label="国家/地区代码" />
        </Form.Item>
      </Form>
    </Modal>
  );
};

// ---------------------------------------------------------------------------
// TeamModal
// ---------------------------------------------------------------------------

const TeamModal: React.FC<{
  visible: boolean;
  mode: 'create' | 'add-member';
  tenantId?: string;
  onCancel: () => void;
  onSubmit: (payload: { name?: string; leader_id?: string; user_id?: string; role?: string }) => void;
  leaders: UserInfo[];
}> = ({ visible, mode, onCancel, onSubmit, leaders }) => {
  const [form] = Form.useForm<{ name: string; leader_id: string }>();

  useEffect(() => {
    if (visible) form.resetFields();
  }, [visible, form]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (mode === 'create') {
        onSubmit(values);
      } else {
        onSubmit(values as unknown as { user_id: string; role: string });
      }
    } catch {
      // validation failed
    }
  };

  const selectOptions = useMemo(
    () => leaders.map((u) => ({ label: u.username, value: u.id })),
    [leaders]
  );

  if (mode === 'add-member') {
    return (
      <Modal
        title="添加团队成员"
        open={visible}
        onCancel={onCancel}
        onOk={handleSubmit}
      >
        <Form<{ user_id: string; role: string }> layout="vertical">
          <Form.Item
            name="user_id"
            label="选择用户"
            rules={[{ required: true, message: '请选择用户' }]}
          >
            <Select
              placeholder="搜索用户…"
              options={selectOptions}
              aria-label="选择用户"
            />
          </Form.Item>
          <Form.Item
            name="role"
            label="角色"
            rules={[{ required: true, message: '请选择角色' }]}
          >
            <Select
              placeholder="选择角色"
              options={[
                { label: '管理员', value: 'admin' },
                { label: '编辑者', value: 'editor' },
                { label: '查看者', value: 'viewer' },
              ]}
              aria-label="角色"
            />
          </Form.Item>
        </Form>
      </Modal>
    );
  }

  return (
    <Modal
      title="新建团队"
      open={visible}
      onCancel={onCancel}
      onOk={handleSubmit}
    >
      <Form<{ name: string; leader_id: string }> form={form} layout="vertical">
        <Form.Item
          name="name"
          label="团队名称"
          rules={[{ required: true, message: '请输入团队名称' }]}
        >
          <Input placeholder="例如：内容审核组" aria-label="团队名称" />
        </Form.Item>
        <Form.Item
          name="leader_id"
          label="负责人"
          rules={[{ required: true, message: '请选择负责人' }]}
        >
          <Select
            placeholder="搜索用户…"
            options={selectOptions}
            aria-label="负责人"
          />
        </Form.Item>
      </Form>
    </Modal>
  );
};

// ---------------------------------------------------------------------------
// Dashboard Stats
// ---------------------------------------------------------------------------

const DashboardStatsCard: React.FC<{ stats: DashboardStats; loading: boolean }> = ({ stats, loading }) => (
  <Card
    title="业务看板"
    loading={loading}
    style={{ marginBottom: SPACING.lg }}
    styles={{ body: { padding: `${SPACING.base} ${SPACING.lg}` } }}
  >
    <Row gutter={[SPACING.base, SPACING.base]}>
      <Col xs={12} sm={8} md={4}>
        <Statistic title="今日已审" value={stats.today_reviewed} suffix="张" />
      </Col>
      <Col xs={12} sm={8} md={4}>
        <Statistic title="通过率" value={stats.approval_rate} precision={1} suffix="%" />
      </Col>
      <Col xs={12} sm={8} md={4}>
        <Statistic title="驳回率" value={stats.rejection_rate} precision={1} suffix="%" />
      </Col>
      <Col xs={12} sm={8} md={4}>
        <Statistic title="平均风险分" value={stats.avg_risk_score} precision={1} />
      </Col>
      <Col xs={12} sm={8} md={4}>
        <Statistic title="待审元素" value={stats.pending_elements} suffix="个" />
      </Col>
      <Col xs={12} sm={8} md={4}>
        <Statistic title="分歧数" value={stats.conflict_count} suffix="个" />
      </Col>
    </Row>
  </Card>
);

// ---------------------------------------------------------------------------
// Reviewer Performance
// ---------------------------------------------------------------------------

const ReviewerPerformanceTable: React.FC<{
  performers: ReviewerPerformance[];
  loading: boolean;
}> = ({ performers, loading }) => {
  const columns: ColumnsType<ReviewerPerformance> = useMemo(
    () => [
      { title: '审核员', dataIndex: 'reviewer_name', key: 'reviewer_name' },
      { title: '审核量', dataIndex: 'total_reviews', key: 'total_reviews' },
      { title: '通过', dataIndex: 'approved', key: 'approved' },
      { title: '驳回', dataIndex: 'rejected', key: 'rejected' },
      { title: '申诉', dataIndex: 'appeals', key: 'appeals' },
      { title: '准确率', dataIndex: 'accuracy', key: 'accuracy', render: (val: number) => `${val}%` },
      { title: '平均耗时', dataIndex: 'avg_time_sec', key: 'avg_time_sec', render: (val: number) => `${val}s` },
    ],
    []
  );

  return (
    <Card title="审核员绩效" style={{ marginTop: SPACING.lg }}>
      <Table
        columns={columns}
        dataSource={performers}
        rowKey="reviewer_id"
        loading={loading}
        pagination={TABLE.pagination}
      />
    </Card>
  );
};

// ---------------------------------------------------------------------------
// Main Dashboard
// ---------------------------------------------------------------------------

const Dashboard: React.FC = () => {
  const [activeTab, setActiveTab] = useState<string>('tenant');

  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [tenantsLoading, setTenantsLoading] = useState(false);
  const [tenantModalVisible, setTenantModalVisible] = useState(false);
  const [tenantEditMode, setTenantEditMode] = useState<'create' | 'edit'>('create');
  const [editingTenant, setEditingTenant] = useState<Tenant | undefined>();

  const [teams, setTeams] = useState<Team[]>([]);
  const [teamsLoading, setTeamsLoading] = useState(false);
  const [teamModalVisible, setTeamModalVisible] = useState(false);
  const [teamMode, setTeamMode] = useState<'create' | 'add-member'>('create');
  const [selectedTeamId, setSelectedTeamId] = useState<string>();

  const [dashboardStats, setDashboardStats] = useState<DashboardStats>({
    total_reviewed: 0, today_reviewed: 0, approval_rate: 0, rejection_rate: 0,
    avg_risk_score: 0, appeal_count: 0, active_streams: 0, pending_elements: 0,
    conflict_count: 0, accuracy_rate: 0,
  });
  const [statsLoading, setStatsLoading] = useState(false);

  const [dailyTrend, setDailyTrend] = useState<DailyTrendPoint[]>([]);
  const [trendLoading, setTrendLoading] = useState(false);

  const [reviewerPerformances, setReviewerPerformances] = useState<ReviewerPerformance[]>([]);
  const [perfLoading, setPerfLoading] = useState(false);

  const currentUser = useAuthStore((s) => s.user);
  const leaders: UserInfo[] = useMemo(
    () => [{
      id: currentUser?.id ?? '',
      username: currentUser?.username ?? '',
      role: currentUser?.role ?? 'admin',
      tenantId: currentUser?.tenantId ?? '',
    }],
    [currentUser]
  );

  const fetchTenants = useCallback(async () => {
    setTenantsLoading(true);
    try {
      const data = await getTenants(1, 100);
      setTenants(data.items ?? []);
    } catch {
      message.error('获取租户列表失败');
    } finally {
      setTenantsLoading(false);
    }
  }, []);

  const fetchTeams = useCallback(async () => {
    setTeamsLoading(true);
    try {
      const data = await getTeams(currentUser?.tenantId ?? '');
      setTeams(Array.isArray(data) ? data : []);
    } catch {
      message.error('获取团队列表失败');
    } finally {
      setTeamsLoading(false);
    }
  }, []);

  const fetchDashboardStats = useCallback(async () => {
    setStatsLoading(true);
    try {
      const data = await getDashboardStats();
      setDashboardStats(data);
    } catch {
      message.error('获取业务看板数据失败');
    } finally {
      setStatsLoading(false);
    }
  }, []);

  const fetchDailyTrend = useCallback(async () => {
    setTrendLoading(true);
    try {
      const data = await getDailyTrend();
      setDailyTrend(data);
    } catch {
      message.error('获取趋势数据失败');
    } finally {
      setTrendLoading(false);
    }
  }, []);

  const fetchReviewerPerformance = useCallback(async () => {
    setPerfLoading(true);
    try {
      const data = await getReviewerPerformance(1, 20);
      setReviewerPerformances(data.items ?? []);
    } catch {
      message.error('获取审核员绩效数据失败');
      setReviewerPerformances([]);
    } finally {
      setPerfLoading(false);
    }
  }, []);

  useEffect(() => {
    Promise.allSettled([
      fetchTenants(),
      fetchTeams(),
      fetchDashboardStats(),
      fetchReviewerPerformance(),
      fetchDailyTrend(),
    ]);
  }, [fetchTenants, fetchTeams, fetchDashboardStats, fetchReviewerPerformance, fetchDailyTrend]);

  const handleTenantSubmit = async (payload: { name: string; country_code: string }) => {
    try {
      if (tenantEditMode === 'create') {
        await createTenant(payload.name, payload.country_code);
        message.success('租户创建成功');
      } else if (editingTenant) {
        await updateTenant(editingTenant.id, payload);
        message.success('租户更新成功');
      }
      setTenantModalVisible(false);
      fetchTenants();
    } catch {
      message.error('操作失败');
    }
  };

  const handleTeamSubmit = async (payload: { name?: string; leader_id?: string; user_id?: string; role?: string }) => {
    try {
      if (teamMode === 'create' && payload.name && payload.leader_id) {
        await createTeam(payload.name, payload.leader_id);
        message.success('团队创建成功');
      } else if (teamMode === 'add-member' && selectedTeamId && payload.user_id) {
        await addTeamMember(selectedTeamId, payload.user_id, payload.role ?? 'editor');
        message.success('成员添加成功');
      }
      setTeamModalVisible(false);
      setSelectedTeamId(undefined);
      fetchTeams();
    } catch {
      message.error('操作失败');
    }
  };

  const handleDeleteTenant = async (id: string) => {
    try {
      await deleteTenant(id);
      message.success('租户已删除');
      fetchTenants();
    } catch {
      message.error('删除租户失败');
    }
  };

  const handleOpenCreateTenant = () => {
    setTenantEditMode('create');
    setEditingTenant(undefined);
    setTenantModalVisible(true);
  };

  const handleOpenEditTenant = (item: Tenant) => {
    setTenantEditMode('edit');
    setEditingTenant(item);
    setTenantModalVisible(true);
  };

  const handleOpenCreateTeam = () => {
    setTeamMode('create');
    setTeamModalVisible(true);
  };

  const handleOpenAddMember = (teamId: string) => {
    setSelectedTeamId(teamId);
    setTeamMode('add-member');
    setTeamModalVisible(true);
  };

  return (
    <AppLayout>
      <Content style={{ padding: SPACING.lg, background: COLORS.bgBase }}>
        {/* Pending review reminder */}
        {dashboardStats.pending_elements > 0 && (
          <div style={{
            marginBottom: SPACING.lg,
            padding: `${SPACING.sm} ${SPACING.base}`,
            background: COLORS.bgContainerHover,
            borderRadius: RADIUS.md,
            display: 'flex',
            alignItems: 'center',
            gap: SPACING.sm,
          }}>
            <WarningOutlined style={{ fontSize: 20, color: COLORS.warning }} />
            <span style={{ color: COLORS.textTertiary, fontSize: FONT.body }}>
              你还有{' '}
              <strong style={{ color: COLORS.textPrimary }}>
                {dashboardStats.pending_elements}
              </strong>{' '}
              个元素待审核
            </span>
          </div>
        )}

        {/* Business Dashboard Stats */}
        <DashboardStatsCard stats={dashboardStats} loading={statsLoading} />

        {/* Daily Trend Chart */}
        <Card title="近 7 天审核趋势" loading={trendLoading} style={{ marginBottom: SPACING.lg }}>
          {dailyTrend.length > 0 ? (
            <div style={{ padding: `${SPACING.sm} 0` }}>
              {dailyTrend.map((point) => {
                const maxTotal = Math.max(...dailyTrend.map((p) => p.total_reviewed), 1);
                const barHeight = Math.max((point.total_reviewed / maxTotal) * 120, 4);
                return (
                  <div key={point.date} style={{
                    display: 'flex',
                    alignItems: 'flex-end',
                    gap: SPACING.base,
                    marginBottom: SPACING.base,
                  }}>
                    <div style={{
                      width: 48,
                      textAlign: 'right',
                      fontSize: FONT.caption,
                      color: COLORS.textSecondary,
                      flexShrink: 0,
                    }}>
                      {point.date}
                    </div>
                    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: SPACING.xs }}>
                      <div
                        style={{
                          height: barHeight,
                          background: `linear-gradient(to top, ${COLORS.brandBlue}, #69b1ff)`,
                          borderRadius: RADIUS.sm,
                          transition: `height ${ANIM.normal}`,
                        }}
                        title={`审核 ${point.total_reviewed} 件 | 通过 ${point.approval_rate}% | 驳回 ${point.rejection_rate}%`}
                      />
                      <div style={{ display: 'flex', gap: SPACING.sm, fontSize: FONT.caption }}>
                        <span style={{ color: COLORS.success }}>
                          ✓ {point.approval_rate}%
                        </span>
                        <span style={{ color: COLORS.danger }}>
                          ✗ {point.rejection_rate}%
                        </span>
                      </div>
                    </div>
                    <div style={{
                      width: 36,
                      textAlign: 'center',
                      fontSize: FONT.small,
                      color: COLORS.textTertiary,
                      flexShrink: 0,
                    }}>
                      {point.total_reviewed}
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <Empty description="暂无趋势数据" />
          )}
        </Card>

        {/* ---- Tenant / Team Tabs ---- */}
        <Tabs
          activeKey={activeTab}
          onChange={(key) => setActiveTab(key)}
          items={[
            {
              key: 'tenant',
              label: '租户管理',
              children: (
                <TenantTable
                  tenants={tenants}
                  loading={tenantsLoading}
                  onCreateModal={handleOpenCreateTenant}
                  onEditModal={handleOpenEditTenant}
                  onDelete={handleDeleteTenant}
                />
              ),
            },
            {
              key: 'team',
              label: '团队管理',
              children: (
                <TeamTable
                  teams={teams}
                  leaders={leaders}
                  loading={teamsLoading}
                  onCreateModal={handleOpenCreateTeam}
                  onAddMember={handleOpenAddMember}
                />
              ),
            },
          ]}
        />

        {/* Reviewer Performance */}
        <ReviewerPerformanceTable performers={reviewerPerformances} loading={perfLoading} />
      </Content>

      {/* -- Modals -- */}
      <TenantModal
        visible={tenantModalVisible}
        mode={tenantEditMode}
        initial={editingTenant}
        onCancel={() => {
          setTenantModalVisible(false);
          setEditingTenant(undefined);
        }}
        onSubmit={handleTenantSubmit}
      />
      <TeamModal
        visible={teamModalVisible}
        mode={teamMode}
        onCancel={() => {
          setTeamModalVisible(false);
          setSelectedTeamId(undefined);
        }}
        onSubmit={handleTeamSubmit}
        leaders={leaders}
      />
    </AppLayout>
  );
};

export default Dashboard;
