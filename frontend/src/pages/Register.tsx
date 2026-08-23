import React, { useEffect, useState } from 'react';
import { Form, Input, Button, Card, message, Typography, Select, Radio } from 'antd';
import { UserOutlined, LockOutlined, MailOutlined } from '@ant-design/icons';
import { useNavigate, Link } from 'react-router-dom';
import { postRegister, getTenants, createTenant } from '@/services/api';
import useAuthStore, { type UserInfo } from '@/stores/auth';
import { COLORS, SPACING, FONT, RADIUS, SHADOW, ANIMATION } from '@/utils/constants';
import { PasswordStrength } from '@/components/PasswordStrength';

const { Title, Text } = Typography;

const RegisterPage: React.FC = () => {
  const navigate = useNavigate();
  const login = useAuthStore((s) => s.login);
  const [loading, setLoading] = useState(false);
  const [tenantMode, setTenantMode] = useState<'existing' | 'create'>('existing');
  const [tenants, setTenants] = useState<Array<{ id: string; name: string }>>([]);
  const [tenantsLoading, setTenantsLoading] = useState(false);
  const [password, setPassword] = useState('');

  useEffect(() => {
    setTenantsLoading(true);
    getTenants(1, 100)
      .then((res) => {
        setTenants(
          res.items.map((t) => ({ id: t.id, name: t.name }))
        );
      })
      .catch(() => setTenants([]))
      .finally(() => setTenantsLoading(false));
  }, []);

  const onFinish = async (values: {
    username: string;
    password: string;
    display_name?: string;
    email?: string;
    tenant_id?: string;
    new_tenant_name?: string;
    new_tenant_country?: string;
  }) => {
    setLoading(true);
    try {
      let tenantId = '';

      if (tenantMode === 'create') {
        const res = await createTenant(
          values.new_tenant_name || values.username,
          values.new_tenant_country || 'CN'
        );
        tenantId = res.id;
      } else {
        tenantId = values.tenant_id || '';
      }

      const result = await postRegister(
        values.username,
        values.password,
        values.display_name,
        values.email,
      );
      const userInfo: UserInfo = {
        id: result.user?.id || '',
        username: values.username,
        role: result.user?.role || 'reviewer',
        tenantId: tenantId,
      };
      login(result.token, userInfo);
      message.success('注册成功，已自动登录');
      navigate('/', { replace: true });
    } catch (err: unknown) {
      const msg =
        err instanceof Error
          ? err.message
          : '注册失败，请检查输入';
      message.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: '100vh',
        background: COLORS.bgBase,
      }}
    >
      <Card
        style={{
          width: 480,
          background: COLORS.bgContainer,
          borderColor: COLORS.borderDefault,
          borderRadius: RADIUS.lg,
          boxShadow: SHADOW.card,
          transition: `box-shadow ${ANIMATION.normal}`,
        }}
        onMouseEnter={(e) => { (e.currentTarget as HTMLDivElement).style.boxShadow = SHADOW.float; }}
        onMouseLeave={(e) => { (e.currentTarget as HTMLDivElement).style.boxShadow = SHADOW.card; }}
      >
        <div style={{ textAlign: 'center', marginBottom: SPACING.lg }}>
          <Title level={3} style={{ color: COLORS.textPrimary, fontSize: FONT.h3, margin: '0 0 8px' }}>
            注册账号
          </Title>
          <Text style={{ color: COLORS.textSecondary }}>创建你的审核平台账号</Text>
        </div>

        <Form.Item label="所属租户" style={{ marginBottom: SPACING.sm }}>
          <Radio.Group
            value={tenantMode}
            onChange={(e) => setTenantMode(e.target.value)}
            optionType="button"
            buttonStyle="solid"
          >
            <Radio.Button value="existing">加入现有租户</Radio.Button>
            <Radio.Button value="create">创建新租户</Radio.Button>
          </Radio.Group>
        </Form.Item>

        <Form
          layout="vertical"
          onFinish={onFinish}
          autoComplete="off"
        >
          {tenantMode === 'existing' ? (
            <Form.Item
              name="tenant_id"
              label="选择租户"
              rules={[{ required: true, message: '请选择租户' }]}
            >
              <Select
                loading={tenantsLoading}
                placeholder="选择租户"
                options={tenants.map((t) => ({ label: t.name, value: t.id }))}
                aria-label="选择租户"
              />
            </Form.Item>
          ) : (
            <>
              <Form.Item
                name="new_tenant_name"
                label="租户名称"
                rules={[{ required: true, message: '请输入租户名称' }]}
              >
                <Input placeholder="例如：某某公司" aria-label="租户名称" />
              </Form.Item>
              <Form.Item
                name="new_tenant_country"
                label="国家代码"
                rules={[
                  { required: true, message: '请输入国家代码' },
                  { pattern: /^[A-Z]{2}$/, message: '请输入2字母国家代码（如 CN、US）' },
                ]}
              >
                <Input placeholder="CN" maxLength={2} aria-label="国家代码" />
              </Form.Item>
            </>
          )}

          <Form.Item
            name="username"
            label="用户名"
            rules={[
              { required: true, message: '请输入用户名' },
              { min: 3, message: '用户名至少3个字符' },
            ]}
          >
            <Input
              prefix={<UserOutlined style={{ color: COLORS.textSecondary }} />}
              placeholder="请输入用户名"
              size="large"
              aria-label="用户名"
            />
          </Form.Item>

          <Form.Item
            name="display_name"
            label="显示名称"
            rules={[{ required: true, message: '请输入显示名称' }]}
          >
            <Input
              prefix={<UserOutlined style={{ color: COLORS.textSecondary }} />}
              placeholder="请输入显示名称"
              size="large"
              aria-label="显示名称"
            />
          </Form.Item>

          <Form.Item
            name="email"
            label="邮箱（可选）"
            rules={[
              { type: 'email', message: '请输入有效的邮箱地址', whitespace: true },
            ]}
          >
            <Input
              prefix={<MailOutlined style={{ color: COLORS.textSecondary }} />}
              placeholder="email@example.com"
              size="large"
              aria-label="邮箱"
            />
          </Form.Item>

          <Form.Item
            name="password"
            label="密码"
            rules={[
              { required: true, message: '请输入密码' },
              { min: 6, message: '密码至少6个字符' },
            ]}
          >
            <Input.Password
              prefix={<LockOutlined style={{ color: COLORS.textSecondary }} />}
              placeholder="请输入密码"
              size="large"
              aria-label="密码"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
            <PasswordStrength password={password} />
          </Form.Item>

          <Form.Item
            name="confirm"
            label="确认密码"
            dependencies={['password']}
            rules={[
              { required: true, message: '请确认密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('password') === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error('两次输入的密码不一致'));
                },
              }),
            ]}
          >
            <Input.Password
              prefix={<LockOutlined style={{ color: COLORS.textSecondary }} />}
              placeholder="请再次输入密码"
              size="large"
              aria-label="确认密码"
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: SPACING.base }}>
            <Button
              type="primary"
              htmlType="submit"
              loading={loading}
              block
              size="large"
            >
              注册
            </Button>
          </Form.Item>
        </Form>

        <div style={{ textAlign: 'center' }}>
          <Link to="/login" style={{ color: COLORS.brandBlue, fontSize: FONT.small }}>
            已有账号？登录
          </Link>
        </div>
      </Card>
    </div>
  );
};

export default RegisterPage;
