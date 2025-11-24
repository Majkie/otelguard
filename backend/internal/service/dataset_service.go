package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"go.uber.org/zap"
)

// DatasetService handles dataset business logic
type DatasetService struct {
	datasetRepo *postgres.DatasetRepository
	logger      *zap.Logger
}

// NewDatasetService creates a new dataset service
func NewDatasetService(datasetRepo *postgres.DatasetRepository, logger *zap.Logger) *DatasetService {
	return &DatasetService{
		datasetRepo: datasetRepo,
		logger:      logger,
	}
}

// Create creates a new dataset
func (s *DatasetService) Create(ctx context.Context, input *domain.DatasetCreate) (*domain.Dataset, error) {
	dataset := &domain.Dataset{
		ID:          uuid.New(),
		ProjectID:   input.ProjectID,
		Name:        input.Name,
		Description: input.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.datasetRepo.Create(ctx, dataset); err != nil {
		s.logger.Error("failed to create dataset", zap.Error(err))
		return nil, fmt.Errorf("failed to create dataset: %w", err)
	}

	return dataset, nil
}

// GetByID retrieves a dataset by ID
func (s *DatasetService) GetByID(ctx context.Context, id string) (*domain.Dataset, error) {
	return s.datasetRepo.GetByID(ctx, id)
}

// List returns datasets for a project
func (s *DatasetService) List(ctx context.Context, projectID string, opts *ListOptions) ([]*domain.Dataset, int, error) {
	return s.datasetRepo.List(ctx, projectID, &postgres.ListOptions{
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

// Update updates a dataset
func (s *DatasetService) Update(ctx context.Context, id string, input *domain.DatasetUpdate) (*domain.Dataset, error) {
	dataset, err := s.datasetRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		dataset.Name = *input.Name
	}
	if input.Description != nil {
		dataset.Description = *input.Description
	}
	dataset.UpdatedAt = time.Now()

	if err := s.datasetRepo.Update(ctx, dataset); err != nil {
		s.logger.Error("failed to update dataset", zap.Error(err))
		return nil, fmt.Errorf("failed to update dataset: %w", err)
	}

	return dataset, nil
}

// Delete soft-deletes a dataset
func (s *DatasetService) Delete(ctx context.Context, id string) error {
	return s.datasetRepo.Delete(ctx, id)
}

// CreateItem creates a new dataset item
func (s *DatasetService) CreateItem(ctx context.Context, input *domain.DatasetItemCreate) (*domain.DatasetItem, error) {
	// Marshal input to JSON
	inputJSON, err := json.Marshal(input.Input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	// Marshal expected output to JSON if provided
	var expectedOutputJSON []byte
	if input.ExpectedOutput != nil {
		expectedOutputJSON, err = json.Marshal(input.ExpectedOutput)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal expected output: %w", err)
		}
	}

	// Marshal metadata to JSON if provided
	var metadataJSON []byte
	if input.Metadata != nil {
		metadataJSON, err = json.Marshal(input.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	item := &domain.DatasetItem{
		ID:             uuid.New(),
		DatasetID:      input.DatasetID,
		Input:          inputJSON,
		ExpectedOutput: expectedOutputJSON,
		Metadata:       metadataJSON,
		CreatedAt:      time.Now(),
	}

	if err := s.datasetRepo.CreateItem(ctx, item); err != nil {
		s.logger.Error("failed to create dataset item", zap.Error(err))
		return nil, fmt.Errorf("failed to create dataset item: %w", err)
	}

	return item, nil
}

// GetItemByID retrieves a dataset item by ID
func (s *DatasetService) GetItemByID(ctx context.Context, id string) (*domain.DatasetItem, error) {
	return s.datasetRepo.GetItemByID(ctx, id)
}

// ListItems returns items for a dataset
func (s *DatasetService) ListItems(ctx context.Context, datasetID string, opts *ListOptions) ([]*domain.DatasetItem, int, error) {
	return s.datasetRepo.ListItems(ctx, datasetID, &postgres.ListOptions{
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

// UpdateItem updates a dataset item
func (s *DatasetService) UpdateItem(ctx context.Context, id string, input *domain.DatasetItemUpdate) (*domain.DatasetItem, error) {
	item, err := s.datasetRepo.GetItemByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Input != nil {
		inputJSON, err := json.Marshal(input.Input)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal input: %w", err)
		}
		item.Input = inputJSON
	}

	if input.ExpectedOutput != nil {
		expectedOutputJSON, err := json.Marshal(input.ExpectedOutput)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal expected output: %w", err)
		}
		item.ExpectedOutput = expectedOutputJSON
	}

	if input.Metadata != nil {
		metadataJSON, err := json.Marshal(input.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		item.Metadata = metadataJSON
	}

	if err := s.datasetRepo.UpdateItem(ctx, item); err != nil {
		s.logger.Error("failed to update dataset item", zap.Error(err))
		return nil, fmt.Errorf("failed to update dataset item: %w", err)
	}

	return item, nil
}

// DeleteItem deletes a dataset item
func (s *DatasetService) DeleteItem(ctx context.Context, id string) error {
	return s.datasetRepo.DeleteItem(ctx, id)
}

// Import imports dataset items from JSON or CSV
func (s *DatasetService) Import(ctx context.Context, input *domain.DatasetImport) (int, error) {
	var items []domain.DatasetItemInput

	switch input.Format {
	case "json":
		if input.Items != nil {
			items = input.Items
		} else if input.Data != "" {
			if err := json.Unmarshal([]byte(input.Data), &items); err != nil {
				return 0, fmt.Errorf("failed to parse JSON data: %w", err)
			}
		} else {
			return 0, fmt.Errorf("no items or data provided for JSON import")
		}

	case "csv":
		if input.Data == "" {
			return 0, fmt.Errorf("no data provided for CSV import")
		}
		parsedItems, err := s.parseCSV(input.Data)
		if err != nil {
			return 0, fmt.Errorf("failed to parse CSV data: %w", err)
		}
		items = parsedItems

	default:
		return 0, fmt.Errorf("unsupported import format: %s", input.Format)
	}

	if len(items) == 0 {
		return 0, fmt.Errorf("no items to import")
	}

	// Convert to domain.DatasetItem
	datasetItems := make([]*domain.DatasetItem, 0, len(items))
	for _, item := range items {
		inputJSON, err := json.Marshal(item.Input)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal input: %w", err)
		}

		var expectedOutputJSON []byte
		if item.ExpectedOutput != nil {
			expectedOutputJSON, err = json.Marshal(item.ExpectedOutput)
			if err != nil {
				return 0, fmt.Errorf("failed to marshal expected output: %w", err)
			}
		}

		var metadataJSON []byte
		if item.Metadata != nil {
			metadataJSON, err = json.Marshal(item.Metadata)
			if err != nil {
				return 0, fmt.Errorf("failed to marshal metadata: %w", err)
			}
		}

		datasetItems = append(datasetItems, &domain.DatasetItem{
			ID:             uuid.New(),
			DatasetID:      input.DatasetID,
			Input:          inputJSON,
			ExpectedOutput: expectedOutputJSON,
			Metadata:       metadataJSON,
			CreatedAt:      time.Now(),
		})
	}

	// Bulk insert
	if err := s.datasetRepo.BulkCreateItems(ctx, datasetItems); err != nil {
		s.logger.Error("failed to bulk create dataset items", zap.Error(err))
		return 0, fmt.Errorf("failed to import dataset items: %w", err)
	}

	s.logger.Info("successfully imported dataset items",
		zap.String("dataset_id", input.DatasetID.String()),
		zap.Int("count", len(datasetItems)),
	)

	return len(datasetItems), nil
}

// parseCSV parses CSV data into dataset items
// Expected format: input columns, expected_output (optional), metadata (optional)
func (s *DatasetService) parseCSV(data string) ([]domain.DatasetItemInput, error) {
	reader := csv.NewReader(strings.NewReader(data))
	reader.TrimLeadingSpace = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	if len(header) == 0 {
		return nil, fmt.Errorf("CSV header is empty")
	}

	// Identify special columns
	expectedOutputCol := -1
	metadataCol := -1
	inputCols := make([]int, 0)

	for i, col := range header {
		switch strings.ToLower(strings.TrimSpace(col)) {
		case "expected_output", "expectedoutput", "output":
			expectedOutputCol = i
		case "metadata":
			metadataCol = i
		default:
			inputCols = append(inputCols, i)
		}
	}

	if len(inputCols) == 0 {
		return nil, fmt.Errorf("no input columns found in CSV")
	}

	items := make([]domain.DatasetItemInput, 0)

	// Read data rows
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row: %w", err)
		}

		if len(row) == 0 {
			continue
		}

		// Build input map
		input := make(map[string]interface{})
		for _, colIdx := range inputCols {
			if colIdx < len(row) {
				key := strings.TrimSpace(header[colIdx])
				value := strings.TrimSpace(row[colIdx])
				input[key] = value
			}
		}

		item := domain.DatasetItemInput{
			Input: input,
		}

		// Parse expected output if present
		if expectedOutputCol >= 0 && expectedOutputCol < len(row) {
			expectedOutput := strings.TrimSpace(row[expectedOutputCol])
			if expectedOutput != "" {
				// Try to parse as JSON, otherwise treat as string
				var outputMap map[string]interface{}
				if err := json.Unmarshal([]byte(expectedOutput), &outputMap); err == nil {
					item.ExpectedOutput = outputMap
				} else {
					item.ExpectedOutput = map[string]interface{}{
						"value": expectedOutput,
					}
				}
			}
		}

		// Parse metadata if present
		if metadataCol >= 0 && metadataCol < len(row) {
			metadata := strings.TrimSpace(row[metadataCol])
			if metadata != "" {
				var metadataMap map[string]interface{}
				if err := json.Unmarshal([]byte(metadata), &metadataMap); err == nil {
					item.Metadata = metadataMap
				}
			}
		}

		items = append(items, item)
	}

	return items, nil
}

// GetItemCount returns the total number of items in a dataset
func (s *DatasetService) GetItemCount(ctx context.Context, datasetID string) (int, error) {
	return s.datasetRepo.GetItemCount(ctx, datasetID)
}
