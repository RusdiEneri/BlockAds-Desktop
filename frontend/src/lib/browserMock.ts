import { models } from '../../wailsjs/go/models';

export function setupBrowserMockIfNeeded() {
  if (typeof window === 'undefined') return;

  const w = window as any;
  if (w.go && w.go.main && w.go.main.App) {
    return; // Real Wails runtime is present
  }

  console.info('[BlockAds Desktop] Wails runtime not detected; initializing browser fallback mock.');

  let filterLists: models.FilterList[] = [
    {
      id: 'filter-1',
      name: 'AdAway Default Hosts',
      sourceUrl: 'https://adaway.org/hosts.txt',
      enabled: true,
      format: 'hosts',
      ruleCount: 8742,
      lastUpdated: new Date().toISOString(),
      lastAttemptAt: new Date().toISOString(),
      lastSuccessAt: new Date().toISOString(),
      lastStatus: 'ok',
      lastError: '',
      contentChecksum: 'sha256-mock-adaway',
      etag: '"v1.0"',
      description: 'Blocks mobile and desktop advertisement networks and telemetry servers.',
      category: 'Advertising',
    },
    {
      id: 'filter-2',
      name: 'StevenBlack Unified Hosts',
      sourceUrl: 'https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts',
      enabled: true,
      format: 'hosts',
      ruleCount: 184291,
      lastUpdated: new Date().toISOString(),
      lastAttemptAt: new Date().toISOString(),
      lastSuccessAt: new Date().toISOString(),
      lastStatus: 'ok',
      lastError: '',
      contentChecksum: 'sha256-mock-stevenblack',
      etag: '"sb-v4"',
      description: 'Consolidated hosts file blocking adware, spam, and telemetry from multiple curated sources.',
      category: 'General / Tracking',
    },
    {
      id: 'filter-3',
      name: 'OISD Basic',
      sourceUrl: 'https://basic.oisd.nl/',
      enabled: false,
      format: 'plain',
      ruleCount: 45120,
      lastUpdated: new Date().toISOString(),
      lastAttemptAt: new Date().toISOString(),
      lastSuccessAt: new Date().toISOString(),
      lastStatus: 'ok',
      lastError: '',
      contentChecksum: 'sha256-mock-oisd',
      etag: '"oisd-basic"',
      description: 'Lightweight, high-reliability list designed to prevent false positives across smart TVs and desktops.',
      category: 'Balanced Privacy',
    },
    {
      id: 'filter-4',
      name: 'URLhaus Malware Hosts',
      sourceUrl: 'https://urlhaus.abuse.ch/downloads/hostfile/',
      enabled: true,
      format: 'hosts',
      ruleCount: 12840,
      lastUpdated: new Date().toISOString(),
      lastAttemptAt: new Date().toISOString(),
      lastSuccessAt: new Date().toISOString(),
      lastStatus: 'ok',
      lastError: '',
      contentChecksum: 'sha256-mock-urlhaus',
      etag: '"urlhaus-v2"',
      description: 'Real-time threat intelligence list blocking verified malware distribution sites and botnet C2s.',
      category: 'Security / Malware',
    },
  ];

  let customRules: models.CustomRule[] = [
    {
      id: 'rule-1',
      domain: 'telemetry.internal-test.local',
      action: 'BLOCK',
      enabled: true,
      createdAt: '2026-10-20 14:30',
      notes: 'Block local diagnostics tracker',
    },
    {
      id: 'rule-2',
      domain: 'nas.home.local',
      action: 'ALLOW',
      enabled: true,
      createdAt: '2026-10-21 09:15',
      notes: 'Bypass filter for local storage appliance',
    },
    {
      id: 'rule-3',
      domain: 'tracking.metrics.vendor.com',
      action: 'BLOCK',
      enabled: true,
      createdAt: '2026-10-22 18:00',
      notes: 'Block third-party metrics collector',
    },
  ];

  let settings: models.AppSettings = {
    upstreamDohUrl: 'https://dns.caliph.dev/dns-query',
    bootstrapDns: '1.1.1.1',
    cacheEnabled: true,
    cacheSize: 4096,
    autoStart: false,
    systemDnsIntegrated: false,
  };

  const sampleLogs: models.QueryLogEntry[] = [
    {
      id: 'log-101',
      timestamp: '21:45:10',
      domain: 'adservice.google.com',
      clientIp: '127.0.0.1',
      queryType: 'A',
      status: 'BLOCKED',
      upstreamLatencyMs: 1,
      filterRule: 'StevenBlack Unified',
      reason: 'Matched advertising blocklist',
    },
    {
      id: 'log-102',
      timestamp: '21:44:55',
      domain: 'github.com',
      clientIp: '127.0.0.1',
      queryType: 'A',
      status: 'ALLOWED',
      upstreamLatencyMs: 24,
      filterRule: 'Passthrough',
      reason: 'Clean domain, resolved via DoH',
    },
    {
      id: 'log-103',
      timestamp: '21:44:30',
      domain: 'telemetry.microsoft.com',
      clientIp: '127.0.0.1',
      queryType: 'AAAA',
      status: 'BLOCKED',
      upstreamLatencyMs: 1,
      filterRule: 'AdAway Default Hosts',
      reason: 'Matched telemetry blocklist',
    },
    {
      id: 'log-104',
      timestamp: '21:44:02',
      domain: 'api.github.com',
      clientIp: '127.0.0.1',
      queryType: 'HTTPS',
      status: 'ALLOWED',
      upstreamLatencyMs: 28,
      filterRule: 'Passthrough',
      reason: 'Clean domain, resolved via DoH',
    },
    {
      id: 'log-105',
      timestamp: '21:43:35',
      domain: 'analytics.tiktok.com',
      clientIp: '127.0.0.1',
      queryType: 'A',
      status: 'BLOCKED',
      upstreamLatencyMs: 1,
      filterRule: 'StevenBlack Unified',
      reason: 'Matched social tracker blocklist',
    },
    {
      id: 'log-106',
      timestamp: '21:43:00',
      domain: 'cdn.jsdelivr.net',
      clientIp: '127.0.0.1',
      queryType: 'A',
      status: 'ALLOWED',
      upstreamLatencyMs: 19,
      filterRule: 'Passthrough',
      reason: 'Clean CDN endpoint',
    },
    {
      id: 'log-107',
      timestamp: '21:42:15',
      domain: 'badware-c2-node.xyz',
      clientIp: '127.0.0.1',
      queryType: 'A',
      status: 'BLOCKED',
      upstreamLatencyMs: 1,
      filterRule: 'URLhaus Malware Hosts',
      reason: 'Matched malware distribution indicator',
    },
    {
      id: 'log-108',
      timestamp: '21:40:50',
      domain: 'unresponsive-internal-dns.invalid',
      clientIp: '127.0.0.1',
      queryType: 'A',
      status: 'FAILED',
      upstreamLatencyMs: 2500,
      filterRule: 'None',
      reason: 'Upstream resolution timeout / NXDOMAIN',
    },
  ];

  w.go = {
    main: {
      App: {
        GetAppInfo: async () => ({
          name: 'BlockAds Desktop',
          version: '0.1.0-alpha',
          platform: 'windows/amd64 (Browser Mode)',
          milestone: 'Milestone 01: Desktop Foundation',
          goVersion: 'go1.27.1',
          webView2Version: '155.0.4283.62',
        }),
        GetProtectionStatus: async () => ({
          isActive: false,
          state: 'INACTIVE_MILESTONE_1',
          upstreamUrl: settings.upstreamDohUrl,
          listenAddr: '127.0.0.1:53 (Standby - Engine in M2)',
          message: 'DNS Engine foundation active. Local network adapter configuration has NOT been altered.',
        }),
        GetSampleQueryLogs: async () => sampleLogs,
        GetFilterLists: async () => filterLists,
        ToggleFilterList: async (id: string, enabled: boolean) => {
          filterLists = filterLists.map((f) => (f.id === id ? { ...f, enabled } : f));
          return filterLists;
        },
        AddFilterSubscription: async (name: string, sourceUrl: string, format: string) => {
          const newSub: models.FilterList = {
            id: `filter-${Date.now()}`,
            name,
            sourceUrl,
            enabled: true,
            format: format || 'hosts',
            ruleCount: 1500,
            lastUpdated: new Date().toISOString(),
            lastAttemptAt: new Date().toISOString(),
            lastSuccessAt: new Date().toISOString(),
            lastStatus: 'ok',
            lastError: '',
            contentChecksum: 'sha256-mock-new',
            etag: '"v1"',
            description: 'Custom added filter subscription',
            category: 'Custom',
          };
          filterLists = [...filterLists, newSub];
          return newSub;
        },
        DeleteFilterSubscription: async (id: string) => {
          filterLists = filterLists.filter((f) => f.id !== id);
        },
        RefreshFilterList: async (id: string) => {
          filterLists = filterLists.map((f) => {
            if (f.id === id) {
              return {
                ...f,
                lastAttemptAt: new Date().toISOString(),
                lastSuccessAt: new Date().toISOString(),
                lastStatus: 'ok',
                lastError: '',
              };
            }
            return f;
          });
          return { subscription_id: id, status: 'updated', status_code: 200, rule_count: 5000 };
        },
        RefreshAllFilterLists: async () => {
          return filterLists.map((f) => ({
            subscription_id: f.id,
            status: 'updated',
            status_code: 200,
            rule_count: f.ruleCount,
          }));
        },
        GetCustomRules: async () => customRules,
        AddCustomRule: async (domain: string, action: string, notes: string) => {
          const cleanDomain = domain.toLowerCase().trim();
          if (customRules.some((r) => r.domain === cleanDomain)) {
            throw new Error(`A rule for domain "${cleanDomain}" already exists`);
          }
          const newRule: models.CustomRule = {
            id: `rule-${Date.now()}`,
            domain: cleanDomain,
            action,
            enabled: true,
            createdAt: new Date().toISOString().replace('T', ' ').substring(0, 16),
            notes,
          };
          customRules = [newRule, ...customRules];
          return newRule;
        },
        DeleteCustomRule: async (id: string) => {
          customRules = customRules.filter((r) => r.id !== id);
        },
        UpdateCustomRule: async (
          id: string,
          action: string,
          notes: string,
          enabled: boolean
        ) => {
          const idx = customRules.findIndex((r) => r.id === id);
          if (idx === -1) throw new Error('Rule not found');
          customRules[idx] = {
            ...customRules[idx],
            action: action.toUpperCase(),
            notes,
            enabled,
          };
        },
        ToggleCustomRule: async (id: string, enabled: boolean) => {
          const idx = customRules.findIndex((r) => r.id === id);
          if (idx === -1) throw new Error('Rule not found');
          customRules[idx] = {
            ...customRules[idx],
            enabled,
          };
        },
        GetSettings: async () => settings,
        UpdateUpstreamDoH: async (url: string) => {
          if (!url.startsWith('https://')) {
            throw new Error("DoH URL must use the 'https://' scheme");
          }
          settings.upstreamDohUrl = url;
        },
        RunDiagnostics: async () => [
          {
            name: 'Loopback Port 53 Status',
            category: 'Networking',
            status: 'PASS',
            details: '127.0.0.1:53 UDP is free and available for local DNS listener.',
          },
          {
            name: 'Upstream DoH Reachability',
            category: 'Encrypted DNS',
            status: 'PASS',
            details: `Connected to ${settings.upstreamDohUrl} successfully (HTTP status: 200). TLS certificate validated.`,
          },
          {
            name: 'Windows Adapter DNS Integration',
            category: 'Operating System',
            status: 'INFO',
            details: 'Adapter configuration is in default system state. In accordance with Milestone 1 safety rules, no network adapters have been modified.',
          },
          {
            name: 'DNS Engine Architectural State',
            category: 'Core Engine',
            status: 'INFO',
            details: 'Desktop Foundation active. Engine separation verified. Ready for Milestone 2 standalone proxy implementation.',
          },
        ],
      },
    },
  };
}
