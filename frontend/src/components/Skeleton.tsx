import React from 'react';
import { COLORS, ANIMATION } from '@/utils/constants';

// Shared skeleton block used by all page skeletons.
const SkeletonBlock: React.FC<{
  width?: string;
  height?: string;
  radius?: string;
  style?: React.CSSProperties;
}> = ({ width = '100%', height = '16px', radius = '4px', style }) => (
  <div
    style={{
      width,
      height,
      borderRadius: radius,
      background: `linear-gradient(90deg, ${COLORS.bgContainer} 25%, ${COLORS.bgBase} 50%, ${COLORS.bgContainer} 75%)`,
      backgroundSize: '200% 100%',
      animation: `skeleton-loading ${ANIMATION.normal} ease-in-out infinite`,
      ...style,
    }}
  />
);

// Card skeleton — mimics an AuditCard layout.
export const CardSkeleton: React.FC<{ count?: number }> = ({ count = 3 }) =>
  Array.from({ length: count }).map((_, i) => (
    <div
      key={i}
      style={{
        background: COLORS.bgContainer,
        borderRadius: '8px',
        padding: '16px',
        border: `1px solid ${COLORS.borderDefault}`,
      }}
    >
      <div style={{ display: 'flex', gap: '12px', marginBottom: '12px' }}>
        <SkeletonBlock width="48px" height="48px" radius="8px" />
        <div style={{ flex: 1 }}>
          <SkeletonBlock width="80%" height="14px" style={{ marginBottom: '8px' }} />
          <SkeletonBlock width="60%" height="12px" />
        </div>
      </div>
      <SkeletonBlock width="100%" height="80px" style={{ marginBottom: '8px' }} />
      <div style={{ display: 'flex', gap: '8px' }}>
        <SkeletonBlock width="60px" height="20px" radius="10px" />
        <SkeletonBlock width="60px" height="20px" radius="10px" />
      </div>
    </div>
  ));

// Table skeleton — mimics an Ant Design Table with rows.
export const TableSkeleton: React.FC<{ rows?: number }> = ({ rows = 5 }) => (
  <div>
    {Array.from({ length: rows }).map((_, i) => (
      <div
        key={i}
        style={{
          display: 'flex',
          gap: '16px',
          padding: '12px 16px',
          borderBottom: `1px solid ${COLORS.borderDefault}`,
        }}
      >
        <SkeletonBlock width="120px" height="14px" />
        <SkeletonBlock width="80px" height="14px" />
        <SkeletonBlock width="100px" height="14px" />
        <SkeletonBlock width="60px" height="14px" />
        <SkeletonBlock width="40px" height="14px" />
      </div>
    ))}
  </div>
);

// Stats card skeleton — for dashboard stats row.
export const StatsSkeleton: React.FC = () => (
  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '16px' }}>
    {Array.from({ length: 4 }).map((_, i) => (
      <div
        key={i}
        style={{
          background: COLORS.bgContainer,
          borderRadius: '8px',
          padding: '20px',
          border: `1px solid ${COLORS.borderDefault}`,
        }}
      >
        <SkeletonBlock width="40%" height="12px" style={{ marginBottom: '12px' }} />
        <SkeletonBlock width="60%" height="28px" radius="4px" />
      </div>
    ))}
  </div>
);

// Page skeleton — generic loading state for full pages.
export const PageSkeleton: React.FC = () => (
  <div style={{ padding: '24px' }}>
    {/* Header */}
    <SkeletonBlock width="200px" height="24px" style={{ marginBottom: '24px' }} />
    {/* Stats row */}
    <div style={{ marginBottom: '24px' }}>
      <StatsSkeleton />
    </div>
    {/* Content */}
    <TableSkeleton rows={5} />
  </div>
);
