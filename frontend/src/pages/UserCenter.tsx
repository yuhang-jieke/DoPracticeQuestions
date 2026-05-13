import React, { useEffect, useState } from 'react';
import { Card, Tabs, List, Tag, Typography, Spin, Empty, Statistic, Row, Col, Progress, Space, Form, Input, Button, App } from 'antd';
import { useNavigate } from 'react-router-dom';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  StarOutlined,
  EditOutlined,
  BarChartOutlined,
  UploadOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { userAPI } from '../api';
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
  const [aiConfig, setAIConfig] = useState({ ai_api_key: '', ai_api_url: '', ai_model: '' });
  const [savingAI, setSavingAI] = useState(false);
  const [loading, setLoading] = useState(true);
  const [aiForm] = Form.useForm();
  const { message } = App.useApp();

  useEffect(() => {
    setLoading(true);
    Promise.allSettled([
      userAPI.getAnswers(),
      userAPI.getWrongAnswers(),
      userAPI.getBookmarks(),
      userAPI.getStats(),
      userAPI.getUploads(),
      userAPI.getAIConfig(),
    ]).then((results) => {
      if (results[0].status === 'fulfilled') {
        setAnswers(results[0].value.data.answers);
      }
      if (results[1].status === 'fulfilled') {
        setWrongAnswers(results[1].value.data.answers);
      }
      if (results[2].status === 'fulfilled') {
        setBookmarks(results[2].value.data.bookmarks);
      }
      if (results[3].status === 'fulfilled') {
        setStats(results[3].value.data);
      }
      if (results[4].status === 'fulfilled') {
        setUploads(results[4].value.data.questions || []);
      }
      if (results[5].status === 'fulfilled') {
        setAIConfig(results[5].value.data);
      }
    }).finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    aiForm.setFieldsValue(aiConfig);
  }, [aiConfig, aiForm]);

  const handleSaveAIConfig = async (values: { ai_api_key: string; ai_api_url: string; ai_model: string }) => {
    setSavingAI(true);
    try {
      await userAPI.updateAIConfig(values);
      setAIConfig(values);
      message.success('AI 配置已保存');
    } catch {
      message.error('保存失败');
    } finally {
      setSavingAI(false);
    }
  };

  const renderAnswerList = (data: UserAnswer[], emptyText: string) => (
    <List
      dataSource={data}
      locale={{ emptyText }}
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
              <Row gutter={24} style={{ marginBottom: 24 }}>
                <Col span={6}>
                  <Card>
                    <Statistic title="总答题数" value={stats.total_answers} prefix={<EditOutlined />} />
                  </Card>
                </Col>
                <Col span={6}>
                  <Card>
                    <Statistic
                      title="合格"
                      value={stats.qualified_count}
                      prefix={<CheckCircleOutlined style={{ color: '#52c41a' }} />}
                      suffix={`/ ${stats.total_answers}`}
                    />
                  </Card>
                </Col>
                <Col span={6}>
                  <Card>
                    <Statistic
                      title="需改进"
                      value={stats.wrong_count}
                      prefix={<CloseCircleOutlined style={{ color: '#ff4d4f' }} />}
                    />
                  </Card>
                </Col>
                <Col span={6}>
                  <Card>
                    <Statistic title="收藏" value={stats.total_bookmarks} prefix={<StarOutlined style={{ color: '#faad14' }} />} />
                  </Card>
                </Col>
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
      label: <span><EditOutlined /> 我的回答 ({answers.length})</span>,
      children: renderAnswerList(answers, '还没有回答过题目'),
    },
    {
      key: 'wrong',
      label: <span><CloseCircleOutlined style={{ color: '#ff4d4f' }} /> 我的错题 ({wrongAnswers.length})</span>,
      children: renderAnswerList(wrongAnswers, '太棒了！没有需要改进的题目'),
    },
    {
      key: 'bookmarks',
      label: <span><StarOutlined style={{ color: '#faad14' }} /> 我的收藏 ({bookmarks.length})</span>,
      children: (
        <List
          dataSource={bookmarks}
          locale={{ emptyText: '还没有收藏题目' }}
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
    {
      key: 'uploads',
      label: <span><UploadOutlined /> 我上传的题目 ({uploads.length})</span>,
      children: (
        <List
          dataSource={uploads}
          locale={{ emptyText: '还没有上传题目，去首页上传吧' }}
          renderItem={(item: Question) => (
            <List.Item
              style={{ cursor: 'pointer' }}
              onClick={() => navigate(`/question/${item.id}`)}
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
          <Form.Item
            name="ai_api_key"
            label="API Key"
            rules={[{ required: true, message: '请输入 API Key' }]}
          >
            <Input.Password placeholder="sk-..." />
          </Form.Item>
          <Form.Item
            name="ai_api_url"
            label="API 地址（完整端点）"
            rules={[{ required: true, message: '请输入 API 地址' }]}
          >
            <Input placeholder="https://api.deepseek.com/chat/completions" />
          </Form.Item>
          <Form.Item
            name="ai_model"
            label="模型名称"
            rules={[{ required: true, message: '请输入模型名称' }]}
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
