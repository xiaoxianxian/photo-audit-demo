import React from 'react';
import { Typography } from 'antd';
import { COLORS, FONT, SPACING } from '@/utils/constants';

const { Text } = Typography;

interface PasswordStrengthProps {
  password: string;
}

export const PasswordStrength: React.FC<PasswordStrengthProps> = ({ password }) => {
  if (!password) return null;

  const { score, label, color, tips } = evaluateStrength(password);

  return (
    <div style={{ marginTop: SPACING.xs }}>
      {/* Strength bar */}
      <div style={{
        display: 'flex',
        gap: 4,
        marginBottom: SPACING.xs,
      }}>
        {[1, 2, 3, 4].map((i) => (
          <div key={i} style={{
            flex: 1,
            height: 4,
            borderRadius: 2,
            background: i <= score ? color : COLORS.borderDefault,
            transition: 'background 0.2s',
          }} />
        ))}
      </div>
      {/* Label */}
      <Text style={{ color, fontSize: FONT.caption }}>
        密码强度：{label}
      </Text>
      {/* Tips */}
      <div style={{ marginTop: SPACING.xs }}>
        {tips.map((tip, i) => (
          <div key={i} style={{
            display: 'flex',
            alignItems: 'center',
            gap: SPACING.xs,
            fontSize: FONT.caption,
            color: COLORS.textMuted,
          }}>
            <span style={{ color: tip.met ? COLORS.success : COLORS.textMuted }}>
              {tip.met ? '✓' : '○'}
            </span>
            <span>{tip.text}</span>
          </div>
        ))}
      </div>
    </div>
  );
};

function evaluateStrength(password: string) {
  const checks = [
    { met: password.length >= 8, text: '至少8个字符' },
    { met: /[a-z]/.test(password), text: '包含小写字母' },
    { met: /[A-Z]/.test(password), text: '包含大写字母' },
    { met: /\d/.test(password), text: '包含数字' },
    { met: /[^a-zA-Z0-9]/.test(password), text: '包含特殊字符' },
  ];

  const score = checks.filter(c => c.met).length;
  const level = score <= 1 ? { label: '弱', color: COLORS.danger } :
    score <= 2 ? { label: '较弱', color: '#faad14' } :
    score <= 3 ? { label: '中等', color: COLORS.info } :
    { label: '强', color: COLORS.success };

  return { score: Math.min(score, 4), label: level.label, color: level.color, tips: checks };
}
