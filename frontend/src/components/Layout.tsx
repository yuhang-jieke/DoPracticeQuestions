import React from 'react';
import { Layout as AntLayout, Menu, Button, Space, Dropdown, Typography } from 'antd';
import { Link, Outlet, useNavigate, useLocation } from 'react-router-dom';
import {
  MenuOutlined,
  UserOutlined,
  LogoutOutlined,
  BookOutlined,
  HomeOutlined,
  BarChartOutlined,
} from '@ant-design/icons';
import { useAuthStore } from '../store/auth';

const { Header, Content, Footer } = AntLayout;
const { Text } = Typography;

const AppLayout: React.FC = () => {
  const { isAuthenticated, user, logout } = useAuthStore();
  const navigate = useNavigate();
  const location = useLocation();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const userMenuItems = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: '个人中心',
      onClick: () => navigate('/user'),
    },
    { type: 'divider' as const },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      onClick: handleLogout,
    },
  ];

  return (
    <AntLayout style={{ minHeight: '100vh', background: '#f5f5f5' }}>
      <Header
        style={{
          background: '#fff',
          padding: '0 40px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: '1px solid #f0f0f0',
          position: 'sticky',
          top: 0,
          zIndex: 100,
          boxShadow: '0 1px 4px rgba(0,0,0,0.04)',
        }}
      >
        <Space size={24}>
          <Link to="/" style={{ textDecoration: 'none' }}>
            <Space>
              <BookOutlined style={{ fontSize: 24, color: '#1677ff' }} />
              <Text strong style={{ fontSize: 18, color: '#1677ff' }}>
                面试刷题
              </Text>
            </Space>
          </Link>
        </Space>

        <Space size={16}>
          {isAuthenticated ? (
            <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
              <Button type="text" icon={<UserOutlined />}>
                {user?.username || '用户'}
              </Button>
            </Dropdown>
          ) : (
            <Space>
              <Button type="text" onClick={() => navigate('/login')}>
                登录
              </Button>
              <Button type="primary" onClick={() => navigate('/register')}>
                注册
              </Button>
            </Space>
          )}
        </Space>
      </Header>

      <Content style={{ padding: '24px 40px', maxWidth: 1200, margin: '0 auto', width: '100%' }}>
        <Outlet />
      </Content>

      <Footer style={{ textAlign: 'center', color: '#999', background: '#f5f5f5', padding: '16px' }}>
        面试刷题平台 ©2026
      </Footer>
    </AntLayout>
  );
};

export default AppLayout;
