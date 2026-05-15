import axios from 'axios';

const api = axios.create({
  baseURL: '/api',
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(err);
  }
);

export interface Category {
  id: number;
  name: string;
  parent_id: number | null;
  children: Category[];
  type: 'tech' | 'non-tech';
  icon: string;
}

export interface Question {
  id: number;
  category_id: number;
  category: Category;
  title: string;
  content: string;
  tags: string;
  difficulty: 'easy' | 'medium' | 'hard';
  type: 'tech' | 'non-tech';
  answer_count: number;
  uploader_id?: number;
  uploader?: { id: number; username: string };
  created_at: string;
}

export interface UserAnswer {
  id: number;
  user_id: number;
  question_id: number;
  content: string;
  score: number;
  previous_score: number | null;
  is_qualified: boolean;
  created_at: string;
  updated_at: string;
  question?: Question;
}

export interface TopAnswer {
  id: number;
  question_id: number;
  user_id: number;
  user: { id: number; username: string };
  content: string;
  score: number;
  likes_count: number;
  comments_count: number;
  is_anonymous: boolean;
  liked: boolean;
  created_at: string;
}

export interface Comment {
  id: number;
  top_answer_id: number;
  user_id: number;
  user: { id: number; username: string };
  content: string;
  created_at: string;
}

export interface AIEvaluation {
  score: number;
  is_qualified: boolean;
  analysis: string;
  strengths: string;
  weaknesses: string;
  reference: string;
  improvements: string;
  score_drop: boolean;
  score_drop_msg: string;
  has_existing: boolean;
  previous_score: number | null;
}

// Auth API
export const authAPI = {
  register: (data: { username: string; email: string; password: string }) =>
    api.post('/auth/register', data),
  login: (data: { email: string; password: string }) =>
    api.post('/auth/login', data),
  getMe: () => api.get('/auth/me'),
};

// Category API
export const categoryAPI = {
  getAll: () => api.get<{ categories: Category[] }>('/categories'),
};

// Question API
export const questionAPI = {
  getAll: (params?: { category_id?: string; search?: string; page?: number; page_size?: number }) =>
    api.get<{ questions: Question[]; total: number; page: number; page_size: number }>('/questions', { params }),
  getById: (id: string) => api.get<{ question: Question }>(`/questions/${id}`),
  delete: (id: number) => api.delete(`/questions/${id}`),
};

// Answer API
export const answerAPI = {
  submit: (questionId: string, data: { content: string; is_anonymous?: boolean }) =>
    api.post<AIEvaluation>(`/questions/${questionId}/answers`, data),
  getMyAnswer: (questionId: string) =>
    api.get<{ answered: boolean; answer?: UserAnswer; feedback?: string }>(`/questions/${questionId}/answers`),
  getHistory: (answerId: number) =>
    api.get<{ histories: any[] }>(`/answers/${answerId}/history`),
  submitStream: (
    questionId: string,
    data: { content: string },
    callbacks: {
      onScore: (score: number, isQualified: boolean) => void;
      onChunk: (field: string, text: string) => void;
      onDone: (meta: { score_drop: boolean; score_drop_msg: string; has_existing: boolean; previous_score: number | null }) => void;
      onError: (message: string) => void;
    }
  ) => {
    const token = localStorage.getItem('token');
    return fetch(`/api/questions/${questionId}/answers/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: JSON.stringify(data),
    }).then(async (response) => {
      if (!response.ok) {
        const err = await response.json().catch(() => ({ error: '请求失败' }));
        callbacks.onError(err.error || '请求失败');
        return;
      }
      const reader = response.body?.getReader();
      if (!reader) { callbacks.onError('不支持流式响应'); return; }
      const decoder = new TextDecoder();
      let buffer = '';
      let eventType = '';
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';
        for (const line of lines) {
          if (line.startsWith('event: ')) {
            eventType = line.slice(7).trim();
          } else if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6));
              switch (eventType) {
                case 'score':
                  callbacks.onScore(data.score, data.is_qualified);
                  break;
                case 'chunk':
                  callbacks.onChunk(data.field, data.text);
                  break;
                case 'done':
                  callbacks.onDone(data);
                  break;
                case 'error':
                  callbacks.onError(data.message || 'AI评估失败');
                  break;
              }
            } catch { /* skip malformed JSON */ }
          }
        }
      }
    });
  },
};

// Top Answers API
export const topAnswerAPI = {
  getByQuestion: (questionId: string) =>
    api.get<{ top_answers: TopAnswer[] }>(`/questions/${questionId}/top-answers`),
  like: (answerId: number) =>
    api.post<{ liked: boolean }>(`/top-answers/${answerId}/like`),
  addComment: (answerId: number, content: string) =>
    api.post(`/top-answers/${answerId}/comments`, { content }),
  getComments: (answerId: number) =>
    api.get<{ comments: Comment[] }>(`/top-answers/${answerId}/comments`),
};

// Bookmark API
export const bookmarkAPI = {
  toggle: (questionId: string) =>
    api.post<{ bookmarked: boolean }>(`/questions/${questionId}/bookmark`),
  check: (questionId: string) =>
    api.get<{ bookmarked: boolean }>(`/questions/${questionId}/bookmark`),
};

// Upload API
export interface PreviewRow {
  index: number;
  content: string;
  tags: string;
  category: string;
  status: string;
  rewritten: string;
  reason: string;
}

export const uploadAPI = {
  uploadQuestions: (file: File) => {
    const form = new FormData();
    form.append('file', file);
    return api.post<{ imported: number; skipped: number; message: string }>('/questions/upload', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },
  previewUpload: (file: File) => {
    const form = new FormData();
    form.append('file', file);
    return api.post<{ preview: PreviewRow[]; summary: { valid: number; rewritten: number; invalid: number; total: number; importable: number } }>('/questions/preview', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },
  confirmImport: (items: { content: string; tags: string; category: string; rewritten: string }[]) =>
    api.post<{ message: string; imported: number }>('/questions/import', { items }),
  downloadTemplate: async () => {
    const res = await api.get('/questions/template', { responseType: 'blob' });
    const url = URL.createObjectURL(res.data);
    const a = document.createElement('a');
    a.href = url;
    a.download = '题目上传模板.xlsx';
    a.click();
    URL.revokeObjectURL(url);
  },
  getCategoryNames: () => api.get<string[]>('/categories/names'),
};

// User API
export interface QuestionScore {
  score: number;
  is_qualified: boolean;
}

export const userAPI = {
  getAnswers: (params?: { page?: number; page_size?: number }) =>
    api.get('/user/answers', { params }),
  getWrongAnswers: (params?: { page?: number; page_size?: number }) =>
    api.get('/user/wrong-answers', { params }),
  getBookmarks: (params?: { page?: number; page_size?: number }) =>
    api.get('/user/bookmarks', { params }),
  getStats: () => api.get('/user/stats'),
  getQuestionScores: (ids: string) =>
    api.get<{ scores: Record<string, QuestionScore> }>('/user/question-scores', { params: { ids } }),
  getUploads: (params?: { page?: number; page_size?: number }) =>
    api.get('/user/uploads', { params }),
  getAIConfig: () => api.get<{ has_config: boolean }>('/user/ai-config'),
  updateAIConfig: (data: { ai_api_key: string; ai_api_url: string; ai_model: string }) =>
    api.put('/user/ai-config', data),
  changePassword: (data: { old_password: string; new_password: string }) =>
    api.put('/user/password', data),
};

// Teacher API
export const teacherAPI = {
  getOverview: (params?: { class_id?: string }) => api.get<{ overview: { student_count: number; total_answers: number; average_score: number; qualified_rate: number } }>('/teacher/overview', { params }),
  getStudents: (params?: { page?: number; page_size?: number; class_id?: string }) =>
    api.get('/teacher/students', { params }),
  getStudentAnswers: (userId: string) =>
    api.get<{ answers: UserAnswer[] }>(`/teacher/students/${userId}/answers`),
  getHotQuestions: (type: 'error' | 'mastered') =>
    api.get<{ questions: { id: number; title: string; avg_score: number; answer_count: number; fail_rate: number }[] }>('/teacher/hot-questions', { params: { type } }),
  analyzeQuestionErrors: (questionId: string, force?: boolean) =>
    api.post<{ analysis: string; analyzed_at?: string; cached?: boolean }>(`/teacher/questions/${questionId}/analyze-errors`, null, { params: force ? { force: 'true' } : {} }),
  getQuestionAnswers: (questionId: string) =>
    api.get<{ question: Question; answers: UserAnswer[] }>(`/teacher/questions/${questionId}/answers`),
  getCategoryStats: (params?: { class_id?: string }) =>
    api.get<{ categories: { category_id: number; category_name: string; avg_score: number; fail_rate: number; answer_count: number; question_count: number }[] }>('/teacher/category-stats', { params }),
  getCategoryQuestions: (categoryId: string, params?: { page?: number; page_size?: number; class_id?: string }) =>
    api.get<{ questions: { id: number; title: string; avg_score: number; fail_rate: number; answer_count: number; error_analysis?: string; error_analysis_at?: string }[]; total: number; page: number; page_size: number; category_name: string }>(`/teacher/categories/${categoryId}/questions`, { params }),
};

// Class API
export interface ClassItem {
  id: number;
  name: string;
  teacher_id: number;
  student_count: number;
  created_at: string;
  updated_at: string;
}

export interface ImportRowResult {
  row: number;
  name: string;
  class_name: string;
  email: string;
  status: 'importable' | 'need_confirm' | 'invalid';
  reason?: string;
  conflict_type?: 'cross_class' | 'same_name';
  existing_class?: string;
}

export interface ClassSummary {
  class_name: string;
  importable: number;
  need_confirm: number;
  invalid: number;
}

export interface PreviewImportResponse {
  results: ImportRowResult[];
  total: number;
  importable: number;
  need_confirm: number;
  invalid: number;
  class_summary: ClassSummary[];
}

export interface ConfirmRowResult {
  row: number;
  name: string;
  status: 'created' | 'moved' | 'skipped';
  reason?: string;
}

export interface ClassResultSummary {
  class_name: string;
  created: number;
  moved: number;
}

export interface ConfirmImportResponse {
  message: string;
  created: number;
  moved: number;
  skipped: number;
  results: ConfirmRowResult[];
  class_results: ClassResultSummary[];
}

export const classAPI = {
  create: (name: string) => api.post<{ class: ClassItem }>('/classes', { name }),
  getAll: () => api.get<{ classes: ClassItem[] }>('/classes'),
  update: (id: number, name: string) => api.put<{ class: ClassItem }>(`/classes/${id}`, { name }),
  delete: (id: number) => api.delete(`/classes/${id}`),
  addStudent: (classId: number, userId: number) => api.post(`/classes/${classId}/students`, { user_id: userId }),
  removeStudent: (classId: number, userId: number) => api.delete(`/classes/${classId}/students/${userId}`),
  getStudents: (classId: number, params?: { page?: number; page_size?: number }) =>
    api.get<{ students: { id: number; username: string; email: string; role: string }[]; total: number; page: number; page_size: number }>(`/classes/${classId}/students`, { params }),
  assignTeacher: (classId: number, teacherId: number) => api.put(`/classes/${classId}/teacher`, { teacher_id: teacherId }),
  downloadStudentTemplate: async () => {
    const res = await api.get('/classes/student-template', { responseType: 'blob' });
    const url = URL.createObjectURL(res.data as Blob);
    const a = document.createElement('a');
    a.href = url; a.download = 'student_template.xlsx'; a.click();
    URL.revokeObjectURL(url);
  },
  previewImportStudents: (file: File) => {
    const form = new FormData();
    form.append('file', file);
    return api.post<PreviewImportResponse>('/classes/import-students', form, { headers: { 'Content-Type': 'multipart/form-data' } });
  },
  confirmImportStudents: (items: { row: number; name: string; class_name: string; email: string; action: string }[]) =>
    api.post<ConfirmImportResponse>('/classes/import-students/confirm', { items }),
};

// Admin API
export interface AdminUser {
  id: number;
  username: string;
  email: string;
  role: string;
  class_id: number | null;
  class_name: string | null;
  created_at: string;
  updated_at: string;
}

export const adminAPI = {
  getUsers: (params?: { page?: number; role?: string; search?: string; class_id?: string }) => api.get<{ users: AdminUser[]; total: number; page: number; page_size: number; can_edit: boolean }>('/admin/users', { params }),
  createUser: (data: { username: string; email: string; password: string; role: string }) =>
    api.post('/admin/users', data),
  updateRole: (userId: number, role: string) => api.put(`/admin/users/${userId}/role`, { role }),
  resetPassword: (userId: number) => api.put(`/admin/users/${userId}/reset-password`),
  deleteUser: (userId: number) => api.delete(`/admin/users/${userId}`),
  createCategory: (data: { name: string; type: string; parent_id?: number; sort_order?: number; icon?: string }) => api.post('/admin/categories', data),
  updateCategory: (id: number, data: { name?: string; type?: string; sort_order?: number; icon?: string }) => api.put(`/admin/categories/${id}`, data),
  deleteCategory: (id: number) => api.delete(`/admin/categories/${id}`),
};

export default api;
