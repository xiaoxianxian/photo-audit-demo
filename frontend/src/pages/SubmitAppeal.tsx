import React, { useState } from 'react';
import { Form, Input, Button, Card, message, Typography } from 'antd';
import { ArrowLeftOutlined, SendOutlined } from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import { submitAppeal } from '@/services/content-api';
import useAuthStore from '@/stores/auth';
import {
  COLORS,
  SPACING,
  FONT,
} from '@/utils/constants';

const { Title, Text } = Typography;

const SubmitAppealPage: React.FC = () => {
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);
  const [loading, setLoading] = useState(false);
  const { contentId } = useParams<{ contentId: string }>();

  if (!contentId) {
    message.error('缺少内容 ID');
    navigate('/appeals');
    return null;
  }

  if (!user) {
    message.error('请先登录');
    navigate('/login');
    return null;
  }

  const onFinish = async (values: { reason: string; evidence_urls?: string }) => {
    setLoading(true);
    try {
      const evidenceUrls = values.evidence_urls
        ? values.evidence_urls.split('\n').filter((u) => u.trim())
        : [];
      await submitAppeal({
        content_id: contentId,
        reason: values.reason,
        evidence_urls: evidenceUrls,
        applicant_id: user.id,
      });
      message.success('申诉提交成功');
      navigate('/appeals');
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '申诉提交失败';
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
        padding: SPACING.lg,
      }}
    >
      <Card
        style={{
          width: 560,
          background: COLORS.bgContainer,
          borderColor: COLORS.borderDefault,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: SPACING.lg }}>
          <Button
            type="text"
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate(-1)}
            style={{ color: COLORS.textSecondary, marginRight: SPACING.md }}
            aria-label="返回上一页"
          />
          <div>
            <Title level={3} style={{ color: COLORS.textPrimary, margin: 0 }}>
              提交申诉
            </Title>
            <Text style={{ color: COLORS.textSecondary }}>对审核结果有异议？请提供申诉理由</Text>
          </div>
        </div>

        <Form layout="vertical" onFinish={onFinish} autoComplete="off">
          <Form.Item
            name="reason"
            label="申诉理由"
            rules={[
              { required: true, message: '请输入申诉理由' },
              { min: 10, message: '申诉理由至少10个字符' },
            ]}
          >
            <Input.TextArea
              rows={6}
              placeholder="请详细说明您的申诉理由，例如：内容不存在违规、被误判等"
              aria-label="申诉理由"
            />
          </Form.Item>

          <Form.Item
            name="evidence_urls"
            label="证明材料链接（可选）"
          >
            <Input.TextArea
              rows={3}
              placeholder="每行一个链接，例如图片/文件的下载地址"
              aria-label="证明材料链接"
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: SPACING.base }}>
            <Button
              type="primary"
              htmlType="submit"
              loading={loading}
              icon={<SendOutlined />}
              block
              size="large"
            >
              提交申诉
            </Button>
          </Form.Item>
        </Form>

        <div style={{ color: COLORS.textMuted, fontSize: FONT.small, lineHeight: 1.6 }}>
          <Text style={{ color: COLORS.textSecondary }}>提示：</Text>
          <br />
          {'• 每条内容仅可提交一次申诉'}
          <br />
          {'• 申诉将由审核员处理，结果将通知您'}
          <br />
          {'• 请如实填写，虚假申诉可能导致账号受限'}
        </div>
      </Card>
    </div>
  );
};

export default SubmitAppealPage;
