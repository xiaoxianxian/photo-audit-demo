import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { ConfigProvider, theme } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import './styles/global.css';
import App from './App';
import { COLORS } from './utils/constants';

/**
 * Entry point: wraps the app with BrowserRouter and Ant Design ConfigProvider
 * using the design library's dark theme tokens.
 */
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <ConfigProvider
        locale={zhCN}
        theme={{
          algorithm: theme.darkAlgorithm,
          token: {
            colorBgContainer: COLORS.bgContainer,
            colorBgBase: COLORS.bgBase,
            colorPrimary: COLORS.brandBlue,
            colorSuccess: COLORS.success,
            colorError: COLORS.danger,
            colorWarning: COLORS.warning,
            colorInfo: COLORS.info,
            borderRadius: 8,
            fontFamily: "'Inter', 'Noto Sans SC', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
          },
        }}
      >
        <App />
      </ConfigProvider>
    </BrowserRouter>
  </React.StrictMode>
);
