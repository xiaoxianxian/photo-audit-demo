package service

import (
	"fmt"
	"os"

	"audit-platform/internal/config"
	"audit-platform/internal/queue"
	"audit-platform/internal/repository"
	"audit-platform/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Services holds all repositories and service instances, providing a single
// wiring point for dependency injection in main.go.
type Services struct {
	UserRepo       *repository.UserRepository
	TenantRepo     *repository.TenantRepository
	TeamRepo       *repository.TeamRepository

	// Content/element repositories.
	ContentRepo    *repository.ContentRepository
	ElementRepo    *repository.ElementRepository
	AuditLogRepo   *repository.LogRepository

	// Tenant config repositories.
	RuleRepo  *repository.TenantRuleRepository
	LevelRepo *repository.TenantLevelRepository
	WordRepo  *repository.TenantWordRepository
	AIConfigRepo *repository.AIConfigRepository

	AuthService     *AuthService
	TenantService   *TenantService
	TeamService     *TeamService

	// Content, review, appeal, dashboard, quality, live wall services.
	IngestionService    *IngestionService
	AIService           *AIService
	ReviewService       *ReviewService
	AppealService       *AppealService
	DashboardService    *DashboardService
	QualityAuditService *QualityAuditService
	LiveWallService     *LiveWallService
	WsHub               *Hub
	StreamScheduler     *StreamScheduler
	KafkaProducer       *queue.Producer // nil when KAFKA_BROKERS unset (queue disabled)

	// Tenant config services.
	RuleService  *TenantRuleService
	LevelService *TenantLevelService
	WordService  *TenantWordService
	AIConfigService *AIConfigService

	// Storage.
	MinIO *storage.MinIOStorage
}

// NewServices initializes all repositories and services from a pgxpool,
// a JWT signing secret, and a minio config.
func NewServices(pool *pgxpool.Pool, jwtSecret string, cfg *config.Config) *Services {
	// Existing repositories.
	userRepo := repository.NewUserRepository(pool)
	tenantRepo := repository.NewTenantRepository(pool)
	teamRepo := repository.NewTeamRepository(pool)

	// Content/element repositories.
	contentRepo  := repository.NewContentRepository(pool)
	elementRepo  := repository.NewElementRepository(pool)
	auditLogRepo := repository.NewLogRepository(pool)
	appealRepo   := repository.NewAppealRepository(pool)
	qualityRepo := repository.NewQualityAuditRepository(pool)
	liveWallRepo := repository.NewLiveWallRepository(pool)

	// Tenant config repositories.
	ruleRepo     := repository.NewTenantRuleRepository(pool)
	levelRepo    := repository.NewTenantLevelRepository(pool)
	wordRepo     := repository.NewTenantWordRepository(pool)
	aiConfigRepo := repository.NewAIConfigRepository(pool)

	// Existing services.
	authSvc := NewAuthService(userRepo, jwtSecret)
	tenantSvc := NewTenantService(tenantRepo, userRepo)
	teamSvc := NewTeamService(teamRepo, userRepo)

	// AI Service.
	agnesKey := os.Getenv("AGNES_API_KEY")
	deepseekKey := os.Getenv("DEEPSEEK_API_KEY")
	aiSvc := NewAIService(agnesKey, deepseekKey)
	// Attach local rule-based fallback for quota exhaustion / API key missing.
	// Enabled by default; disable via FALLBACK_ENABLED=false env var.
	if cfg.FallbackEnabled {
		aiSvc.WithFallback(NewLocalFallbackService())
	}

	// Notifier must be created BEFORE services that depend on it.
	notifier := NewMultiNotifier(&ConsoleNotifier{})

	// WebSocket hub must be created before services that reference it.
	wsHub := NewHub()

	// Content-related services (wire AI service into ingestion first).
	ingestionSvc := NewIngestionService(contentRepo, elementRepo, aiSvc, wsHub)

	// Phase 2: Kafka queue — producer attaches to ingestion; the consumer runs
	// in main.go after Services returns. nil producer = in-process goroutine path.
	var kafkaProducer *queue.Producer
	if cfg != nil && len(cfg.KafkaBrokers) > 0 {
		kafkaProducer = queue.NewProducer(cfg.KafkaBrokers)
		ingestionSvc.WithProducer(kafkaProducer)
	}

	// Video processor for short_video preprocessing.
	videoProc := NewVideoProcessor(elementRepo)
	ingestionSvc.WithVideoProcessor(videoProc)
	// Attach fingerprint service for video deduplication.
	videoProc.SetFingerprintService(NewFingerprintService())

	// Wire AI config repo into AIService for dynamic key retrieval.
	aiSvc.WithAIConfigRepo(aiConfigRepo)

	reviewSvc := NewReviewService(elementRepo, appealRepo, auditLogRepo, ingestionSvc, notifier, wsHub, contentRepo)
	appealSvc := NewAppealService(appealRepo, contentRepo, notifier)
	dashSvc := NewDashboardService(auditLogRepo, elementRepo)
	qualitySvc := NewQualityAuditService(qualityRepo, elementRepo, auditLogRepo)
	liveWallSvc := NewLiveWallService(liveWallRepo)

	// Stream scheduler — manages periodic frame snapshots for active live streams.
	streamScheduler := NewStreamScheduler(liveWallSvc, wsHub)

	// Tenant config services.
	ruleSvc      := NewTenantRuleService(ruleRepo)
	levelSvc     := NewTenantLevelService(levelRepo)
	wordSvc      := NewTenantWordService(wordRepo)
	aiConfigSvc  := NewAIConfigService(aiConfigRepo)

	// MinIO storage (optional — nil if not configured).
	var minioStorage *storage.MinIOStorage
	if cfg != nil && cfg.MinIOEndpoint != "" && cfg.MinIOAccessKey != "" && cfg.MinIOSecretKey != "" {
		s, err := storage.NewMinIOStorage(
			cfg.MinIOEndpoint,
			cfg.MinIOAccessKey,
			cfg.MinIOSecretKey,
			cfg.MinIOBucket,
			"", // region
			false, // useSSL
		)
		if err != nil {
			fmt.Println("[WARN] minio storage init failed:", err)
		} else {
			minioStorage = s
		}
	}

	return &Services{
		UserRepo:            userRepo,
		TenantRepo:          tenantRepo,
		TeamRepo:            teamRepo,
		ContentRepo:         contentRepo,
		ElementRepo:         elementRepo,
		AuditLogRepo:        auditLogRepo,
		RuleRepo:            ruleRepo,
		LevelRepo:           levelRepo,
		WordRepo:            wordRepo,
		AIConfigRepo:        aiConfigRepo,
		AuthService:         authSvc,
		TenantService:       tenantSvc,
		TeamService:         teamSvc,
		IngestionService:    ingestionSvc,
		AIService:           aiSvc,
		ReviewService:       reviewSvc,
		AppealService:       appealSvc,
		DashboardService:    dashSvc,
		QualityAuditService: qualitySvc,
		LiveWallService:     liveWallSvc,
		WsHub:               wsHub,
		StreamScheduler:     streamScheduler,
		KafkaProducer:       kafkaProducer,
		RuleService:         ruleSvc,
		LevelService:        levelSvc,
		WordService:         wordSvc,
		AIConfigService:     aiConfigSvc,
		MinIO:               minioStorage,
	}
}
