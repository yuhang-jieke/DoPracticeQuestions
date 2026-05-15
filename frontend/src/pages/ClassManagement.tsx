import React, { useEffect, useState } from 'react';
import { Card, Table, Tag, Typography, Space, Button, Modal, Input, Popconfirm, App, Select, Upload, List, Spin } from 'antd';
import { PlusOutlined, TeamOutlined, DownloadOutlined, UploadOutlined, InboxOutlined } from '@ant-design/icons';
import { classAPI, adminAPI, type ImportRowResult, type ConfirmRowResult, type PreviewImportResponse, type ConfirmImportResponse } from '../api';
import type { ClassItem } from '../api';
import { useAuthStore } from '../store/auth';

const { Title, Text } = Typography;

const ClassManagement: React.FC = () => {
  const { user } = useAuthStore();
  const [classes, setClasses] = useState<ClassItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [createModal, setCreateModal] = useState(false);
  const [newName, setNewName] = useState('');
  const [studentModal, setStudentModal] = useState<{ classId: number } | null>(null);
  const [students, setStudents] = useState<any[]>([]);
  const [studentPage, setStudentPage] = useState(1);
  const [studentTotal, setStudentTotal] = useState(0);
  const [studentLoading, setStudentLoading] = useState(false);
  const [addStudentModal, setAddStudentModal] = useState<number | null>(null);
  const [addStudentId, setAddStudentId] = useState('');
  const [teachers, setTeachers] = useState<any[]>([]);
  const [importOpen, setImportOpen] = useState(false);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importStep, setImportStep] = useState<'upload' | 'preview' | 'loading' | 'result'>('upload');
  const [previewData, setPreviewData] = useState<PreviewImportResponse | null>(null);
  const [confirmData, setConfirmData] = useState<ConfirmImportResponse | null>(null);
  const [decisions, setDecisions] = useState<Record<number, string>>({});
  const { message } = App.useApp();

  const isDirectorOrAbove = user?.role === 'director' || user?.role === 'principal';

  useEffect(() => {
    loadClasses();
  }, []);

  useEffect(() => {
    if (isDirectorOrAbove) {
      adminAPI.getUsers({ role: 'teacher' }).then((r) => setTeachers(r.data.users));
    }
  }, [user]);

  const loadClasses = () => {
    classAPI.getAll().then((res) => setClasses(res.data.classes)).finally(() => setLoading(false));
  };

  const handleCreate = async () => {
    if (!newName.trim()) { message.warning('请输入班级名称'); return; }
    await classAPI.create(newName);
    message.success('班级已创建');
    setNewName(''); setCreateModal(false);
    loadClasses();
  };

  const handleDelete = async (id: number) => {
    await classAPI.delete(id);
    message.success('班级已删除');
    loadClasses();
  };

  const loadStudents = async (classId: number, page = 1) => {
    setStudentLoading(true);
    const res = await classAPI.getStudents(classId, { page, page_size: 15 });
    setStudents(res.data.students);
    setStudentTotal(res.data.total);
    setStudentPage(page);
    setStudentModal({ classId });
    setStudentLoading(false);
  };

  const handleAddStudent = async () => {
    if (addStudentModal === null || !addStudentId.trim()) return;
    try {
      await classAPI.addStudent(addStudentModal, Number(addStudentId));
      message.success('学生已添加');
      setAddStudentModal(null); setAddStudentId('');
      if (studentModal) loadStudents(studentModal.classId);
    } catch (err: any) { message.error(err.response?.data?.error || '添加失败'); }
  };

  const handleRemoveStudent = async (classId: number, userId: number) => {
    await classAPI.removeStudent(classId, userId);
    message.success('学生已移除');
    if (studentModal) loadStudents(studentModal.classId);
  };

  const handleAssignTeacher = async (classId: number, teacherId: string) => {
    if (!teacherId) return;
    await classAPI.assignTeacher(classId, Number(teacherId));
    message.success('教师已分配');
    loadClasses();
  };

  const handlePreview = async () => {
    if (!importFile) { message.warning('请选择文件'); return; }
    setImportStep('loading');
    try {
      const res = await classAPI.previewImportStudents(importFile);
      setPreviewData(res.data);
      const defaults: Record<number, string> = {};
      res.data.results.forEach((r) => {
        if (r.status === 'need_confirm') {
          defaults[r.row] = r.conflict_type === 'cross_class' ? 'move' : 'create';
        }
      });
      setDecisions(defaults);
      setImportStep('preview');
    } catch (err: any) { message.error(err.response?.data?.error || '预览失败'); setImportStep('upload'); }
  };

  const handleConfirm = async () => {
    if (!previewData) return;
    const items = previewData.results
      .filter((r) => r.status !== 'invalid')
      .map((r) => ({
        row: r.row,
        name: r.name,
        class_name: r.class_name,
        email: r.email,
        action: r.status === 'importable' ? 'create' : (decisions[r.row] || 'skip'),
      }));
    setImportStep('loading');
    try {
      const res = await classAPI.confirmImportStudents(items);
      setConfirmData(res.data);
      setImportStep('result');
      loadClasses();
    } catch (err: any) { message.error(err.response?.data?.error || '导入失败'); setImportStep('preview'); }
  };

  return (
    <div style={{ maxWidth: 900, margin: '0 auto' }}>
      <Title level={3}><TeamOutlined style={{ marginRight: 8 }} />班级管理</Title>
      <Card style={{ borderRadius: 8 }}>
        <Space style={{ marginBottom: 16 }} wrap>
          {isDirectorOrAbove && (
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateModal(true)}>创建班级</Button>
          )}
          <Button icon={<DownloadOutlined />} onClick={() => classAPI.downloadStudentTemplate()}>下载模板</Button>
          <Button icon={<UploadOutlined />} onClick={() => { setImportOpen(true); setImportFile(null); setImportStep('upload'); setPreviewData(null); setConfirmData(null); setDecisions({}); }}>批量导入学生</Button>
        </Space>
        <Table
          dataSource={classes}
          rowKey="id"
          loading={loading}
          pagination={false}
          columns={[
            { title: 'ID', dataIndex: 'id', width: 60 },
            { title: '班级名称', dataIndex: 'name' },
            { title: '人数', dataIndex: 'student_count', width: 60 },
            { title: '教师', dataIndex: 'teacher_id', width: 120, render: (v: number) => {
              if (v === user?.id) return <span>{user?.username}</span>;
              const t = teachers.find((t: any) => t.id === v);
              return <span>{t?.username || `ID:${v}`}</span>;
            }},
            { title: '操作', width: 280, render: (_: any, c: ClassItem) => (
              <Space size="small">
                <Button size="small" onClick={() => loadStudents(c.id)}>学生列表</Button>
                {isDirectorOrAbove && (
                  <>
                    <Button size="small" onClick={() => setAddStudentModal(c.id)}>添加学生</Button>
                    <Select
                      size="small"
                      style={{ width: 80 }}
                      placeholder="分配老师"
                      value=""
                      onChange={(v) => handleAssignTeacher(c.id, v)}
                      options={teachers.map((t) => ({ value: String(t.id), label: t.username }))}
                    />
                    <Popconfirm title="确定删除？" onConfirm={() => handleDelete(c.id)}>
                      <Button size="small" danger>删除</Button>
                    </Popconfirm>
                  </>
                )}
              </Space>
            )},
          ]}
        />
      </Card>

      <Modal title="创建班级" open={createModal} onCancel={() => setCreateModal(false)} onOk={handleCreate}>
        <Input placeholder="班级名称" value={newName} onChange={(e) => setNewName(e.target.value)} />
      </Modal>

      <Modal title="班级学生" open={studentModal !== null} onCancel={() => setStudentModal(null)} footer={null} width={500}>
        <Table
          dataSource={students}
          rowKey="id"
          loading={studentLoading}
          pagination={{
            current: studentPage,
            total: studentTotal,
            pageSize: 15,
            showTotal: (t: number) => `共 ${t} 人`,
            onChange: (p: number) => studentModal && loadStudents(studentModal.classId, p),
          }}
          columns={[
            { title: 'ID', dataIndex: 'id', width: 60 },
            { title: '用户名', dataIndex: 'username' },
            { title: '邮箱', dataIndex: 'email', ellipsis: true },
            { title: '角色', dataIndex: 'role', width: 80, render: (v: string) => <Tag>{v}</Tag> },
            { title: '操作', width: 80, render: (_: any, s: any) => (
              <Popconfirm title="移除该学生？" onConfirm={() => handleRemoveStudent(studentModal!.classId, s.id)}>
                <Button size="small" danger>移除</Button>
              </Popconfirm>
            )},
          ]}
        />
        <Button type="dashed" block style={{ marginTop: 16 }} onClick={() => studentModal && setAddStudentModal(studentModal.classId)}>添加学生</Button>
      </Modal>

      <Modal title="添加学生" open={addStudentModal !== null} onCancel={() => setAddStudentModal(null)} onOk={handleAddStudent}>
        <Input placeholder="输入学生 ID" value={addStudentId} onChange={(e) => setAddStudentId(e.target.value)} />
      </Modal>

      <Modal
        title="批量导入学生"
        open={importOpen}
        onCancel={() => setImportOpen(false)}
        width={700}
        footer={null}
      >
        {importStep === 'upload' && (
          <>
            <Upload.Dragger
              accept=".xlsx"
              maxCount={1}
              beforeUpload={(file) => { setImportFile(file); return false; }}
              onRemove={() => setImportFile(null)}
            >
              <p className="ant-upload-drag-icon"><InboxOutlined /></p>
              <p className="ant-upload-text">点击或拖拽 .xlsx 文件到此处</p>
              <p className="ant-upload-hint">A列：学生姓名 | B列：班级 | C列：邮箱号</p>
            </Upload.Dragger>
            <div style={{ marginTop: 16, textAlign: 'right' }}>
              <Button onClick={() => setImportOpen(false)} style={{ marginRight: 8 }}>取消</Button>
              <Button type="primary" onClick={handlePreview} disabled={!importFile}>预览</Button>
            </div>
          </>
        )}

        {importStep === 'loading' && (
          <div style={{ textAlign: 'center', padding: '40px 0' }}>
            <Spin size="large" />
            <p style={{ marginTop: 12, color: '#999' }}>正在解析文件...</p>
          </div>
        )}

        {importStep === 'preview' && previewData && (
          <>
            <div style={{ marginBottom: 16 }}>
              <Space size="middle">
                <Tag color="green">可导入 {previewData.importable}</Tag>
                <Tag color="orange">需确认 {previewData.need_confirm}</Tag>
                <Tag color="red">无效 {previewData.invalid}</Tag>
              </Space>
              <Text type="secondary" style={{ marginLeft: 8 }}>共 {previewData.total} 条</Text>
            </div>

            {previewData.class_summary.length > 0 && (
              <Card size="small" title="按班级汇总" style={{ marginBottom: 16 }}>
                {previewData.class_summary.map((cs) => (
                  <div key={cs.class_name} style={{ marginBottom: 4 }}>
                    <Text strong>{cs.class_name}</Text>
                    <Space size="small" style={{ marginLeft: 12 }}>
                      {cs.importable > 0 && <Tag color="green" style={{ fontSize: 12 }}>可导入 {cs.importable}</Tag>}
                      {cs.need_confirm > 0 && <Tag color="orange" style={{ fontSize: 12 }}>待确认 {cs.need_confirm}</Tag>}
                      {cs.invalid > 0 && <Tag color="red" style={{ fontSize: 12 }}>无效 {cs.invalid}</Tag>}
                    </Space>
                  </div>
                ))}
              </Card>
            )}

            <div style={{ maxHeight: 360, overflow: 'auto' }}>
              <List
                size="small"
                dataSource={previewData.results}
                renderItem={(r: ImportRowResult) => (
                  <List.Item
                    style={{
                      background: r.status === 'invalid' ? '#fff2f0' : r.status === 'need_confirm' ? '#fffbe6' : '#f6ffed',
                      padding: '8px 12px', marginBottom: 2, borderRadius: 4,
                    }}
                  >
                    <div style={{ width: '100%' }}>
                      <Space>
                        <Text type="secondary" style={{ fontSize: 12 }}>第{r.row}行</Text>
                        <Text>{r.name}</Text>
                        <Text type="secondary">{r.class_name}</Text>
                        <Text type="secondary" style={{ fontSize: 12 }}>{r.email}</Text>
                      </Space>
                      <div style={{ marginTop: 4 }}>
                        {r.status === 'importable' && <Tag color="green">可导入</Tag>}
                        {r.status === 'invalid' && (
                          <Space>
                            <Tag color="red">无法导入</Tag>
                            <Text type="secondary" style={{ fontSize: 12 }}>{r.reason}</Text>
                          </Space>
                        )}
                        {r.status === 'need_confirm' && (
                          <Space>
                            <Tag color="orange">需确认</Tag>
                            <Text type="secondary" style={{ fontSize: 12 }}>{r.reason}</Text>
                            <Select
                              size="small"
                              value={decisions[r.row] || (r.conflict_type === 'cross_class' ? 'move' : 'create')}
                              onChange={(v) => setDecisions((prev) => ({ ...prev, [r.row]: v }))}
                              style={{ width: 140 }}
                              options={
                                r.conflict_type === 'cross_class'
                                  ? [{ value: 'move', label: '移动到新班级' }, { value: 'skip', label: '跳过' }]
                                  : [{ value: 'create', label: '是另一个人' }, { value: 'skip', label: '跳过（重复）' }]
                              }
                            />
                          </Space>
                        )}
                      </div>
                    </div>
                  </List.Item>
                )}
              />
            </div>

            <div style={{ marginTop: 16, textAlign: 'right' }}>
              <Button onClick={() => setImportOpen(false)} style={{ marginRight: 8 }}>取消</Button>
              <Button type="primary" onClick={handleConfirm} disabled={previewData.importable + previewData.need_confirm === 0}>确认导入</Button>
            </div>
          </>
        )}

        {importStep === 'result' && confirmData && (
          <>
            <div style={{ marginBottom: 16 }}>
              <Space size="middle">
                <Tag color="green">创建 {confirmData.created} 人</Tag>
                <Tag color="blue">移动 {confirmData.moved} 人</Tag>
                <Tag color="orange">跳过 {confirmData.skipped} 条</Tag>
              </Space>
            </div>

            {confirmData.class_results.length > 0 && (
              <Card size="small" title="按班级汇总" style={{ marginBottom: 16 }}>
                {confirmData.class_results.map((cr) => (
                  <div key={cr.class_name} style={{ marginBottom: 4 }}>
                    <Text strong>{cr.class_name}</Text>
                    <Space size="small" style={{ marginLeft: 12 }}>
                      {cr.created > 0 && <Tag color="green" style={{ fontSize: 12 }}>新创建 {cr.created}</Tag>}
                      {cr.moved > 0 && <Tag color="blue" style={{ fontSize: 12 }}>移动转入 {cr.moved}</Tag>}
                    </Space>
                  </div>
                ))}
              </Card>
            )}

            <div style={{ maxHeight: 300, overflow: 'auto' }}>
              <List
                size="small"
                dataSource={confirmData.results}
                renderItem={(r: ConfirmRowResult) => (
                  <List.Item style={{ padding: '6px 12px' }}>
                    <Space>
                      <Text type="secondary" style={{ fontSize: 12 }}>第{r.row}行</Text>
                      <Text>{r.name}</Text>
                      <Tag color={r.status === 'created' ? 'green' : r.status === 'moved' ? 'blue' : 'orange'}>
                        {r.status === 'created' ? '已创建' : r.status === 'moved' ? '已移动' : '已跳过'}
                      </Tag>
                      {r.reason && <Text type="secondary" style={{ fontSize: 12 }}>{r.reason}</Text>}
                    </Space>
                  </List.Item>
                )}
              />
            </div>

            <div style={{ marginTop: 16, textAlign: 'right' }}>
              <Button type="primary" onClick={() => setImportOpen(false)}>完成</Button>
            </div>
          </>
        )}
      </Modal>
    </div>
  );
};

export default ClassManagement;
