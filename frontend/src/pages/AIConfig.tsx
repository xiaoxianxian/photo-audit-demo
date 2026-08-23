import React, { useState, useEffect } from 'react';
import {
  Card,
  Form,
  Input,
  Button,
  Switch,
  InputNumber,
  message,
  Space,
  Descriptions,
  Alert,
  Divider,
  Tag,
  Select,
  Typography,
} from 'antd';
import {
  SaveOutlined,
  KeyOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import AppLayout, { Content } from '@/components/Layout';
import { getDashboardStats, getAIConfig, saveAIConfig, type DashboardStats } from '@/services/content-api';
import { COLORS, SPACING } from '@/utils/constants';

const { Title, Text } = Typography;

const AIConfigPage: React.FC = () => {
  const [form] = Form.useForm();
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadConfig();
    loadStats();
  }, []);

  const loadConfig = async () => {
    try {
      const cfg = await getAIConfig();
      if (cfg) {
        form.setFieldsValue({
          agnes_api_key: '',
          agnes_endpoint: cfg.agnes_endpoint,
          agnes_concurrency: cfg.agnes_concurrency,
          deepseek_api_key: '',
          deepseek_model: cfg.deepseek_model,
          fallback_enabled: cfg.fallback_enabled,
        });
      }
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      const data = await getDashboardStats();
      setStats(data);
    } catch {
      // ignore
    }
  };

  const handleSave = async () => {
    try {
      setSaving(true);
      const values = await form.validateFields();
      await saveAIConfig(values);
      message.success('AI 模型配置已保存');
    } catch (err: unknown) {
      const msg = err instanceof Error ? (err as { response?: { data?: { message?: string } } }).response?.data?.message || '保存配置失败' : '保存配置失败';
      message.error(msg);
    } finally {
      setSaving(false);
    }
  };

  const agnesKey = Form.useWatch('agnes_api_key', form);
  const deepseekKey = Form.useWatch('deepseek_api_key', form);
  const fallbackEnabled = Form.useWatch('fallback_enabled', form);

  if (loading) return (
    <AppLayout>
      <div style={{ padding: SPACING.lg, textAlign: 'center', color: COLORS.textSecondary }}>加载中...</div>
    </AppLayout>
  );

  return (
    <AppLayout>
      <Content style={{ padding: SPACING.lg, background: COLORS.bgBase }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: SPACING.lg }}>
          <Title level={4} style={{ color: COLORS.textPrimary, margin: 0 }}>
            AI 模型配置
          </Title>
          <Button
            type="primary"
            icon={<SaveOutlined />}
            loading={saving}
            onClick={handleSave}
          >
            保存配置
          </Button>
        </div>

        {/* Current status */}
        {stats && (
          <Card
            style={{ marginBottom: SPACING.lg, background: COLORS.bgContainer, borderColor: COLORS.borderDefault }}
            bordered={false}
          >
            <Descriptions
              bordered
              size="small"
              column={3}
            >
              <Descriptions.Item label="Agnes AI">
                {agnesKey ? (
                  <Tag color="success" icon={<CheckCircleOutlined />}>已配置</Tag>
                ) : (
                  <Tag color="error" icon={<CloseCircleOutlined />}>未配置</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="DeepSeek 裁判">
                {deepseekKey ? (
                  <Tag color="success" icon={<CheckCircleOutlined />}>已配置</Tag>
                ) : (
                  <Tag color="error" icon={<CloseCircleOutlined />}>未配置</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="本地降级">
                {fallbackEnabled !== false ? (
                  <Tag color="processing">已启用</Tag>
                ) : (
                  <Tag color="default">已禁用</Tag>
                )}
              </Descriptions.Item>
            </Descriptions>
          </Card>
        )}

        {/* Agnes AI Config */}
        <Card
          title={
            <Space>
              <KeyOutlined />
              <Text style={{ color: COLORS.textPrimary }}>Agnes AI 多模态模型</Text>
            </Space>
          }
          style={{ marginBottom: SPACING.lg, background: COLORS.bgContainer, borderColor: COLORS.borderDefault }}
          bordered={false}
        >
          <Form form={form} layout="vertical">
            <Form.Item
              label="API Key"
              name="agnes_api_key"
              tooltip="留空则保持现有密钥不变"
            >
              <Input.Password
                placeholder="sk-xxx（留空保持不变）"
                autoComplete="off"
                aria-label="Agnes API Key"
              />
            </Form.Item>
            <Form.Item
              label="API 端点"
              name="agnes_endpoint"
            >
              <Input placeholder="https://api.agnes.ai/v1/review" aria-label="API 端点" />
            </Form.Item>
            <Form.Item
              label="并发限制"
              name="agnes_concurrency"
              tooltip="同时发送到 Agnes AI 的最大请求数"
            >
              <InputNumber min={1} max={100} style={{ width: '100%' }} aria-label="并发限制" />
            </Form.Item>
          </Form>
        </Card>

        {/* DeepSeek Config */}
        <Card
          title={
            <Space>
              <KeyOutlined />
              <Text style={{ color: COLORS.textPrimary }}>DeepSeek 裁判模型</Text>
            </Space>
          }
          style={{ marginBottom: SPACING.lg, background: COLORS.bgContainer, borderColor: COLORS.borderDefault }}
          bordered={false}
        >
          <Form form={form} layout="vertical">
            <Form.Item
              label="API Key"
              name="deepseek_api_key"
              tooltip="留空则保持现有密钥不变"
            >
              <Input.Password
                placeholder="sk-xxx（留空保持不变）"
                autoComplete="off"
                aria-label="DeepSeek API Key"
              />
            </Form.Item>
            <Form.Item
              label="模型"
              name="deepseek_model"
            >
              <Select
                placeholder="选择模型"
                aria-label="DeepSeek 模型"
              >
                <Select.Option value="deepseek-chat">deepseek-chat</Select.Option>
                <Select.Option value="deepseek-coder">deepseek-coder</Select.Option>
                <Select.Option value="deepseek-v3">deepseek-v3</Select.Option>
                <Select.Option value="deepseek-r1">deepseek-r1</Select.Option>
              </Select>
            </Form.Item>
          </Form>
        </Card>

        {/* Fallback Settings */}
        <Card
          title={
            <Space>
              <SaveOutlined />
              <Text style={{ color: COLORS.textPrimary }}>降级策略</Text>
            </Space>
          }
          style={{ marginBottom: SPACING.lg, background: COLORS.bgContainer, borderColor: COLORS.borderDefault }}
          bordered={false}
        >
          <Form form={form} layout="vertical">
            <Form.Item
              label="启用本地规则引擎降级"
              name="fallback_enabled"
              valuePropName="checked"
              initialValue={true}
            >
              <Switch aria-label="启用降级" />
            </Form.Item>
            <Alert
              message="当 Agnes AI API 返回 402/429 或 API Key 未配置时，系统将自动使用本地规则引擎进行内容审核。"
              type="info"
              showIcon
              style={{ marginTop: SPACING.md }}
            />
          </Form>
        </Card>

        <Divider style={{ borderColor: COLORS.borderDefault }} />

        {/* Quick Actions */}
        <Card
          title="快速操作"
          style={{ background: COLORS.bgContainer, borderColor: COLORS.borderDefault }}
          bordered={false}
        >
          <Space>
            <Button onClick={loadStats}>
              刷新状态
            </Button>
          </Space>
        </Card>
      </Content>
    </AppLayout>
  );
};

export default AIConfigPage;
