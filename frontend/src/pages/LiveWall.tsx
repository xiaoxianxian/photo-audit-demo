import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  Tag,
  Row,
  Col,
  Statistic,
  Typography,
  Empty,
  Alert,
  Modal,
  Form,
  Input,
  Button,
  message,
  Tooltip,
} from 'antd';
import {
  VideoCameraOutlined,
  UserOutlined,
  WarningOutlined,
  PlusOutlined,
  CopyOutlined,
} from '@ant-design/icons';
import {
  getLiveWall,
  startLiveStream,
  stopLiveStream,
  type LiveStreamWithSnapshot,
} from '@/services/content-api';
import AppLayout, { Content } from '@/components/Layout';
import { useNavigate } from 'react-router-dom';
import {
  COLORS,
  SPACING,
  FONT,
  RADIUS,
  ANIMATION as ANIM,
  SIZE,
  riskColor,
  riskLabel,
  riskTagColor,
} from '@/utils/constants';

const { Text } = Typography;

// ---------------------------------------------------------------------------
// Stream Tile Component
// ---------------------------------------------------------------------------

const StreamTile: React.FC<{
  stream: LiveStreamWithSnapshot;
  onClick?: () => void;
  onStop?: () => void;
}> = ({ stream, onClick, onStop }) => {
  const latestSnapshot = stream.latest_snapshot;
  const riskScore = latestSnapshot?.ai_risk_score ?? 0;
  const riskTypes = latestSnapshot?.ai_risk_types ?? [];
  const isOffline = stream.status !== 'streaming' || !latestSnapshot?.snapshot_url;

  const handleCopyRTMP = () => {
    navigator.clipboard.writeText(stream.stream_url);
    message.success('RTMP 推流地址已复制');
  };

  return (
    <div
      style={{
        position: 'relative',
        borderRadius: RADIUS.md,
        overflow: 'hidden',
        background: isOffline ? COLORS.bgOffline : COLORS.bgContainer,
        border: isOffline
          ? `1px dashed ${COLORS.borderDefault}`
          : riskScore > 60
            ? `2px solid ${COLORS.conflictOrange}`
            : `1px solid ${COLORS.borderDefault}`,
        transition: `border-color ${ANIM.normal}`,
        opacity: isOffline ? 0.5 : 1,
        cursor: onClick ? 'pointer' : 'default',
      }}
      onClick={onClick}
      onMouseEnter={(e) => {
        if (!isOffline) (e.currentTarget as HTMLDivElement).style.borderColor = COLORS.borderHover;
      }}
      onMouseLeave={(e) => {
        (e.currentTarget as HTMLDivElement).style.borderColor = isOffline
          ? COLORS.borderDefault
          : riskScore > 60
            ? COLORS.conflictOrange
            : COLORS.borderDefault;
      }}
    >
      {/* Snapshot image */}
      <div style={{ height: 140, background: COLORS.bgBase, position: 'relative' }}>
        {latestSnapshot?.snapshot_url ? (
          <img
            src={latestSnapshot.snapshot_url}
            alt="直播截帧"
            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
            onError={(e) => {
              (e.target as HTMLImageElement).style.display = 'none';
            }}
          />
        ) : (
          <div
            style={{
              width: '100%',
              height: '100%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: COLORS.textMuted,
            }}
          >
            <VideoCameraOutlined style={{ fontSize: 32 }} />
          </div>
        )}

        {/* Status badge */}
        <div
          style={{
            position: 'absolute',
            top: SPACING.xs,
            left: SPACING.xs,
            display: 'flex',
            gap: SPACING.xs,
          }}
        >
          <Tag color={isOffline ? 'default' : 'success'} style={{ margin: 0, fontSize: FONT.caption }}>
            {isOffline ? '○ OFFLINE' : '● LIVE'}
          </Tag>
          {riskTypes.map((t) => (
            <Tag key={t} color="error" style={{ margin: 0, fontSize: FONT.caption }}>
              {t}
            </Tag>
          ))}
        </div>

        {/* Risk score overlay */}
        <div
          style={{
            position: 'absolute',
            bottom: SPACING.xs,
            right: SPACING.xs,
            background: riskColor(riskScore),
            color: COLORS.textPrimary,
            borderRadius: RADIUS.sm,
            padding: '2px 8px',
            fontSize: FONT.small,
            fontWeight: 600,
          }}
          aria-label={`风险分 ${riskScore} - ${riskLabel(riskScore)}`}
        >
          {riskLabel(riskScore)} {riskScore}
        </div>
      </div>

      {/* Info bar */}
      <div style={{ padding: `${SPACING.sm} ${SPACING.base}`, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: SPACING.xs }}>
          <UserOutlined style={{ color: COLORS.textSecondary }} />
          <Text style={{ color: COLORS.textTertiary, fontSize: FONT.small }}>
            {latestSnapshot?.ai_confidence ? `${(latestSnapshot.ai_confidence * 100).toFixed(0)}%` : '--'}
          </Text>
        </div>
        <Tag color={riskTagColor(riskScore)} style={{ margin: 0, fontSize: FONT.caption }}>
          {riskLabel(riskScore)}
        </Tag>
      </div>

      {/* RTMP URL + Stop button for active streams */}
      {!isOffline && (
        <div style={{ padding: `0 ${SPACING.base} ${SPACING.sm}`, display: 'flex', gap: SPACING.xs, alignItems: 'center' }}>
          <Text
            style={{ color: COLORS.textMuted, fontSize: FONT.caption, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
            title={stream.stream_url}
          >
            {stream.stream_url}
          </Text>
          <Tooltip title="复制 RTMP 地址">
            <Button
              size="small"
              icon={<CopyOutlined />}
              style={{ padding: '0 6px', height: SIZE.buttonSm, fontSize: FONT.caption }}
              onClick={(e) => {
                e.stopPropagation();
                handleCopyRTMP();
              }}
              aria-label="复制 RTMP 推流地址"
            />
          </Tooltip>
          <Tooltip title="停止直播">
            <Button
              size="small"
              danger
              style={{ padding: '0 6px', height: SIZE.buttonSm, fontSize: FONT.caption }}
              onClick={(e) => {
                e.stopPropagation();
                onStop?.();
              }}
              aria-label="停止直播"
            >
              停止
            </Button>
          </Tooltip>
        </div>
      )}
    </div>
  );
};

// ---------------------------------------------------------------------------
// Live Wall Page
// ---------------------------------------------------------------------------

const LiveWallPage: React.FC = () => {
  const navigate = useNavigate();
  const [streams, setStreams] = useState<LiveStreamWithSnapshot[]>([]);
  const [loading, setLoading] = useState(false);
  const cancelledRef = useRef(false);
  const wsRef = useRef<WebSocket | null>(null);

  const isMounted = useRef(true);

  // Start stream modal
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();

  useEffect(() => {
    return () => {
      isMounted.current = false;
    };
  }, []);

  const fetchStreams = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getLiveWall();
      if (cancelledRef.current) return;
      setStreams(data ?? []);
    } catch {
      message.error('获取直播流失败');
      setStreams([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    cancelledRef.current = false;
    fetchStreams();
    return () => {
      cancelledRef.current = true;
    };
  }, [fetchStreams]);

  // WebSocket connection for real-time updates
  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws/live-wall`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {};
    ws.onclose = () => {};
    ws.onerror = () => {};
    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === 'snapshot_update' && !cancelledRef.current) {
          fetchStreams();
        }
      } catch {
        // ignore parse errors
      }
    };

    return () => {
      ws.close();
    };
  }, [fetchStreams]);

  // Auto-refresh every 10 seconds as fallback
  useEffect(() => {
    const interval = setInterval(() => {
      if (!cancelledRef.current) {
        fetchStreams();
      }
    }, 10000);
    return () => clearInterval(interval);
  }, [fetchStreams]);

  const highRiskCount = streams.filter(
    (s) => (s.latest_snapshot?.ai_risk_score ?? 0) > 60,
  ).length;

  const handleStartStream = async () => {
    try {
      const values = await form.validateFields();
      await startLiveStream({
        content_id: values.content_id || '',
        stream_key: values.stream_key,
        stream_url: '',
        play_url: '',
      });
      message.success('直播间已启动');
      setModalOpen(false);
      form.resetFields();
      fetchStreams();
    } catch {
      message.error('启动直播间失败');
    }
  };

  const handleStopStream = async (id: string) => {
    try {
      await stopLiveStream(id);
      message.success('直播间已停止');
      fetchStreams();
    } catch {
      message.error('停止直播间失败');
    }
  };

  return (
    <AppLayout>
      <Content style={{ padding: SPACING.lg, background: COLORS.bgBase }}>
        {/* Header */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: SPACING.lg, flexWrap: 'wrap', gap: SPACING.sm }}>
          <Typography.Title level={4} style={{ color: COLORS.textPrimary, margin: 0 }}>
            直播电视墙
          </Typography.Title>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setModalOpen(true)}
          >
            新建直播间
          </Button>
        </div>

        {/* Stats bar */}
        <div style={{ marginBottom: SPACING.lg, display: 'flex', gap: SPACING.lg, flexWrap: 'wrap' }}>
          <Statistic
            title="在线直播间"
            value={streams.filter((s) => s.status === 'streaming').length}
            valueStyle={{ color: COLORS.textPrimary }}
            prefix={<VideoCameraOutlined />}
          />
          <Statistic
            title="高风险直播间"
            value={highRiskCount}
            valueStyle={{ color: highRiskCount > 0 ? COLORS.danger : COLORS.textPrimary }}
            prefix={<WarningOutlined />}
          />
          <Statistic
            title="总观看人数"
            value={streams.reduce((sum, s) => sum + (s.viewer_count || 0), 0)}
            valueStyle={{ color: COLORS.textPrimary }}
            prefix={<UserOutlined />}
          />
        </div>

        {highRiskCount > 0 && (
          <Alert
            type="warning"
            showIcon
            icon={<WarningOutlined />}
            message={`${highRiskCount} 个直播间存在高风险，请及时处理`}
            style={{ marginBottom: SPACING.lg }}
          />
        )}

        {/* Stream Grid */}
        <Row gutter={[SPACING.base, SPACING.base]}>
          {streams.map((stream) => (
            <Col xs={24} sm={12} md={8} lg={6} key={stream.id}>
              <StreamTile
                stream={stream}
                onClick={() => {
                  if ((stream.latest_snapshot?.ai_risk_score ?? 0) > 60) {
                    navigate('/review', { state: { fromLiveWall: stream.content_id } });
                  }
                }}
                onStop={() => handleStopStream(stream.id)}
              />
            </Col>
          ))}
        </Row>

        {streams.length === 0 && !loading && (
          <Empty
            description="暂无正在直播的流，点击「新建直播间」开始"
            style={{ marginTop: SPACING.xxl, color: COLORS.textMuted }}
          />
        )}

        {/* Start Stream Modal */}
        <Modal
          title="新建直播间"
          open={modalOpen}
          onCancel={() => {
            setModalOpen(false);
            form.resetFields();
          }}
          onOk={handleStartStream}
          okText="启动"
          cancelText="取消"
        >
          <Form form={form} layout="vertical">
            <Form.Item
              label="内容 ID（可选）"
              name="content_id"
            >
              <Input placeholder="关联的内容 ID，留空则跳过" aria-label="内容 ID" />
            </Form.Item>
            <Form.Item
              label="推流密钥"
              name="stream_key"
              rules={[{ required: true, message: '请输入推流密钥' }]}
            >
              <Input placeholder="自动生成 RTMP 推流地址" aria-label="推流密钥" />
            </Form.Item>
          </Form>
          <Text type="secondary" style={{ display: 'block', marginTop: SPACING.base, fontSize: FONT.caption, lineHeight: 1.8 }}>
            启动后将自动生成 RTMP 推流地址：rtmp://localhost:1935/live/&lt;stream_key&gt;
            <br />
            请使用 OBS 等推流软件向该地址推送 RTMP 流。
          </Text>
        </Modal>
      </Content>
    </AppLayout>
  );
};

export default LiveWallPage;
