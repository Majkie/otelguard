package postgres

import (
	"context"
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/otelguard/otelguard/internal/domain"
)

// DatasetRepository handles dataset data access
type DatasetRepository struct {
	db *pgxpool.Pool
}

// NewDatasetRepository creates a new dataset repository
func NewDatasetRepository(db *pgxpool.Pool) *DatasetRepository {
	return &DatasetRepository{db: db}
}

// Create creates a new dataset
func (r *DatasetRepository) Create(ctx context.Context, dataset *domain.Dataset) error {
	query := `
		INSERT INTO datasets (id, project_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query,
		dataset.ID,
		dataset.ProjectID,
		dataset.Name,
		dataset.Description,
		dataset.CreatedAt,
		dataset.UpdatedAt,
	)
	return err
}

// GetByID retrieves a dataset by ID
func (r *DatasetRepository) GetByID(ctx context.Context, id string) (*domain.Dataset, error) {
	var dataset domain.Dataset
	query := `
		SELECT id, project_id, name, description, created_at, updated_at
		FROM datasets
		WHERE id = $1 AND deleted_at IS NULL
	`
	err := pgxscan.Get(ctx, r.db, &dataset, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &dataset, nil
}

// List returns datasets for a project with pagination
func (r *DatasetRepository) List(ctx context.Context, projectID string, opts *ListOptions) ([]*domain.Dataset, int, error) {
	datasets := make([]*domain.Dataset, 0)
	var total int

	// Use default values if opts is nil
	limit := 50
	offset := 0
	if opts != nil {
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		if opts.Offset > 0 {
			offset = opts.Offset
		}
	}

	// Ensure reasonable limits
	if limit > 100 {
		limit = 100
	}

	// Count query
	countQuery := `SELECT COUNT(*) FROM datasets WHERE project_id = $1 AND deleted_at IS NULL`
	if err := pgxscan.Get(ctx, r.db, &total, countQuery, projectID); err != nil {
		return nil, 0, err
	}

	// List query with pagination
	listQuery := `
		SELECT id, project_id, name, description, created_at, updated_at
		FROM datasets
		WHERE project_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	if err := pgxscan.Select(ctx, r.db, &datasets, listQuery, projectID, limit, offset); err != nil {
		return nil, 0, err
	}

	return datasets, total, nil
}

// Update updates a dataset
func (r *DatasetRepository) Update(ctx context.Context, dataset *domain.Dataset) error {
	query := `
		UPDATE datasets
		SET name = $2, description = $3, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`
	result, err := r.db.Exec(ctx, query,
		dataset.ID,
		dataset.Name,
		dataset.Description,
		dataset.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Delete soft-deletes a dataset
func (r *DatasetRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE datasets SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// CreateItem creates a new dataset item
func (r *DatasetRepository) CreateItem(ctx context.Context, item *domain.DatasetItem) error {
	query := `
		INSERT INTO dataset_items (id, dataset_id, input, expected_output, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query,
		item.ID,
		item.DatasetID,
		item.Input,
		item.ExpectedOutput,
		item.Metadata,
		item.CreatedAt,
	)
	return err
}

// GetItemByID retrieves a dataset item by ID
func (r *DatasetRepository) GetItemByID(ctx context.Context, id string) (*domain.DatasetItem, error) {
	var item domain.DatasetItem
	query := `
		SELECT id, dataset_id, input, expected_output, metadata, created_at
		FROM dataset_items
		WHERE id = $1
	`
	err := pgxscan.Get(ctx, r.db, &item, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

// ListItems returns items for a dataset with pagination
func (r *DatasetRepository) ListItems(ctx context.Context, datasetID string, opts *ListOptions) ([]*domain.DatasetItem, int, error) {
	items := make([]*domain.DatasetItem, 0)
	var total int

	// Use default values if opts is nil
	limit := 50
	offset := 0
	if opts != nil {
		if opts.Limit > 0 {
			limit = opts.Limit
		}
		if opts.Offset > 0 {
			offset = opts.Offset
		}
	}

	// Ensure reasonable limits
	if limit > 100 {
		limit = 100
	}

	// Count query
	countQuery := `SELECT COUNT(*) FROM dataset_items WHERE dataset_id = $1`
	if err := pgxscan.Get(ctx, r.db, &total, countQuery, datasetID); err != nil {
		return nil, 0, err
	}

	// List query with pagination
	listQuery := `
		SELECT id, dataset_id, input, expected_output, metadata, created_at
		FROM dataset_items
		WHERE dataset_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`
	if err := pgxscan.Select(ctx, r.db, &items, listQuery, datasetID, limit, offset); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// UpdateItem updates a dataset item
func (r *DatasetRepository) UpdateItem(ctx context.Context, item *domain.DatasetItem) error {
	query := `
		UPDATE dataset_items
		SET input = $2, expected_output = $3, metadata = $4
		WHERE id = $1
	`
	result, err := r.db.Exec(ctx, query,
		item.ID,
		item.Input,
		item.ExpectedOutput,
		item.Metadata,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// DeleteItem deletes a dataset item
func (r *DatasetRepository) DeleteItem(ctx context.Context, id string) error {
	query := `DELETE FROM dataset_items WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// BulkCreateItems creates multiple dataset items in a transaction
func (r *DatasetRepository) BulkCreateItems(ctx context.Context, items []*domain.DatasetItem) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO dataset_items (id, dataset_id, input, expected_output, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for _, item := range items {
		_, err := tx.Exec(ctx, query,
			item.ID,
			item.DatasetID,
			item.Input,
			item.ExpectedOutput,
			item.Metadata,
			item.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetItemCount returns the total number of items in a dataset
func (r *DatasetRepository) GetItemCount(ctx context.Context, datasetID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM dataset_items WHERE dataset_id = $1`
	err := pgxscan.Get(ctx, r.db, &count, query, datasetID)
	return count, err
}
