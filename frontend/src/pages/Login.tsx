import React, { useState } from 'react';
import { Form, Input, Button, Card, message, Typography } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useNavigate, Link } from 'react-router-dom';
import { postLogin } from '@/services/api';
import useAuthStore, { type UserInfo } from '@/stores/auth';
import { COLORS, SPACING, RADIUS, SHADOW, ANIMATION, SIZE } from '@/utils/constants';

const { Title } = Typography;

const LoginPage: React.FC = () => {
  const navigate = useNavigate();
  const login = useAuthStore((s) => s.login);
  const [loading, setLoading] = useState(false);

  const onFinish = async (values: { username: string; password: string }) => {
    setLoading(true);
    try {
      const result = await postLogin(values.username, values.password);
      const userInfo: UserInfo = {
        id: result.user?.id || '',
        username: values.username,
        role: result.user?.role || 'viewer',
        tenantId: result.user?.tenant_id || '',
      };
      login(result.token, userInfo);
      message.success('登录成功');
      navigate('/', { replace: true });
    } catch (err: unknown) {
      const msg =
        err instanceof Error
          ? err.message
          : '登录失败，请检查用户名和密码';
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
          width: 400,
          background: COLORS.bgContainer,
          borderColor: COLORS.borderDefault,
          borderRadius: RADIUS.lg,
          boxShadow: SHADOW.card,
          transition: `box-shadow ${ANIMATION.normal}`,
        }}
        hoverable
      >
        <div style={{ textAlign: 'center', marginBottom: SPACING.lg }}>
          <Title level={3} className="pa-h2" style={{ color: COLORS.textPrimary, margin: '0 0 8px' }}>
            Photo Audit Platform
          </Title>
        </div>

        <Form
          layout="vertical"
          onFinish={onFinish}
          autoComplete="off"
        >
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
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: SPACING.base }}>
            <Button
              type="primary"
              htmlType="submit"
              loading={loading}
              block
              size="large"
              style={{ height: SIZE.buttonLg }}
            >
              登录
            </Button>
          </Form.Item>
        </Form>

        <div style={{ textAlign: 'center' }}>
          <Link to="/register" style={{ color: COLORS.brandBlue }} className="pa-body">
            还没有账号？注册
          </Link>
        </div>
      </Card>
    </div>
  );
};

export default LoginPage;
