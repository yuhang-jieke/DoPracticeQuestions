import axios from 'axios';

const api = axios.create({
  baseURL: 'http://localhost:8080/api',
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
  getAll: (params?: { category_id?: string; page?: number; page_size?: number }) =>
    api.get<{ questions: Question[]; total: number; page: number; page_size: number }>('/questions', { params }),
  getById: (id: string) => api.get<{ question: Question }>(`/questions/${id}`),
};

// Answer API
export const answerAPI = {
  submit: (questionId: string, data: { content: string; is_anonymous?: boolean }) =>
    api.post<AIEvaluation>(`/questions/${questionId}/answers`, data),
  getMyAnswer: (questionId: string) =>
    api.get<{ answered: boolean; answer?: UserAnswer; feedback?: string }>(`/questions/${questionId}/answers`),
  getHistory: (answerId: number) =>
    api.get<{ histories: any[] }>(`/answers/${answerId}/history`),
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
export const uploadAPI = {
  uploadQuestions: (file: File) => {
    const form = new FormData();
    form.append('file', file);
    return api.post<{ imported: number; skipped: number; message: string }>('/questions/upload', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },
  downloadTemplate: async () => {
    const res = await api.get('/questions/template', { responseType: 'blob' });
    const url = URL.createObjectURL(res.data);
    const a = document.createElement('a');
    a.href = url;
    a.download = '题目上传模板.xlsx';
    a.click();
    URL.revokeObjectURL(url);
  },
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
  getAIConfig: () => api.get<{ ai_api_key: string; ai_api_url: string; ai_model: string }>('/user/ai-config'),
  updateAIConfig: (data: { ai_api_key: string; ai_api_url: string; ai_model: string }) =>
    api.put('/user/ai-config', data),
};

export default api;
