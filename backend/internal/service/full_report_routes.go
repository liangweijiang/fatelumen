package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"fatelumen/backend/internal/llm"
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/logger"
)

type FullReportModelRoute struct {
	RouteNo          uint8   `json:"route_no"`
	ProviderConfigID uint64  `json:"provider_config_id"`
	ModelConfigID    uint64  `json:"model_config_id"`
	ProviderCode     string  `json:"provider_code"`
	ProviderName     string  `json:"provider_name"`
	BaseURL          string  `json:"base_url"`
	ModelName        string  `json:"model_name"`
	Model            string  `json:"model"`
	Priority         int     `json:"priority"`
	MaxRetries       int     `json:"max_retries"`
	MaxAttempts      int     `json:"max_attempts"`
	TimeoutSeconds   int     `json:"timeout_seconds"`
	Temperature      float64 `json:"temperature"`
}

type ResolvedFullReportRoute struct {
	Frozen   FullReportModelRoute
	Provider llm.LLMProvider
}

type FullReportRouteResolver interface {
	Resolve(ctx context.Context) ([]ResolvedFullReportRoute, error)
	ResolveFrozen(ctx context.Context, frozen []FullReportModelRoute) ([]ResolvedFullReportRoute, error)
}

type fullReportRouteStore interface {
	ListEnabledRoutes(ctx context.Context) ([]model.LLMModelConfig, error)
	ListRoutesByIDs(ctx context.Context, ids []uint64) ([]model.LLMModelConfig, error)
}

type databaseFullReportRouteResolver struct {
	store          fullReportRouteStore
	secretCipher   *llm.ConfigSecretCipher
	defaultTimeout time.Duration
}

func NewDatabaseFullReportRouteResolver(store fullReportRouteStore, secretCipher *llm.ConfigSecretCipher, defaultTimeout time.Duration) FullReportRouteResolver {
	if defaultTimeout <= 0 {
		defaultTimeout = 180 * time.Second
	}
	return &databaseFullReportRouteResolver{store: store, secretCipher: secretCipher, defaultTimeout: defaultTimeout}
}

func (r *databaseFullReportRouteResolver) Resolve(ctx context.Context) ([]ResolvedFullReportRoute, error) {
	if r == nil || r.store == nil || r.secretCipher == nil {
		return nil, errors.New("full report route resolver is not configured")
	}
	rows, err := r.store.ListEnabledRoutes(ctx)
	if err != nil {
		logger.FromCtx(ctx).Error("load full report model routes failed", "err", err)
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("no enabled full report model route")
	}
	if len(rows) > 255 {
		return nil, errors.New("full report model route count exceeds 255")
	}
	routes := make([]ResolvedFullReportRoute, 0, len(rows))
	for i, row := range rows {
		if !row.Enabled || !row.Provider.Enabled || strings.TrimSpace(row.ModelID) == "" || strings.TrimSpace(row.Provider.BaseURL) == "" {
			return nil, fmt.Errorf("enabled model route %d is incomplete", row.ID)
		}
		apiKey, err := r.secretCipher.Decrypt(row.Provider.APIKeyCiphertext)
		if err != nil {
			logger.FromCtx(ctx).Error("decrypt full report provider credential failed", "err", err, "provider_id", row.ProviderID, "model_config_id", row.ID)
			return nil, fmt.Errorf("decrypt provider %d credential: %w", row.ProviderID, err)
		}
		maxRetries := row.MaxRetries
		if maxRetries < 0 {
			maxRetries = 0
		}
		frozen := FullReportModelRoute{
			RouteNo: uint8(i + 1), ProviderConfigID: row.ProviderID, ModelConfigID: row.ID,
			ProviderCode: row.Provider.Code, ProviderName: row.Provider.Name, BaseURL: row.Provider.BaseURL,
			ModelName: row.Name, Model: row.ModelID, Priority: row.Priority,
			MaxRetries: maxRetries, MaxAttempts: maxRetries + 1,
			TimeoutSeconds: int(r.defaultTimeout.Seconds()), Temperature: 0.5,
		}
		routes = append(routes, ResolvedFullReportRoute{
			Frozen:   frozen,
			Provider: llm.NewOpenAICompatibleProvider(row.Provider.Code, apiKey, row.Provider.BaseURL, row.ModelID),
		})
	}
	return routes, nil
}

func (r *databaseFullReportRouteResolver) ResolveFrozen(ctx context.Context, frozen []FullReportModelRoute) ([]ResolvedFullReportRoute, error) {
	if r == nil || r.store == nil || r.secretCipher == nil || len(frozen) == 0 {
		return nil, errors.New("frozen full report route resolver is not configured")
	}
	ids := make([]uint64, len(frozen))
	for i := range frozen {
		ids[i] = frozen[i].ModelConfigID
	}
	rows, err := r.store.ListRoutesByIDs(ctx, ids)
	if err != nil {
		logger.FromCtx(ctx).Error("load frozen full report credentials failed", "err", err)
		return nil, err
	}
	byID := make(map[uint64]model.LLMModelConfig, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	resolved := make([]ResolvedFullReportRoute, 0, len(frozen))
	for _, route := range frozen {
		row, ok := byID[route.ModelConfigID]
		if !ok || row.ProviderID != route.ProviderConfigID {
			return nil, fmt.Errorf("frozen model route %d is no longer available", route.ModelConfigID)
		}
		apiKey, decryptErr := r.secretCipher.Decrypt(row.Provider.APIKeyCiphertext)
		if decryptErr != nil {
			logger.FromCtx(ctx).Error("decrypt frozen full report credential failed", "err", decryptErr, "provider_id", route.ProviderConfigID, "model_config_id", route.ModelConfigID)
			return nil, fmt.Errorf("decrypt frozen provider %d credential: %w", route.ProviderConfigID, decryptErr)
		}
		resolved = append(resolved, ResolvedFullReportRoute{Frozen: route, Provider: llm.NewOpenAICompatibleProvider(route.ProviderCode, apiKey, route.BaseURL, route.Model)})
	}
	return resolved, nil
}

func frozenRoutes(routes []ResolvedFullReportRoute) []FullReportModelRoute {
	result := make([]FullReportModelRoute, len(routes))
	for i := range routes {
		result[i] = routes[i].Frozen
	}
	return result
}

func totalRouteAttempts(routes []FullReportModelRoute) int {
	total := 0
	for _, route := range routes {
		if route.MaxAttempts > 0 {
			total += route.MaxAttempts
		}
	}
	return total
}

// routeForConsumedAttempts maps the immutable chapter attempt count onto the
// frozen provider chain. It makes aggregate retries resume at the correct
// route instead of restarting the first provider.
func routeForConsumedAttempts(routes []ResolvedFullReportRoute, consumed int) (ResolvedFullReportRoute, int, bool) {
	for _, route := range routes {
		if consumed < route.Frozen.MaxAttempts {
			return route, consumed + 1, true
		}
		consumed -= route.Frozen.MaxAttempts
	}
	return ResolvedFullReportRoute{}, 0, false
}
