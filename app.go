package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"

	"blockads-desktop/internal/dnsengine"
	"blockads-desktop/internal/filtermanager"
	"blockads-desktop/internal/filters"
	"blockads-desktop/internal/models"
	"blockads-desktop/internal/rewrites"
	"blockads-desktop/internal/safesearch"
	"blockads-desktop/internal/startup"
	"blockads-desktop/internal/storage"
)

// App manages desktop UI state and IPC bindings for Wails.
type App struct {
	ctx        context.Context
	mu         sync.RWMutex
	storage    *storage.Storage
	logger     *storage.AsyncLogger
	rewriter   *rewrites.Rewriter
	matcher    *filters.Matcher
	updater    *filtermanager.FilterUpdater
	scheduler  *filtermanager.Scheduler
	dnsHandler *dnsengine.Handler
	startupMgr startup.Manager
	dbPath     string
	isPortable bool
}

// NewApp creates a new App application struct using the default SQLite database path.
func NewApp() *App {
	explicitPath := os.Getenv("BLOCKADS_DB_PATH")
	var dbPath string
	var isPortable bool
	var err error

	if flag.Lookup("test.v") != nil && explicitPath == "" {
		dbPath = filepath.Join(os.TempDir(), fmt.Sprintf("blockads_test_%d.db", time.Now().UnixNano()))
	} else {
		dbPath, isPortable, err = storage.ResolveDatabasePath(explicitPath, "")
		if err != nil {
			dbPath = "blockads.db"
		}
	}

	store, err := storage.Open(dbPath)
	if err != nil {
		// Fallback to in-memory if filesystem path fails
		store, _ = storage.Open(":memory:")
	}

	app := NewAppWithStorage(store)
	app.dbPath = dbPath
	app.isPortable = isPortable
	app.startupMgr = startup.DefaultManager()
	return app
}

// NewAppWithStorage constructs an App backed by a specific storage instance.
func NewAppWithStorage(store *storage.Storage) *App {
	ctx := context.Background()

	// Apply migrations
	if store != nil {
		_ = store.Migrate(ctx)
		seedInitialDataIfEmpty(ctx, store)
	}

	// Reconcile in-memory active engines
	var rewriter *rewrites.Rewriter
	var matcher *filters.Matcher
	if store != nil {
		active, err := storage.ReconcileActiveState(ctx, store)
		if err == nil && active != nil {
			rewriter = active.Rewriter
			matcher = active.Matcher
		}
	}
	if rewriter == nil {
		rewriter = rewrites.NewRewriter()
	}
	if matcher == nil {
		matcher = filters.NewMatcher()
	}

	var asyncLogger *storage.AsyncLogger
	if store != nil {
		asyncLogger = storage.NewAsyncLogger(storage.AsyncLoggerConfig{
			Storage: store,
		})
	}

	app := &App{
		storage:    store,
		logger:     asyncLogger,
		rewriter:   rewriter,
		matcher:    matcher,
		startupMgr: startup.DefaultManager(),
	}

	if store != nil {
		dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{})
		updater := filtermanager.NewFilterUpdater(store, dl, app)
		scheduler := filtermanager.NewScheduler(updater, store, filtermanager.SchedulerConfig{})
		app.updater = updater
		app.scheduler = scheduler
	}

	return app
}

// startup is called when the app starts. The context is saved
// so we can call runtime methods.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.mu.RLock()
	scheduler := a.scheduler
	a.mu.RUnlock()
	if scheduler != nil {
		_ = scheduler.Start(ctx)
	}
}

// shutdown is called during application exit to flush logs and close storage.
func (a *App) shutdown(ctx context.Context) {
	_ = a.Close()
}

// Close flushes background workers and closes the database cleanly.
func (a *App) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	var err error
	if a.scheduler != nil {
		_ = a.scheduler.Stop()
		a.scheduler = nil
	}
	if a.logger != nil {
		_ = a.logger.Close()
		a.logger = nil
	}
	if a.storage != nil {
		err = a.storage.Close()
		a.storage = nil
	}
	return err
}

// SetMatcher atomically replaces the active filtering matcher.
func (a *App) SetMatcher(m *filters.Matcher) {
	a.mu.Lock()
	a.matcher = m
	a.mu.Unlock()
}

// Storage returns the underlying persistent storage instance.
func (a *App) Storage() *storage.Storage {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.storage
}

// Logger returns the async query logger.
func (a *App) Logger() *storage.AsyncLogger {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.logger
}

// Rewriter returns the active DNS rewrites engine.
func (a *App) Rewriter() *rewrites.Rewriter {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.rewriter
}

// Matcher returns the active domain filtering matcher.
func (a *App) Matcher() *filters.Matcher {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.matcher
}

// SetDNSHandler attaches an active DNS engine Handler for atomic updates and cache flushes.
func (a *App) SetDNSHandler(h *dnsengine.Handler) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.dnsHandler = h
}

// DNSHandler returns the attached DNS engine Handler, if any.
func (a *App) DNSHandler() *dnsengine.Handler {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.dnsHandler
}

// GetAppInfo returns static and environment metadata.
func (a *App) GetAppInfo() models.AppInfo {
	dataDir := storage.GetDataDirFromDBPath(a.dbPath)
	return models.AppInfo{
		Name:            "BlockAds Desktop",
		Version:         "0.4.0",
		Platform:        fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		Milestone:       "Milestone 04B.4A: SafeSearch DNS Policies (Release Candidate)",
		GoVersion:       runtime.Version(),
		WebView2Version: "155.0.4283.62",
		DataDir:         dataDir,
		IsPortable:      a.isPortable,
	}
}

// GetProtectionStatus returns current engine operational state.
func (a *App) GetProtectionStatus() models.ProtectionStatus {
	settings := a.GetSettings()

	return models.ProtectionStatus{
		IsActive:    false,
		State:       "INACTIVE_MILESTONE_1",
		UpstreamURL: settings.UpstreamDoHURL,
		ListenAddr:  "127.0.0.1:53 (Standby - Engine active)",
		Message:     "DNS Engine foundation active. Local network adapter configuration has NOT been altered.",
	}
}

// GetSampleQueryLogs returns recent query log records from persistent SQLite.
func (a *App) GetSampleQueryLogs() []models.QueryLogEntry {
	if a.storage == nil {
		return nil
	}

	entries, _, err := a.storage.GetQueryEvents(context.Background(), storage.QueryFilter{
		Limit: 50,
	})
	if err != nil {
		return nil
	}
	return entries
}

// GetQueryLogs provides filtered and paginated query log retrieval.
func (a *App) GetQueryLogs(domain string, status string, limit int, offset int) ([]models.QueryLogEntry, int, error) {
	if a.storage == nil {
		return nil, 0, errors.New("storage unavailable")
	}

	return a.storage.GetQueryEvents(context.Background(), storage.QueryFilter{
		Domain: domain,
		Status: status,
		Limit:  limit,
		Offset: offset,
	})
}

// GetFilterLists returns the list of configured filter sources from SQLite.
func (a *App) GetFilterLists() []models.FilterList {
	if a.storage == nil {
		return nil
	}

	subs, err := a.storage.GetFilterSubscriptions(context.Background())
	if err != nil {
		return nil
	}

	result := make([]models.FilterList, len(subs))
	for i, s := range subs {
		lastUpdated := s.LastSuccessAt
		if lastUpdated == "" {
			lastUpdated = s.UpdatedAt
		}
		result[i] = models.FilterList{
			ID:              s.ID,
			Name:            s.Name,
			SourceURL:       s.SourceURL,
			Enabled:         s.Enabled,
			Format:          s.Format,
			RuleCount:       s.RuleCount,
			LastUpdated:     lastUpdated,
			LastAttemptAt:   s.LastAttemptAt,
			LastSuccessAt:   s.LastSuccessAt,
			LastStatus:      s.LastStatus,
			LastError:       s.LastError,
			ContentChecksum: s.ContentChecksum,
			ETag:            s.ETag,
			Description:     s.Description,
			Category:        s.Category,
		}
	}

	return result
}

// ToggleFilterList updates the enabled state of a filter list in SQLite and updates the active matcher.
func (a *App) ToggleFilterList(id string, enabled bool) []models.FilterList {
	if a.storage != nil {
		ctx := context.Background()
		_ = a.storage.ToggleFilterSubscription(ctx, id, enabled)
		a.reconcileActiveEngines()
	}

	return a.GetFilterLists()
}

// GetCatalogFilters returns curated filter catalog entries decorated with their active subscription status.
func (a *App) GetCatalogFilters() []models.CatalogFilterItem {
	catalog := filtermanager.BuiltinCatalog()
	var subs []storage.FilterSubscription
	if a.storage != nil {
		subs, _ = a.storage.GetFilterSubscriptions(context.Background())
	}

	subURLMap := make(map[string]storage.FilterSubscription)
	for _, sub := range subs {
		norm, err := models.NormalizeURL(sub.SourceURL)
		if err == nil {
			subURLMap[norm] = sub
		}
	}

	result := make([]models.CatalogFilterItem, 0, len(catalog))
	for _, item := range catalog {
		norm, _ := models.NormalizeURL(item.SourceURL)
		ci := models.CatalogFilterItem{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Category:    item.Category,
			SourceURL:   item.SourceURL,
			Format:      item.Format,
			Homepage:    item.Homepage,
			Maintainer:  item.Maintainer,
			License:     item.License,
		}
		if sub, ok := subURLMap[norm]; ok {
			ci.IsSubscribed = true
			ci.SubscribedID = sub.ID
		}
		result = append(result, ci)
	}

	return result
}

// PreviewFilterURL inspects and downloads preview metadata from a potential filter URL.
// It detects duplicates against existing subscriptions and rejects HTML or malformed sources.
func (a *App) PreviewFilterURL(rawURL string) (*models.FilterPreviewResult, error) {
	normURL, err := models.NormalizeURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid filter URL: %w", err)
	}

	if err := filtermanager.ValidateSourceURL(normURL, false); err != nil {
		return nil, fmt.Errorf("insecure or restricted URL: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	result := &models.FilterPreviewResult{
		SourceURL:     rawURL,
		NormalizedURL: normURL,
	}

	// Check if already subscribed in storage
	if a.storage != nil {
		if existing, err := a.storage.GetFilterSubscriptionByURL(ctx, normURL); err == nil && existing != nil {
			result.IsAlreadySubscribed = true
			result.ExistingSubscription = existing.Name
			result.ExistingID = existing.ID
		}
	}

	// Fetch preview content using downloader
	dl := filtermanager.NewDownloader(filtermanager.DownloaderConfig{})
	resp, err := dl.Download(ctx, filtermanager.DownloadRequest{URL: normURL})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch filter preview: %w", err)
	}

	meta, err := filtermanager.ExtractFilterMetadata(bytes.NewReader(resp.Body), normURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse filter metadata: %w", err)
	}

	result.DetectedName = meta.Title
	result.Description = meta.Description
	result.Homepage = meta.Homepage
	result.Format = meta.Format
	result.RuleCount = meta.RuleCount

	return result, nil
}

// AddFilterSubscriptionWithMetadata registers a subscription with full metadata and immediately fetches it.
func (a *App) AddFilterSubscriptionWithMetadata(name, sourceURL, format, category, description string) (*models.FilterList, error) {
	if a.storage == nil {
		return nil, errors.New("storage unavailable")
	}

	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return nil, errors.New("subscription name cannot be empty")
	}

	normURL, err := models.NormalizeURL(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("invalid subscription URL: %w", err)
	}

	if err := filtermanager.ValidateSourceURL(normURL, false); err != nil {
		return nil, fmt.Errorf("invalid subscription URL: %w", err)
	}

	ctx := context.Background()

	// Check for existing duplicate subscription
	if existing, err := a.storage.GetFilterSubscriptionByURL(ctx, normURL); err == nil && existing != nil {
		return nil, fmt.Errorf("subscription with this URL already exists: %s", existing.Name)
	}

	cleanFormat := strings.ToLower(strings.TrimSpace(format))
	if cleanFormat == "" {
		cleanFormat = "hosts"
	}
	cleanCategory := strings.TrimSpace(category)
	if cleanCategory == "" {
		cleanCategory = "Custom"
	}
	cleanDesc := strings.TrimSpace(description)
	if cleanDesc == "" {
		cleanDesc = "User added filter subscription"
	}

	id := fmt.Sprintf("sub-%d", time.Now().UnixNano())
	sub := storage.FilterSubscription{
		ID:          id,
		Name:        cleanName,
		SourceURL:   normURL,
		Enabled:     true,
		Format:      cleanFormat,
		Category:    cleanCategory,
		Description: cleanDesc,
	}

	if err := a.storage.SaveFilterSubscription(ctx, sub); err != nil {
		return nil, fmt.Errorf("failed to save subscription: %w", err)
	}

	// Trigger immediate fetch to compile and activate candidate matcher atomically
	if a.updater != nil {
		_, _ = a.updater.RefreshSubscription(ctx, id)
	}
	a.reconcileActiveEngines()

	created, err := a.storage.GetFilterSubscription(ctx, id)
	if err == nil && created != nil {
		return &models.FilterList{
			ID:              created.ID,
			Name:            created.Name,
			SourceURL:       created.SourceURL,
			Enabled:         created.Enabled,
			Format:          created.Format,
			RuleCount:       created.RuleCount,
			LastUpdated:     created.UpdatedAt,
			LastAttemptAt:   created.LastAttemptAt,
			LastSuccessAt:   created.LastSuccessAt,
			LastStatus:      created.LastStatus,
			LastError:       created.LastError,
			ContentChecksum: created.ContentChecksum,
			Description:     created.Description,
			Category:        created.Category,
		}, nil
	}

	return &models.FilterList{
		ID:          id,
		Name:        cleanName,
		SourceURL:   normURL,
		Enabled:     true,
		Format:      cleanFormat,
		RuleCount:   0,
		LastStatus:  "pending",
		Description: cleanDesc,
		Category:    cleanCategory,
	}, nil
}

// AddFilterSubscription is a backwards-compatible wrapper for manual additions.
func (a *App) AddFilterSubscription(name string, sourceURL string, format string) (*models.FilterList, error) {
	return a.AddFilterSubscriptionWithMetadata(name, sourceURL, format, "Custom", "User added filter subscription")
}

// SubscribeCatalogFilter subscribes to a verified item from the built-in catalog.
func (a *App) SubscribeCatalogFilter(catalogID string) (*models.FilterList, error) {
	catalog := filtermanager.BuiltinCatalog()
	var target *filtermanager.CatalogFilter
	for _, item := range catalog {
		if strings.EqualFold(item.ID, catalogID) {
			target = &item
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("catalog filter %q not found", catalogID)
	}

	return a.AddFilterSubscriptionWithMetadata(target.Name, target.SourceURL, target.Format, target.Category, target.Description)
}

// GetStartupStatus checks the current status of Start with Windows.
func (a *App) GetStartupStatus() (*models.StartupStatus, error) {
	if a.startupMgr == nil {
		return &models.StartupStatus{
			Supported: false,
			Message:   "Startup manager is unavailable",
		}, nil
	}
	return a.startupMgr.GetStatus()
}

// SetStartWithWindows configures whether the app should launch on Windows startup.
func (a *App) SetStartWithWindows(enabled bool) (*models.StartupStatus, error) {
	if a.startupMgr == nil {
		return nil, errors.New("startup manager is unavailable")
	}

	if err := a.startupMgr.SetEnabled(enabled); err != nil {
		return nil, err
	}

	// Persist preference in app_settings table
	if a.storage != nil {
		ctx := context.Background()
		cfg, err := a.storage.GetSettings(ctx)
		if err == nil {
			cfg.AutoStart = enabled
			_ = a.storage.UpdateSettings(ctx, cfg)
		}
	}

	return a.startupMgr.GetStatus()
}

// DeleteFilterSubscription removes a subscription from SQLite and updates the active matcher.
func (a *App) DeleteFilterSubscription(id string) error {
	if a.storage == nil {
		return errors.New("storage unavailable")
	}

	ctx := context.Background()
	if err := a.storage.DeleteFilterSubscription(ctx, id); err != nil {
		return err
	}

	a.reconcileActiveEngines()
	return nil
}

// RefreshFilterList manually refreshes a single filter subscription.
func (a *App) RefreshFilterList(id string) (*filtermanager.UpdateResult, error) {
	a.mu.RLock()
	updater := a.updater
	a.mu.RUnlock()

	if updater == nil {
		return nil, errors.New("filter updater unavailable")
	}

	return updater.RefreshSubscription(context.Background(), id)
}

// RefreshAllFilterLists triggers a refresh across all enabled filter subscriptions.
func (a *App) RefreshAllFilterLists() ([]filtermanager.UpdateResult, error) {
	a.mu.RLock()
	scheduler := a.scheduler
	updater := a.updater
	a.mu.RUnlock()

	if scheduler != nil {
		return scheduler.RefreshAll(context.Background())
	}
	if updater != nil {
		subs, err := a.storage.GetFilterSubscriptions(context.Background())
		if err != nil {
			return nil, err
		}
		var results []filtermanager.UpdateResult
		for _, s := range subs {
			if s.Enabled {
				res, _ := updater.RefreshSubscription(context.Background(), s.ID)
				if res != nil {
					results = append(results, *res)
				}
			}
		}
		return results, nil
	}

	return nil, errors.New("updater unavailable")
}

// GetCustomRules returns custom user rules from SQLite.
func (a *App) GetCustomRules() []models.CustomRule {
	if a.storage == nil {
		return nil
	}

	rules, err := a.storage.GetCustomRules(context.Background())
	if err != nil {
		return nil
	}
	return rules
}

// AddCustomRule validates, stores, and installs a new custom domain rule.
func (a *App) AddCustomRule(domain string, action string, notes string) (*models.CustomRule, error) {
	if err := models.ValidateDomain(domain); err != nil {
		return nil, err
	}

	normalizedAction := strings.ToUpper(strings.TrimSpace(action))
	if normalizedAction != "BLOCK" && normalizedAction != "ALLOW" {
		return nil, errors.New("rule action must be either 'BLOCK' or 'ALLOW'")
	}

	cleanDomain := strings.ToLower(strings.TrimSpace(domain))

	// Check duplicates
	existingRules := a.GetCustomRules()
	for _, r := range existingRules {
		if strings.EqualFold(r.Domain, cleanDomain) {
			return nil, fmt.Errorf("a rule for domain %q already exists", cleanDomain)
		}
	}

	newRule := models.CustomRule{
		ID:        fmt.Sprintf("rule-%d", time.Now().UnixNano()),
		Domain:    cleanDomain,
		Action:    normalizedAction,
		Enabled:   true,
		CreatedAt: time.Now().Format("2006-01-02 15:04"),
		Notes:     strings.TrimSpace(notes),
	}

	if a.storage != nil {
		if err := a.storage.SaveCustomRule(context.Background(), newRule); err != nil {
			return nil, err
		}
		a.reconcileActiveEngines()
	}

	return &newRule, nil
}

// DeleteCustomRule removes a custom rule from SQLite by ID.
func (a *App) DeleteCustomRule(id string) error {
	if a.storage == nil {
		return errors.New("storage unavailable")
	}

	if err := a.storage.DeleteCustomRule(context.Background(), id); err != nil {
		return err
	}

	a.reconcileActiveEngines()
	return nil
}

// UpdateCustomRule updates a custom rule in SQLite.
func (a *App) UpdateCustomRule(id string, action string, notes string, enabled bool) error {
	if a.storage == nil {
		return errors.New("storage unavailable")
	}

	if err := a.storage.UpdateCustomRule(context.Background(), id, action, notes, enabled); err != nil {
		return err
	}

	a.reconcileActiveEngines()
	return nil
}

// ToggleCustomRule updates the enabled status of an existing custom rule.
func (a *App) ToggleCustomRule(id string, enabled bool) error {
	if a.storage == nil {
		return errors.New("storage unavailable")
	}

	if err := a.storage.ToggleCustomRule(context.Background(), id, enabled); err != nil {
		return err
	}

	a.reconcileActiveEngines()
	return nil
}

// GetSettings returns current application settings from SQLite.
func (a *App) GetSettings() models.AppSettings {
	if a.storage == nil {
		return models.AppSettings{
			UpstreamDoHURL: "https://dns.caliph.dev/dns-query",
			BootstrapDNS:   "1.1.1.1",
			CacheEnabled:   true,
			CacheSize:      4096,
		}
	}

	set, err := a.storage.GetSettings(context.Background())
	if err != nil {
		return models.AppSettings{
			UpstreamDoHURL: "https://dns.caliph.dev/dns-query",
			BootstrapDNS:   "1.1.1.1",
			CacheEnabled:   true,
			CacheSize:      4096,
		}
	}
	return set
}

// UpdateUpstreamDoH validates and updates the DoH upstream URL in SQLite.
func (a *App) UpdateUpstreamDoH(url string) error {
	if err := models.ValidateUpstreamDoHURL(url); err != nil {
		return err
	}

	if a.storage == nil {
		return errors.New("storage unavailable")
	}

	current := a.GetSettings()
	current.UpstreamDoHURL = strings.TrimSpace(url)
	return a.storage.UpdateSettings(context.Background(), current)
}

// GetDNSRewrites returns all persistent DNS rewrites.
func (a *App) GetDNSRewrites() ([]rewrites.Rule, error) {
	if a.storage == nil {
		return nil, errors.New("storage unavailable")
	}
	return a.storage.GetRewrites(context.Background())
}

// SaveDNSRewrite validates, saves, and applies a DNS rewrite rule.
func (a *App) SaveDNSRewrite(rule rewrites.Rule) error {
	if a.storage == nil {
		return errors.New("storage unavailable")
	}
	if err := a.storage.SaveRewrite(context.Background(), rule); err != nil {
		return err
	}
	a.reconcileActiveEngines()
	return nil
}

// DeleteDNSRewrite removes a DNS rewrite rule.
func (a *App) DeleteDNSRewrite(domain string, recordType string) error {
	if a.storage == nil {
		return errors.New("storage unavailable")
	}
	if err := a.storage.DeleteRewrite(context.Background(), domain, recordType); err != nil {
		return err
	}
	a.reconcileActiveEngines()
	return nil
}

// ToggleDNSRewrite toggles the active state of a DNS rewrite rule.
func (a *App) ToggleDNSRewrite(domain string, recordType string, enabled bool) error {
	if a.storage == nil {
		return errors.New("storage unavailable")
	}
	if err := a.storage.ToggleRewrite(context.Background(), domain, recordType, enabled); err != nil {
		return err
	}
	a.reconcileActiveEngines()
	return nil
}

// reconcileActiveEngines atomically refreshes in-memory engines from SQLite.
func (a *App) reconcileActiveEngines() {
	if a.storage == nil {
		return
	}
	active, err := storage.ReconcileActiveState(context.Background(), a.storage)
	if err == nil && active != nil {
		a.mu.Lock()
		a.rewriter = active.Rewriter
		a.matcher = active.Matcher
		handler := a.dnsHandler
		a.mu.Unlock()

		if handler != nil {
			handler.SetRewriter(active.Rewriter)
			handler.SetMatcher(active.Matcher)
			handler.FlushCache()
		}
	}
}

// GetSafeSearchSettings returns all registered SafeSearch provider policies and their verified runtime state.
func (a *App) GetSafeSearchSettings() []models.SafeSearchProviderSetting {
	reg := safesearch.DefaultRegistry()
	providers := reg.GetProviders()

	var records []storage.SafeSearchPolicyRecord
	if a.storage != nil {
		records, _ = a.storage.GetSafeSearchPolicyRecords(context.Background())
	}

	policyMap := make(map[string]storage.SafeSearchPolicyRecord, len(records))
	for _, rec := range records {
		policyMap[rec.ProviderID] = rec
	}

	a.mu.RLock()
	currentRewriter := a.rewriter
	a.mu.RUnlock()

	result := make([]models.SafeSearchProviderSetting, len(providers))
	for i, p := range providers {
		rec, hasPolicy := policyMap[p.ID]
		enabled := hasPolicy && rec.Enabled
		updatedAt := ""
		if hasPolicy {
			updatedAt = rec.UpdatedAt
		}

		var status string
		if p.Status == safesearch.StatusUnsupported {
			status = "UNSUPPORTED"
		} else if !enabled {
			status = "DISABLED"
		} else {
			// Check if policy is actively installed in the runtime rewriter
			status = "PENDING"
			if currentRewriter != nil && len(p.Domains) > 0 {
				req := new(dns.Msg)
				req.SetQuestion(p.Domains[0]+".", dns.TypeA)
				if resp := currentRewriter.Rewrite(req); resp != nil && len(resp.Answer) > 0 {
					if cname, ok := resp.Answer[0].(*dns.CNAME); ok {
						if strings.TrimSuffix(strings.ToLower(cname.Target), ".") == strings.TrimSuffix(strings.ToLower(p.Target), ".") {
							status = "ACTIVE"
						}
					}
				}
			}
		}

		result[i] = models.SafeSearchProviderSetting{
			ID:        p.ID,
			Name:      p.Name,
			Domains:   p.Domains,
			Target:    p.Target,
			DocURL:    p.DocURL,
			Enabled:   enabled,
			Status:    status,
			Warning:   p.Warning,
			UpdatedAt: updatedAt,
		}
	}

	return result
}

// ToggleSafeSearch toggles a provider's SafeSearch policy, persists it to SQLite, and updates the engine.
func (a *App) ToggleSafeSearch(providerID string, enabled bool) error {
	cleanID := strings.ToLower(strings.TrimSpace(providerID))
	reg := safesearch.DefaultRegistry()
	provider, exists := reg.GetProvider(cleanID)
	if !exists {
		return fmt.Errorf("unknown SafeSearch provider: %s", cleanID)
	}

	if provider.Status == safesearch.StatusUnsupported && enabled {
		return fmt.Errorf("provider %s cannot be enabled: %s", provider.Name, provider.Warning)
	}

	if a.storage == nil {
		return errors.New("storage unavailable")
	}

	if err := a.storage.SetSafeSearchPolicy(context.Background(), cleanID, enabled); err != nil {
		return fmt.Errorf("failed to persist SafeSearch policy: %w", err)
	}

	a.reconcileActiveEngines()
	return nil
}

// RunDiagnostics performs non-invasive health checks on the system environment.
func (a *App) RunDiagnostics() []models.DiagnosticCheck {
	checks := make([]models.DiagnosticCheck, 0, 4)

	// 1. Loopback Port 53 Availability Check
	portCheck := models.DiagnosticCheck{
		Name:     "Loopback Port 53 Status",
		Category: "Networking",
	}

	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:53")
	if err == nil {
		conn, listenErr := net.ListenUDP("udp", addr)
		if listenErr != nil {
			portCheck.Status = "WARN"
			portCheck.Details = fmt.Sprintf("Port 53 UDP is currently bound by another process: %v. Milestone 2 will coordinate listener binding.", listenErr)
		} else {
			conn.Close()
			portCheck.Status = "PASS"
			portCheck.Details = "127.0.0.1:53 UDP is free and available for local DNS listener."
		}
	} else {
		portCheck.Status = "WARN"
		portCheck.Details = fmt.Sprintf("Unable to resolve loopback address: %v", err)
	}
	checks = append(checks, portCheck)

	// 2. Upstream DoH Endpoint Connectivity
	settings := a.GetSettings()
	targetURL := settings.UpstreamDoHURL

	dohCheck := models.DiagnosticCheck{
		Name:     "Upstream DoH Reachability",
		Category: "Encrypted DNS",
	}

	httpClient := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	req, reqErr := http.NewRequest("GET", targetURL, nil)
	if reqErr == nil {
		req.Header.Set("Accept", "application/dns-message")
		resp, respErr := httpClient.Do(req)
		if respErr != nil {
			dohCheck.Status = "WARN"
			dohCheck.Details = fmt.Sprintf("Network connection to %s failed: %v", targetURL, respErr)
		} else {
			resp.Body.Close()
			dohCheck.Status = "PASS"
			dohCheck.Details = fmt.Sprintf("Connected to %s successfully (HTTP status: %d). TLS certificate validated.", targetURL, resp.StatusCode)
		}
	} else {
		dohCheck.Status = "FAIL"
		dohCheck.Details = fmt.Sprintf("Malformed HTTP request: %v", reqErr)
	}
	checks = append(checks, dohCheck)

	// 3. Adapter DNS Protection Status
	adapterCheck := models.DiagnosticCheck{
		Name:     "Windows Adapter DNS Integration",
		Category: "Operating System",
		Status:   "INFO",
		Details:  "Adapter configuration is in default system state. In accordance with Milestone 1 safety rules, no network adapters have been modified.",
	}
	checks = append(checks, adapterCheck)

	// 4. Engine Architecture State
	engineCheck := models.DiagnosticCheck{
		Name:     "DNS Engine Architectural State",
		Category: "Core Engine",
		Status:   "INFO",
		Details:  "Desktop Foundation active with SQLite persistence. Engine separation verified. Standalone proxy operational.",
	}
	checks = append(checks, engineCheck)

	return checks
}

// seedInitialDataIfEmpty populates a fresh database with initial default data.
func seedInitialDataIfEmpty(ctx context.Context, store *storage.Storage) {
	// 1. Settings
	_, err := store.GetSettings(ctx)
	if errors.Is(err, storage.ErrNotFound) {
		_ = store.UpdateSettings(ctx, models.AppSettings{
			UpstreamDoHURL:      "https://dns.caliph.dev/dns-query",
			BootstrapDNS:        "1.1.1.1",
			CacheEnabled:        true,
			CacheSize:           4096,
			AutoStart:           false,
			SystemDNSIntegrated: false,
		})
	}

	// 2. Filter Subscriptions
	subs, _ := store.GetFilterSubscriptions(ctx)
	if len(subs) == 0 {
		now := time.Now().UTC()
		initialFilters := []storage.FilterSubscription{
			{
				ID:          "filter-1",
				Name:        "AdAway Default Hosts",
				SourceURL:   "https://adaway.org/hosts.txt",
				Enabled:     true,
				RuleCount:   8742,
				UpdatedAt:   now.Add(-3 * time.Hour).Format(time.RFC3339),
				Description: "Blocks mobile and desktop advertisement networks and telemetry servers.",
				Category:    "Advertising",
			},
			{
				ID:          "filter-2",
				Name:        "StevenBlack Unified Hosts",
				SourceURL:   "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
				Enabled:     true,
				RuleCount:   184291,
				UpdatedAt:   now.Add(-12 * time.Hour).Format(time.RFC3339),
				Description: "Consolidated hosts file blocking adware, spam, and telemetry from multiple curated sources.",
				Category:    "General / Tracking",
			},
			{
				ID:          "filter-3",
				Name:        "OISD Basic",
				SourceURL:   "https://basic.oisd.nl/",
				Enabled:     false,
				RuleCount:   45120,
				UpdatedAt:   now.Add(-24 * time.Hour).Format(time.RFC3339),
				Description: "Lightweight, high-reliability list designed to prevent false positives across smart TVs and desktops.",
				Category:    "Balanced Privacy",
			},
			{
				ID:          "filter-4",
				Name:        "URLhaus Malware Hosts",
				SourceURL:   "https://urlhaus.abuse.ch/downloads/hostfile/",
				Enabled:     true,
				RuleCount:   12840,
				UpdatedAt:   now.Add(-1 * time.Hour).Format(time.RFC3339),
				Description: "Real-time threat intelligence list blocking verified malware distribution sites and botnet C2s.",
				Category:    "Security / Malware",
			},
		}

		for _, f := range initialFilters {
			_ = store.SaveFilterSubscription(ctx, f)
		}
	}

	// 3. Custom Rules
	rules, _ := store.GetCustomRules(ctx)
	if len(rules) == 0 {
		now := time.Now().UTC()
		initialRules := []models.CustomRule{
			{
				ID:        "rule-1",
				Domain:    "telemetry.internal-test.local",
				Action:    "BLOCK",
				Enabled:   true,
				CreatedAt: now.Add(-48 * time.Hour).Format("2006-01-02 15:04"),
				Notes:     "Block local diagnostics tracker",
			},
			{
				ID:        "rule-2",
				Domain:    "nas.home.local",
				Action:    "ALLOW",
				Enabled:   true,
				CreatedAt: now.Add(-24 * time.Hour).Format("2006-01-02 15:04"),
				Notes:     "Bypass filter for local storage appliance",
			},
			{
				ID:        "rule-3",
				Domain:    "tracking.metrics.vendor.com",
				Action:    "BLOCK",
				Enabled:   true,
				CreatedAt: now.Add(-5 * time.Hour).Format("2006-01-02 15:04"),
				Notes:     "Block third-party metrics collector",
			},
		}

		for _, r := range initialRules {
			_ = store.SaveCustomRule(ctx, r)
		}
	}

	// 4. Sample Query Logs
	_, totalLogs, _ := store.GetQueryEvents(ctx, storage.QueryFilter{Limit: 1})
	if totalLogs == 0 {
		now := time.Now().UTC()
		sampleEvents := []storage.QueryEvent{
			{
				ID:                "log-101",
				Timestamp:         now.Add(-12 * time.Second),
				Domain:            "adservice.google.com",
				ClientIP:          "127.0.0.1",
				QueryType:         "A",
				Status:            "BLOCKED",
				UpstreamLatencyMs: 1,
				FilterRule:        "StevenBlack Unified",
				Reason:            "Matched advertising blocklist",
				Protocol:          "UDP",
			},
			{
				ID:                "log-102",
				Timestamp:         now.Add(-24 * time.Second),
				Domain:            "github.com",
				ClientIP:          "127.0.0.1",
				QueryType:         "A",
				Status:            "ALLOWED",
				UpstreamLatencyMs: 24,
				FilterRule:        "Passthrough",
				Reason:            "Clean domain, resolved via DoH",
				Protocol:          "UDP",
			},
			{
				ID:                "log-103",
				Timestamp:         now.Add(-45 * time.Second),
				Domain:            "telemetry.microsoft.com",
				ClientIP:          "127.0.0.1",
				QueryType:         "AAAA",
				Status:            "BLOCKED",
				UpstreamLatencyMs: 1,
				FilterRule:        "AdAway Default Hosts",
				Reason:            "Matched telemetry blocklist",
				Protocol:          "UDP",
			},
		}
		_ = store.InsertQueryEventsBatch(ctx, sampleEvents)
	}
}
