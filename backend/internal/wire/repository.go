package wire

import (
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	chrepo "github.com/otelguard/otelguard/internal/repository/clickhouse"
	pgrepo "github.com/otelguard/otelguard/internal/repository/postgres"
)

// RepositorySet provides all repository instances.
var RepositorySet = wire.NewSet(
	// PostgreSQL repositories
	ProvideUserRepository,
	ProvideOrganizationRepository,
	ProvideProjectRepository,
	ProvidePromptRepository,
	ProvideGuardrailRepository,
	ProvideAnnotationRepository,
	ProvideFeedbackRepository,
	ProvideFeedbackScoreMappingRepository,
	ProvideEvaluatorRepository,
	ProvideEvaluationJobRepository,
	ProvideDatasetRepository,
	ProvideExperimentRepository,
	ProvideDashboardRepository,
	// ClickHouse repositories
	ProvideTraceRepository,
	ProvideGuardrailEventRepository,
	ProvideAgentRepository,
	ProvideEvaluationResultRepository,
)

// PostgreSQL Repositories

// ProvideUserRepository creates a new UserRepository.
func ProvideUserRepository(db *pgxpool.Pool) *pgrepo.UserRepository {
	return pgrepo.NewUserRepository(db)
}

// ProvideOrganizationRepository creates a new OrganizationRepository.
func ProvideOrganizationRepository(db *pgxpool.Pool) *pgrepo.OrganizationRepository {
	return pgrepo.NewOrganizationRepository(db)
}

// ProvideProjectRepository creates a new ProjectRepository.
func ProvideProjectRepository(db *pgxpool.Pool) *pgrepo.ProjectRepository {
	return pgrepo.NewProjectRepository(db)
}

// ProvidePromptRepository creates a new PromptRepository.
func ProvidePromptRepository(db *pgxpool.Pool) *pgrepo.PromptRepository {
	return pgrepo.NewPromptRepository(db)
}

// ProvideGuardrailRepository creates a new GuardrailRepository.
func ProvideGuardrailRepository(db *pgxpool.Pool) *pgrepo.GuardrailRepository {
	return pgrepo.NewGuardrailRepository(db)
}

// ProvideAnnotationRepository creates a new AnnotationRepository.
func ProvideAnnotationRepository(db *pgxpool.Pool) *pgrepo.AnnotationRepository {
	return pgrepo.NewAnnotationRepository(db)
}

// ProvideFeedbackRepository creates a new FeedbackRepository.
func ProvideFeedbackRepository(db *pgxpool.Pool) *pgrepo.FeedbackRepository {
	return pgrepo.NewFeedbackRepository(db)
}

// ProvideFeedbackScoreMappingRepository creates a new FeedbackScoreMappingRepository.
func ProvideFeedbackScoreMappingRepository(db *pgxpool.Pool) *pgrepo.FeedbackScoreMappingRepository {
	return pgrepo.NewFeedbackScoreMappingRepository(db)
}

// ProvideEvaluatorRepository creates a new EvaluatorRepository.
func ProvideEvaluatorRepository(db *pgxpool.Pool) *pgrepo.EvaluatorRepository {
	return pgrepo.NewEvaluatorRepository(db)
}

// ProvideEvaluationJobRepository creates a new EvaluationJobRepository.
func ProvideEvaluationJobRepository(db *pgxpool.Pool) *pgrepo.EvaluationJobRepository {
	return pgrepo.NewEvaluationJobRepository(db)
}

// ProvideDatasetRepository creates a new DatasetRepository.
func ProvideDatasetRepository(db *pgxpool.Pool) *pgrepo.DatasetRepository {
	return pgrepo.NewDatasetRepository(db)
}

// ProvideExperimentRepository creates a new ExperimentRepository.
func ProvideExperimentRepository(db *pgxpool.Pool) *pgrepo.ExperimentRepository {
	return pgrepo.NewExperimentRepository(db)
}

// ProvideDashboardRepository creates a new DashboardRepository.
func ProvideDashboardRepository(db *pgxpool.Pool) *pgrepo.DashboardRepository {
	return pgrepo.NewDashboardRepository(db)
}

// ClickHouse Repositories

// ProvideTraceRepository creates a new TraceRepository.
func ProvideTraceRepository(conn clickhouse.Conn) *chrepo.TraceRepository {
	return chrepo.NewTraceRepository(conn)
}

// ProvideGuardrailEventRepository creates a new GuardrailEventRepository.
func ProvideGuardrailEventRepository(conn clickhouse.Conn) *chrepo.GuardrailEventRepository {
	return chrepo.NewGuardrailEventRepository(conn)
}

// ProvideAgentRepository creates a new AgentRepository.
func ProvideAgentRepository(conn clickhouse.Conn) *chrepo.AgentRepository {
	return chrepo.NewAgentRepository(conn)
}

// ProvideEvaluationResultRepository creates a new EvaluationResultRepository.
func ProvideEvaluationResultRepository(conn clickhouse.Conn) *chrepo.EvaluationResultRepository {
	return chrepo.NewEvaluationResultRepository(conn)
}
