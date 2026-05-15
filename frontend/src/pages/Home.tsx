import React, { useEffect, useState } from 'react';
import { Layout, Card, Tag, Typography, Spin, Empty, Pagination, Space, Button, Modal, Upload, App, Input, Drawer, Grid } from 'antd';

import { useNavigate } from 'react-router-dom';
import {
  CodeOutlined,
  TeamOutlined,
  RightOutlined,
  CheckCircleFilled,
  CloseCircleFilled,
  UploadOutlined,
  DownloadOutlined,
  InboxOutlined,
} from '@ant-design/icons';
import { questionAPI, categoryAPI, userAPI, uploadAPI } from '../api';
import type { Category, Question, QuestionScore, PreviewRow } from '../api';
import { useAuthStore } from '../store/auth';

const { Sider, Content } = Layout;
const { Title, Text, Paragraph } = Typography;

const difficultyColors: Record<string, string> = {
  easy: 'green',
  medium: 'orange',
  hard: 'red',
};

const difficultyLabels: Record<string, string> = {
  easy: '简单',
  medium: '中等',
  hard: '困难',
};

const Home: React.FC = () => {
  const [categories, setCategories] = useState<Category[]>([]);
  const [questions, setQuestions] = useState<Question[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(false);
  const [scores, setScores] = useState<Record<string, QuestionScore>>({});
  const [uploadOpen, setUploadOpen] = useState(false);
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewRows, setPreviewRows] = useState<PreviewRow[]>([]);
  const [previewChecked, setPreviewChecked] = useState<Set<number>>(new Set());
  const [previewSummary, setPreviewSummary] = useState<{ valid: number; rewritten: number; invalid: number; total: number; importable: number } | null>(null);
  const [allCategories, setAllCategories] = useState<string[]>([]);
  const { isAuthenticated, user } = useAuthStore();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const screens = Grid.useBreakpoint();
  const isMobile = !screens.md;
  const [drawerOpen, setDrawerOpen] = useState(false);

  useEffect(() => {
    categoryAPI.getAll().then((res) => setCategories(res.data.categories));
  }, []);

  useEffect(() => {
    setLoading(true);
    questionAPI
      .getAll({
        category_id: selectedCategory || undefined,
        search: search || undefined,
        page,
        page_size: 15,
      })
      .then((res) => {
        setQuestions(res.data.questions);
        setTotal(res.data.total);
        // Fetch scores for the loaded questions
        if (isAuthenticated && res.data.questions.length > 0) {
          const ids = res.data.questions.map((q) => q.id).join(',');
          userAPI.getQuestionScores(ids).then((sRes) => {
            setScores(sRes.data.scores);
          }).catch(() => {});
        }
      })
      .finally(() => setLoading(false));
  }, [selectedCategory, page, search, isAuthenticated]);

  const handleCategoryClick = (categoryId?: string) => {
    setSelectedCategory(categoryId || '');
    setPage(1);
  };

  const handlePreview = async () => {
    if (!isAuthenticated) { message.warning('请先登录'); navigate('/login'); return; }
    if (!uploadFile) { message.warning('请选择文件'); return; }
    setUploading(true);
    try {
      const res = await uploadAPI.previewUpload(uploadFile);
      setPreviewRows(res.data.preview);
      setPreviewSummary(res.data.summary);
      const checked = new Set<number>();
      res.data.preview.forEach((r) => {
        if (r.status !== 'invalid') checked.add(r.index);
      });
      setPreviewChecked(checked);
      const catRes = await uploadAPI.getCategoryNames();
      setAllCategories((catRes as any).data?.categories || []);
      setUploadOpen(false);
      setPreviewOpen(true);
    } catch (err: any) {
      message.error(err.response?.data?.error || '解析失败');
    } finally {
      setUploading(false);
    }
  };

  const handleConfirmImport = async () => {
    setUploading(true);
    try {
      const toImport = previewRows
        .filter((r) => previewChecked.has(r.index))
        .map((r) => ({ content: r.content, tags: r.tags, category: r.category, rewritten: r.rewritten }));
      if (toImport.length === 0) { message.warning('请至少勾选一条题目'); setUploading(false); return; }
      const res = await uploadAPI.confirmImport(toImport);
      message.success(res.data.message);
      setPreviewOpen(false);
      setPreviewRows([]);
      setPreviewChecked(new Set());
      setPreviewSummary(null);
      setUploadFile(null);
      setPage(1);
      setSelectedCategory('');
    } catch (err: any) {
      message.error(err.response?.data?.error || '导入失败');
    } finally {
      setUploading(false);
    }
  };

  const updatePreviewCategory = (index: number, category: string) => {
    setPreviewRows((prev) => prev.map((r) => (r.index === index ? { ...r, category } : r)));
  };

  const updatePreviewRewritten = (index: number, rewritten: string) => {
    setPreviewRows((prev) => prev.map((r) => (r.index === index ? { ...r, rewritten } : r)));
  };

  const togglePreviewCheck = (index: number) => {
    setPreviewChecked((prev) => {
      const next = new Set(prev);
      if (next.has(index)) next.delete(index); else next.add(index);
      return next;
    });
  };

  const toggleAll = (checked: boolean) => {
    if (checked) setPreviewChecked(new Set(previewRows.map((r) => r.index)));
    else setPreviewChecked(new Set());
  };

  const renderCategoryTree = (cats: Category[], parentType?: string) => (
    <div>
      {parentType && (
        <div
          style={{
            padding: '8px 12px',
            marginBottom: 4,
            color: '#1677ff',
            fontWeight: 500,
            fontSize: 13,
            borderBottom: '1px solid #f0f0f0',
          }}
        >
          {parentType === 'tech' ? (
            <Space><CodeOutlined /> 技术类</Space>
          ) : (
            <Space><TeamOutlined /> 非技术类</Space>
          )}
        </div>
      )}
      {cats.map((cat) => (
        <React.Fragment key={cat.id}>
          <div
            onClick={() => handleCategoryClick(String(cat.id))}
            style={{
              padding: '6px 12px 6px 28px',
              cursor: 'pointer',
              borderRadius: 4,
              margin: '2px 0',
              background: selectedCategory === String(cat.id) ? '#e6f4ff' : 'transparent',
              color: selectedCategory === String(cat.id) ? '#1677ff' : '#333',
              fontWeight: selectedCategory === String(cat.id) ? 500 : 400,
              fontSize: 14,
              transition: 'all 0.2s',
            }}
            onMouseEnter={(e) => {
              if (selectedCategory !== String(cat.id)) {
                e.currentTarget.style.background = '#f5f5f5';
              }
            }}
            onMouseLeave={(e) => {
              if (selectedCategory !== String(cat.id)) {
                e.currentTarget.style.background = 'transparent';
              }
            }}
          >
            {cat.name}
          </div>
          {cat.children &&
            cat.children.map((child) => (
              <div
                key={child.id}
                onClick={() => handleCategoryClick(String(child.id))}
                style={{
                  padding: '5px 12px 5px 44px',
                  cursor: 'pointer',
                  borderRadius: 4,
                  fontSize: 13,
                  background: selectedCategory === String(child.id) ? '#e6f4ff' : 'transparent',
                  color: selectedCategory === String(child.id) ? '#1677ff' : '#666',
                  transition: 'all 0.2s',
                }}
                onMouseEnter={(e) => {
                  if (selectedCategory !== String(child.id)) e.currentTarget.style.background = '#f5f5f5';
                }}
                onMouseLeave={(e) => {
                  if (selectedCategory !== String(child.id)) e.currentTarget.style.background = 'transparent';
                }}
              >
                {child.name}
              </div>
            ))}
        </React.Fragment>
      ))}
    </div>
  );

  const techCategories = categories.filter((c) => c.type === 'tech');
  const nonTechCategories = categories.filter((c) => c.type === 'non-tech');

  return (
    <Layout style={{ background: 'transparent', gap: 24 }}>
      {!isMobile && (
        <Sider
          width={240}
          style={{
            background: '#fff',
            borderRadius: 8,
            padding: '12px 0',
            boxShadow: '0 1px 4px rgba(0,0,0,0.04)',
            height: 'fit-content',
            position: 'sticky',
            top: 80,
          }}
        >
        <div
          onClick={() => handleCategoryClick()}
          style={{
            padding: '8px 16px',
            cursor: 'pointer',
            fontWeight: selectedCategory === '' ? 600 : 400,
            color: selectedCategory === '' ? '#1677ff' : '#333',
            borderBottom: '1px solid #f0f0f0',
            marginBottom: 8,
          }}
        >
          <Space>
            <RightOutlined rotate={90} style={{ fontSize: 12 }} />
            全部分类
          </Space>
        </div>
        {renderCategoryTree(techCategories, 'tech')}
        {renderCategoryTree(nonTechCategories, 'non-tech')}
        {(user?.role === 'teacher' || user?.role === 'director' || user?.role === 'principal') && (
          <div style={{ borderTop: '1px solid #f0f0f0', margin: '8px 16px 0', padding: '12px 0' }}>
            <Button
              type="dashed"
              icon={<UploadOutlined />}
              block
              onClick={() => setUploadOpen(true)}
            >
              上传题目
            </Button>
          </div>
        )}
      </Sider>
      )}

      <Content>
        {isMobile && (
          <Button
            block
            style={{ marginBottom: 12 }}
            onClick={() => setDrawerOpen(true)}
          >
            📂 分类筛选
          </Button>
        )}
        <Input.Search
          placeholder="搜索题目..."
          allowClear
          onSearch={(value) => { setSearch(value); setPage(1); }}
          style={{ marginBottom: 16, maxWidth: 480 }}
        />
        <div style={{ marginBottom: 16 }}>
          <Title level={4} style={{ margin: 0 }}>
            {search
              ? `搜索"${search}"`
              : selectedCategory
                ? questions[0]?.category?.name || '题目列表'
                : '全部题目'}
            <Text type="secondary" style={{ fontSize: 14, fontWeight: 400, marginLeft: 8 }}>
              共 {total} 题
            </Text>
          </Title>
        </div>

        <Spin spinning={loading}>
          {questions.length === 0 ? (
            <Empty description="暂无题目" style={{ paddingTop: 80 }} />
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
              {questions.map((q) => {
                const myScore = scores[String(q.id)];
                return (
                  <Card
                    key={q.id}
                    hoverable
                    style={{ borderRadius: 8 }}
                    onClick={() => navigate(`/question/${q.id}`)}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                      <div style={{ flex: 1 }}>
                        <div style={{ marginBottom: 8 }}>
                          <Tag color={difficultyColors[q.difficulty]} style={{ borderRadius: 4 }}>
                            {difficultyLabels[q.difficulty]}
                          </Tag>
                          <Tag style={{ borderRadius: 4 }}>{q.category?.name}</Tag>
                          {q.tags && q.tags.split(',').slice(0, 3).map((t, i) => (
                            <Tag key={i} style={{ borderRadius: 4, color: '#666', fontSize: 11 }}>{t.trim()}</Tag>
                          ))}
                        </div>
                        <Text strong style={{ fontSize: 15, lineHeight: '24px' }}>
                          {q.title}
                        </Text>
                        <Paragraph
                          ellipsis={{ rows: 2 }}
                          type="secondary"
                          style={{ margin: '8px 0 0', fontSize: 13, lineHeight: '20px' }}
                        >
                          {q.content}
                        </Paragraph>
                        {myScore && (
                          <div style={{ marginTop: 8 }}>
                            <Tag
                              color={myScore.is_qualified ? 'green' : 'red'}
                              style={{ borderRadius: 4, fontSize: 12 }}
                            >
                              {myScore.is_qualified ? (
                                <CheckCircleFilled style={{ marginRight: 4 }} />
                              ) : (
                                <CloseCircleFilled style={{ marginRight: 4 }} />
                              )}
                              我的得分: {myScore.score}/10
                            </Tag>
                          </div>
                        )}
                      </div>
                      <Text type="secondary" style={{ fontSize: 12, whiteSpace: 'nowrap', marginLeft: 16 }}>
                        {q.answer_count} 人作答
                      </Text>
                    </div>
                  </Card>
                );
              })}
            </div>
          )}

          {total > 15 && (
            <div style={{ display: 'flex', justifyContent: 'center', marginTop: 24 }}>
              <Pagination
                current={page}
                total={total}
                pageSize={15}
                onChange={(p) => setPage(p)}
                showSizeChanger={false}
              />
            </div>
          )}
        </Spin>
      </Content>

      <Modal
        title="上传题目"
        open={uploadOpen}
        onCancel={() => { setUploadOpen(false); setUploadFile(null); }}
        footer={[
          <Button key="cancel" onClick={() => { setUploadOpen(false); setUploadFile(null); }}>
            取消
          </Button>,
          <Button key="preview" type="primary" loading={uploading} onClick={handlePreview}>
            预览
          </Button>,
        ]}
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <Button icon={<DownloadOutlined />} onClick={() => uploadAPI.downloadTemplate()}>
            下载模板文件
          </Button>
          <Upload.Dragger
            accept=".xlsx"
            maxCount={1}
            beforeUpload={(file) => { setUploadFile(file); return false; }}
            onRemove={() => setUploadFile(null)}
          >
            <p className="ant-upload-drag-icon">
              <InboxOutlined />
            </p>
            <p className="ant-upload-text">点击或拖拽 .xlsx 文件到此处</p>
            <p className="ant-upload-hint">A列：题目内容，B列：标签关键词</p>
          </Upload.Dragger>
        </Space>
      </Modal>

      <Modal
        title="预览导入"
        open={previewOpen}
        onCancel={() => { setPreviewOpen(false); setPreviewRows([]); setPreviewChecked(new Set()); setPreviewSummary(null); }}
        width={1000}
        footer={[
          <Button key="cancel" onClick={() => { setPreviewOpen(false); setPreviewRows([]); setPreviewChecked(new Set()); setPreviewSummary(null); }}>
            取消
          </Button>,
          <Button key="import" type="primary" loading={uploading} onClick={handleConfirmImport}>
            确认导入（{previewChecked.size} 条）
          </Button>,
        ]}
      >
        {previewSummary && (
          <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
            <Card size="small" style={{ flex: 1, textAlign: 'center', borderLeft: '3px solid #52c41a' }}>
              <div style={{ fontSize: 20, fontWeight: 600, color: '#52c41a' }}>{previewSummary.valid}</div>
              <div style={{ fontSize: 12, color: '#999' }}>✅ 合格</div>
            </Card>
            <Card size="small" style={{ flex: 1, textAlign: 'center', borderLeft: '3px solid #1677ff' }}>
              <div style={{ fontSize: 20, fontWeight: 600, color: '#1677ff' }}>{previewSummary.rewritten}</div>
              <div style={{ fontSize: 12, color: '#999' }}>✏️ AI改写</div>
            </Card>
            <Card size="small" style={{ flex: 1, textAlign: 'center', borderLeft: '3px solid #ff4d4f' }}>
              <div style={{ fontSize: 20, fontWeight: 600, color: '#ff4d4f' }}>{previewSummary.invalid}</div>
              <div style={{ fontSize: 12, color: '#999' }}>❌ 无效（默认不导入）</div>
            </Card>
            <Card size="small" style={{ flex: 1, textAlign: 'center', background: '#f6ffed' }}>
              <div style={{ fontSize: 20, fontWeight: 600, color: '#1677ff' }}>{previewSummary.importable}</div>
              <div style={{ fontSize: 12, color: '#999' }}>可导入共计</div>
            </Card>
          </div>
        )}
        <div style={{ maxHeight: 400, overflow: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
            <thead>
              <tr style={{ background: '#fafafa' }}>
                <th style={{ padding: '8px 6px', textAlign: 'center', borderBottom: '2px solid #f0f0f0', width: 40 }}>
                  <input type="checkbox" checked={previewRows.length > 0 && previewRows.every((r) => previewChecked.has(r.index))} onChange={(e) => toggleAll(e.target.checked)} />
                </th>
                <th style={{ padding: '8px 12px', textAlign: 'left', borderBottom: '2px solid #f0f0f0', width: 40 }}>#</th>
                <th style={{ padding: '8px 12px', textAlign: 'left', borderBottom: '2px solid #f0f0f0' }}>原始内容</th>
                <th style={{ padding: '8px 12px', textAlign: 'left', borderBottom: '2px solid #f0f0f0', minWidth: 180 }}>AI 改写 / 无效原因</th>
                <th style={{ padding: '8px 12px', textAlign: 'left', borderBottom: '2px solid #f0f0f0', minWidth: 100 }}>分类</th>
                <th style={{ padding: '8px 12px', textAlign: 'center', borderBottom: '2px solid #f0f0f0', width: 70 }}>状态</th>
              </tr>
            </thead>
            <tbody>
              {previewRows.map((r) => (
                <tr key={r.index} style={{ background: r.status === 'invalid' && !previewChecked.has(r.index) ? '#fff7f7' : undefined }}>
                  <td style={{ padding: '8px 6px', borderBottom: '1px solid #f5f5f5', textAlign: 'center' }}>
                    <input type="checkbox" checked={previewChecked.has(r.index)} onChange={() => togglePreviewCheck(r.index)} />
                  </td>
                  <td style={{ padding: '8px 12px', borderBottom: '1px solid #f5f5f5' }}>{r.index}</td>
                  <td style={{ padding: '8px 12px', borderBottom: '1px solid #f5f5f5' }}>
                    <Paragraph ellipsis={{ rows: 2 }} style={{ margin: 0, maxWidth: 250 }}>{r.content}</Paragraph>
                    {r.tags && <Text type="secondary" style={{ fontSize: 11 }}>标签：{r.tags}</Text>}
                  </td>
                  <td style={{ padding: '4px 12px', borderBottom: '1px solid #f5f5f5' }}>
                    {r.status === 'rewritten' ? (
                      <Input.TextArea rows={2} value={r.rewritten} onChange={(e) => updatePreviewRewritten(r.index, e.target.value)} style={{ fontSize: 12 }} />
                    ) : r.status === 'invalid' ? (
                      <Text type="danger" style={{ fontSize: 12 }}>{r.reason}</Text>
                    ) : (
                      <Text type="secondary" style={{ fontSize: 12 }}>—</Text>
                    )}
                  </td>
                  <td style={{ padding: '4px 12px', borderBottom: '1px solid #f5f5f5' }}>
                    <select
                      value={r.category}
                      onChange={(e) => updatePreviewCategory(r.index, e.target.value)}
                      style={{ width: '100%', padding: '4px 8px', borderRadius: 4, border: '1px solid #d9d9d9', fontSize: 12 }}
                    >
                      {allCategories.map((c) => (
                        <option key={c} value={c}>{c}</option>
                      ))}
                    </select>
                  </td>
                  <td style={{ padding: '8px 12px', borderBottom: '1px solid #f5f5f5', textAlign: 'center' }}>
                    {r.status === 'valid' && <Tag color="green" style={{ fontSize: 11 }}>合格</Tag>}
                    {r.status === 'rewritten' && <Tag color="blue" style={{ fontSize: 11 }}>已改写</Tag>}
                    {r.status === 'invalid' && <Tag color="red" style={{ fontSize: 11 }}>无效</Tag>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Modal>

      <Drawer
        title="分类筛选"
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        styles={{ body: { padding: 0 } }}
      >
        <div onClick={() => { handleCategoryClick(); setDrawerOpen(false); }} style={{ padding: '8px 16px', cursor: 'pointer', fontWeight: selectedCategory === '' ? 600 : 400, color: selectedCategory === '' ? '#1677ff' : '#333', borderBottom: '1px solid #f0f0f0' }}>
          📚 全部分类
        </div>
        {renderCategoryTree(techCategories, 'tech')}
        {renderCategoryTree(nonTechCategories, 'non-tech')}
        {(user?.role === 'teacher' || user?.role === 'director' || user?.role === 'principal') && (
          <div style={{ borderTop: '1px solid #f0f0f0', margin: '8px 16px 0', padding: '12px 0' }}>
            <Button type="dashed" icon={<UploadOutlined />} block onClick={() => { setDrawerOpen(false); setUploadOpen(true); }}>
              上传题目
            </Button>
          </div>
        )}
      </Drawer>
    </Layout>
  );
};

export default Home;
