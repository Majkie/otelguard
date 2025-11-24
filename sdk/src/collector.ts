import type { FeedbackConfig, FeedbackData, APIResponse } from './types';

/**
 * FeedbackCollector handles communication with the OTelGuard API
 */
export class FeedbackCollector {
  private config: FeedbackConfig;

  constructor(config: FeedbackConfig) {
    this.config = config;
  }

  /**
   * Submit feedback to the OTelGuard API
   */
  async submitFeedback(feedback: FeedbackData): Promise<void> {
    const url = `${this.config.apiUrl}/v1/feedback`;

    const payload = {
      projectId: feedback.projectId,
      userId: feedback.userId,
      sessionId: feedback.sessionId,
      traceId: feedback.traceId,
      spanId: feedback.spanId,
      itemType: feedback.itemType,
      itemId: feedback.itemId,
      thumbsUp: feedback.thumbsUp,
      rating: feedback.rating,
      comment: feedback.comment,
      metadata: {
        ...feedback.metadata,
        userAgent: navigator.userAgent,
        url: window.location.href,
        referrer: document.referrer,
        timestamp: new Date().toISOString(),
      },
    };

    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${this.config.apiKey}`,
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const errorData: APIResponse = await response.json().catch(() => ({}));
        throw new Error(errorData.error?.message || `HTTP ${response.status}`);
      }

      const result: APIResponse = await response.json();

      if (result.error) {
        throw new Error(result.error.message);
      }

      // Call success callback if provided
      if (this.config.onSubmit) {
        this.config.onSubmit(feedback);
      }
    } catch (error) {
      console.error('Failed to submit feedback:', error);
      throw error;
    }
  }

  /**
   * Get existing feedback for an item (if user is authenticated)
   */
  async getFeedback(itemType: string, itemId: string): Promise<any | null> {
    if (!this.config.userId) {
      return null; // Can't get feedback without user ID
    }

    const url = `${this.config.apiUrl}/v1/feedback?itemType=${itemType}&itemId=${itemId}&userId=${this.config.userId}`;

    try {
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${this.config.apiKey}`,
        },
      });

      if (!response.ok) {
        return null;
      }

      const result: APIResponse<any[]> = await response.json();

      if (result.error || !result.data || result.data.length === 0) {
        return null;
      }

      return result.data[0];
    } catch (error) {
      console.error('Failed to get feedback:', error);
      return null;
    }
  }
}
