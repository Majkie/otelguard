import { FeedbackCollector } from './collector';
import type { FeedbackConfig, FeedbackData } from './types';

/**
 * FeedbackWidget provides a UI widget for collecting user feedback
 */
export class FeedbackWidget {
  private config: Required<FeedbackConfig>;
  private collector: FeedbackCollector;
  private container: HTMLElement | null = null;
  private isOpen = false;
  private isSubmitted = false;

  // Default configuration
  private static readonly DEFAULT_CONFIG: Partial<FeedbackConfig> = {
    position: 'bottom-right',
    primaryColor: '#3b82f6',
    showThumbs: true,
    showRating: true,
    showComment: true,
    labels: {
      title: 'How was your experience?',
      thumbsUp: 'Good',
      thumbsDown: 'Poor',
      rating: 'Rate your experience',
      comment: 'Additional comments (optional)',
      submit: 'Submit Feedback',
      thankYou: 'Thank you for your feedback!',
    },
  };

  constructor(config: FeedbackConfig) {
    this.config = { ...FeedbackWidget.DEFAULT_CONFIG, ...config } as Required<FeedbackConfig>;
    this.collector = new FeedbackCollector(config);

    this.init();
  }

  /**
   * Initialize the widget
   */
  private init(): void {
    // Create widget container
    this.container = document.createElement('div');
    this.container.className = `otelguard-feedback-widget otelguard-position-${this.config.position}`;
    this.container.innerHTML = this.getWidgetHTML();

    // Apply custom styles
    this.applyStyles();

    // Attach to DOM
    this.attachToDOM();

    // Bind events
    this.bindEvents();

    // Load existing feedback if user is authenticated
    this.loadExistingFeedback();
  }

  /**
   * Get the widget HTML structure
   */
  private getWidgetHTML(): string {
    const { labels, showThumbs, showRating, showComment } = this.config;

    return `
      <div class="otelguard-feedback-trigger">
        <button class="otelguard-feedback-button" type="button">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
          </svg>
          Feedback
        </button>
      </div>
      <div class="otelguard-feedback-modal">
        <div class="otelguard-feedback-overlay"></div>
        <div class="otelguard-feedback-content">
          <button class="otelguard-feedback-close" type="button">×</button>
          <div class="otelguard-feedback-body">
            <h3 class="otelguard-feedback-title">${labels.title}</h3>

            ${showThumbs ? `
              <div class="otelguard-feedback-thumbs">
                <button class="otelguard-feedback-thumb otelguard-thumb-up" type="button" data-value="true">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M14 9V5a3 3 0 0 0-3-3l-4 9v11h11.28a2 2 0 0 0 2-1.7l1.38-9a2 2 0 0 0-2-2.3zM7 22H4a2 2 0 0 1-2-2v-7a2 2 0 0 1 2-2h3"/>
                  </svg>
                  ${labels.thumbsUp}
                </button>
                <button class="otelguard-feedback-thumb otelguard-thumb-down" type="button" data-value="false">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M10 15v4a3 3 0 0 0 3 3l4-9V3H5.72a2 2 0 0 0-2 1.7l-1.38 9a2 2 0 0 0 2 2.3zM17 2h3a2 2 0 0 1 2 2v7a2 2 0 0 1-2 2h-3"/>
                  </svg>
                  ${labels.thumbsDown}
                </button>
              </div>
            ` : ''}

            ${showRating ? `
              <div class="otelguard-feedback-rating">
                <label class="otelguard-feedback-label">${labels.rating}</label>
                <div class="otelguard-stars">
                  ${[1, 2, 3, 4, 5].map(star => `
                    <button class="otelguard-star" type="button" data-rating="${star}">
                      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
                      </svg>
                    </button>
                  `).join('')}
                </div>
              </div>
            ` : ''}

            ${showComment ? `
              <div class="otelguard-feedback-comment">
                <label class="otelguard-feedback-label">${labels.comment}</label>
                <textarea class="otelguard-feedback-textarea" placeholder="Tell us more..." rows="3"></textarea>
              </div>
            ` : ''}

            <div class="otelguard-feedback-actions">
              <button class="otelguard-feedback-submit" type="button">${labels.submit}</button>
            </div>

            <div class="otelguard-feedback-success" style="display: none;">
              <div class="otelguard-feedback-success-message">${labels.thankYou}</div>
            </div>
          </div>
        </div>
      </div>
    `;
  }

  /**
   * Apply custom styles to the widget
   */
  private applyStyles(): void {
    if (!this.container) return;

    const style = document.createElement('style');
    style.textContent = this.getWidgetCSS();
    document.head.appendChild(style);
  }

  /**
   * Get the widget CSS styles
   */
  private getWidgetCSS(): string {
    const { primaryColor, position } = this.config;

    return `
      .otelguard-feedback-widget {
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        position: fixed;
        z-index: 9999;
      }

      .otelguard-position-bottom-right {
        bottom: 20px;
        right: 20px;
      }

      .otelguard-position-bottom-left {
        bottom: 20px;
        left: 20px;
      }

      .otelguard-position-top-right {
        top: 20px;
        right: 20px;
      }

      .otelguard-position-top-left {
        top: 20px;
        left: 20px;
      }

      .otelguard-feedback-trigger {
        margin-bottom: 10px;
      }

      .otelguard-feedback-button {
        background: ${primaryColor};
        color: white;
        border: none;
        border-radius: 8px;
        padding: 12px 16px;
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 8px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        transition: all 0.2s ease;
      }

      .otelguard-feedback-button:hover {
        transform: translateY(-1px);
        box-shadow: 0 6px 16px rgba(0, 0, 0, 0.2);
      }

      .otelguard-feedback-modal {
        display: none;
      }

      .otelguard-feedback-modal.open {
        display: block;
      }

      .otelguard-feedback-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.5);
        backdrop-filter: blur(2px);
      }

      .otelguard-feedback-content {
        position: fixed;
        ${position.includes('right') ? 'right: 20px;' : 'left: 20px;'}
        ${position.includes('bottom') ? 'bottom: 80px;' : 'top: 80px;'}
        background: white;
        border-radius: 12px;
        box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
        width: 320px;
        max-width: calc(100vw - 40px);
        max-height: calc(100vh - 120px);
        overflow: hidden;
      }

      .otelguard-feedback-close {
        position: absolute;
        top: 12px;
        right: 12px;
        background: none;
        border: none;
        font-size: 24px;
        cursor: pointer;
        color: #6b7280;
        padding: 4px;
        line-height: 1;
      }

      .otelguard-feedback-body {
        padding: 24px;
      }

      .otelguard-feedback-title {
        margin: 0 0 20px 0;
        font-size: 18px;
        font-weight: 600;
        color: #111827;
      }

      .otelguard-feedback-thumbs {
        display: flex;
        gap: 12px;
        margin-bottom: 20px;
      }

      .otelguard-feedback-thumb {
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        padding: 16px;
        border: 2px solid #e5e7eb;
        border-radius: 8px;
        background: white;
        cursor: pointer;
        transition: all 0.2s ease;
        font-size: 14px;
        font-weight: 500;
        color: #6b7280;
      }

      .otelguard-feedback-thumb:hover {
        border-color: ${primaryColor};
        color: ${primaryColor};
      }

      .otelguard-feedback-thumb.selected {
        border-color: ${primaryColor};
        background: ${primaryColor}10;
        color: ${primaryColor};
      }

      .otelguard-feedback-rating {
        margin-bottom: 20px;
      }

      .otelguard-feedback-label {
        display: block;
        margin-bottom: 8px;
        font-size: 14px;
        font-weight: 500;
        color: #374151;
      }

      .otelguard-stars {
        display: flex;
        gap: 4px;
      }

      .otelguard-star {
        background: none;
        border: none;
        cursor: pointer;
        color: #d1d5db;
        transition: color 0.2s ease;
        padding: 2px;
      }

      .otelguard-star:hover,
      .otelguard-star.selected {
        color: #fbbf24;
      }

      .otelguard-feedback-comment {
        margin-bottom: 20px;
      }

      .otelguard-feedback-textarea {
        width: 100%;
        padding: 12px;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        font-size: 14px;
        resize: vertical;
        font-family: inherit;
      }

      .otelguard-feedback-textarea:focus {
        outline: none;
        border-color: ${primaryColor};
        box-shadow: 0 0 0 3px ${primaryColor}20;
      }

      .otelguard-feedback-actions {
        text-align: right;
      }

      .otelguard-feedback-submit {
        background: ${primaryColor};
        color: white;
        border: none;
        border-radius: 6px;
        padding: 10px 20px;
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s ease;
      }

      .otelguard-feedback-submit:hover {
        background: ${primaryColor}dd;
      }

      .otelguard-feedback-submit:disabled {
        opacity: 0.6;
        cursor: not-allowed;
      }

      .otelguard-feedback-success {
        text-align: center;
        padding: 20px;
      }

      .otelguard-feedback-success-message {
        color: #059669;
        font-weight: 500;
      }

      @media (max-width: 480px) {
        .otelguard-feedback-content {
          left: 20px;
          right: 20px;
          width: auto;
        }
      }
    `;
  }

  /**
   * Attach the widget to the DOM
   */
  private attachToDOM(): void {
    if (!this.container) return;

    const target = this.config.selector
      ? document.querySelector(this.config.selector)
      : document.body;

    if (target) {
      target.appendChild(this.container);
    }
  }

  /**
   * Bind event listeners
   */
  private bindEvents(): void {
    if (!this.container) return;

    const triggerButton = this.container.querySelector('.otelguard-feedback-button') as HTMLButtonElement;
    const closeButton = this.container.querySelector('.otelguard-feedback-close') as HTMLButtonElement;
    const overlay = this.container.querySelector('.otelguard-feedback-overlay') as HTMLElement;
    const submitButton = this.container.querySelector('.otelguard-feedback-submit') as HTMLButtonElement;

    // Trigger button
    triggerButton?.addEventListener('click', () => this.open());

    // Close events
    closeButton?.addEventListener('click', () => this.close());
    overlay?.addEventListener('click', () => this.close());

    // Thumbs up/down
    const thumbButtons = this.container.querySelectorAll('.otelguard-feedback-thumb');
    thumbButtons.forEach(button => {
      button.addEventListener('click', (e) => {
        const target = e.currentTarget as HTMLElement;
        const value = target.dataset.value === 'true';

        // Remove selected class from all thumbs
        thumbButtons.forEach(btn => btn.classList.remove('selected'));
        // Add selected class to clicked button
        target.classList.add('selected');

        this.selectedThumbs = value;
      });
    });

    // Star rating
    const starButtons = this.container.querySelectorAll('.otelguard-star');
    starButtons.forEach((button, index) => {
      button.addEventListener('click', () => {
        const rating = index + 1;

        // Update selected state
        starButtons.forEach((star, i) => {
          star.classList.toggle('selected', i < rating);
        });

        this.selectedRating = rating;
      });
    });

    // Submit
    submitButton?.addEventListener('click', () => this.submit());
  }

  /**
   * Load existing feedback for the current user/item
   */
  private async loadExistingFeedback(): Promise<void> {
    try {
      const existing = await this.collector.getFeedback(this.config.itemType, this.config.itemId);
      if (existing) {
        this.populateExistingFeedback(existing);
      }
    } catch (error) {
      console.warn('Failed to load existing feedback:', error);
    }
  }

  /**
   * Populate the widget with existing feedback data
   */
  private populateExistingFeedback(feedback: any): void {
    if (!this.container) return;

    // Set thumbs
    if (feedback.thumbsUp !== undefined) {
      const thumbButton = this.container.querySelector(
        `.otelguard-feedback-thumb[data-value="${feedback.thumbsUp}"]`
      ) as HTMLElement;
      thumbButton?.classList.add('selected');
      this.selectedThumbs = feedback.thumbsUp;
    }

    // Set rating
    if (feedback.rating) {
      const starButtons = this.container.querySelectorAll('.otelguard-star');
      starButtons.forEach((star, index) => {
        star.classList.toggle('selected', index < feedback.rating);
      });
      this.selectedRating = feedback.rating;
    }

    // Set comment
    if (feedback.comment) {
      const textarea = this.container.querySelector('.otelguard-feedback-textarea') as HTMLTextAreaElement;
      if (textarea) {
        textarea.value = feedback.comment;
      }
    }

    this.isSubmitted = true;
    this.showSuccess();
  }

  /**
   * Open the feedback widget
   */
  open(): void {
    if (!this.container || this.isOpen) return;

    const modal = this.container.querySelector('.otelguard-feedback-modal') as HTMLElement;
    modal?.classList.add('open');
    this.isOpen = true;

    if (this.config.onOpen) {
      this.config.onOpen();
    }
  }

  /**
   * Close the feedback widget
   */
  close(): void {
    if (!this.container || !this.isOpen) return;

    const modal = this.container.querySelector('.otelguard-feedback-modal') as HTMLElement;
    modal?.classList.remove('open');
    this.isOpen = false;

    if (this.config.onClose) {
      this.config.onClose();
    }
  }

  /**
   * Submit the feedback
   */
  private async submit(): Promise<void> {
    if (!this.container) return;

    const submitButton = this.container.querySelector('.otelguard-feedback-submit') as HTMLButtonElement;
    const textarea = this.container.querySelector('.otelguard-feedback-textarea') as HTMLTextAreaElement;

    // Disable submit button
    submitButton.disabled = true;
    submitButton.textContent = 'Submitting...';

    try {
      const feedbackData: FeedbackData = {
        projectId: this.config.projectId,
        userId: this.config.userId,
        itemType: this.config.itemType,
        itemId: this.config.itemId,
        traceId: this.config.traceId,
        sessionId: this.config.sessionId,
        spanId: this.config.spanId,
        thumbsUp: this.selectedThumbs === undefined ? null : this.selectedThumbs,
        rating: this.selectedRating,
        comment: textarea?.value || undefined,
      };

      await this.collector.submitFeedback(feedbackData);

      this.isSubmitted = true;
      this.showSuccess();

      // Auto-close after 2 seconds
      setTimeout(() => this.close(), 2000);

    } catch (error) {
      console.error('Failed to submit feedback:', error);
      // Re-enable submit button
      submitButton.disabled = false;
      submitButton.textContent = this.config.labels.submit || 'Submit Feedback';

      // Show error (you could add a proper error display here)
      alert('Failed to submit feedback. Please try again.');
    }
  }

  /**
   * Show success message
   */
  private showSuccess(): void {
    if (!this.container) return;

    const body = this.container.querySelector('.otelguard-feedback-body') as HTMLElement;
    const success = this.container.querySelector('.otelguard-feedback-success') as HTMLElement;

    if (body && success) {
      body.style.display = 'none';
      success.style.display = 'block';
    }
  }

  // Internal state
  private selectedThumbs?: boolean;
  private selectedRating?: number;
}

/**
 * Initialize the feedback widget
 */
export function initFeedbackWidget(config: FeedbackConfig): FeedbackWidget {
  return new FeedbackWidget(config);
}

/**
 * Auto-initialize widgets from DOM data attributes
 */
export function autoInitFeedbackWidgets(): FeedbackWidget[] {
  const widgets: FeedbackWidget[] = [];
  const containers = document.querySelectorAll('[data-otelguard-feedback]');

  containers.forEach(container => {
    const configElement = container as HTMLElement;
    const config: FeedbackConfig = {
      apiUrl: configElement.dataset.apiUrl || '',
      apiKey: configElement.dataset.apiKey || '',
      projectId: configElement.dataset.projectId || '',
      userId: configElement.dataset.userId,
      itemType: (configElement.dataset.itemType as any) || 'trace',
      itemId: configElement.dataset.itemId || '',
      traceId: configElement.dataset.traceId,
      sessionId: configElement.dataset.sessionId,
      spanId: configElement.dataset.spanId,
      position: (configElement.dataset.position as any) || 'bottom-right',
      primaryColor: configElement.dataset.primaryColor,
      selector: `[data-otelguard-feedback="${configElement.dataset.otelguardFeedback}"]`,
    };

    widgets.push(new FeedbackWidget(config));
  });

  return widgets;
}
