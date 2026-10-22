import React, { useState, useEffect, useCallback } from 'react';
import { Sidebar, NavTab } from './components/Sidebar';
import { DashboardView } from './features/dashboard/DashboardView';
import { QueryLogView } from './features/querylog/QueryLogView';
import { FilterListsView } from './features/filters/FilterListsView';
import { CustomRulesView } from './features/rules/CustomRulesView';
import { SettingsView } from './features/settings/SettingsView';
import { DiagnosticsView } from './features/diagnostics/DiagnosticsView';
import { SplashScreen } from './components/SplashScreen';
import { NeoBadge } from './components/NeoBadge';
import { RefreshCw, AlertTriangle } from 'lucide-react';

import { ipc } from './lib/ipc';
import { models } from '../wailsjs/go/models';

export function App() {
  const [currentTab, setCurrentTab] = useState<NavTab>('dashboard');
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [showSplash, setShowSplash] = useState(true);

  // Application Data States
  const [appInfo, setAppInfo] = useState<models.AppInfo | null>(null);
  const [protectionStatus, setProtectionStatus] = useState<models.ProtectionStatus | null>(null);
  const [queryLogs, setQueryLogs] = useState<models.QueryLogEntry[]>([]);
  const [filterLists, setFilterLists] = useState<models.FilterList[]>([]);
  const [catalog, setCatalog] = useState<models.CatalogFilterItem[]>([]);
  const [customRules, setCustomRules] = useState<models.CustomRule[]>([]);
  const [settings, setSettings] = useState<models.AppSettings | null>(null);
  const [diagnostics, setDiagnostics] = useState<models.DiagnosticCheck[]>([]);
  const [safeSearchSettings, setSafeSearchSettings] = useState<models.SafeSearchProviderSetting[]>([]);
  const [startupStatus, setStartupStatus] = useState<models.StartupStatus | null>(null);

  // Initial Data Fetching from Go Backend / Unified IPC
  const loadData = useCallback(async () => {
    try {
      setIsLoading(true);
      setLoadError(null);

      const [info, status, logs, filters, rules, cfg, diag, ss, cat, st] = await Promise.all([
        ipc.getAppInfo(),
        ipc.getProtectionStatus(),
        ipc.getSampleQueryLogs(),
        ipc.getFilterLists(),
        ipc.getCustomRules(),
        ipc.getSettings(),
        ipc.runDiagnostics(),
        ipc.getSafeSearchSettings(),
        ipc.getCatalogFilters(),
        ipc.getStartupStatus(),
      ]);

      setAppInfo(info);
      setProtectionStatus(status);
      setQueryLogs(logs);
      setFilterLists(filters);
      setCustomRules(rules);
      setSettings(cfg);
      setDiagnostics(diag);
      setSafeSearchSettings(ss);
      setCatalog(cat);
      setStartupStatus(st);
    } catch (err: any) {
      console.error('Failed to load initial application state from backend:', err);
      setLoadError(err?.message || 'Failed to connect to backend process.');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  // Action Handlers bound to IPC
  const handleToggleProtection = async (enabled: boolean) => {
    try {
      const status = await ipc.toggleProtection(enabled);
      setProtectionStatus(status);
    } catch (err) {
      console.error('Failed to toggle protection:', err);
    }
  };

  const handleSimulateQuery = async (domain: string, queryType: string = 'A') => {
    try {
      const entry = await ipc.simulateQuery(domain, queryType);
      setQueryLogs((prev) => [entry, ...prev.slice(0, 49)]);
    } catch (err) {
      console.error('Failed to simulate test query:', err);
    }
  };

  const handleToggleFilter = async (id: string, enabled: boolean) => {
    try {
      const updated = await ipc.toggleFilterList(id, enabled);
      setFilterLists(updated);
    } catch (err) {
      console.error('Failed to toggle filter list:', err);
    }
  };

  const handleRefreshAllFilters = async () => {
    try {
      await ipc.refreshAllFilterLists();
      const updated = await ipc.getFilterLists();
      setFilterLists(updated);
    } catch (err) {
      console.error('Failed to refresh all filter lists:', err);
    }
  };

  const handleRefreshSingleFilter = async (id: string) => {
    try {
      await ipc.refreshFilterList(id);
      const updated = await ipc.getFilterLists();
      setFilterLists(updated);
    } catch (err) {
      console.error(`Failed to refresh filter list ${id}:`, err);
    }
  };

  const handleAddFilterSubscription = async (
    name: string,
    url: string,
    format: string,
    category: string = 'Custom',
    description: string = 'User added filter subscription'
  ) => {
    await ipc.addFilterSubscriptionWithMetadata(name, url, format, category, description);
    const [updatedLists, updatedCat] = await Promise.all([
      ipc.getFilterLists(),
      ipc.getCatalogFilters(),
    ]);
    setFilterLists(updatedLists);
    setCatalog(updatedCat);
  };

  const handleSubscribeCatalog = async (catalogId: string) => {
    await ipc.subscribeCatalogFilter(catalogId);
    const [updatedLists, updatedCat] = await Promise.all([
      ipc.getFilterLists(),
      ipc.getCatalogFilters(),
    ]);
    setFilterLists(updatedLists);
    setCatalog(updatedCat);
  };

  const handleDeleteFilterSubscription = async (id: string) => {
    await ipc.deleteFilterSubscription(id);
    const [updatedLists, updatedCat] = await Promise.all([
      ipc.getFilterLists(),
      ipc.getCatalogFilters(),
    ]);
    setFilterLists(updatedLists);
    setCatalog(updatedCat);
  };

  const handleAddCustomRule = async (domain: string, action: string, notes: string) => {
    const newRule = await ipc.addCustomRule(domain, action, notes);
    setCustomRules((prev) => [newRule, ...prev]);
  };

  const handleUpdateCustomRule = async (
    id: string,
    action: string,
    notes: string,
    enabled: boolean
  ) => {
    await ipc.updateCustomRule(id, action, notes, enabled);
    setCustomRules((prev) =>
      prev.map((r) =>
        r.id === id ? new models.CustomRule({ ...r, action, notes, enabled }) : r
      )
    );
  };

  const handleToggleCustomRule = async (id: string, enabled: boolean) => {
    await ipc.toggleCustomRule(id, enabled);
    setCustomRules((prev) =>
      prev.map((r) => (r.id === id ? new models.CustomRule({ ...r, enabled }) : r))
    );
  };

  const handleDeleteCustomRule = async (id: string) => {
    await ipc.deleteCustomRule(id);
    setCustomRules((prev) => prev.filter((r) => r.id !== id));
  };

  const handleUpdateUpstream = async (url: string) => {
    await ipc.updateUpstreamDoH(url);
    const updatedCfg = await ipc.getSettings();
    setSettings(updatedCfg);
    const updatedStatus = await ipc.getProtectionStatus();
    setProtectionStatus(updatedStatus);
  };

  const handleRefreshDiagnostics = async () => {
    const diag = await ipc.runDiagnostics();
    setDiagnostics(diag);
  };

  const handleToggleSafeSearch = async (providerId: string, enabled: boolean) => {
    try {
      const updated = await ipc.toggleSafeSearch(providerId, enabled);
      setSafeSearchSettings(updated);
    } catch (err) {
      console.error(`Failed to toggle SafeSearch for ${providerId}:`, err);
      throw err;
    }
  };

  const handleToggleStartup = async (enabled: boolean) => {
    try {
      const updated = await ipc.setStartWithWindows(enabled);
      setStartupStatus(updated);
    } catch (err) {
      console.error('Failed to update startup configuration:', err);
      throw err;
    }
  };

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-[#F5F1E8] text-[#121212]">
      {/* NeoBrutalist Splash Screen Overlay */}
      {showSplash && (
        <SplashScreen
          version={appInfo?.version || 'v0.4.0'}
          isReady={!isLoading && !loadError}
          error={loadError}
          onRetry={loadData}
          onDismiss={() => setShowSplash(false)}
        />
      )}

      {/* Desktop Sidebar */}
      <Sidebar
        currentTab={currentTab}
        onSelectTab={setCurrentTab}
        appVersion={appInfo?.version || 'v0.4.0'}
        isProtected={protectionStatus?.isActive ?? true}
        onToggleProtection={handleToggleProtection}
      />

      {/* Main Content Workspace */}
      <div className="flex-1 flex flex-col h-screen overflow-hidden">
        {/* Top Desktop Bar */}
        <header className="h-14 neo-border-b bg-white flex items-center justify-between px-6 shrink-0 select-none">
          <div className="flex items-center gap-3">
            <span className="text-xs font-black uppercase tracking-wider text-[#121212]">
              BlockAds Desktop
            </span>
            <span className="text-xs text-[#9CA3AF]">/</span>
            <span className="text-xs font-bold uppercase tracking-wider text-[#B38B6D]">
              {currentTab}
            </span>
          </div>

          <div className="flex items-center gap-3">
            <NeoBadge variant={protectionStatus?.isActive ?? true ? 'success' : 'warning'} size="sm">
              <span className="w-1.5 h-1.5 rounded-full bg-current mr-1 inline-block animate-ping" />
              {protectionStatus?.isActive ?? true ? 'PROTECTION ACTIVE' : 'PROTECTION PAUSED'}
            </NeoBadge>

            <button
              onClick={loadData}
              title="Refresh telemetry & status from Go engine"
              className="p-1.5 border-2 border-[#121212] bg-[#F5F1E8] hover:bg-white transition-all neo-shadow-hover cursor-pointer"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${isLoading ? 'animate-spin' : ''}`} />
            </button>
          </div>
        </header>

        {/* Scrollable View Container */}
        <main className="flex-1 overflow-y-auto p-6">
          {loadError ? (
            <div className="p-6 bg-[#FEECEB] neo-border neo-shadow-md space-y-3 max-w-lg mx-auto mt-12">
              <div className="flex items-center gap-2 text-sm font-bold text-[#D9381E]">
                <AlertTriangle className="w-5 h-5" />
                <span>Backend Initialization Error</span>
              </div>
              <p className="text-xs text-[#4B5563] leading-relaxed font-mono bg-white p-3 border border-[#121212]">
                {loadError}
              </p>
              <button
                onClick={loadData}
                className="px-4 py-2 bg-[#121212] text-[#F5F1E8] font-bold text-xs uppercase border-2 border-[#121212] neo-shadow-hover cursor-pointer"
              >
                Retry Connection
              </button>
            </div>
          ) : (
            <>
              {currentTab === 'dashboard' && (
                <DashboardView
                  protectionStatus={protectionStatus}
                  sampleLogs={queryLogs}
                  filterLists={filterLists}
                  onNavigate={setCurrentTab}
                  onToggleProtection={handleToggleProtection}
                  onSimulateQuery={handleSimulateQuery}
                />
              )}

              {currentTab === 'querylog' && (
                <QueryLogView logs={queryLogs} />
              )}

              {currentTab === 'filters' && (
                <FilterListsView
                  filterLists={filterLists}
                  catalog={catalog}
                  onToggleFilter={handleToggleFilter}
                  onRefreshAll={handleRefreshAllFilters}
                  onRefreshSingle={handleRefreshSingleFilter}
                  onAddSubscription={handleAddFilterSubscription}
                  onSubscribeCatalog={handleSubscribeCatalog}
                  onDeleteSubscription={handleDeleteFilterSubscription}
                />
              )}

              {currentTab === 'rules' && (
                <CustomRulesView
                  rules={customRules}
                  onAddRule={handleAddCustomRule}
                  onUpdateRule={handleUpdateCustomRule}
                  onToggleRule={handleToggleCustomRule}
                  onDeleteRule={handleDeleteCustomRule}
                />
              )}

              {currentTab === 'settings' && (
                <SettingsView
                  settings={settings}
                  safeSearchSettings={safeSearchSettings}
                  startupStatus={startupStatus}
                  appInfo={appInfo}
                  onUpdateUpstream={handleUpdateUpstream}
                  onToggleSafeSearch={handleToggleSafeSearch}
                  onToggleStartup={handleToggleStartup}
                />
              )}

              {currentTab === 'diagnostics' && (
                <DiagnosticsView
                  appInfo={appInfo}
                  diagnostics={diagnostics}
                  onRefreshDiagnostics={handleRefreshDiagnostics}
                />
              )}
            </>
          )}
        </main>
      </div>
    </div>
  );
}

export default App;
