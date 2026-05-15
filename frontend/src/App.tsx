import React from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { ConfigProvider, App as AntApp } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import Layout from './components/Layout';
import Home from './pages/Home';
import Login from './pages/Login';
import Register from './pages/Register';
import QuestionDetail from './pages/QuestionDetail';
import UserCenter from './pages/UserCenter';
import TeacherDashboard from './pages/TeacherDashboard';
import TeacherCategory from './pages/TeacherCategory';
import ClassManagement from './pages/ClassManagement';
import Admin from './pages/Admin';

const App: React.FC = () => {
  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        token: {
          colorPrimary: '#1677ff',
          borderRadius: 6,
        },
      }}
    >
      <AntApp>
        <BrowserRouter>
          <Routes>
            <Route path="/" element={<Layout />}>
              <Route index element={<Home />} />
              <Route path="question/:id" element={<QuestionDetail />} />
              <Route path="login" element={<Login />} />
              <Route path="register" element={<Register />} />
              <Route path="user" element={<UserCenter />} />
              <Route path="teacher" element={<TeacherDashboard />} />
              <Route path="teacher/category/:id" element={<TeacherCategory />} />
              <Route path="classes" element={<ClassManagement />} />
              <Route path="admin" element={<Admin />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </AntApp>
    </ConfigProvider>
  );
};

export default App;
