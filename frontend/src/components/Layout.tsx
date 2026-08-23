import React, { useState } from 'react';
import { Layout, Menu, Button, Typography } from 'antd';
import {
  DashboardOutlined,
  AuditOutlined,
  AlertOutlined,
  VideoCameraOutlined,
  SettingOutlined,
  CheckSquareOutlined,
  FileTextOutlined,
  RobotOutlined,
  LogoutOutlined,
  QuestionCircleOutlined,
} from '@ant-design/icons';
import { useNavigate, useLocation } from 'react-router-dom';
import useAuthStore from '@/stores/auth';
import { COLORS, FONT, SIZE, GUTTER, ANIMATION } from '@/utils/constants';
import HelpModal from '@/components/HelpModal';

const { Sider, Content } = Layout;
const { Text } = Typography;

export { Content };

const MENU_ITEMS = [
  { key: '/', icon: <DashboardOutlined />, label: '仪表盘' },
  { key: '/review', icon: <AuditOutlined />, label: '审核工作台' },
  { key: '/review/video', icon: <FileTextOutlined />, label: '短视频审核' },
  { key: '/appeals', icon: <AlertOutlined />, label: '申诉管理' },
  { key: '/live-wall', icon: <VideoCameraOutlined />, label: '直播电视墙' },
  { key: '/tenant-config', icon: <SettingOutlined />, label: '租户配置' },
  { key: '/quality-audit', icon: <CheckSquareOutlined />, label: '质量抽检' },
  { key: '/audit-log', icon: <FileTextOutlined />, label: '审核日志' },
  { key: '/ai-config', icon: <RobotOutlined />, label: 'AI 模型配置' },
];

const AppLayout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const [helpOpen, setHelpOpen] = useState(false);

  return (
    <Layout style={{ minHeight: '100vh', background: COLORS.bgBase }}>
      <Sider
        width={SIZE.sidebarWidth}
        style={{
          background: COLORS.bgContainer,
          borderRight: `1px solid ${COLORS.borderDefault}`,
          transition: `width ${ANIMATION.normal}`,
        }}
      >
        <div style={{ padding: `${GUTTER.md} ${GUTTER.sm}` }}>
          <Typography.Title level={3} style={{ color: COLORS.textPrimary, fontSize: FONT.h2, margin: '0 0 4px' }}>
            Photo Audit
          </Typography.Title>
          <Text style={{ color: COLORS.textSecondary, fontSize: FONT.caption }}>
            欢迎, {user?.username || '审核员'}
          </Text>
        </div>
        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          items={MENU_ITEMS}
          onSelect={({ key }) => navigate(key)}
          style={{ background: 'transparent' }}
        />
        <div style={{ padding: `${GUTTER.sm} ${GUTTER.sm}`, borderTop: `1px solid ${COLORS.borderDefault}` }}>
          <Button
            block
            type="default"
            icon={<LogoutOutlined />}
            onClick={logout}
            danger
            style={{
              borderColor: COLORS.danger,
              color: COLORS.danger,
              height: SIZE.buttonMd,
            }}
            aria-label="退出登录"
          >
            退出登录
          </Button>
        </div>
      </Sider>
      <Content style={{ background: COLORS.bgBase }}>
        {/* Global help button */}
        <div style={{
          position: 'fixed',
          bottom: GUTTER.lg,
          right: GUTTER.lg,
          zIndex: 1000,
        }}>
          <Button
            type="primary"
            shape="circle"
            icon={<QuestionCircleOutlined />}
            onClick={() => setHelpOpen(true)}
            size="large"
            aria-label="帮助"
            style={{
              width: 44,
              height: 44,
              fontSize: 20,
              boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
            }}
          />
        </div>
        {children}
        <HelpModal open={helpOpen} onClose={() => setHelpOpen(false)} />
      </Content>
    </Layout>
  );
};

export default AppLayout;
