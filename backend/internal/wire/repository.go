package wire

import (
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
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
	// ClickHouse repositories
	ProvideTraceRepository,
	ProvideGuardrailEventRepository,
)

// PostgreSQL Repositories

// ProvideUserRepository creates a new UserRepository.
func ProvideUserRepository(db *sqlx.DB) *pgrepo.UserRepository {
	return pgrepo.NewUserRepository(db)
}

// ProvideOrganizationRepository creates a new OrganizationRepository.
func ProvideOrganizationRepository(db *sqlx.DB) *pgrepo.OrganizationRepository {
	return pgrepo.NewOrganizationRepository(db)
}

// ProvideProjectRepository creates a new ProjectRepository.
func ProvideProjectRepository(db *sqlx.DB) *pgrepo.ProjectRepository {
	return pgrepo.NewProjectRepository(db)
}

// ProvidePromptRepository creates a new PromptRepository.
func ProvidePromptRepository(db *sqlx.DB) *pgrepo.PromptRepository {
	return pgrepo.NewPromptRepository(db)
}

// ProvideGuardrailRepository creates a new GuardrailRepository.
func ProvideGuardrailRepository(db *sqlx.DB) *pgrepo.GuardrailRepository {
	return pgrepo.NewGuardrailRepository(db)
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
