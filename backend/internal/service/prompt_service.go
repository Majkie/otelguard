package service

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"go.uber.org/zap"
)

// PromptService handles prompt business logic
type PromptService struct {
	promptRepo *postgres.PromptRepository
	logger     *zap.Logger
}

// NewPromptService creates a new prompt service
func NewPromptService(promptRepo *postgres.PromptRepository, logger *zap.Logger) *PromptService {
	return &PromptService{
		promptRepo: promptRepo,
		logger:     logger,
	}
}

// ListOptions contains options for listing resources
type ListOptions struct {
	Limit  int
	Offset int
}

// Create creates a new prompt
func (s *PromptService) Create(ctx context.Context, prompt *domain.Prompt) error {
	return s.promptRepo.Create(ctx, prompt)
}

// GetByID retrieves a prompt by ID
func (s *PromptService) GetByID(ctx context.Context, id string) (*domain.Prompt, error) {
	return s.promptRepo.GetByID(ctx, id)
}

// List returns prompts for a project
func (s *PromptService) List(ctx context.Context, projectID string, opts *ListOptions) ([]*domain.Prompt, int, error) {
	return s.promptRepo.List(ctx, projectID, &postgres.ListOptions{
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

// Update updates a prompt
func (s *PromptService) Update(ctx context.Context, prompt *domain.Prompt) error {
	return s.promptRepo.Update(ctx, prompt)
}

// Delete soft-deletes a prompt
func (s *PromptService) Delete(ctx context.Context, id string) error {
	return s.promptRepo.Delete(ctx, id)
}

// CreateVersion creates a new version of a prompt
func (s *PromptService) CreateVersion(ctx context.Context, promptID string, content string, config []byte, labels []string, createdBy *uuid.UUID) (*domain.PromptVersion, error) {
	// Verify prompt exists
	_, err := s.promptRepo.GetByID(ctx, promptID)
	if err != nil {
		return nil, err
	}

	// Get the next version number
	latestVersion, err := s.promptRepo.GetLatestVersion(ctx, promptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	promptUUID, err := uuid.Parse(promptID)
	if err != nil {
		return nil, fmt.Errorf("invalid prompt ID: %w", err)
	}

	version := &domain.PromptVersion{
		ID:        uuid.New(),
		PromptID:  promptUUID,
		Version:   latestVersion + 1,
		Content:   content,
		Config:    config,
		Labels:    labels,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}

	if err := s.promptRepo.CreateVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	return version, nil
}

// GetVersion retrieves a specific version of a prompt
func (s *PromptService) GetVersion(ctx context.Context, promptID string, version int) (*domain.PromptVersion, error) {
	return s.promptRepo.GetVersion(ctx, promptID, version)
}

// GetLatestVersion retrieves the latest version of a prompt
func (s *PromptService) GetLatestVersion(ctx context.Context, promptID string) (*domain.PromptVersion, error) {
	latestVersionNum, err := s.promptRepo.GetLatestVersion(ctx, promptID)
	if err != nil {
		return nil, err
	}
	if latestVersionNum == 0 {
		return nil, domain.ErrNotFound
	}
	return s.promptRepo.GetVersion(ctx, promptID, latestVersionNum)
}

// ListVersions returns all versions of a prompt
func (s *PromptService) ListVersions(ctx context.Context, promptID string) ([]*domain.PromptVersion, error) {
	// Verify prompt exists
	_, err := s.promptRepo.GetByID(ctx, promptID)
	if err != nil {
		return nil, err
	}
	return s.promptRepo.ListVersions(ctx, promptID)
}

// GetVersionCount returns the number of versions for a prompt
func (s *PromptService) GetVersionCount(ctx context.Context, promptID string) (int, error) {
	return s.promptRepo.GetLatestVersion(ctx, promptID)
}

// Duplicate creates a copy of a prompt with a new name
func (s *PromptService) Duplicate(ctx context.Context, promptID string, newName string) (*domain.Prompt, error) {
	// Get the original prompt
	original, err := s.promptRepo.GetByID(ctx, promptID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	newPrompt := &domain.Prompt{
		ID:          uuid.New(),
		ProjectID:   original.ProjectID,
		Name:        newName,
		Description: original.Description,
		Tags:        original.Tags,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.promptRepo.Create(ctx, newPrompt); err != nil {
		return nil, fmt.Errorf("failed to create duplicate prompt: %w", err)
	}

	// Copy all versions from the original prompt
	versions, err := s.promptRepo.ListVersions(ctx, promptID)
	if err != nil {
		s.logger.Warn("failed to list versions for duplication", zap.Error(err))
		return newPrompt, nil // Return the prompt even if versions fail to copy
	}

	for _, v := range versions {
		newVersion := &domain.PromptVersion{
			ID:        uuid.New(),
			PromptID:  newPrompt.ID,
			Version:   v.Version,
			Content:   v.Content,
			Config:    v.Config,
			Labels:    v.Labels,
			CreatedBy: v.CreatedBy,
			CreatedAt: time.Now(),
		}
		if err := s.promptRepo.CreateVersion(ctx, newVersion); err != nil {
			s.logger.Warn("failed to copy version", zap.Int("version", v.Version), zap.Error(err))
		}
	}

	return newPrompt, nil
}

// CompileResult contains the result of compiling a prompt template
type CompileResult struct {
	Compiled  string   `json:"compiled"`
	Variables []string `json:"variables"`
	Missing   []string `json:"missing,omitempty"`
	Errors    []string `json:"errors,omitempty"`
}

// CompileTemplate compiles a prompt template with the given variables
func (s *PromptService) CompileTemplate(ctx context.Context, promptID string, version int, variables map[string]interface{}) (*CompileResult, error) {
	var content string

	if version > 0 {
		// Get specific version
		v, err := s.promptRepo.GetVersion(ctx, promptID, version)
		if err != nil {
			return nil, err
		}
		content = v.Content
	} else {
		// Get latest version
		latestVersion, err := s.promptRepo.GetLatestVersion(ctx, promptID)
		if err != nil {
			return nil, err
		}
		if latestVersion == 0 {
			return nil, domain.ErrNotFound
		}
		v, err := s.promptRepo.GetVersion(ctx, promptID, latestVersion)
		if err != nil {
			return nil, err
		}
		content = v.Content
	}

	return s.compileContent(content, variables)
}

// compileContent compiles template content with variables
func (s *PromptService) compileContent(content string, variables map[string]interface{}) (*CompileResult, error) {
	result := &CompileResult{
		Variables: extractVariables(content),
	}

	// Check for missing variables
	if variables == nil {
		variables = make(map[string]interface{})
	}

	for _, v := range result.Variables {
		if _, ok := variables[v]; !ok {
			result.Missing = append(result.Missing, v)
		}
	}

	// Convert Jinja2-style {{variable}} to Go template style {{.variable}}
	goTemplateContent := convertToGoTemplate(content)

	// Parse and execute the template
	tmpl, err := template.New("prompt").Parse(goTemplateContent)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("template parse error: %v", err))
		result.Compiled = content // Return original content on error
		return result, nil
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("template execution error: %v", err))
		result.Compiled = content // Return original content on error
		return result, nil
	}

	result.Compiled = buf.String()
	return result, nil
}

// extractVariables extracts variable names from a template
// Supports {{variable_name}} syntax and {{#if variable}} and {{#each variable}} syntax
func extractVariables(content string) []string {
	// Match simple variables: {{variable}}
	simpleVarRe := regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)
	// Match if conditions: {{#if variable}}
	ifVarRe := regexp.MustCompile(`\{\{#if\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)
	// Match unless conditions: {{#unless variable}}
	unlessVarRe := regexp.MustCompile(`\{\{#unless\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)
	// Match each loops: {{#each variable}}
	eachVarRe := regexp.MustCompile(`\{\{#each\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)
	// Match with blocks: {{#with variable}}
	withVarRe := regexp.MustCompile(`\{\{#with\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)

	// Use map to deduplicate
	seen := make(map[string]bool)
	var variables []string

	// Helper to add matches
	addMatches := func(re *regexp.Regexp) {
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 && !seen[match[1]] {
				seen[match[1]] = true
				variables = append(variables, match[1])
			}
		}
	}

	addMatches(simpleVarRe)
	addMatches(ifVarRe)
	addMatches(unlessVarRe)
	addMatches(eachVarRe)
	addMatches(withVarRe)

	return variables
}

// convertToGoTemplate converts Jinja2/Mustache-style templates to Go template syntax
// Supports:
//   - {{variable}} -> {{.variable}}
//   - {{#if condition}}...{{/if}} -> {{if .condition}}...{{end}}
//   - {{#unless condition}}...{{/unless}} -> {{if not .condition}}...{{end}}
//   - {{#each array}}...{{/each}} -> {{range .array}}...{{end}}
//   - {{#with object}}...{{/with}} -> {{with .object}}...{{end}}
//   - {{else}} -> {{else}}
func convertToGoTemplate(content string) string {
	// Order matters - process control structures first, then simple variables

	// Convert {{#if variable}} to {{if .variable}}
	content = regexp.MustCompile(`\{\{#if\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`).
		ReplaceAllString(content, "{{if .${1}}}")

	// Convert {{/if}} to {{end}}
	content = regexp.MustCompile(`\{\{/if\s*\}\}`).
		ReplaceAllString(content, "{{end}}")

	// Convert {{#unless variable}} to {{if not .variable}}
	content = regexp.MustCompile(`\{\{#unless\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`).
		ReplaceAllString(content, "{{if not .${1}}}")

	// Convert {{/unless}} to {{end}}
	content = regexp.MustCompile(`\{\{/unless\s*\}\}`).
		ReplaceAllString(content, "{{end}}")

	// Convert {{#each array}} to {{range .array}}
	content = regexp.MustCompile(`\{\{#each\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`).
		ReplaceAllString(content, "{{range .${1}}}")

	// Convert {{/each}} to {{end}}
	content = regexp.MustCompile(`\{\{/each\s*\}\}`).
		ReplaceAllString(content, "{{end}}")

	// Convert {{#with object}} to {{with .object}}
	content = regexp.MustCompile(`\{\{#with\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`).
		ReplaceAllString(content, "{{with .${1}}}")

	// Convert {{/with}} to {{end}}
	content = regexp.MustCompile(`\{\{/with\s*\}\}`).
		ReplaceAllString(content, "{{end}}")

	// Convert {{else}} - this stays the same in Go templates
	// No conversion needed for {{else}}

	// Convert special loop variables {{this}} to {{.}}
	content = regexp.MustCompile(`\{\{\s*this\s*\}\}`).
		ReplaceAllString(content, "{{.}}")

	// Convert {{@index}} to {{$index}} (loop index in range)
	// Note: Go range provides index automatically when using: {{range $index, $element := .array}}
	// For simplicity, we'll leave @index as a placeholder that users should adapt

	// Finally, convert simple variables {{variable}} to {{.variable}}
	// But avoid converting already-converted Go template syntax
	simpleVarRe := regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)
	content = simpleVarRe.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the variable name
		submatch := simpleVarRe.FindStringSubmatch(match)
		if len(submatch) > 1 {
			varName := submatch[1]
			// Don't convert Go template keywords
			keywords := map[string]bool{
				"if": true, "else": true, "end": true, "range": true,
				"with": true, "define": true, "template": true, "block": true,
				"not": true, "and": true, "or": true, "eq": true, "ne": true,
				"lt": true, "le": true, "gt": true, "ge": true,
			}
			if keywords[varName] {
				return match
			}
			return "{{." + varName + "}}"
		}
		return match
	})

	return content
}

// UpdateVersionLabels updates the labels for a specific version
func (s *PromptService) UpdateVersionLabels(ctx context.Context, promptID string, version int, labels []string) error {
	return s.promptRepo.UpdateVersionLabels(ctx, promptID, version, labels)
}

// PromoteVersion promotes a version to a new environment label
// It removes the label from any other version and adds it to the target version
func (s *PromptService) PromoteVersion(ctx context.Context, promptID string, version int, targetLabel string) error {
	// Valid promotion labels
	validLabels := map[string]bool{
		"production":  true,
		"staging":     true,
		"development": true,
	}

	if !validLabels[targetLabel] {
		return fmt.Errorf("invalid promotion target: %s", targetLabel)
	}

	// Get all versions to remove the label from others
	versions, err := s.promptRepo.ListVersions(ctx, promptID)
	if err != nil {
		return err
	}

	// Remove the target label from all other versions
	for _, v := range versions {
		if v.Version != version && containsLabel(v.Labels, targetLabel) {
			newLabels := removeLabel(v.Labels, targetLabel)
			if err := s.promptRepo.UpdateVersionLabels(ctx, promptID, v.Version, newLabels); err != nil {
				s.logger.Warn("failed to remove label from version",
					zap.Int("version", v.Version),
					zap.String("label", targetLabel),
					zap.Error(err))
			}
		}
	}

	// Get the target version and add the label
	targetVersion, err := s.promptRepo.GetVersion(ctx, promptID, version)
	if err != nil {
		return err
	}

	// Add the label if not already present
	if !containsLabel(targetVersion.Labels, targetLabel) {
		newLabels := append(targetVersion.Labels, targetLabel)
		return s.promptRepo.UpdateVersionLabels(ctx, promptID, version, newLabels)
	}

	return nil
}

// GetVersionByLabel retrieves the version with a specific label (e.g., "production")
func (s *PromptService) GetVersionByLabel(ctx context.Context, promptID string, label string) (*domain.PromptVersion, error) {
	versions, err := s.promptRepo.ListVersions(ctx, promptID)
	if err != nil {
		return nil, err
	}

	for _, v := range versions {
		if containsLabel(v.Labels, label) {
			return v, nil
		}
	}

	return nil, domain.ErrNotFound
}

func containsLabel(labels []string, target string) bool {
	for _, l := range labels {
		if l == target {
			return true
		}
	}
	return false
}

func removeLabel(labels []string, target string) []string {
	result := make([]string, 0, len(labels))
	for _, l := range labels {
		if l != target {
			result = append(result, l)
		}
	}
	return result
}
