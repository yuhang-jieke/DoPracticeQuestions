import React, { useEffect, useRef, useState } from 'react';
import {
  Card,
  Typography,
  Tag,
  Button,
  Input,
  Spin,
  Space,
  Progress,
  List,
  Tooltip,
  Modal,
  Collapse,
  Alert,
  App,
} from 'antd';
import { useParams, useNavigate } from 'react-router-dom';
import {
  LikeOutlined,
  LikeFilled,
  StarOutlined,
  StarFilled,
  MessageOutlined,
  AudioOutlined,
} from '@ant-design/icons';
import ReactMarkdown from 'react-markdown';
import { questionAPI, answerAPI, topAnswerAPI, bookmarkAPI } from '../api';
import type { Question, TopAnswer, Comment as CommentType } from '../api';
import { useAuthStore } from '../store/auth';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;

const difficultyColors: Record<string, string> = { easy: 'green', medium: 'orange', hard: 'red' };
const difficultyLabels: Record<string, string> = { easy: '简单', medium: '中等', hard: '困难' };

const QuestionDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { isAuthenticated } = useAuthStore();
  const { message } = App.useApp();

  const [question, setQuestion] = useState<Question | null>(null);
  const [loading, setLoading] = useState(true);

  // Answer
  const [answer, setAnswer] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [streaming, setStreaming] = useState(false);
  const [streamScore, setStreamScore] = useState<number | null>(null);
  const [streamQualified, setStreamQualified] = useState<boolean | null>(null);
  const [streamFields, setStreamFields] = useState<Record<string, string>>({});
  const [streamDone, setStreamDone] = useState(false);
  const [streamMeta, setStreamMeta] = useState<any>(null);
  const [evaluation, setEvaluation] = useState<any>(null);
  const [hasExisting, setHasExisting] = useState(false);
  const [showEval, setShowEval] = useState(false);

  // Top answers
  const [topAnswers, setTopAnswers] = useState<TopAnswer[]>([]);
  const [likedMap, setLikedMap] = useState<Record<number, boolean>>({});
  const [bookmarked, setBookmarked] = useState(false);

  // Comments
  const [commentModal, setCommentModal] = useState<number | null>(null);
  const [comments, setComments] = useState<Record<number, CommentType[]>>({});
  const [newComment, setNewComment] = useState('');
  const [listening, setListening] = useState(false);
  const recognitionRef = useRef<any>(null);

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    Promise.all([
      questionAPI.getById(id),
      topAnswerAPI.getByQuestion(id),
    ])
      .then(([qRes, tRes]) => {
        setQuestion(qRes.data.question);
        setTopAnswers(tRes.data.top_answers);
        // Initialize liked state from server
        const initialLiked: Record<number, boolean> = {};
        tRes.data.top_answers.forEach((ta: any) => {
          if (ta.liked) initialLiked[ta.id] = true;
        });
        setLikedMap(initialLiked);
      })
      .finally(() => setLoading(false));

    if (isAuthenticated) {
      answerAPI.getMyAnswer(id).then((res) => {
        if (res.data.answered && res.data.answer) {
          setHasExisting(true);
          setAnswer(res.data.answer.content);
          if (res.data.answer.score) {
            // Restore previous AI evaluation from saved feedback
            try {
              const saved = JSON.parse(res.data.feedback || '{}');
              setEvaluation({
                score: res.data.answer.score,
                is_qualified: res.data.answer.is_qualified,
                analysis: saved.analysis || '',
                strengths: saved.strengths || '',
                weaknesses: saved.weaknesses || '',
                reference: saved.reference_answer || '',
                improvements: saved.improvements || '',
              });
            } catch {
              setEvaluation({
                score: res.data.answer.score,
                is_qualified: res.data.answer.is_qualified,
                analysis: '之前已作答，可重新提交获取新的评估',
                strengths: '',
                weaknesses: '',
                reference: '',
                improvements: '',
              });
            }
            setShowEval(true);
          }
        }
      });
      bookmarkAPI.check(id).then((res) => setBookmarked(res.data.bookmarked));
    }
  }, [id, isAuthenticated]);

  const handleSubmit = async () => {
    if (!isAuthenticated) {
      message.warning('请先登录');
      navigate('/login');
      return;
    }
    if (!answer.trim()) {
      message.warning('请输入回答内容');
      return;
    }
    if (!id) return;

    setSubmitting(true);
    setShowEval(true);
    setStreaming(true);
    setStreamScore(null);
    setStreamQualified(null);
    setStreamFields({ analysis: '', strengths: '', weaknesses: '', improvements: '', reference: '' });
    setStreamDone(false);
    setStreamMeta(null);
    setEvaluation(null);

    const fieldLabel: Record<string, string> = {
      analysis: 'analysis', strengths: 'strengths', weaknesses: 'weaknesses',
      improvements: 'improvements', reference: 'reference',
    };

    try {
      await answerAPI.submitStream(id, { content: answer }, {
        onScore: (score, isQualified) => {
          setStreamScore(score);
          setStreamQualified(isQualified);
        },
        onChunk: (field, text) => {
          const key = fieldLabel[field] || 'analysis';
          setStreamFields((prev) => ({ ...prev, [key]: (prev[key] || '') + text }));
        },
        onDone: (meta) => {
          setStreamDone(true);
          setStreamMeta(meta);
          setHasExisting(true);
          setStreaming(false);
          setSubmitting(false);
          // Refresh top answers
          topAnswerAPI.getByQuestion(id).then((tRes) => setTopAnswers(tRes.data.top_answers));
        },
        onError: (msg) => {
          message.error(msg);
          setStreaming(false);
          setSubmitting(false);
        },
      });
    } catch (err: any) {
      message.error('提交失败');
      setStreaming(false);
      setSubmitting(false);
    }
  };

  const toggleVoice = () => {
    const SpeechRecognition = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
    if (!SpeechRecognition) {
      message.error('当前浏览器不支持语音识别，请使用 Chrome 或 Edge');
      return;
    }
    if (listening) {
      recognitionRef.current?.stop();
      setListening(false);
      return;
    }
    const recognition = new SpeechRecognition();
    recognition.lang = 'zh-CN';
    recognition.interimResults = false;
    recognition.continuous = false;
    recognition.onresult = (event: any) => {
      const text = event.results[0][0].transcript;
      setAnswer((prev) => prev + (prev ? '\n' : '') + text);
    };
    recognition.onerror = () => {
      message.error('语音识别失败，请重试');
      setListening(false);
    };
    recognition.onend = () => setListening(false);
    recognitionRef.current = recognition;
    recognition.start();
    setListening(true);
  };

  const handleLike = async (answerId: number) => {
    if (!isAuthenticated) { message.warning('请先登录'); return; }
    const res = await topAnswerAPI.like(answerId);
    setLikedMap((prev) => ({ ...prev, [answerId]: res.data.liked }));
    setTopAnswers((prev) =>
      prev.map((ta) =>
        ta.id === answerId
          ? { ...ta, likes_count: ta.likes_count + (res.data.liked ? 1 : -1) }
          : ta
      )
    );
  };

  const handleBookmark = async () => {
    if (!isAuthenticated || !id) { message.warning('请先登录'); return; }
    const res = await bookmarkAPI.toggle(id);
    setBookmarked(res.data.bookmarked);
    message.success(res.data.bookmarked ? '已收藏' : '已取消收藏');
  };

  const handleShowComments = async (answerId: number) => {
    setCommentModal(answerId);
    if (!comments[answerId]) {
      const res = await topAnswerAPI.getComments(answerId);
      setComments((prev) => ({ ...prev, [answerId]: res.data.comments }));
    }
  };

  const handleAddComment = async () => {
    if (!newComment.trim() || commentModal === null) return;
    await topAnswerAPI.addComment(commentModal, newComment);
    setNewComment('');
    const res = await topAnswerAPI.getComments(commentModal);
    setComments((prev) => ({ ...prev, [commentModal!]: res.data.comments }));
    message.success('评论成功');
  };

  if (loading) return <Spin size="large" style={{ display: 'flex', justifyContent: 'center', paddingTop: 120 }} />;
  if (!question) return <div style={{ padding: 40, textAlign: 'center' }}>题目不存在</div>;

  return (
    <div style={{ maxWidth: 900, margin: '0 auto' }}>
      {/* Question */}
      <Card style={{ borderRadius: 8, marginBottom: 16 }}>
        <div style={{ marginBottom: 12 }}>
          <Space>
            <Tag color={difficultyColors[question.difficulty]}>{difficultyLabels[question.difficulty]}</Tag>
            <Tag>{question.category?.name}</Tag>
            {question.tags && question.tags.split(',').slice(0, 5).map((t, i) => (
              <Tag key={i} style={{ color: '#666', fontSize: 11 }}>{t.trim()}</Tag>
            ))}
            <Text type="secondary" style={{ fontSize: 12 }}>{question.answer_count} 人作答</Text>
            {question.uploader && <Text type="secondary" style={{ fontSize: 12 }}>上传者：{question.uploader.username}</Text>}
          </Space>
        </div>
        <Title level={4} style={{ marginBottom: 16 }}>{question.title}</Title>
        <Paragraph style={{ fontSize: 14, lineHeight: 1.8, whiteSpace: 'pre-wrap' }}>{question.content}</Paragraph>

        {isAuthenticated && (
          <Space>
            <Tooltip title={bookmarked ? '取消收藏' : '收藏'}>
              <Button
                type="text"
                icon={bookmarked ? <StarFilled style={{ color: '#faad14' }} /> : <StarOutlined />}
                onClick={handleBookmark}
              >
                {bookmarked ? '已收藏' : '收藏'}
              </Button>
            </Tooltip>
          </Space>
        )}
      </Card>

      {/* Answer Area */}
      <Card title={hasExisting ? '编辑你的回答' : '你的回答'} style={{ borderRadius: 8, marginBottom: 16 }}>
        <TextArea
          rows={6}
          value={answer}
          onChange={(e) => setAnswer(e.target.value)}
          placeholder="在此输入你的回答... (支持 Markdown 格式，点击麦克风可语音输入)"
          style={{ marginBottom: 12, fontSize: 14 }}
        />
        <Space>
          <Button
            type={listening ? 'primary' : 'default'}
            danger={listening}
            icon={<AudioOutlined />}
            onClick={toggleVoice}
          >
            {listening ? '录音中...' : '语音输入'}
          </Button>
          <Button type="primary" loading={submitting} onClick={handleSubmit} size="large">
          {hasExisting ? '更新回答' : '提交回答'}
        </Button>
        </Space>
      </Card>

      {/* AI Evaluation */}
      {showEval && (evaluation || streaming || streamScore !== null) && (
        <Card
          title={
            <Space>
              <span>AI 评估结果</span>
              {streamQualified !== null ? (
                <Tag color={streamQualified ? 'green' : 'red'}>{streamQualified ? '合格' : '需改进'}</Tag>
              ) : evaluation ? (
                <Tag color={evaluation.is_qualified ? 'green' : 'red'}>{evaluation.is_qualified ? '合格' : '需改进'}</Tag>
              ) : (
                <Tag>分析中...</Tag>
              )}
            </Space>
          }
          style={{ borderRadius: 8, marginBottom: 16, borderLeft: `4px solid ${(streamQualified ?? evaluation?.is_qualified) ? '#52c41a' : '#ff4d4f'}` }}
        >
          {(evaluation?.score_drop || streamMeta?.score_drop) && (
            <Alert
              type="warning"
              message={(evaluation || streamMeta)?.score_drop_msg}
              style={{ marginBottom: 16 }}
              showIcon
            />
          )}

          <div style={{ display: 'flex', gap: 40, marginBottom: 24 }}>
            <div style={{ textAlign: 'center' }}>
              <Progress
                type="circle"
                percent={Math.round(((streamScore ?? evaluation?.score ?? 0) / 10) * 100)}
                format={() => {
                  if (streamScore !== null) return `${streamScore}/10`;
                  if (evaluation) return `${evaluation.score}/10`;
                  return '--/10';
                }}
                strokeColor={(streamScore ?? evaluation?.score ?? 0) >= 7 ? '#52c41a' : '#ff4d4f'}
                size={100}
              />
              <div style={{ marginTop: 8 }}>
                <Text type="secondary">评分</Text>
              </div>
            </div>
            <div style={{ flex: 1 }}>
{(streamFields.analysis || evaluation?.analysis) && (
                <div style={{ marginBottom: 12 }}>
                  <Text strong>综合分析：</Text>
                  <div style={{ margin: '4px 0', lineHeight: 1.8 }}>
                    <ReactMarkdown>{streamFields.analysis || evaluation?.analysis || ''}</ReactMarkdown>
                    {streaming && !streamDone && <Text type="secondary">▌</Text>}
                  </div>
                </div>
              )}
              {(streamFields.strengths || evaluation?.strengths) && (
                <div style={{ marginBottom: 8 }}>
                  <Text style={{ color: '#52c41a' }}>✅ 优点：</Text>
                  <div style={{ margin: '4px 0', lineHeight: 1.8 }}>
                    <ReactMarkdown>{streamFields.strengths || evaluation?.strengths || ''}</ReactMarkdown>
                  </div>
                </div>
              )}
              {(streamFields.weaknesses || evaluation?.weaknesses) && (
                <div style={{ marginBottom: 8 }}>
                  <Text style={{ color: '#ff4d4f' }}>📌 不足：</Text>
                  <div style={{ margin: '4px 0', lineHeight: 1.8 }}>
                    <ReactMarkdown>{streamFields.weaknesses || evaluation?.weaknesses || ''}</ReactMarkdown>
                  </div>
                </div>
              )}
              {(streamFields.improvements || evaluation?.improvements) && (
                <div>
                  <Text style={{ color: '#1677ff' }}>💡 改进建议：</Text>
                  <div style={{ margin: '4px 0', lineHeight: 1.8 }}>
                    <ReactMarkdown>{streamFields.improvements || evaluation?.improvements || ''}</ReactMarkdown>
                  </div>
                </div>
              )}
            </div>
          </div>

          {(streamFields.reference || evaluation?.reference) && (
            <Collapse
              items={[{
                key: 'reference',
                label: '查看参考答案',
                children: <div style={{ lineHeight: 1.8 }}><ReactMarkdown>{streamFields.reference || evaluation?.reference || ''}</ReactMarkdown></div>,
              }]}
              style={{ background: '#fafafa' }}
            />
          )}
        </Card>
      )}

      {/* Top 10 Answers - only show after user submits their answer */}
      {showEval && (
        <Card title={`优质回答 Top ${topAnswers.length}`} style={{ borderRadius: 8 }}>
          {topAnswers.length === 0 ? (
            <Text type="secondary">暂无优质回答，快来第一个作答吧！</Text>
          ) : (
            <List
              dataSource={topAnswers}
              renderItem={(item, index) => (
                <List.Item
                  style={{ padding: '16px 0' }}
                  actions={[
                    <Tooltip title="点赞">
                      <Button
                        type="text"
                        icon={likedMap[item.id] ? <LikeFilled style={{ color: '#1677ff' }} /> : <LikeOutlined />}
                        onClick={() => handleLike(item.id)}
                      >
                        {item.likes_count}
                      </Button>
                    </Tooltip>,
                    <Button
                      type="text"
                      icon={<MessageOutlined />}
                      onClick={() => handleShowComments(item.id)}
                    >
                      {item.comments_count}
                    </Button>,
                  ]}
                >
                  <List.Item.Meta
                    avatar={
                      <div
                        style={{
                          width: 32,
                          height: 32,
                          borderRadius: '50%',
                          background: index < 3 ? '#1677ff' : '#f0f0f0',
                          color: index < 3 ? '#fff' : '#666',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          fontWeight: 600,
                          fontSize: 14,
                        }}
                      >
                        {index + 1}
                      </div>
                    }
                    title={
                      <Space>
                        <Text strong>{item.is_anonymous ? '匿名用户' : item.user?.username || '用户'}</Text>
                        <Tag color="blue" style={{ fontSize: 12 }}>{item.score}分</Tag>
                      </Space>
                    }
                    description={
                      <Paragraph style={{ margin: '8px 0 0', whiteSpace: 'pre-wrap', fontSize: 14, lineHeight: 1.7 }}>
                        {item.content}
                      </Paragraph>
                    }
                  />
                </List.Item>
              )}
            />
          )}
        </Card>
      )}

      {/* Comments Modal */}
      <Modal
        title="评论"
        open={commentModal !== null}
        onCancel={() => setCommentModal(null)}
        footer={null}
        width={500}
      >
        {commentModal !== null && (
          <div>
            <List
              dataSource={comments[commentModal] || []}
              locale={{ emptyText: '暂无评论' }}
              renderItem={(c: CommentType) => (
                <List.Item>
                  <List.Item.Meta
                    title={<Text strong>{c.user?.username}</Text>}
                    description={c.content}
                  />
                </List.Item>
              )}
              style={{ marginBottom: 16 }}
            />
            <Space style={{ width: '100%' }}>
              <Input
                value={newComment}
                onChange={(e) => setNewComment(e.target.value)}
                placeholder="输入评论..."
                onPressEnter={handleAddComment}
                style={{ flex: 1 }}
              />
              <Button type="primary" onClick={handleAddComment}>发表</Button>
            </Space>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default QuestionDetail;
