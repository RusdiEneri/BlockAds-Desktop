import { models } from '../../wailsjs/go/models';
import * as WailsApp from '../../wailsjs/go/main/App';

function isWailsAvailable(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof (window as any)?.go?.main?.App?.GetAppInfo === 'function'
  );
}

// Fallback in-memory state for standalone browser preview / testing
let fallbackAppInfo: models.AppInfo = new models.AppInfo({
  name: 'BlockAds Desktop',
  version: '0.1.0-alpha',
  platform: 'windows/amd64',
  milestone: 'Milestone 01: Desktop Foundation',
  goVersion: 'go1.27.1',
  webView2Version: '155.0.4283.62',
});

let fallbackSettings: models.AppSettings = new models.AppSettings({
  upstreamDohUrl: 'https://dns.caliph.dev/dns-query',
  bootstrapDns: '1.1.1.1',
  cacheEnabled: true,
  cacheSize: 4096,
  autoStart: false,
  systemDnsIntegrated: false,
});

let fallbackProtectionStatus: models.ProtectionStatus = new models.ProtectionStatus({
  isActive: true,
  state: 'ACTIVE',
  upstreamUrl: 'https://dns.caliph.dev/dns-query',
  listenAddr: '127.0.0.1:53',
  message: 'DNS filtering and ad-blocking protection is ACTIVE.',
});

let fallbackFilterLists: models.FilterList[] = [
  new models.FilterList({
    id: 'filter-1',
    name: 'AdAway Default Hosts',
    sourceUrl: 'https://adaway.org/hosts.txt',
    enabled: true,
    ruleCount: 8742,
    lastUpdated: new Date(Date.now() - 3 * 3600 * 1000).toISOString(),
    description: 'Blocks mobile and desktop advertisement networks and telemetry servers.',
    category: 'Advertising',
  }),
  new models.FilterList({
    id: 'filter-2',
    name: 'StevenBlack Unified Hosts',
    sourceUrl: 'https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts',
    enabled: true,
    ruleCount: 184291,
    lastUpdated: new Date(Date.now() - 12 * 3600 * 1000).toISOString(),
    description: 'Consolidated hosts file blocking adware, spam, and telemetry from multiple curated sources.',
    category: 'General / Tracking',
  }),
  new models.FilterList({
    id: 'filter-3',
    name: 'OISD Basic',
    sourceUrl: 'https://basic.oisd.nl/',
    enabled: false,
    ruleCount: 45120,
    lastUpdated: new Date(Date.now() - 24 * 3600 * 1000).toISOString(),
    description: 'Lightweight, high-reliability list designed to prevent false positives across smart TVs and desktops.',
    category: 'Balanced Privacy',
  }),
  new models.FilterList({
    id: 'filter-4',
    name: 'URLhaus Malware Hosts',
    sourceUrl: 'https://urlhaus.abuse.ch/downloads/hostfile/',
    enabled: true,
    ruleCount: 12840,
    lastUpdated: new Date(Date.now() - 1 * 3600 * 1000).toISOString(),
    description: 'Real-time threat intelligence list blocking verified malware distribution sites and botnet C2s.',
    category: 'Security / Malware',
  }),
];

let fallbackCustomRules: models.CustomRule[] = [
  new models.CustomRule({
    id: 'rule-1',
    domain: 'telemetry.internal-test.local',
    action: 'BLOCK',
    enabled: true,
    createdAt: '2026-10-20 14:30',
    notes: 'Block local diagnostics tracker',
  }),
  new models.CustomRule({
    id: 'rule-2',
    domain: 'nas.home.local',
    action: 'ALLOW',
    enabled: true,
    createdAt: '2026-10-21 09:15',
    notes: 'Bypass filter for local storage appliance',
  }),
  new models.CustomRule({
    id: 'rule-3',
    domain: 'tracking.metrics.vendor.com',
    action: 'BLOCK',
    enabled: true,
    createdAt: '2026-10-22 18:45',
    notes: 'Block third-party metrics collector',
  }),
];

let fallbackQueryLogs: models.QueryLogEntry[] = [
  new models.QueryLogEntry({
    id: 'log-101',
    timestamp: '22:04:17',
    domain: 'adservice.google.com',
    clientIp: '127.0.0.1',
    queryType: 'A',
    status: 'BLOCKED',
    upstreamLatencyMs: 1,
    filterRule: 'StevenBlack Unified',
    reason: 'Matched advertising blocklist',
  }),
  new models.QueryLogEntry({
    id: 'log-102',
    timestamp: '22:04:05',
    domain: 'github.com',
    clientIp: '127.0.0.1',
    queryType: 'A',
    status: 'ALLOWED',
    upstreamLatencyMs: 24,
    filterRule: 'Passthrough',
    reason: 'Clean domain, resolved via DoH',
  }),
  new models.QueryLogEntry({
    id: 'log-103',
    timestamp: '22:03:44',
    domain: 'telemetry.microsoft.com',
    clientIp: '127.0.0.1',
    queryType: 'AAAA',
    status: 'BLOCKED',
    upstreamLatencyMs: 1,
    filterRule: 'AdAway Default Hosts',
    reason: 'Matched telemetry blocklist',
  }),
  new models.QueryLogEntry({
    id: 'log-104',
    timestamp: '22:03:21',
    domain: 'api.github.com',
    clientIp: '127.0.0.1',
    queryType: 'HTTPS',
    status: 'ALLOWED',
    upstreamLatencyMs: 28,
    filterRule: 'Passthrough',
    reason: 'Clean domain, resolved via DoH',
  }),
  new models.QueryLogEntry({
    id: 'log-105',
    timestamp: '22:02:57',
    domain: 'analytics.tiktok.com',
    clientIp: '127.0.0.1',
    queryType: 'A',
    status: 'BLOCKED',
    upstreamLatencyMs: 1,
    filterRule: 'StevenBlack Unified',
    reason: 'Matched social tracker blocklist',
  }),
  new models.QueryLogEntry({
    id: 'log-106',
    timestamp: '22:02:34',
    domain: 'cdn.jsdelivr.net',
    clientIp: '127.0.0.1',
    queryType: 'A',
    status: 'ALLOWED',
    upstreamLatencyMs: 19,
    filterRule: 'Passthrough',
    reason: 'Clean CDN endpoint',
  }),
  new models.QueryLogEntry({
    id: 'log-107',
    timestamp: '22:02:09',
    domain: 'badware-c2-node.xyz',
    clientIp: '127.0.0.1',
    queryType: 'A',
    status: 'BLOCKED',
    upstreamLatencyMs: 1,
    filterRule: 'URLhaus Malware Hosts',
    reason: 'Matched malware distribution indicator',
  }),
  new models.QueryLogEntry({
    id: 'log-108',
    timestamp: '22:01:29',
    domain: 'unresponsive-internal-dns.invalid',
    clientIp: '127.0.0.1',
    queryType: 'A',
    status: 'FAILED',
    upstreamLatencyMs: 2500,
    filterRule: 'None',
    reason: 'Upstream resolution timeout / NXDOMAIN',
  }),
];

let fallbackDiagnostics: models.DiagnosticCheck[] = [
  new models.DiagnosticCheck({
    name: 'Loopback Port 53 Status',
    category: 'Networking',
    status: 'PASS',
    details: '127.0.0.1:53 UDP is free and available for local DNS listener.',
  }),
  new models.DiagnosticCheck({
    name: 'Upstream DoH Reachability',
    category: 'Encrypted DNS',
    status: 'PASS',
    details: 'Connected to https://dns.caliph.dev/dns-query successfully (HTTP status: 400). TLS certificate validated.',
  }),
  new models.DiagnosticCheck({
    name: 'Windows Adapter DNS Integration',
    category: 'Operating System',
    status: 'INFO',
    details: 'Adapter configuration is in default system state. In accordance with Milestone 1 safety rules, no network adapters have been modified.',
  }),
  new models.DiagnosticCheck({
    name: 'DNS Engine Architectural State',
    category: 'Core Engine',
    status: 'INFO',
    details: 'Desktop Foundation active. Engine separation verified. Ready for Milestone 2 standalone proxy implementation.',
  }),
];

let fallbackSafeSearchSettings: models.SafeSearchProviderSetting[] = [
  new models.SafeSearchProviderSetting({
    id: 'google',
    name: 'Google Search',
    domains: [
      'www.google.com',
      'www.google.co.uk',
      'www.google.ca',
      'www.google.de',
      'www.google.fr',
      'www.google.co.jp',
    ],
    target: 'forcesafesearch.google.com',
    docUrl: 'https://support.google.com/websearch/answer/186669',
    enabled: true,
    status: 'ACTIVE',
    updatedAt: new Date(Date.now() - 3600 * 1000).toISOString(),
  }),
  new models.SafeSearchProviderSetting({
    id: 'bing',
    name: 'Microsoft Bing',
    domains: ['www.bing.com', 'edgeservices.bing.com'],
    target: 'strict.bing.com',
    docUrl: 'https://support.microsoft.com/en-us/bing/blocking-explicit-content-with-safesearch',
    enabled: true,
    status: 'ACTIVE',
    updatedAt: new Date(Date.now() - 3600 * 1000).toISOString(),
  }),
  new models.SafeSearchProviderSetting({
    id: 'duckduckgo',
    name: 'DuckDuckGo',
    domains: ['duckduckgo.com', 'www.duckduckgo.com'],
    target: 'safe.duckduckgo.com',
    docUrl: 'https://duckduckgo.com/duckduckgo-help-pages/features/safe-search',
    enabled: false,
    status: 'DISABLED',
    updatedAt: '',
  }),
  new models.SafeSearchProviderSetting({
    id: 'youtube',
    name: 'YouTube',
    domains: [
      'www.youtube.com',
      'm.youtube.com',
      'youtubei.googleapis.com',
      'youtube.googleapis.com',
      'www.youtube-nocookie.com',
    ],
    target: 'restrict.youtube.com',
    docUrl: 'https://support.google.com/a/answer/6214622',
    enabled: false,
    status: 'DISABLED',
    updatedAt: '',
  }),
  new models.SafeSearchProviderSetting({
    id: 'yandex',
    name: 'Yandex',
    domains: ['yandex.ru', 'www.yandex.ru', 'yandex.com', 'www.yandex.com'],
    target: '',
    docUrl: 'https://yandex.com/support/search/settings/family-search.html',
    enabled: false,
    status: 'UNSUPPORTED',
    warning:
      'Yandex Family Search requires routing through dedicated upstream DNS resolvers (77.88.8.7 / 77.88.8.3) rather than standard CNAME host rewrites. CNAME rewriting to family.yandex.ru is not officially supported by Yandex.',
    updatedAt: '',
  }),
];

// Unified API exposing Wails bindings when running in desktop app or fallback when in browser
export const ipc = {
  getAppInfo: async (): Promise<models.AppInfo> => {
    if (isWailsAvailable()) {
      return WailsApp.GetAppInfo();
    }
    return fallbackAppInfo;
  },

  getProtectionStatus: async (): Promise<models.ProtectionStatus> => {
    if (isWailsAvailable()) {
      return WailsApp.GetProtectionStatus();
    }
    return fallbackProtectionStatus;
  },

  toggleProtection: async (enabled: boolean): Promise<models.ProtectionStatus> => {
    if (isWailsAvailable()) {
      return WailsApp.ToggleProtection(enabled);
    }
    fallbackProtectionStatus = new models.ProtectionStatus({
      ...fallbackProtectionStatus,
      isActive: enabled,
      state: enabled ? 'ACTIVE' : 'PAUSED',
      listenAddr: enabled ? '127.0.0.1:53' : '127.0.0.1:53 (Paused)',
      message: enabled
        ? 'DNS filtering and ad-blocking protection is ACTIVE.'
        : 'Protection is PAUSED. Ad-blocking and filter lists are temporarily bypassed.',
    });
    return fallbackProtectionStatus;
  },

  simulateQuery: async (domain: string, queryType: string = 'A'): Promise<models.QueryLogEntry> => {
    if (isWailsAvailable()) {
      return WailsApp.SimulateQuery(domain, queryType);
    }
    const isAd = domain.includes('ad') || domain.includes('track') || domain.includes('doubleclick') || domain.includes('analytics');
    const newEntry = new models.QueryLogEntry({
      id: `sim-${Date.now()}`,
      timestamp: new Date().toISOString(),
      domain,
      clientIp: '127.0.0.1',
      queryType,
      status: fallbackProtectionStatus.isActive && isAd ? 'BLOCKED' : 'ALLOWED',
      upstreamLatencyMs: Math.floor(Math.random() * 25) + 2,
      filterRule: isAd ? 'StevenBlack Unified / AdAway' : 'Passthrough',
      reason: isAd ? 'Matched advertising blocklist' : 'Clean domain resolved via DoH',
    });
    fallbackQueryLogs = [newEntry, ...fallbackQueryLogs.slice(0, 49)];
    return newEntry;
  },

  onQueryEvent: (callback: (entry: models.QueryLogEntry) => void): (() => void) => {
    if (typeof window !== 'undefined' && (window as any)?.runtime?.EventsOn) {
      (window as any).runtime.EventsOn('query:event', callback);
      return () => {
        if ((window as any)?.runtime?.EventsOff) {
          (window as any).runtime.EventsOff('query:event');
        }
      };
    }
    return () => {};
  },

  onProtectionStatus: (callback: (status: models.ProtectionStatus) => void): (() => void) => {
    if (typeof window !== 'undefined' && (window as any)?.runtime?.EventsOn) {
      (window as any).runtime.EventsOn('protection:status', callback);
      return () => {
        if ((window as any)?.runtime?.EventsOff) {
          (window as any).runtime.EventsOff('protection:status');
        }
      };
    }
    return () => {};
  },

  onSettingsUpdated: (callback: (settings: models.AppSettings) => void): (() => void) => {
    if (typeof window !== 'undefined' && (window as any)?.runtime?.EventsOn) {
      (window as any).runtime.EventsOn('settings:updated', callback);
      return () => {
        if ((window as any)?.runtime?.EventsOff) {
          (window as any).runtime.EventsOff('settings:updated');
        }
      };
    }
    return () => {};
  },

  getSampleQueryLogs: async (): Promise<models.QueryLogEntry[]> => {
    if (isWailsAvailable()) {
      return WailsApp.GetSampleQueryLogs();
    }
    return [...fallbackQueryLogs];
  },

  getFilterLists: async (): Promise<models.FilterList[]> => {
    if (isWailsAvailable()) {
      return WailsApp.GetFilterLists();
    }
    return [...fallbackFilterLists];
  },

  toggleFilterList: async (id: string, enabled: boolean): Promise<models.FilterList[]> => {
    if (isWailsAvailable()) {
      return WailsApp.ToggleFilterList(id, enabled);
    }
    fallbackFilterLists = fallbackFilterLists.map((f) =>
      f.id === id ? new models.FilterList({ ...f, enabled }) : f
    );
    return [...fallbackFilterLists];
  },

  addFilterSubscription: async (
    name: string,
    sourceUrl: string,
    format: string
  ): Promise<models.FilterList> => {
    if (isWailsAvailable()) {
      return WailsApp.AddFilterSubscription(name, sourceUrl, format);
    }
    const newSub = new models.FilterList({
      id: `filter-${Date.now()}`,
      name,
      sourceUrl,
      enabled: true,
      format,
      ruleCount: 0,
      lastUpdated: new Date().toISOString(),
      lastAttemptAt: new Date().toISOString(),
      lastSuccessAt: '',
      lastStatus: 'pending',
      lastError: '',
      contentChecksum: '',
      description: 'Custom added filter subscription',
      category: 'Custom',
    });
    fallbackFilterLists.push(newSub);
    return newSub;
  },

  deleteFilterSubscription: async (id: string): Promise<void> => {
    if (isWailsAvailable()) {
      return WailsApp.DeleteFilterSubscription(id);
    }
    fallbackFilterLists = fallbackFilterLists.filter((f) => f.id !== id);
  },

  refreshFilterList: async (id: string): Promise<any> => {
    if (isWailsAvailable()) {
      return WailsApp.RefreshFilterList(id);
    }
    fallbackFilterLists = fallbackFilterLists.map((f) => {
      if (f.id === id) {
        return new models.FilterList({
          ...f,
          lastAttemptAt: new Date().toISOString(),
          lastSuccessAt: new Date().toISOString(),
          lastStatus: 'ok',
          lastError: '',
        });
      }
      return f;
    });
    return { subscription_id: id, status: 'updated', status_code: 200, rule_count: 1000 };
  },

  refreshAllFilterLists: async (): Promise<any[]> => {
    if (isWailsAvailable()) {
      return WailsApp.RefreshAllFilterLists();
    }
    const results = fallbackFilterLists.map((f) => ({
      subscription_id: f.id,
      status: 'updated',
      status_code: 200,
      rule_count: f.ruleCount || 1000,
    }));
    return results;
  },

  getCustomRules: async (): Promise<models.CustomRule[]> => {
    if (isWailsAvailable()) {
      return WailsApp.GetCustomRules();
    }
    return [...fallbackCustomRules];
  },

  addCustomRule: async (
    domain: string,
    action: string,
    notes: string
  ): Promise<models.CustomRule> => {
    if (isWailsAvailable()) {
      return WailsApp.AddCustomRule(domain, action, notes);
    }
    const cleanDomain = domain.trim().toLowerCase();
    const cleanAction = action.trim().toUpperCase();
    if (fallbackCustomRules.some((r) => r.domain.toLowerCase() === cleanDomain)) {
      throw new Error(`A rule for domain "${cleanDomain}" already exists`);
    }
    const newRule = new models.CustomRule({
      id: `rule-${Date.now()}`,
      domain: cleanDomain,
      action: cleanAction,
      enabled: true,
      createdAt: new Date().toISOString().replace('T', ' ').slice(0, 16),
      notes: notes.trim(),
    });
    fallbackCustomRules = [newRule, ...fallbackCustomRules];
    return newRule;
  },

  updateCustomRule: async (
    id: string,
    action: string,
    notes: string,
    enabled: boolean
  ): Promise<void> => {
    if (isWailsAvailable()) {
      return WailsApp.UpdateCustomRule(id, action, notes, enabled);
    }
    const idx = fallbackCustomRules.findIndex((r) => r.id === id);
    if (idx === -1) {
      throw new Error('Rule not found');
    }
    fallbackCustomRules[idx] = new models.CustomRule({
      ...fallbackCustomRules[idx],
      action: action.trim().toUpperCase(),
      notes: notes.trim(),
      enabled,
    });
  },

  toggleCustomRule: async (id: string, enabled: boolean): Promise<void> => {
    if (isWailsAvailable()) {
      return WailsApp.ToggleCustomRule(id, enabled);
    }
    const idx = fallbackCustomRules.findIndex((r) => r.id === id);
    if (idx === -1) {
      throw new Error('Rule not found');
    }
    fallbackCustomRules[idx] = new models.CustomRule({
      ...fallbackCustomRules[idx],
      enabled,
    });
  },

  deleteCustomRule: async (id: string): Promise<void> => {
    if (isWailsAvailable()) {
      return WailsApp.DeleteCustomRule(id);
    }
    fallbackCustomRules = fallbackCustomRules.filter((r) => r.id !== id);
  },

  getSettings: async (): Promise<models.AppSettings> => {
    if (isWailsAvailable()) {
      return WailsApp.GetSettings();
    }
    return fallbackSettings;
  },

  updateUpstreamDoH: async (url: string): Promise<void> => {
    if (isWailsAvailable()) {
      return WailsApp.UpdateUpstreamDoH(url);
    }
    const trimmed = url.trim();
    if (!trimmed.startsWith('https://')) {
      throw new Error("DoH URL must use the 'https://' scheme");
    }
    fallbackSettings = new models.AppSettings({
      ...fallbackSettings,
      upstreamDohUrl: trimmed,
    });
    fallbackProtectionStatus = new models.ProtectionStatus({
      ...fallbackProtectionStatus,
      upstreamUrl: trimmed,
    });
  },

  runDiagnostics: async (): Promise<models.DiagnosticCheck[]> => {
    if (isWailsAvailable()) {
      return WailsApp.RunDiagnostics();
    }
    return [...fallbackDiagnostics];
  },

  getSafeSearchSettings: async (): Promise<models.SafeSearchProviderSetting[]> => {
    if (isWailsAvailable()) {
      return WailsApp.GetSafeSearchSettings();
    }
    return [...fallbackSafeSearchSettings];
  },

  toggleSafeSearch: async (providerId: string, enabled: boolean): Promise<models.SafeSearchProviderSetting[]> => {
    if (isWailsAvailable()) {
      await WailsApp.ToggleSafeSearch(providerId, enabled);
      return WailsApp.GetSafeSearchSettings();
    }
    const idx = fallbackSafeSearchSettings.findIndex((p) => p.id === providerId);
    if (idx !== -1) {
      if (fallbackSafeSearchSettings[idx].status === 'UNSUPPORTED' && enabled) {
        throw new Error('Unsupported provider cannot be enabled');
      }
      fallbackSafeSearchSettings[idx] = new models.SafeSearchProviderSetting({
        ...fallbackSafeSearchSettings[idx],
        enabled,
        status: enabled ? 'ACTIVE' : 'DISABLED',
        updatedAt: new Date().toISOString(),
      });
    }
    return [...fallbackSafeSearchSettings];
  },

  getCatalogFilters: async (): Promise<models.CatalogFilterItem[]> => {
    if (isWailsAvailable()) {
      return WailsApp.GetCatalogFilters();
    }
    return [...fallbackCatalog];
  },

  previewFilterURL: async (url: string): Promise<models.FilterPreviewResult> => {
    if (isWailsAvailable()) {
      return WailsApp.PreviewFilterURL(url);
    }
    const clean = url.trim().toLowerCase();
    const isSub = fallbackFilterLists.some((f) => f.sourceUrl.toLowerCase() === clean);
    return new models.FilterPreviewResult({
      sourceUrl: url,
      normalizedUrl: url,
      detectedName: clean.split('/').pop()?.replace('.txt', '') || 'Custom Filter',
      description: 'Extracted rule list from remote source',
      format: 'hosts',
      ruleCount: 1542,
      isAlreadySubscribed: isSub,
      existingSubscription: isSub ? 'Existing Filter' : undefined,
    });
  },

  addFilterSubscriptionWithMetadata: async (
    name: string,
    url: string,
    format: string,
    category: string,
    description: string
  ): Promise<models.FilterList> => {
    if (isWailsAvailable()) {
      return WailsApp.AddFilterSubscriptionWithMetadata(name, url, format, category, description);
    }
    const newSub = new models.FilterList({
      id: `filter-${Date.now()}`,
      name,
      sourceUrl: url,
      enabled: true,
      format,
      ruleCount: 2500,
      lastUpdated: new Date().toISOString(),
      description,
      category,
      lastStatus: 'ok',
    });
    fallbackFilterLists.push(newSub);
    return newSub;
  },

  subscribeCatalogFilter: async (catalogId: string): Promise<models.FilterList> => {
    if (isWailsAvailable()) {
      return WailsApp.SubscribeCatalogFilter(catalogId);
    }
    const item = fallbackCatalog.find((c) => c.id === catalogId);
    if (!item) throw new Error('Catalog filter not found');
    item.isSubscribed = true;
    return ipc.addFilterSubscriptionWithMetadata(
      item.name,
      item.sourceUrl,
      item.format,
      item.category,
      item.description
    );
  },

  getStartupStatus: async (): Promise<models.StartupStatus> => {
    if (isWailsAvailable()) {
      return WailsApp.GetStartupStatus();
    }
    return fallbackStartupStatus;
  },

  setStartWithWindows: async (enabled: boolean): Promise<models.StartupStatus> => {
    if (isWailsAvailable()) {
      return WailsApp.SetStartWithWindows(enabled);
    }
    fallbackStartupStatus = new models.StartupStatus({
      ...fallbackStartupStatus,
      enabled,
      message: enabled
        ? 'Application is configured to start on Windows login (Preview mock).'
        : 'Startup launch is currently disabled.',
    });
    return fallbackStartupStatus;
  },

  getTelemetryStats: async (): Promise<models.TelemetryStats> => {
    if (isWailsAvailable() && typeof WailsApp.GetTelemetryStats === 'function') {
      return WailsApp.GetTelemetryStats();
    }
    return new models.TelemetryStats({
      queriesPerSec: 91.8,
      peakQueriesPerSec: 101,
      cacheHitRate: 21.7,
      cachedEntries: 21600,
      medianLatencyMs: 2.11,
      blockedPercent: 8.70,
      totalAnswers: 1760000,
      totalBlocked: 153120,
      resolverLoad: 6.30,
      memoryResidentMB: 266,
      uptimeSeconds: 352800,
      engineVersion: 'dnsdist 2.0.11',
    });
  },
};

let fallbackStartupStatus: models.StartupStatus = new models.StartupStatus({
  supported: true,
  enabled: false,
  executablePath: 'C:\\Program Files\\BlockAds Desktop\\BlockAds-Desktop.exe',
  isPathValid: true,
  message: 'Startup launch is currently disabled in preview mock.',
});

let fallbackCatalog: models.CatalogFilterItem[] = [
  new models.CatalogFilterItem({
    id: 'adaway',
    name: 'AdAway Default Hosts',
    description: 'Blocks mobile and desktop advertisement networks and telemetry servers.',
    category: 'Advertising',
    sourceUrl: 'https://adaway.org/hosts.txt',
    format: 'hosts',
    homepage: 'https://adaway.org/',
    maintainer: 'AdAway Team',
    license: 'CC BY 3.0',
    isSubscribed: true,
    subscribedId: 'filter-1',
  }),
  new models.CatalogFilterItem({
    id: 'stevenblack',
    name: 'StevenBlack Unified Hosts',
    description: 'Consolidated hosts file blocking adware, spam, and telemetry from multiple curated sources.',
    category: 'General Protection',
    sourceUrl: 'https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts',
    format: 'hosts',
    homepage: 'https://github.com/StevenBlack/hosts',
    maintainer: 'Steven Black',
    license: 'MIT',
    isSubscribed: true,
    subscribedId: 'filter-2',
  }),
  new models.CatalogFilterItem({
    id: 'oisd-basic',
    name: 'OISD Basic',
    description: 'Lightweight, high-reliability list designed to prevent false positives across smart TVs and desktops.',
    category: 'Privacy & Tracking',
    sourceUrl: 'https://basic.oisd.nl/',
    format: 'domains',
    homepage: 'https://oisd.nl/',
    maintainer: 'Stephan van Velzen (OISD)',
    license: 'Free for personal use',
    isSubscribed: false,
  }),
  new models.CatalogFilterItem({
    id: 'urlhaus',
    name: 'URLhaus Malware Hosts',
    description: 'Real-time threat intelligence list blocking verified malware distribution sites and botnet C2s.',
    category: 'Security & Malware',
    sourceUrl: 'https://urlhaus.abuse.ch/downloads/hostfile/',
    format: 'hosts',
    homepage: 'https://urlhaus.abuse.ch/',
    maintainer: 'abuse.ch',
    license: 'CC0',
    isSubscribed: true,
    subscribedId: 'filter-4',
  }),
  new models.CatalogFilterItem({
    id: 'dan-pollock',
    name: "Dan Pollock's Hosts",
    description: 'Maintained since 1999, blocks intrusive advertising, web tracking, and shock sites.',
    category: 'Advertising',
    sourceUrl: 'https://someonewhocares.org/hosts/hosts',
    format: 'hosts',
    homepage: 'https://someonewhocares.org/hosts/',
    maintainer: 'Dan Pollock',
    license: 'Free',
    isSubscribed: false,
  }),
  new models.CatalogFilterItem({
    id: 'peter-lowe',
    name: "Peter Lowe's Ad & Tracking List",
    description: 'Strictly curated list of adservers and tracking domains with near-zero false positive rate.',
    category: 'Privacy & Tracking',
    sourceUrl: 'https://pgl.yoyo.org/adservers/serverlist.php?hostformat=hosts&mimetype=plaintext',
    format: 'hosts',
    homepage: 'https://pgl.yoyo.org/adservers/',
    maintainer: 'Peter Lowe',
    license: 'Free for personal use',
    isSubscribed: false,
  }),
];

