import React from 'react';
import { Modal, Button, Tag, Typography } from 'antd';
import { QuestionCircleOutlined } from '@ant-design/icons';
import { COLORS, SPACING, FONT } from '@/utils/constants';

const { Text, Title } = Typography;

interface HelpModalProps {
  open: boolean;
  onClose: () => void;
}

const HelpModal: React.FC<HelpModalProps> = ({ open, onClose }) => (
  <Modal
    title={
      <span>
        <QuestionCircleOutlined style={{ marginRight: SPACING.xs }} />
        帮助与快捷键
      </span>
    }
    open={open}
    onCancel={onClose}
    footer={
      <Button type="primary" onClick={onClose}>
        知道了
      </Button>
    }
    width={560}
  >
    <div style={{ marginBottom: SPACING.lg }}>
      <Title level={5} style={{ marginBottom: SPACING.sm }}>
        审核工作台快捷键
      </Title>
      <div style={{ display: 'flex', flexDirection: 'column', gap: SPACING.sm }}>
        <ShortcutRow key="Enter" label="Enter / Space" action="通过当前元素" />
        <ShortcutRow key="Escape" label="Esc" action="打回当前元素" />
        <ShortcutRow key="ArrowLeft" label="←" action="上一个元素" />
        <ShortcutRow key="ArrowRight" label="→" action="下一个元素" />
        <ShortcutRow key="ShiftEnter" label="Shift + Enter" action="批量通过已选元素" />
      </div>
    </div>

    <div style={{ marginBottom: SPACING.lg }}>
      <Title level={5} style={{ marginBottom: SPACING.sm }}>
        页面导航
      </Title>
      <div style={{ display: 'flex', flexDirection: 'column', gap: SPACING.sm }}>
        <NavItem href="/review" label="审核工作台" desc="供稿审核" />
        <NavItem href="/review/video" label="短视频审核" desc="视频 + ASR 转写" />
        <NavItem href="/live-wall" label="直播电视墙" desc="实时监控" />
        <NavItem href="/appeals" label="申诉管理" desc="处理用户申诉" />
        <NavItem href="/quality-audit" label="质量抽检" desc="质检批次" />
        <NavItem href="/audit-log" label="审核日志" desc="操作记录" />
      </div>
    </div>

    <div>
      <Title level={5} style={{ marginBottom: SPACING.sm }}>
        风险分说明
      </Title>
      <div style={{ display: 'flex', gap: SPACING.sm, flexWrap: 'wrap' }}>
        <RiskBadge label="低危" range="0-20" color="green" />
        <RiskBadge label="中危" range="20-60" color="orange" />
        <RiskBadge label="高危" range="60-80" color="red" />
        <RiskBadge label="严重" range="80-100" color="#a8071a" />
      </div>
    </div>
  </Modal>
);

const ShortcutRow: React.FC<{ key: string; label: string; action: string }> = ({ label, action }) => (
  <div style={{ display: 'flex', alignItems: 'center', gap: SPACING.sm }}>
    <Tag style={{ fontFamily: 'monospace', fontSize: FONT.caption, margin: 0 }}>{label}</Tag>
    <Text style={{ fontSize: FONT.small }}>{action}</Text>
  </div>
);

const NavItem: React.FC<{ href: string; label: string; desc: string }> = ({ href, label, desc }) => (
  <a
    href={href}
    style={{
      display: 'flex',
      alignItems: 'center',
      gap: SPACING.sm,
      textDecoration: 'none',
      color: COLORS.textPrimary,
      fontSize: FONT.small,
    }}
  >
    <Text style={{ fontWeight: 500 }}>{label}</Text>
    <Text style={{ color: COLORS.textMuted }}>({desc})</Text>
  </a>
);

const RiskBadge: React.FC<{ label: string; range: string; color: string }> = ({ label, range, color }) => (
  <Tag color={color} style={{ margin: 0 }}>
    {label} {range}
  </Tag>
);

// Global help button in the header area.
export const HelpButton: React.FC<{ onOpen: () => void }> = ({ onOpen }) => (
  <Button
    type="text"
    icon={<QuestionCircleOutlined />}
    onClick={onOpen}
    aria-label="打开帮助"
    style={{ color: COLORS.textSecondary }}
  />
);

export default HelpModal;
