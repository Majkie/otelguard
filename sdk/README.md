# OTelGuard Feedback SDK

A lightweight JavaScript/TypeScript SDK for collecting user feedback in web applications with seamless integration to OTelGuard.

## Features

- 🎯 **Easy Integration** - Drop-in widget or programmatic API
- 🎨 **Customizable UI** - Themes, positioning, and labels
- ⭐ **Multiple Feedback Types** - Thumbs up/down, star ratings, and comments
- 🔄 **Real-time Sync** - Automatic submission to OTelGuard API
- 📱 **Responsive Design** - Works on all screen sizes
- 🔒 **Secure** - API key authentication and data validation

## Installation

### NPM
```bash
npm install @otelguard/feedback-sdk
```

### CDN
```html
<script src="https://cdn.otelguard.dev/feedback-sdk/v1/index.js"></script>
```

## Quick Start

### Basic Widget

```html
<!DOCTYPE html>
<html>
<head>
  <title>My App</title>
</head>
<body>
  <!-- Your app content -->

  <script src="https://cdn.otelguard.dev/feedback-sdk/v1/index.js"></script>
  <script>
    OTelGuard.initFeedbackWidget({
      apiUrl: 'https://api.otelguard.dev',
      apiKey: 'your-api-key',
      projectId: 'your-project-id',
      itemType: 'trace',
      itemId: 'trace-123',
      position: 'bottom-right'
    });
  </script>
</body>
</html>
```

### Using NPM

```typescript
import { FeedbackWidget } from '@otelguard/feedback-sdk';

const widget = new FeedbackWidget({
  apiUrl: 'https://api.otelguard.dev',
  apiKey: 'your-api-key',
  projectId: 'your-project-id',
  itemType: 'session',
  itemId: 'session-456',
  position: 'bottom-left',
  primaryColor: '#ff6b6b'
});

// Programmatically open
widget.open();
```

### Auto-initialization from HTML

```html
<div data-otelguard-feedback="widget1"
     data-api-url="https://api.otelguard.dev"
     data-api-key="your-api-key"
     data-project-id="your-project-id"
     data-item-type="prompt"
     data-item-id="prompt-789"
     data-position="top-right">
</div>

<script>
  OTelGuard.autoInitFeedbackWidgets();
</script>
```

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `apiUrl` | `string` | - | OTelGuard API endpoint URL |
| `apiKey` | `string` | - | API key for authentication |
| `projectId` | `string` | - | Your OTelGuard project ID |
| `userId` | `string?` | - | User ID for authenticated feedback |
| `itemType` | `'trace' \| 'session' \| 'span' \| 'prompt'` | `'trace'` | Type of item being rated |
| `itemId` | `string` | - | ID of the item being rated |
| `traceId` | `string?` | - | Trace ID for context |
| `sessionId` | `string?` | - | Session ID for context |
| `spanId` | `string?` | - | Span ID for context |
| `selector` | `string?` | - | CSS selector to attach widget to |
| `position` | `'bottom-right' \| 'bottom-left' \| 'top-right' \| 'top-left'` | `'bottom-right'` | Widget position |
| `primaryColor` | `string` | `'#3b82f6'` | Primary color for the widget |
| `showThumbs` | `boolean` | `true` | Show thumbs up/down buttons |
| `showRating` | `boolean` | `true` | Show star rating |
| `showComment` | `boolean` | `true` | Show comment textarea |
| `labels` | `object` | - | Custom text labels |
| `onSubmit` | `function` | - | Callback when feedback is submitted |
| `onOpen` | `function` | - | Callback when widget opens |
| `onClose` | `function` | - | Callback when widget closes |

## API Reference

### FeedbackWidget

#### Methods

- `open()` - Open the feedback widget
- `close()` - Close the feedback widget

#### Events

- `onSubmit(feedback: FeedbackData)` - Fired when feedback is successfully submitted
- `onOpen()` - Fired when the widget is opened
- `onClose()` - Fired when the widget is closed

### FeedbackCollector

Handles direct API communication for custom implementations.

```typescript
import { FeedbackCollector } from '@otelguard/feedback-sdk';

const collector = new FeedbackCollector({
  apiUrl: 'https://api.otelguard.dev',
  apiKey: 'your-api-key',
  projectId: 'your-project-id'
});

await collector.submitFeedback({
  itemType: 'trace',
  itemId: 'trace-123',
  rating: 5,
  comment: 'Great experience!'
});
```

## Styling

The widget comes with default styles but can be customized:

```css
/* Custom widget styles */
.otelguard-feedback-widget {
  --otelguard-primary: #your-color;
  --otelguard-font-family: 'Your Font', sans-serif;
}

/* Hide specific elements */
.otelguard-feedback-rating {
  display: none;
}
```

## Examples

### AI Chatbot Feedback

```typescript
// After each AI response
const widget = new FeedbackWidget({
  apiUrl: 'https://api.otelguard.dev',
  apiKey: 'your-api-key',
  projectId: 'chatbot-project',
  itemType: 'session',
  itemId: currentSessionId,
  traceId: currentTraceId,
  position: 'inline',
  selector: '#feedback-container',
  labels: {
    title: 'How helpful was this response?',
    thumbsUp: 'Helpful',
    thumbsDown: 'Not helpful'
  },
  showRating: false, // Only thumbs and comments
  onSubmit: (feedback) => {
    console.log('Feedback submitted:', feedback);
    // Analytics tracking, etc.
  }
});
```

### Document Q&A Feedback

```html
<!-- Embed in your document viewer -->
<div id="doc-feedback"></div>

<script>
OTelGuard.initFeedbackWidget({
  apiUrl: 'https://api.otelguard.dev',
  apiKey: 'your-api-key',
  projectId: 'docs-project',
  itemType: 'prompt',
  itemId: documentId,
  selector: '#doc-feedback',
  position: 'inline',
  showThumbs: true,
  showRating: true,
  showComment: true,
  labels: {
    title: 'Was this document helpful?',
    comment: 'What could be improved?'
  }
});
</script>
```

## Security

- API keys are required for all requests
- User feedback is validated server-side
- No sensitive data is stored in the widget
- All network requests use HTTPS

## Browser Support

- Chrome 60+
- Firefox 60+
- Safari 12+
- Edge 79+

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions welcome! Please see the main OTelGuard repository for contribution guidelines.
