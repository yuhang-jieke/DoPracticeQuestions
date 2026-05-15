import React, { useEffect, useState } from 'react';
import { Card, Table, Tag, Typography, Spin, Modal, List, Button, Space, App } from 'antd';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { teacherAPI } from '../api';
import type { UserAnswer } from '../api';

const { Title, Text, Paragraph } = Typography;

const TeacherCategory: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { message } = App.useApp();

  const [loading, setLoading] = useState(true);
  const [questions, setQuestions] = useState<any[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [qModalOpen, setQModalOpen] = useState(false);
  const [qModalTitle, setQModalTitle] = useState('');
  const [qAnswers, setQAnswers] = useState<UserAnswer[]>([]);
  const [qLoading, setQLoading] = useState(false);
  const [analyses, setAnalyses] = useState<Record<string, string>>({});
  const [analyzedAt, setAnalyzedAt] = useState<Record<string, string>>({});
  const [analyzing, setAnalyzing] = useState<Record<string, boolean>>({});
  const [categoryName, setCategoryName] = useState('');
  const [analysisModal, setAnalysisModal] = useState<{ id: string; title: string } | null>(null);

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    teacherAPI.getCategoryQuestions(id, { page, page_size: 20 })
      .then((res) => {
        const qs = res.data.questions || [];
        setQuestions(qs);
        setTotal(res.data.total);
        setCategoryName(res.data.category_name || '');
        // Restore cached analyses
        const cachedAnalyses: Record<string, string> = {};
        const cachedTimes: Record<string, string> = {};
        qs.forEach((q: any) => {
          if (q.error_analysis) {
            cachedAnalyses[q.id] = q.error_analysis;
            if (q.error_analysis_at) cachedTimes[q.id] = q.error_analysis_at;
          }
        });
        setAnalyses(cachedAnalyses);
        setAnalyzedAt(cachedTimes);
      }).finally(() => setLoading(false));
  }, [id, page]);

  const handleOpenQuestion = async (questionId: string, title: string) => {
    setQModalOpen(true);
    setQModalTitle(title);
    setQLoading(true);
    const res = await teacherAPI.getQuestionAnswers(questionId);
    setQAnswers(res.data.answers || []);
    setQLoading(false);
  };

  const handleAnalyze = async (questionId: string, force: boolean) => {
    setAnalyzing((prev) => ({ ...prev, [questionId]: true }));
    try {
      const res = await teacherAPI.analyzeQuestionErrors(questionId, force);
      setAnalyses((prev) => ({ ...prev, [questionId]: res.data.analysis }));
      setAnalyzedAt((prev) => ({ ...prev, [questionId]: res.data.analyzed_at || '' }));
      const q = questions.find((q: any) => String(q.id) === questionId);
      setAnalysisModal({ id: questionId, title: q?.title || '' });
      if (force) message.success('已重新分析');
    } catch (err: any) {
      message.error(err.response?.data?.error || '分析失败');
    } finally {
      setAnalyzing((prev) => ({ ...prev, [questionId]: false }));
    }
  };

  const columns = [
    { title: '题目', dataIndex: 'title', ellipsis: true },
    { title: '平均分', dataIndex: 'avg_score', width: 80, render: (v: number) => <Tag color={v >= 7 ? 'green' : 'red'}>{v.toFixed(1)}</Tag> },
    { title: '错误率', dataIndex: 'fail_rate', width: 90, render: (v: number) => <Text type="danger">{v.toFixed(0)}%</Text> },
    { title: '人数', dataIndex: 'answer_count', width: 60 },
    {
      title: '操作', width: 200,
      render: (_: any, r: any) => (
        <Space size="small">
          <Button size="small" onClick={() => handleOpenQuestion(String(r.id), r.title)}>查看回答</Button>
          <Button
            size="small"
            type={analyses[r.id] ? 'default' : 'primary'}
            loading={analyzing[r.id]}
            onClick={() => handleAnalyze(String(r.id), !!analyses[r.id])}
          >
            {analyses[r.id] ? '重新分析' : 'AI 分析'}
          </Button>
          {analyses[r.id] && (
            <Button
              size="small"
              type="text"
              style={{ color: '#1677ff', fontWeight: 'bold', fontSize: 16, lineHeight: 1, padding: '0 4px' }}
              onClick={() => setAnalysisModal({ id: String(r.id), title: r.title })}
            >+</Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div style={{ maxWidth: 1000, margin: '0 auto' }}>
      <Button type="text" icon={<ArrowLeftOutlined />} onClick={() => navigate('/teacher')} style={{ marginBottom: 16 }}>
        返回后台
      </Button>
      <Title level={3} style={{ marginBottom: 24 }}>
        题目分析（按错误率排序）
        {categoryName && <Tag color="blue" style={{ marginLeft: 12, fontSize: 14 }}>{categoryName}</Tag>}
      </Title>

      <Card style={{ borderRadius: 8 }}>
        <Spin spinning={loading}>
          <Table
            dataSource={questions}
            rowKey="id"
            columns={columns}
            scroll={{ x: 'max-content' }}
            pagination={{ current: page, total, pageSize: 20, onChange: (p) => setPage(p) }}
          />
        </Spin>
      </Card>

      <Modal
        title={
          <div style={{ paddingRight: 24 }}>
            <Text type="secondary" style={{ fontSize: 12 }}>题目</Text>
            <Paragraph style={{ marginTop: 4, fontSize: 15, fontWeight: 500 }}>{analysisModal?.title}</Paragraph>
          </div>
        }
        open={analysisModal !== null}
        onCancel={() => setAnalysisModal(null)}
        footer={null}
        width={720}
        closeIcon={<span style={{ fontSize: 18, fontWeight: 'bold' }}>−</span>}
      >
        {analysisModal && analyses[analysisModal.id] && (
          <div>
            {analyzedAt[analysisModal.id] && (
              <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 12 }}>
                分析时间：{new Date(analyzedAt[analysisModal.id]).toLocaleString('zh-CN')}
              </Text>
            )}
            <Paragraph style={{ whiteSpace: 'pre-wrap', margin: 0, fontSize: 14, lineHeight: 1.8 }}>
              {analyses[analysisModal.id]}
            </Paragraph>
          </div>
        )}
      </Modal>

      <Modal title={qModalTitle} open={qModalOpen} onCancel={() => setQModalOpen(false)} footer={null} width={750}>
        <Spin spinning={qLoading}>
          <List
            dataSource={qAnswers}
            locale={{ emptyText: '暂无回答' }}
            renderItem={(item: UserAnswer) => (
              <List.Item>
                <List.Item.Meta
                  avatar={<div style={{ width: 32, height: 32, borderRadius: '50%', background: '#1677ff', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 14, fontWeight: 600 }}>{(item as any).user?.username?.[0] || '?'}</div>}
                  title={<Space><Text strong>{(item as any).user?.username || '用户'}</Text><Tag color={item.is_qualified ? 'green' : 'red'}>{item.score}/10</Tag></Space>}
                  description={
                    <div>
                      <Text type="secondary" style={{ fontSize: 12 }}>{new Date(item.updated_at).toLocaleString('zh-CN')}</Text>
                      <Paragraph ellipsis={{ rows: 2, expandable: true }} style={{ marginTop: 4, fontSize: 13 }}>{item.content}</Paragraph>
                    </div>
                  }
                />
              </List.Item>
            )}
          />
        </Spin>
      </Modal>
    </div>
  );
};

export default TeacherCategory;
