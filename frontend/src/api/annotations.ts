import { api } from './client';

// Types
export interface AnnotationQueue {
  id: string;
  projectId: string;
  name: string;
  description: string;
  scoreConfigs: any[];
  config: Record<string, any>;
  itemSource: string;
  itemSourceConfig: Record<string, any>;
  assignmentStrategy: string;
  maxAnnotationsPerItem: number;
  instructions: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CreateAnnotationQueueRequest {
  name: string;
  description: string;
  scoreConfigs?: any[];
  config?: Record<string, any>;
  itemSource?: string;
  itemSourceConfig?: Record<string, any>;
  assignmentStrategy?: string;
  maxAnnotationsPerItem?: number;
  instructions?: string;
}

export interface UpdateAnnotationQueueRequest {
  name?: string;
  description?: string;
  scoreConfigs?: any[];
  config?: Record<string, any>;
  itemSource?: string;
  itemSourceConfig?: Record<string, any>;
  assignmentStrategy?: string;
  maxAnnotationsPerItem?: number;
  instructions?: string;
  isActive?: boolean;
}

export interface AnnotationQueueItem {
  id: string;
  queueId: string;
  itemType: string;
  itemId: string;
  itemData: Record<string, any>;
  metadata: Record<string, any>;
  priority: number;
  maxAnnotations: number;
  createdAt: string;
  updatedAt: string;
}

export interface CreateAnnotationQueueItemRequest {
  queueId: string;
  itemType: string;
  itemId: string;
  itemData?: Record<string, any>;
  metadata?: Record<string, any>;
  priority?: number;
  maxAnnotations?: number;
}

export interface AnnotationAssignment {
  id: string;
  queueItemId: string;
  userId: string;
  status: string;
  assignedAt: string;
  startedAt?: string;
  completedAt?: string;
  skippedAt?: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

export interface Annotation {
  id: string;
  assignmentId: string;
  queueId: string;
  queueItemId: string;
  userId: string;
  scores: Record<string, any>;
  labels: string[];
  notes?: string;
  confidenceScore?: number;
  annotationTime?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateAnnotationRequest {
  assignmentId: string;
  scores?: Record<string, any>;
  labels?: string[];
  notes?: string;
  confidenceScore?: number;
  annotationTime?: string;
}

export interface SkipAssignmentRequest {
  notes?: string;
}

export interface InterAnnotatorAgreement {
  id: string;
  queueId: string;
  queueItemId: string;
  scoreConfigName: string;
  agreementType: string;
  agreementValue?: number;
  annotatorCount: number;
  calculatedAt: string;
}

// Queue Management

export const annotationQueueApi = {
  // Create a new annotation queue
  create: async (projectId: string, data: CreateAnnotationQueueRequest): Promise<AnnotationQueue> => {
    const response = await api.post(`/v1/projects/${projectId}/annotation-queues`, data);
    return response.data;
  },

  // Get a specific queue
  get: async (queueId: string): Promise<AnnotationQueue> => {
    const response = await api.get(`/v1/annotation-queues/${queueId}`);
    return response.data;
  },

  // List queues for a project
  listByProject: async (projectId: string): Promise<AnnotationQueue[]> => {
    const response = await api.get(`/v1/projects/${projectId}/annotation-queues`);
    return response.data;
  },

  // Update a queue
  update: async (queueId: string, data: UpdateAnnotationQueueRequest): Promise<AnnotationQueue> => {
    const response = await api.put(`/v1/annotation-queues/${queueId}`, data);
    return response.data;
  },

  // Delete a queue
  delete: async (queueId: string): Promise<void> => {
    await api.delete(`/v1/annotation-queues/${queueId}`);
  },

  // Get queue statistics
  getStats: async (queueId: string): Promise<Record<string, any>> => {
    const response = await api.get(`/v1/annotation-queues/${queueId}/stats`);
    return response.data;
  },
};

// Queue Item Management

export const annotationQueueItemApi = {
  // Create a queue item
  create: async (data: CreateAnnotationQueueItemRequest): Promise<AnnotationQueueItem> => {
    const response = await api.post(`/v1/annotation-queues/${data.queueId}/items`, data);
    return response.data;
  },

  // List items in a queue
  list: async (queueId: string, params?: { limit?: number; offset?: number }): Promise<AnnotationQueueItem[]> => {
    const response = await api.get(`/v1/annotation-queues/${queueId}/items`, { params });
    return response.data;
  },

  // Get annotations for a queue item
  getAnnotations: async (queueItemId: string): Promise<Annotation[]> => {
    const response = await api.get(`/v1/annotation-queue-items/${queueItemId}/annotations`);
    return response.data;
  },
};

// Assignment Management

export const annotationAssignmentApi = {
  // Assign next item to current user
  assignNext: async (queueId: string): Promise<AnnotationAssignment> => {
    const response = await api.post(`/v1/annotation-queues/${queueId}/assign`);
    return response.data;
  },

  // Start working on an assignment
  start: async (assignmentId: string): Promise<void> => {
    await api.post(`/v1/annotation-assignments/${assignmentId}/start`);
  },

  // Skip an assignment
  skip: async (assignmentId: string, data?: SkipAssignmentRequest): Promise<void> => {
    await api.post(`/v1/annotation-assignments/${assignmentId}/skip`, data);
  },

  // List user assignments
  listUser: async (params?: {
    status?: string;
    limit?: number;
    offset?: number;
  }): Promise<AnnotationAssignment[]> => {
    const response = await api.get('/v1/user/annotation-assignments', { params });
    return response.data;
  },
};

// Annotation Management

export const annotationApi = {
  // Create an annotation
  create: async (data: CreateAnnotationRequest): Promise<Annotation> => {
    const response = await api.post('/v1/annotations', data);
    return response.data;
  },

  // Get a specific annotation
  get: async (annotationId: string): Promise<Annotation> => {
    const response = await api.get(`/v1/annotations/${annotationId}`);
    return response.data;
  },

  // List annotations for a queue
  listByQueue: async (queueId: string, params?: { limit?: number; offset?: number }): Promise<Annotation[]> => {
    const response = await api.get(`/v1/annotation-queues/${queueId}/annotations`, { params });
    return response.data;
  },
};

// User Statistics

export const annotationStatsApi = {
  // Get user annotation statistics
  getUserStats: async (): Promise<Record<string, any>> => {
    const response = await api.get('/v1/user/annotation-stats');
    return response.data;
  },
};

// Inter-annotator agreement API
export const interAnnotatorAgreementApi = {
  // Calculate agreement for a queue item
  calculate: async (queueId: string, queueItemId: string, scoreConfigName: string): Promise<InterAnnotatorAgreement> => {
    const response = await api.post(`/v1/annotation-queues/${queueId}/items/${queueItemId}/agreement`, null, {
      params: { scoreConfigName }
    });
    return response.data;
  },

  // Get agreements for a queue
  listByQueue: async (queueId: string, params?: { limit?: number; offset?: number }): Promise<InterAnnotatorAgreement[]> => {
    const response = await api.get(`/v1/annotation-queues/${queueId}/agreements`, { params });
    return response.data;
  },

  // Get agreement statistics for a queue
  getQueueStats: async (queueId: string): Promise<Record<string, any>> => {
    const response = await api.get(`/v1/annotation-queues/${queueId}/agreement-stats`);
    return response.data;
  },
};

// Export API
export const exportApi = {
  // Export annotations for a queue
  exportAnnotations: async (queueId: string, format: 'json' | 'csv' = 'json'): Promise<Blob> => {
    const response = await api.get(`/v1/annotation-queues/${queueId}/export`, {
      params: { format },
      responseType: 'blob',
    });
    return response.data;
  },
};

// Combined API object for convenience
export const annotationsApi = {
  queues: annotationQueueApi,
  items: annotationQueueItemApi,
  assignments: annotationAssignmentApi,
  annotations: annotationApi,
  stats: annotationStatsApi,
  agreements: interAnnotatorAgreementApi,
  export: exportApi,
};
