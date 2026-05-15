import React from 'react';
import { Layout as AntLayout, Button, Space, Dropdown, Typography, Grid } from 'antd';
import { Link, Outlet, useNavigate } from 'react-router-dom';
import {
  UserOutlined,
  LogoutOutlined,
  BookOutlined,
} from '@ant-design/icons';
import { useAuthStore } from '../store/auth';

const { Header, Content, Footer } = AntLayout;
const { Text } = Typography;
const { useBreakpoint } = Grid;

const AppLayout: React.FC = () => {
  const { isAuthenticated, user, logout } = useAuthStore();
  const navigate = useNavigate();
  const screens = useBreakpoint();
  const isMobile = !screens.md;

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
          padding: isMobile ? '0 12px' : '0 40px',
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
        <Link to="/" style={{ textDecoration: 'none' }}>
          <Space size={8}>
            <BookOutlined style={{ fontSize: isMobile ? 20 : 24, color: '#1677ff' }} />
            <Text strong style={{ fontSize: isMobile ? 15 : 18, color: '#1677ff' }}>
              面试刷题
            </Text>
          </Space>
        </Link>

        <Space size={isMobile ? 4 : 16}>
          {(user?.role === 'teacher' || user?.role === 'director' || user?.role === 'principal') && (
            <>
              <Button type="link" size={isMobile ? 'small' : 'middle'} onClick={() => navigate('/teacher')}>
                教师后台
              </Button>
              <Button type="link" size={isMobile ? 'small' : 'middle'} onClick={() => navigate('/classes')}>
                班级管理
              </Button>
            </>
          )}
          {(user?.role === 'director' || user?.role === 'principal') && (
            <Button type="link" size={isMobile ? 'small' : 'middle'} onClick={() => navigate('/admin')}>
              系统管理
            </Button>
          )}
          {isAuthenticated ? (
            <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
              <Button type="text" size={isMobile ? 'small' : 'middle'} icon={<UserOutlined />}>
                {isMobile ? '' : user?.username || '用户'}
              </Button>
            </Dropdown>
          ) : (
            <Space size={4}>
              <Button type="text" size={isMobile ? 'small' : 'middle'} onClick={() => navigate('/login')}>登录</Button>
              <Button type="primary" size={isMobile ? 'small' : 'middle'} onClick={() => navigate('/register')}>注册</Button>
            </Space>
          )}
        </Space>
      </Header>

      <Content style={{ padding: isMobile ? '12px' : '24px 40px', maxWidth: 1200, margin: '0 auto', width: '100%' }}>
        <Outlet />
      </Content>

      <Footer style={{ textAlign: 'center', color: '#999', background: '#f5f5f5', padding: isMobile ? '12px' : '16px' }}>
        面试刷题平台 ©2026
      </Footer>
    </AntLayout>
  );
};

export default AppLayout;
