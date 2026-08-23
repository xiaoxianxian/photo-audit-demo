import React, { lazy, Suspense } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import useAuthStore from '@/stores/auth';

// Lazy-loaded pages (code-split)
const LoginPage = lazy(() => import('@/pages/Login'));
const RegisterPage = lazy(() => import('@/pages/Register'));
const DashboardPage = lazy(() => import('@/pages/Dashboard'));
const ReviewPage = lazy(() => import('@/pages/Review'));
const AppealPage = lazy(() => import('@/pages/Appeal'));
const LiveWallPage = lazy(() => import('@/pages/LiveWall'));
const TenantConfigPage = lazy(() => import('@/pages/TenantConfig'));
const QualityAuditPage = lazy(() => import('@/pages/QualityAudit'));
const ShortVideoReviewPage = lazy(() => import('@/pages/ShortVideoReview'));
const AuditLogPage = lazy(() => import('@/pages/AuditLog'));
const AIConfigPage = lazy(() => import('@/pages/AIConfig'));
const SubmitAppealPage = lazy(() => import('@/pages/SubmitAppeal'));

// Shared loading fallback
const LoadingFallback = () => (
  <div style={{
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    height: '100vh',
    background: '#f5f5f5',
  }}>
    <div style={{ fontSize: 16, color: '#999' }}>Loading...</div>
  </div>
);

const ProtectedRoute: React.FC<{
  children: React.ReactNode;
  roles?: string[];
}> = ({ children, roles }) => {
  const user = useAuthStore((s) => s.user);
  const isAuthenticated = !!user;
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  if (roles && user && !roles.includes(user.role)) {
    return <Navigate to="/review" replace />;
  }
  return <>{children}</>;
};

const App: React.FC = () => (
  <Suspense fallback={<LoadingFallback />}>
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />

      <Route
        path="/"
        element={
          <ProtectedRoute><DashboardPage /></ProtectedRoute>
        }
      />
      <Route
        path="/review"
        element={
          <ProtectedRoute><ReviewPage /></ProtectedRoute>
        }
      />
      <Route
        path="/appeals"
        element={
          <ProtectedRoute><AppealPage /></ProtectedRoute>
        }
      />
      <Route
        path="/live-wall"
        element={
          <ProtectedRoute><LiveWallPage /></ProtectedRoute>
        }
      />
      <Route
        path="/tenant-config"
        element={
          <ProtectedRoute roles={['platform_admin', 'tenant_admin']}>
            <TenantConfigPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/quality-audit"
        element={
          <ProtectedRoute roles={['quality_checker', 'tenant_admin', 'platform_admin']}>
            <QualityAuditPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/review/video"
        element={
          <ProtectedRoute><ShortVideoReviewPage /></ProtectedRoute>
        }
      />
      <Route
        path="/audit-log"
        element={
          <ProtectedRoute><AuditLogPage /></ProtectedRoute>
        }
      />
      <Route
        path="/ai-config"
        element={
          <ProtectedRoute roles={['platform_admin', 'tenant_admin']}>
            <AIConfigPage />
          </ProtectedRoute>
        }
      />
      <Route
        path="/appeal/new/:contentId"
        element={
          <ProtectedRoute><SubmitAppealPage /></ProtectedRoute>
        }
      />

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  </Suspense>
);

export default App;
