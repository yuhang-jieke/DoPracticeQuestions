import React, { useEffect, useState } from 'react';
import { Card, Table, Tag, Typography, Spin, Statistic, Row, Col, Modal, List, Button, Space, Select } from 'antd';
import { BarChartOutlined, TeamOutlined, FileTextOutlined, PercentageOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { teacherAPI, classAPI } from '../api';
import type { UserAnswer } from '../api';

const { Title, Text, Paragraph } = Typography;

const TeacherDashboard: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [overview, setOverview] = useState<any>(null);
  const [students, setStudents] = useState<any[]>([]);
  const [categoryStats, setCategoryStats] = useState<any[]>([]);
  const [studentAnswers, setStudentAnswers] = useState<UserAnswer[]>([]);
  const [studentModal, setStudentModal] = useState<string | null>(null);
  const [classId, setClassId] = useState('');
  const [classes, setClasses] = useState<any[]>([]);
  const navigate = useNavigate();

  useEffect(() => {
    classAPI.getAll().then((res) => setClasses(res.data.classes)).catch(() => {});
  }, []);

  useEffect(() => {
    setLoading(true);
    const params = classId ? { class_id: classId } : {};
    Promise.allSettled([
      teacherAPI.getOverview(params),
      teacherAPI.getStudents(params),
      teacherAPI.getCategoryStats(params),
    ]).then(([oRes, sRes, cRes]) => {
      if (oRes.status === 'fulfilled') setOverview(oRes.value.data.overview);
      if (sRes.status === 'fulfilled') setStudents(sRes.value.data.students || []);
      if (cRes.status === 'fulfilled') setCategoryStats(cRes.value.data.categories || []);
    }).finally(() => setLoading(false));
  }, [classId]);

  const handleViewStudent = async (userId: string) => {
    setStudentModal(userId);
    const res = await teacherAPI.getStudentAnswers(userId);
    setStudentAnswers(res.data.answers || []);
  };

  if (loading) return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', paddingTop: 120 }} />;

  return (
    <div style={{ maxWidth: 1100, margin: '0 auto' }}>
      <Space style={{ marginBottom: 24, justifyContent: 'space-between', width: '100%' }} align="center">
        <Title level={3} style={{ margin: 0 }}>
          <BarChartOutlined style={{ marginRight: 8 }} />
          教师后台
        </Title>
        {classes.length > 0 && (
          <Select
            allowClear
            placeholder="全部班级"
            style={{ width: 200 }}
            value={classId || undefined}
            onChange={(v) => setClassId(v || '')}
            options={classes.map((c: any) => ({ value: String(c.id), label: c.name }))}
          />
        )}
      </Space>

      {overview && (
        <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
          <Col xs={12} sm={12} md={6}><Card><Statistic title="答题学生数" value={overview.student_count} prefix={<TeamOutlined />} /></Card></Col>
          <Col xs={12} sm={12} md={6}><Card><Statistic title="总答题次数" value={overview.total_answers} prefix={<FileTextOutlined />} /></Card></Col>
          <Col xs={12} sm={12} md={6}><Card><Statistic title="整体平均分" value={`${overview.average_score.toFixed(1)}/10`} /></Card></Col>
          <Col xs={12} sm={12} md={6}><Card><Statistic title="合格率" value={`${overview.qualified_rate.toFixed(1)}%`} prefix={<PercentageOutlined />} /></Card></Col>
        </Row>
      )}

      {categoryStats.length > 0 && (
        <Card title="📈 分类错误率" style={{ borderRadius: 8, marginBottom: 24 }}>
          <LineChart
            data={categoryStats}
            onPointClick={(catId: number) => navigate(`/teacher/category/${catId}`)}
          />
        </Card>
      )}

      <Card title={<span><TeamOutlined /> 学生刷题情况</span>} style={{ borderRadius: 8 }}>
        <Table
          dataSource={students}
          rowKey="user_id"
          scroll={{ x: 'max-content' }}
          pagination={{ pageSize: 20 }}
          columns={[
            { title: '姓名', dataIndex: 'username', width: 120 },
            { title: '邮箱', dataIndex: 'email', ellipsis: true },
            { title: '答题数', dataIndex: 'answer_count', width: 80 },
            { title: '平均分', dataIndex: 'average_score', width: 90, render: (v: number) => <Tag color={v >= 7 ? 'green' : 'red'}>{v.toFixed(1)}</Tag> },
            { title: '合格', dataIndex: 'qualified', width: 70 },
            { title: '错题', dataIndex: 'wrong', width: 70 },
            { title: '最近答题', dataIndex: 'last_answer', width: 120, render: (v: string) => new Date(v).toLocaleDateString('zh-CN') },
            { title: '操作', width: 100, render: (_: any, r: any) => <Button type="link" size="small" onClick={() => handleViewStudent(String(r.user_id))}>查看详情</Button> },
          ]}
        />
      </Card>

      <Modal title="学生答题详情" open={studentModal !== null} onCancel={() => setStudentModal(null)} footer={null} width={700}>
        <List
          dataSource={studentAnswers}
          renderItem={(item: UserAnswer) => (
            <List.Item>
              <List.Item.Meta
                title={<Space><Tag color={item.is_qualified ? 'green' : 'red'}>{item.score}/10</Tag>{item.question?.title || `题目 #${item.question_id}`}</Space>}
                description={
                  <div>
                    <Text type="secondary" style={{ fontSize: 12 }}>{item.question?.category?.name} · {new Date(item.updated_at).toLocaleString('zh-CN')}</Text>
                    <Paragraph ellipsis={{ rows: 2, expandable: true }} style={{ marginTop: 4, fontSize: 13 }}>{item.content}</Paragraph>
                  </div>
                }
              />
            </List.Item>
          )}
        />
      </Modal>
    </div>
  );
};

const LineChart: React.FC<{ data: any[]; onPointClick: (id: number) => void }> = ({ data, onPointClick }) => {
  if (data.length === 0) return <Text type="secondary">暂无数据</Text>;

  const w = 1000, h = 240, padL = 60, padR = 30, padT = 20, padB = 40;
  const chartW = w - padL - padR;
  const chartH = h - padT - padB;

  const maxRate = Math.max(10, ...data.map((d: any) => d.fail_rate));
  const step = chartW / (data.length - 1 || 1);

  const points = data.map((d: any, i: number) => ({
    x: padL + i * step,
    y: padT + chartH - (d.fail_rate / maxRate) * chartH,
    ...d,
  }));

  const pathD = points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ');
  const yTicks = 5;

  return (
    <svg viewBox={`0 0 ${w} ${h}`} style={{ width: '100%', maxWidth: 1050 }}>
      <line x1={padL} y1={padT} x2={padL} y2={padT + chartH} stroke="#ddd" />
      <line x1={padL} y1={padT + chartH} x2={padL + chartW} y2={padT + chartH} stroke="#ddd" />
      {Array.from({ length: yTicks + 1 }, (_, i) => {
        const y = padT + (chartH / yTicks) * i;
        const val = maxRate - (maxRate / yTicks) * i;
        return (
          <g key={i}>
            <line x1={padL - 4} y1={y} x2={padL} y2={y} stroke="#ddd" />
            <text x={padL - 8} y={y + 4} textAnchor="end" fontSize={11} fill="#999">{val.toFixed(0)}%</text>
            <line x1={padL} y1={y} x2={padL + chartW} y2={y} stroke="#f0f0f0" strokeDasharray="4 4" />
          </g>
        );
      })}
      <path d={pathD} fill="none" stroke="#1677ff" strokeWidth={2} />
      {points.map((p) => (
        <g key={p.category_id} style={{ cursor: 'pointer' }} onClick={() => onPointClick(p.category_id)}>
          <circle cx={p.x} cy={p.y} r={5} fill="#1677ff" />
          <circle cx={p.x} cy={p.y} r={12} fill="transparent" />
          <text x={p.x} y={padT + chartH + 18} textAnchor="middle" transform={data.length > 6 ? `rotate(-30, ${p.x}, ${padT + chartH + 18})` : undefined} fontSize={11} fill="#666">{p.category_name}</text>
        </g>
      ))}
      {points.map((p) => (
        <text key={'val-' + p.category_id} x={p.x} y={p.y - 14} textAnchor="middle" fontSize={11} fill="#1677ff" fontWeight={500}>{p.fail_rate.toFixed(1)}%</text>
      ))}
    </svg>
  );
};

export default TeacherDashboard;
