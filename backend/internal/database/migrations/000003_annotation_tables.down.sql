-- ============================================
-- Annotation Tables Rollback Migration
-- ============================================

-- Drop triggers
DROP TRIGGER IF EXISTS update_annotations_updated_at ON annotations;
DROP TRIGGER IF EXISTS update_annotation_assignments_updated_at ON annotation_assignments;
DROP TRIGGER IF EXISTS update_annotation_queue_items_updated_at ON annotation_queue_items;

-- Drop tables
DROP TABLE IF EXISTS inter_annotator_agreements;
DROP TABLE IF EXISTS annotations;
DROP TABLE IF EXISTS annotation_assignments;
DROP TABLE IF EXISTS annotation_queue_items;

-- Remove added columns from annotation_queues
ALTER TABLE annotation_queues DROP COLUMN IF EXISTS config;
ALTER TABLE annotation_queues DROP COLUMN IF EXISTS item_source;
ALTER TABLE annotation_queues DROP COLUMN IF EXISTS item_source_config;
ALTER TABLE annotation_queues DROP COLUMN IF EXISTS assignment_strategy;
ALTER TABLE annotation_queues DROP COLUMN IF EXISTS max_annotations_per_item;
ALTER TABLE annotation_queues DROP COLUMN IF EXISTS instructions;
ALTER TABLE annotation_queues DROP COLUMN IF EXISTS is_active;
