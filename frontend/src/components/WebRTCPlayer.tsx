import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Tag, Button, message } from 'antd';
import { PlayCircleOutlined, PauseCircleOutlined } from '@ant-design/icons';
import { whepView } from '@/services/webrtc-api';
import { COLORS, SPACING, FONT, RADIUS } from '@/utils/constants';

// ---------------------------------------------------------------------------
// WebRTCPlayer — WHEP viewer for a live stream.
//
// Flow: create RTCPeerConnection with a recvonly transceiver → generate an
// SDP answer → POST it to /webrtc/whep/:streamKey → receive the publisher's
// offer → setRemoteDescription. Media then flows P2P (or via the browser's
// ICE candidates); the server only relayed signaling.
// ---------------------------------------------------------------------------

interface WebRTCPlayerProps {
  streamKey: string;
  height?: number;
}

const WebRTCPlayer: React.FC<WebRTCPlayerProps> = ({ streamKey, height = 180 }) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const pcRef = useRef<RTCPeerConnection | null>(null);
  const [state, setState] = useState<'idle' | 'connecting' | 'playing' | 'error'>('idle');

  const stop = useCallback(() => {
    pcRef.current?.close();
    pcRef.current = null;
    if (videoRef.current) videoRef.current.srcObject = null;
    setState('idle');
  }, []);

  const start = useCallback(async () => {
    if (!streamKey) return;
    setState('connecting');
    try {
      const pc = new RTCPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] });
      pcRef.current = pc;

      // We only receive media.
      pc.addTransceiver('video', { direction: 'recvonly' });
      pc.addTransceiver('audio', { direction: 'recvonly' });

      pc.ontrack = (ev) => {
        if (videoRef.current && ev.streams[0]) {
          videoRef.current.srcObject = ev.streams[0];
          void videoRef.current.play().catch(() => {/* autoplay blocked until user gesture */});
        }
      };
      pc.onconnectionstatechange = () => {
        if (pc.connectionState === 'failed' || pc.connectionState === 'disconnected') {
          setState('error');
        } else if (pc.connectionState === 'connected') {
          setState('playing');
        }
      };

      // Build local answer first (WHEP viewer role).
      const answer = await pc.createAnswer();
      await pc.setLocalDescription(answer);

      // Wait for ICE gathering to complete so the SDP is complete (non-trickle).
      await new Promise<void>((resolve) => {
        if (pc.iceGatheringState === 'complete') return resolve();
        const check = () => {
          if (pc.iceGatheringState === 'complete') {
            pc.removeEventListener('icegatheringstatechange', check);
            resolve();
          }
        };
        pc.addEventListener('icegatheringstatechange', check);
        setTimeout(resolve, 1500); // safety timeout
      });

      const offerSdp = await whepView(streamKey, pc.localDescription?.sdp ?? '');
      if (!offerSdp) {
        setState('error');
        message.warning('当前没有发布者在线');
        return;
      }
      await pc.setRemoteDescription({ type: 'offer', sdp: offerSdp });
      // connectionstatechange will move us to 'playing'.
    } catch {
      setState('error');
      message.error('WebRTC 连接失败');
    }
  }, [streamKey]);

  useEffect(() => () => stop(), [stop]);

  return (
    <div
      style={{
        position: 'relative',
        borderRadius: RADIUS.sm,
        overflow: 'hidden',
        background: '#000',
        height,
      }}
    >
      <video ref={videoRef} autoPlay playsInline muted style={{ width: '100%', height: '100%', objectFit: 'contain' }} />

      {state !== 'playing' && (
        <div
          style={{
            position: 'absolute',
            inset: 0,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            gap: SPACING.xs,
            color: COLORS.textMuted,
          }}
        >
          {state === 'connecting' ? (
            <Tag color="processing" style={{ fontSize: FONT.caption }}>连接中…</Tag>
          ) : state === 'error' ? (
            <>
              <Tag color="warning" style={{ fontSize: FONT.caption }}>无信号 / 无发布者</Tag>
              <Button size="small" icon={<PlayCircleOutlined />} onClick={start}>重试</Button>
            </>
          ) : (
            <Button type="primary" size="small" icon={<PlayCircleOutlined />} onClick={start}>
              播放实时画面
            </Button>
          )}
        </div>
      )}

      {state === 'playing' && (
        <Button
          size="small"
          icon={<PauseCircleOutlined />}
          style={{ position: 'absolute', top: SPACING.xs, right: SPACING.xs }}
          onClick={stop}
        >
          断开
        </Button>
      )}
    </div>
  );
};

export default WebRTCPlayer;
