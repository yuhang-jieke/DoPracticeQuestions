import React, { useEffect, useState } from 'react';
import { Card, Table, Tag, Typography, Space, Button, Select, Modal, Input, Popconfirm, App, Form } from 'antd';
import { TeamOutlined, AppstoreOutlined, UserAddOutlined } from '@ant-design/icons';
import { adminAPI, categoryAPI, classAPI } from '../api';
import type { AdminUser } from '../api';

const { Title, Text } = Typography;

const Admin: React.FC = () => {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [roleFilter, setRoleFilter] = useState('');
  const [classFilter, setClassFilter] = useState('');
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [canEdit, setCanEdit] = useState(false);
  const [roleModal, setRoleModal] = useState<{ id: number; role: string } | null>(null);
  const [catModal, setCatModal] = useState(false);
  const [catName, setCatName] = useState('');
  const [catType, setCatType] = useState<'tech' | 'non-tech'>('tech');
  const [categories, setCategories] = useState<any[]>([]);
  const [classes, setClasses] = useState<any[]>([]);
  const [createUserOpen, setCreateUserOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [userForm] = Form.useForm();
  const { message } = App.useApp();

  const isPrincipal = canEdit;

  useEffect(() => {
    setLoading(true);
    adminAPI.getUsers({ page, role: roleFilter || undefined, search: search || undefined, class_id: classFilter || undefined })
      .then((res) => { setUsers(res.data.users); setTotal(res.data.total); setCanEdit(res.data.can_edit); })
      .finally(() => setLoading(false));
  }, [page, roleFilter, search, classFilter]);

  useEffect(() => {
    categoryAPI.getAll().then((res) => setCategories(res.data.categories));
    classAPI.getAll().then((r) => setClasses(r.data.classes)).catch(() => {});
  }, []);

  const handleRoleChange = async () => {
    if (!roleModal) return;
    try {
      await adminAPI.updateRole(roleModal.id, roleModal.role);
      message.success('角色已更新');
      setRoleModal(null);
      setUsers((prev) => prev.map((u) => (u.id === roleModal.id ? { ...u, role: roleModal.role } : u)));
    } catch (err: any) {
      message.error(err.response?.data?.error || '操作失败');
    }
  };

  const handleCreateUser = async (values: { username: string; email: string; password: string; role: string }) => {
    setCreating(true);
    try {
      await adminAPI.createUser(values);
      message.success('用户已创建');
      setCreateUserOpen(false);
      userForm.resetFields();
      setPage(1);
      const res = await adminAPI.getUsers({ page: 1, role: roleFilter || undefined, search: search || undefined, class_id: classFilter || undefined });
      setUsers(res.data.users); setTotal(res.data.total);
    } catch (err: any) { message.error(err.response?.data?.error || '创建失败'); }
    finally { setCreating(false); }
  };

  const handleResetPassword = async (userId: number, username: string) => {
    try {
      await adminAPI.resetPassword(userId);
      message.success(`${username} 的密码已重置为 123456`);
    } catch (err: any) { message.error(err.response?.data?.error || '操作失败'); }
  };

  const handleDelete = async (userId: number) => {
    try {
      await adminAPI.deleteUser(userId);
      message.success('用户已删除');
      setUsers((prev) => prev.filter((u) => u.id !== userId));
    } catch (err: any) {
      message.error(err.response?.data?.error || '删除失败');
    }
  };

  const roleColors: Record<string, string> = { student: 'blue', teacher: 'green', director: 'orange', principal: 'red' };
  const roleLabels: Record<string, string> = { student: '学生', teacher: '老师', director: '主任', principal: '校长' };

  return (
    <div style={{ maxWidth: 1000, margin: '0 auto' }}>
      <Title level={3}><TeamOutlined style={{ marginRight: 8 }} />用户管理</Title>
      <Card style={{ borderRadius: 8, marginBottom: 24 }}>
        <Space style={{ marginBottom: 16 }} wrap>
          <Button type="primary" icon={<UserAddOutlined />} onClick={() => setCreateUserOpen(true)}>创建用户</Button>
          <Input.Search
            placeholder="搜索用户名/邮箱"
            allowClear
            style={{ width: 240 }}
            onSearch={(v) => { setSearch(v); setPage(1); }}
          />
          <Select
            allowClear
            placeholder="角色筛选"
            style={{ width: 120 }}
            value={roleFilter || undefined}
            onChange={(v) => { setRoleFilter(v || ''); setPage(1); }}
            options={[
              { value: 'student', label: '学生' }, { value: 'teacher', label: '老师' },
              { value: 'director', label: '主任' }, { value: 'principal', label: '校长' },
            ]}
          />
          <Select
            allowClear
            placeholder="班级筛选"
            style={{ width: 160 }}
            value={classFilter || undefined}
            onChange={(v) => { setClassFilter(v || ''); setPage(1); }}
            options={classes.map((c: any) => ({ value: String(c.id), label: c.name }))}
          />
        </Space>
        <Table
          dataSource={users}
          rowKey="id"
          loading={loading}
          pagination={{ current: page, total, pageSize: 20, showTotal: (t) => `共 ${t} 人`, onChange: (p) => setPage(p) }}
          scroll={{ x: 'max-content' }}
          columns={[
            { title: 'ID', dataIndex: 'id', width: 60 },
            { title: '用户名', dataIndex: 'username', width: 100 },
            { title: '邮箱', dataIndex: 'email', ellipsis: true },
            { title: '角色', dataIndex: 'role', width: 70, render: (r: string) => <Tag color={roleColors[r]}>{roleLabels[r] || r}</Tag> },
            { title: '班级', dataIndex: 'class_name', width: 100, render: (v: string | null) => v ? <Tag>{v}</Tag> : <Text type="secondary">—</Text> },
            { title: '操作', width: 160, render: (_: any, u: AdminUser) => (
              <Space size="small">
                <Button size="small" onClick={() => setRoleModal({ id: u.id, role: u.role })}>改角色</Button>
                <Popconfirm title={`重置 ${u.username} 的密码为 123456？`} onConfirm={() => handleResetPassword(u.id, u.username)}>
                  <Button size="small">重置密码</Button>
                </Popconfirm>
                {isPrincipal && u.role !== 'principal' && (
                  <Popconfirm title="确定删除此用户？" onConfirm={() => handleDelete(u.id)}>
                    <Button size="small" danger>删除</Button>
                  </Popconfirm>
                )}
              </Space>
            )},
          ]}
        />
      </Card>

      {isPrincipal && (
        <>
          <Title level={3} style={{ marginTop: 32 }}><AppstoreOutlined style={{ marginRight: 8 }} />分类管理</Title>
          <Card style={{ borderRadius: 8 }}>
            <Button type="primary" onClick={() => setCatModal(true)} style={{ marginBottom: 16 }}>新增分类</Button>
            <Table
              dataSource={categories}
              rowKey="id"
              pagination={false}
              columns={[
                { title: '名称', dataIndex: 'name' },
                { title: '类型', dataIndex: 'type', width: 100, render: (t: string) => <Tag>{t === 'tech' ? '技术' : '非技术'}</Tag> },
                { title: '操作', width: 160, render: (_: any, c: any) => (
                  <Space>
                    <Button size="small" onClick={() => {
                      setCatModal(true);
                      // pre-fill not implemented, just use for creating
                    }}>编辑</Button>
                    <Popconfirm title="确定删除？" onConfirm={async () => {
                      await adminAPI.deleteCategory(c.id);
                      setCategories((prev) => prev.filter((x) => x.id !== c.id));
                      message.success('已删除');
                    }}>
                      <Button size="small" danger>删除</Button>
                    </Popconfirm>
                  </Space>
                )},
              ]}
            />
          </Card>
          <Modal title="新增分类" open={catModal} onCancel={() => setCatModal(false)} onOk={async () => {
            try {
              await adminAPI.createCategory({ name: catName, type: catType });
              message.success('分类已创建');
              setCatModal(false);
            } catch (err: any) { message.error(err.response?.data?.error || '创建失败'); }
          }}>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Input placeholder="分类名称" value={catName} onChange={(e) => setCatName(e.target.value)} />
              <Select value={catType} onChange={(v) => setCatType(v)} options={[{ value: 'tech', label: '技术' }, { value: 'non-tech', label: '非技术' }]} />
            </Space>
          </Modal>
        </>
      )}

      <Modal
        title="创建用户"
        open={createUserOpen}
        onCancel={() => { setCreateUserOpen(false); userForm.resetFields(); }}
        onOk={() => userForm.submit()}
        confirmLoading={creating}
      >
        <Form form={userForm} layout="vertical" onFinish={handleCreateUser}>
          <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }, { min: 2, max: 50 }]}>
            <Input placeholder="用户名" />
          </Form.Item>
          <Form.Item name="email" label="邮箱" rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '邮箱格式不正确' }]}>
            <Input placeholder="student@test.com" />
          </Form.Item>
          <Form.Item name="password" label="初始密码" rules={[{ required: true, message: '请输入初始密码' }, { min: 6, message: '密码至少6位' }]}>
            <Input.Password placeholder="初始密码（至少6位）" />
          </Form.Item>
          <Form.Item name="role" label="角色" rules={[{ required: true, message: '请选择角色' }]}>
            <Select
              placeholder="选择角色"
              options={[
                { value: 'student', label: '学生' }, { value: 'teacher', label: '老师' },
                ...(isPrincipal ? [{ value: 'director', label: '主任' }, { value: 'principal', label: '校长' }] : []),
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title="修改角色" open={roleModal !== null} onCancel={() => setRoleModal(null)} onOk={handleRoleChange}>
        <Select
          style={{ width: '100%' }}
          value={roleModal?.role}
          onChange={(v) => setRoleModal((prev) => prev ? { ...prev, role: v } : null)}
          options={[
            { value: 'student', label: '学生' }, { value: 'teacher', label: '老师' },
            ...(isPrincipal ? [{ value: 'director', label: '主任' }, { value: 'principal', label: '校长' }] : []),
          ]}
        />
      </Modal>
    </div>
  );
};

export default Admin;
