/**
 * Configuration for the feedback widget
 */
export interface FeedbackConfig {
  /** OTelGuard API endpoint */
  apiUrl: string;
  /** API key for authentication */
  apiKey: string;
  /** Project ID */
  projectId: string;
  /** User ID (optional, for authenticated feedback) */
  userId?: string;
  /** Item type being rated ('trace', 'session', 'span', 'prompt') */
  itemType: 'trace' | 'session' | 'span' | 'prompt';
  /** Item ID being rated */
  itemId: string;
  /** Optional trace ID for context */
  traceId?: string;
  /** Optional session ID for context */
  sessionId?: string;
  /** Optional span ID for context */
  spanId?: string;
  /** Custom CSS selector to attach the widget to */
  selector?: string;
  /** Position of the widget ('bottom-right', 'bottom-left', 'top-right', 'top-left', 'inline') */
  position?: 'bottom-right' | 'bottom-left' | 'top-right' | 'top-left' | 'inline';
  /** Primary color for the widget */
  primaryColor?: string;
  /** Show thumbs up/down buttons */
  showThumbs?: boolean;
  /** Show star rating */
  showRating?: boolean;
  /** Show comment textarea */
  showComment?: boolean;
  /** Custom labels */
  labels?: {
    title?: string;
    thumbsUp?: string;
    thumbsDown?: string;
    rating?: string;
    comment?: string;
    submit?: string;
    thankYou?: string;
  };
  /** Callback when feedback is submitted */
  onSubmit?: (feedback: FeedbackData) => void;
  /** Callback when widget is opened */
  onOpen?: () => void;
  /** Callback when widget is closed */
  onClose?: () => void;
}

/**
 * Feedback data structure
 */
export interface FeedbackData {
  /** Project ID */
  projectId: string;
  /** User ID (optional) */
  userId?: string;
  /** Item type */
  itemType: 'trace' | 'session' | 'span' | 'prompt';
  /** Item ID */
  itemId: string;
  /** Trace ID for context */
  traceId?: string;
  /** Session ID for context */
  sessionId?: string;
  /** Span ID for context */
  spanId?: string;
  /** Thumbs up/down (true = up, false = down, null = no opinion) */
  thumbsUp?: boolean | null;
  /** Star rating (1-5) */
  rating?: number;
  /** Free-form comment */
  comment?: string;
  /** Additional metadata */
  metadata?: Record<string, any>;
}

/**
 * Feedback item for internal use
 */
export interface FeedbackItem {
  id: string;
  projectId: string;
  userId?: string;
  sessionId?: string;
  traceId?: string;
  spanId?: string;
  itemType: string;
  itemId: string;
  thumbsUp?: boolean;
  rating?: number;
  comment?: string;
  metadata?: Record<string, any>;
  createdAt: string;
  updatedAt: string;
}

/**
 * API response structure
 */
export interface APIResponse<T = any> {
  data?: T;
  error?: {
    type: string;
    message: string;
    details?: any[];
  };
}
