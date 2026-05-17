import React, { useEffect, useState } from 'react';
import { Card, Tabs, List, Tag, Typography, Spin, Statistic, Row, Col, Progress, Space, Form, Input, Button, App, Popconfirm } from 'antd';
import { useNavigate } from 'react-router-dom';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  StarOutlined,
  EditOutlined,
  BarChartOutlined,
  UploadOutlined,
  SettingOutlined,
  LockOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import { userAPI, questionAPI } from '../api';
import type { UserAnswer, Question } from '../api';
import { useAuthStore } from '../store/auth';

const { Title, Text } = Typography;

const UserCenter: React.FC = () => {
  const { user } = useAuthStore();
  const navigate = useNavigate();

  const [answers, setAnswers] = useState<UserAnswer[]>([]);
  const [wrongAnswers, setWrongAnswers] = useState<UserAnswer[]>([]);
  const [bookmarks, setBookmarks] = useState<any[]>([]);
  const [stats, setStats] = useState<any>(null);
  const [uploads, setUploads] = useState<Question[]>([]);
  const [hasAIConfig, setHasAIConfig] = useState(false);
  const [savingAI, setSavingAI] = useState(false);
  const [loading, setLoading] = useState(true);
  const [changingPwd, setChangingPwd] = useState(false);
  const [aiForm] = Form.useForm();
  const [pwdForm] = Form.useForm();
  const { message } = App.useApp();

  const [answerPage, setAnswerPage] = useState(1);
  const [wrongPage, setWrongPage] = useState(1);
  const [bookmarkPage, setBookmarkPage] = useState(1);
  const [uploadPage, setUploadPage] = useState(1);
  const [answerTotal, setAnswerTotal] = useState(0);
  const [wrongTotal, setWrongTotal] = useState(0);
  const [bookmarkTotal, setBookmarkTotal] = useState(0);
  const [uploadTotal, setUploadTotal] = useState(0);

  const fetchAnswers = (page: number) => {
    userAPI.getAnswers({ page, page_size: 20 }).then((res) => {
      setAnswers(res.data.answers);
      setAnswerTotal(res.data.total);
      setAnswerPage(res.data.page);
    });
  };

  const fetchWrongAnswers = (page: number) => {
    userAPI.getWrongAnswers({ page, page_size: 20 }).then((res) => {
      setWrongAnswers(res.data.answers);
      setWrongTotal(res.data.total);
      setWrongPage(res.data.page);
    });
  };

  const fetchBookmarks = (page: number) => {
    userAPI.getBookmarks({ page, page_size: 20 }).then((res) => {
      setBookmarks(res.data.bookmarks);
      setBookmarkTotal(res.data.total);
      setBookmarkPage(res.data.page);
    });
  };

  const fetchUploads = (page: number) => {
    userAPI.getUploads({ page, page_size: 20 }).then((res) => {
      setUploads(res.data.questions || []);
      setUploadTotal(res.data.total);
      setUploadPage(res.data.page);
    });
  };

  useEffect(() => {
    setLoading(true);
    const isUploader = user?.role === 'teacher' || user?.role === 'director' || user?.role === 'principal';
    Promise.allSettled([
      userAPI.getAnswers({ page: 1, page_size: 20 }),
      userAPI.getWrongAnswers({ page: 1, page_size: 20 }),
      userAPI.getBookmarks({ page: 1, page_size: 20 }),
      userAPI.getStats(),
      isUploader ? userAPI.getUploads({ page: 1, page_size: 20 }) : Promise.resolve({ data: { questions: [], total: 0, page: 1, page_size: 20 } }),
      userAPI.getAIConfig(),
    ]).then((results) => {
      if (results[0].status === 'fulfilled') {
        const d = results[0].value.data;
        setAnswers(d.answers);
        setAnswerTotal(d.total);
      }
      if (results[1].status === 'fulfilled') {
        const d = results[1].value.data;
        setWrongAnswers(d.answers);
        setWrongTotal(d.total);
      }
      if (results[2].status === 'fulfilled') {
        const d = results[2].value.data;
        setBookmarks(d.bookmarks);
        setBookmarkTotal(d.total);
      }
      if (results[3].status === 'fulfilled') {
        setStats(results[3].value.data);
      }
      if (results[4].status === 'fulfilled') {
        const d = results[4].value.data;
        setUploads(d.questions || []);
        setUploadTotal(d.total);
      }
      if (results[5].status === 'fulfilled') {
        setHasAIConfig(results[5].value.data.has_config);
      }
    }).finally(() => setLoading(false));
  }, []);

  const handleChangePassword = async (values: { old_password: string; new_password: string }) => {
    setChangingPwd(true);
    try {
      await userAPI.changePassword(values);
      message.success('密码已修改');
      pwdForm.resetFields();
    } catch (err: any) { message.error(err.response?.data?.error || '修改失败'); }
    finally { setChangingPwd(false); }
  };

  const handleDeleteQuestion = async (id: number, title: string) => {
    try {
      await questionAPI.delete(id);
      message.success(`题目「${title}」已删除`);
      setUploads((prev) => prev.filter((q) => q.id !== id));
      setUploadTotal((prev) => prev - 1);
    } catch (err: any) { message.error(err.response?.data?.error || '删除失败'); }
  };

  const handleSaveAIConfig = async (values: { ai_api_key: string; ai_api_url: string; ai_model: string }) => {
    setSavingAI(true);
    try {
      await userAPI.updateAIConfig(values);
      setHasAIConfig(true);
      aiForm.resetFields();
      message.success('AI 配置已保存');
    } catch (err: any) {
      message.error(err.response?.data?.error || '保存失败');
    } finally {
      setSavingAI(false);
    }
  };

  const renderAnswerList = (data: UserAnswer[], emptyText: string, page: number, total: number, onPageChange: (page: number) => void) => (
    <List
      dataSource={data}
      locale={{ emptyText }}
      pagination={total > 20 ? {
        current: page,
        pageSize: 20,
        total,
        onChange: onPageChange,
        showSizeChanger: false,
      } : false}
      renderItem={(item) => (
        <List.Item
          style={{ cursor: 'pointer' }}
          onClick={() => navigate(`/question/${item.question_id}`)}
          actions={[
            <Tag color={item.is_qualified ? 'green' : 'red'}>
              {item.score}/10 {item.is_qualified ? '合格' : '不合格'}
            </Tag>,
          ]}
        >
          <List.Item.Meta
            avatar={<EditOutlined style={{ fontSize: 20, color: '#1677ff' }} />}
            title={item.question?.title || `题目 #${item.question_id}`}
            description={
              <Space>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {item.question?.category?.name}
                </Text>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  作答时间: {new Date(item.updated_at).toLocaleDateString('zh-CN')}
                </Text>
              </Space>
            }
          />
        </List.Item>
      )}
    />
  );

  if (loading) return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', paddingTop: 120 }} />;

  const tabItems = [
    {
      key: 'stats',
      label: <span><BarChartOutlined /> 学习概览</span>,
      children: (
        <div>
          {stats && (
            <>
              <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
                <Col xs={12} sm={12} md={6}><Card><Statistic title="总答题数" value={stats.total_answers} prefix={<EditOutlined />} /></Card></Col>
                <Col xs={12} sm={12} md={6}><Card><Statistic title="合格" value={stats.qualified_count} prefix={<CheckCircleOutlined style={{ color: '#52c41a' }} />} suffix={`/ ${stats.total_answers}`} /></Card></Col>
                <Col xs={12} sm={12} md={6}><Card><Statistic title="需改进" value={stats.wrong_count} prefix={<CloseCircleOutlined style={{ color: '#ff4d4f' }} />} /></Card></Col>
                <Col xs={12} sm={12} md={6}><Card><Statistic title="收藏" value={stats.total_bookmarks} prefix={<StarOutlined style={{ color: '#faad14' }} />} /></Card></Col>
              </Row>
              <Card title="平均得分" style={{ marginBottom: 16 }}>
                <div style={{ textAlign: 'center' }}>
                  <Progress
                    type="dashboard"
                    percent={Math.round((stats.average_score / 10) * 100)}
                    format={() => `${stats.average_score.toFixed(1)}/10`}
                    strokeColor={stats.average_score >= 7 ? '#52c41a' : '#ff4d4f'}
                    size={160}
                  />
                </div>
              </Card>
            </>
          )}
        </div>
      ),
    },
    {
      key: 'answers',
      label: <span><EditOutlined /> 我的回答 ({answerTotal})</span>,
      children: renderAnswerList(answers, '还没有回答过题目', answerPage, answerTotal, fetchAnswers),
    },
    {
      key: 'wrong',
      label: <span><CloseCircleOutlined style={{ color: '#ff4d4f' }} /> 我的错题 ({wrongTotal})</span>,
      children: renderAnswerList(wrongAnswers, '太棒了！没有需要改进的题目', wrongPage, wrongTotal, fetchWrongAnswers),
    },
    {
      key: 'bookmarks',
      label: <span><StarOutlined style={{ color: '#faad14' }} /> 我的收藏 ({bookmarkTotal})</span>,
      children: (
        <List
          dataSource={bookmarks}
          locale={{ emptyText: '还没有收藏题目' }}
          pagination={bookmarkTotal > 20 ? {
            current: bookmarkPage,
            pageSize: 20,
            total: bookmarkTotal,
            onChange: fetchBookmarks,
            showSizeChanger: false,
          } : false}
          renderItem={(item: any) => (
            <List.Item
              style={{ cursor: 'pointer' }}
              onClick={() => navigate(`/question/${item.question_id}`)}
            >
              <List.Item.Meta
                avatar={<StarOutlined style={{ fontSize: 20, color: '#faad14' }} />}
                title={item.question?.title || `题目 #${item.question_id}`}
                description={
                  <Space>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {item.question?.category?.name}
                    </Text>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      收藏于: {new Date(item.created_at).toLocaleDateString('zh-CN')}
                    </Text>
                  </Space>
                }
              />
            </List.Item>
          )}
        />
      ),
    },
    ...(user?.role === 'teacher' || user?.role === 'director' || user?.role === 'principal' ? [{
      key: 'uploads',
      label: <span><UploadOutlined /> 我上传的题目 ({uploadTotal})</span>,
      children: (
        <List
          dataSource={uploads}
          locale={{ emptyText: '还没有上传题目，去首页上传吧' }}
          pagination={uploadTotal > 20 ? {
            current: uploadPage,
            pageSize: 20,
            total: uploadTotal,
            onChange: fetchUploads,
            showSizeChanger: false,
          } : false}
          renderItem={(item: Question) => (
            <List.Item
              style={{ cursor: 'pointer' }}
              onClick={() => navigate(`/question/${item.id}`)}
              actions={[
                <Popconfirm
                  key="del"
                  title={`确定删除题目「${item.title}」？`}
                  description="该题目下的学生回答将被一并清除"
                  onConfirm={(e) => { e?.stopPropagation(); handleDeleteQuestion(item.id, item.title); }}
                  onCancel={(e) => { e?.stopPropagation(); }}
                  onPopupClick={(e) => e.stopPropagation()}
                >
                  <Button
                    type="text"
                    size="small"
                    danger
                    icon={<DeleteOutlined />}
                    onClick={(e) => e.stopPropagation()}
                  >
                    删除
                  </Button>
                </Popconfirm>,
              ]}
            >
              <List.Item.Meta
                avatar={<UploadOutlined style={{ fontSize: 20, color: '#1677ff' }} />}
                title={item.title}
                description={
                  <Space>
                    <Tag>{item.category?.name}</Tag>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      上传于: {new Date(item.created_at).toLocaleDateString('zh-CN')}
                    </Text>
                  </Space>
                }
              />
            </List.Item>
          )}
        />
      ),
    }] : []),
    {
      key: 'security',
      label: <span><LockOutlined /> 账号安全</span>,
      children: (
        <Form
          form={pwdForm}
          layout="vertical"
          onFinish={handleChangePassword}
          style={{ maxWidth: 400 }}
        >
          <Form.Item
            name="old_password"
            label="当前密码"
            rules={[{ required: true, message: '请输入当前密码' }]}
          >
            <Input.Password placeholder="输入当前密码" />
          </Form.Item>
          <Form.Item
            name="new_password"
            label="新密码"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 6, message: '新密码至少6位' },
            ]}
          >
            <Input.Password placeholder="输入新密码（至少6位）" />
          </Form.Item>
          <Form.Item
            name="confirm"
            label="确认新密码"
            dependencies={['new_password']}
            rules={[
              { required: true, message: '请再次输入新密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('new_password') === value) return Promise.resolve();
                  return Promise.reject(new Error('两次输入的密码不一致'));
                },
              }),
            ]}
          >
            <Input.Password placeholder="再次输入新密码" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={changingPwd}>
              修改密码
            </Button>
          </Form.Item>
        </Form>
      ),
    },
    {
      key: 'ai',
      label: <span><SettingOutlined /> AI 设置</span>,
      children: (
        <Form
          form={aiForm}
          layout="vertical"
          onFinish={handleSaveAIConfig}
          style={{ maxWidth: 480 }}
        >
          {hasAIConfig && (
            <div style={{ marginBottom: 16, color: '#52c41a' }}>已配置 AI 接口，修改时仅填写需要更新的字段即可</div>
          )}
          <Form.Item
            name="ai_api_key"
            label="API Key"
            rules={hasAIConfig ? [] : [{ required: true, message: '首次配置请输入 API Key' }]}
          >
            <Input.Password placeholder="sk-..." />
          </Form.Item>
          <Form.Item
            name="ai_api_url"
            label="API 地址（完整端点）"
            rules={hasAIConfig ? [] : [{ required: true, message: '首次配置请输入 API 地址' }]}
          >
            <Input placeholder="https://api.deepseek.com/chat/completions" />
          </Form.Item>
          <Form.Item
            name="ai_model"
            label="模型名称"
          >
            <Input placeholder="deepseek-chat" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={savingAI}>
              保存配置
            </Button>
          </Form.Item>
        </Form>
      ),
    },
  ];

  return (
    <div style={{ maxWidth: 900, margin: '0 auto' }}>
      <Card style={{ borderRadius: 8, marginBottom: 16 }}>
        <Space>
          <div
            style={{
              width: 48,
              height: 48,
              borderRadius: '50%',
              background: '#1677ff',
              color: '#fff',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 20,
              fontWeight: 600,
            }}
          >
            {user?.username?.[0]?.toUpperCase() || 'U'}
          </div>
          <div>
            <Title level={4} style={{ margin: 0 }}>{user?.username}</Title>
            <Text type="secondary">{user?.email}</Text>
          </div>
        </Space>
      </Card>

      <Card style={{ borderRadius: 8 }}>
        <Tabs items={tabItems} />
      </Card>
    </div>
  );
};

export default UserCenter;
