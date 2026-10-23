import React, { useState, useEffect } from 'react';
import {
  Settings,
  Server,
  Zap,
  Cpu,
  Monitor,
  CheckCircle2,
  AlertCircle,
  Info,
  ExternalLink,
  ShieldCheck,
} from 'lucide-react';
import { NeoCard } from '../../components/NeoCard';
import { NeoButton } from '../../components/NeoButton';
import { NeoBadge } from '../../components/NeoBadge';
import { models } from '../../../wailsjs/go/models';

interface SettingsViewProps {
  settings: models.AppSettings | null;
  safeSearchSettings?: models.SafeSearchProviderSetting[];
  startupStatus?: models.StartupStatus | null;
  appInfo?: models.AppInfo | null;
  onUpdateUpstream: (url: string) => Promise<void>;
  onToggleSafeSearch?: (providerId: string, enabled: boolean) => Promise<void>;
  onToggleStartup?: (enabled: boolean) => Promise<void>;
}

export const SettingsView: React.FC<SettingsViewProps> = ({
  settings,
  safeSearchSettings = [],
  startupStatus,
  appInfo,
  onUpdateUpstream,
  onToggleSafeSearch,
  onToggleStartup,
}) => {
  const [upstreamUrl, setUpstreamUrl] = useState(
    settings?.upstreamDohUrl || 'https://dns.caliph.dev/dns-query'
  );

  useEffect(() => {
    if (settings?.upstreamDohUrl) {
      setUpstreamUrl(settings.upstreamDohUrl);
    }
  }, [settings?.upstreamDohUrl]);

  const [statusMessage, setStatusMessage] = useState<{
    type: 'success' | 'error';
    text: string;
  } | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [togglingProvider, setTogglingProvider] = useState<string | null>(null);
  const [safeSearchError, setSafeSearchError] = useState<string | null>(null);
  const [isTogglingStartup, setIsTogglingStartup] = useState(false);
  const [startupError, setStartupError] = useState<string | null>(null);

  const handleToggleStartup = async (enabled: boolean) => {
    if (!onToggleStartup) return;
    setStartupError(null);
    setIsTogglingStartup(true);
    try {
      await onToggleStartup(enabled);
    } catch (err: any) {
      setStartupError(err?.message || 'Failed to update Windows startup registration');
    } finally {
      setIsTogglingStartup(false);
    }
  };

  const presets = [
    {
      name: 'Caliph DNS (Default)',
      url: 'https://dns.caliph.dev/dns-query',
    },
    {
      name: 'Cloudflare',
      url: 'https://cloudflare-dns.com/dns-query',
    },
    {
      name: 'Google DNS',
      url: 'https://dns.google/dns-query',
    },
    {
      name: 'Quad9 Security',
      url: 'https://dns.quad9.net/dns-query',
    },
  ];

  const handleSaveUpstream = async (e: React.FormEvent) => {
    e.preventDefault();
    setStatusMessage(null);
    setIsSaving(true);

    try {
      await onUpdateUpstream(upstreamUrl.trim());
      setStatusMessage({
        type: 'success',
        text: 'Upstream DoH URL updated and validated successfully.',
      });
    } catch (err: any) {
      setStatusMessage({
        type: 'error',
        text: err?.message || String(err),
      });
    } finally {
      setIsSaving(false);
    }
  };

  const handleToggleProvider = async (providerId: string, enabled: boolean) => {
    if (!onToggleSafeSearch) return;
    setSafeSearchError(null);
    setTogglingProvider(providerId);

    try {
      await onToggleSafeSearch(providerId, enabled);
    } catch (err: any) {
      setSafeSearchError(err?.message || String(err));
    } finally {
      setTogglingProvider(null);
    }
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-xl font-black uppercase tracking-wider text-[#121212]">
          Settings & Upstream Configuration
        </h1>
        <p className="text-xs text-[#4B5563] mt-0.5">
          Encrypted DNS Transport, Cache Allocation & Windows System Preferences
        </p>
      </div>

      {/* Scope Disclaimer */}
      <div className="bg-[#FEF3C7] neo-border p-3.5 flex items-start gap-3">
        <Info className="w-5 h-5 text-[#92400E] shrink-0 mt-0.5" />
        <div className="text-xs text-[#92400E]">
          <span className="font-bold">Milestone 1 Scope Discipline: </span>
          Only upstream DoH parameters are actively validated by the Go backend in this release. System adapter hooks and daemon autostart switches are disabled and clearly tagged with their planned milestones to guarantee zero unintended network mutations.
        </div>
      </div>

      {/* Upstream DoH Section (Active) */}
      <NeoCard
        title="Encrypted DNS Upstream (DoH)"
        subtitle="RFC 8484 DNS-over-HTTPS resolver endpoint"
        badge={
          <NeoBadge variant="success" size="sm">
            ACTIVE CONTRACT
          </NeoBadge>
        }
      >
        <form onSubmit={handleSaveUpstream} className="space-y-4">
          <div>
            <label htmlFor="doh-endpoint-input" className="block text-xs font-bold uppercase tracking-wider text-[#121212] mb-1">
              DoH Endpoint URL <span className="text-[#D9381E]">*</span>
            </label>
            <div className="flex items-center gap-2">
              <input
                id="doh-endpoint-input"
                name="upstreamDohUrl"
                aria-label="DoH Endpoint URL"
                type="text"
                value={upstreamUrl}
                onChange={(e) => setUpstreamUrl(e.target.value)}
                placeholder="https://dns.caliph.dev/dns-query"
                className="flex-1 text-xs font-mono p-2.5 border-2 border-[#121212] focus:outline-none focus:shadow-[2px_2px_0px_#B38B6D]"
                required
              />
              <NeoButton
                type="submit"
                variant="primary"
                size="sm"
                isLoading={isSaving}
              >
                Save Upstream
              </NeoButton>
            </div>
            <p className="text-[11px] text-[#6B7280] mt-1.5">
              Must use encrypted HTTPS scheme and specify a path (typically <code className="font-mono">/dns-query</code>).
            </p>
          </div>

          {statusMessage && (
            <div
              className={`p-3 border-2 text-xs flex items-center gap-2 font-medium ${
                statusMessage.type === 'success'
                  ? 'bg-[#EBFBEE] text-[#1E7E34] border-[#1E7E34]'
                  : 'bg-[#FEECEB] text-[#D9381E] border-[#D9381E]'
              }`}
            >
              {statusMessage.type === 'success' ? (
                <CheckCircle2 className="w-4 h-4 shrink-0" />
              ) : (
                <AlertCircle className="w-4 h-4 shrink-0" />
              )}
              <span>{statusMessage.text}</span>
            </div>
          )}

          {/* Upstream Presets */}
          <div className="pt-2 border-t border-[#EBE7DE]">
            <div className="text-[11px] font-bold uppercase tracking-wider text-[#6B7280] mb-2">
              Tested Public Resolvers:
            </div>
            <div className="flex flex-wrap gap-2">
              {presets.map((preset) => (
                <button
                  key={preset.name}
                  type="button"
                  onClick={() => setUpstreamUrl(preset.url)}
                  className={`px-2.5 py-1 text-xs font-medium border-2 transition-all ${
                    upstreamUrl === preset.url
                      ? 'bg-[#B38B6D] text-white border-[#121212] shadow-[1px_1px_0px_#121212]'
                      : 'bg-white text-[#121212] border-[#121212] hover:bg-[#F5F1E8]'
                  }`}
                >
                  {preset.name}
                </button>
              ))}
            </div>
          </div>
        </form>
      </NeoCard>

      {/* SafeSearch DNS Policies (Active Contract - Milestone 04B.4A) */}
      <NeoCard
        title="SafeSearch DNS Enforcement"
        subtitle="Canonical CNAME policy rewriting for verified search and video providers"
        badge={
          <NeoBadge variant="success" size="sm">
            ACTIVE CONTRACT
          </NeoBadge>
        }
      >
        <div className="space-y-4">
          {/* Scope notice */}
          <div className="bg-[#F5F1E8] p-3 border border-[#D1CBC0] text-xs text-[#4B5563] flex items-start gap-2.5">
            <Info className="w-4 h-4 text-[#B38B6D] shrink-0 mt-0.5" />
            <div className="leading-relaxed">
              <span className="font-bold text-[#121212]">DNS-Only Policy Enforcement: </span>
              SafeSearch rewrites DNS queries for supported search and video hostnames to their official canonical safe search targets (e.g.{' '}
              <code className="font-mono text-[#121212] bg-white px-1 border border-[#D1CBC0]">forcesafesearch.google.com</code>).
              It strictly affects DNS resolution and does NOT decrypt, inspect, or intercept HTTPS content, nor does it replace operating system, account, or browser-level controls.
            </div>
          </div>

          {safeSearchError && (
            <div className="p-3 bg-[#FEECEB] border-2 border-[#D9381E] text-xs text-[#D9381E] flex items-center gap-2 font-medium">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{safeSearchError}</span>
            </div>
          )}

          {/* Providers List */}
          <div className="grid grid-cols-1 gap-3">
            {safeSearchSettings.map((provider) => {
              const isUnsupported = provider.status === 'UNSUPPORTED';
              const isActive = provider.status === 'ACTIVE';
              const isPending = provider.status === 'PENDING';

              return (
                <div
                  key={provider.id}
                  className={`p-3.5 border-2 transition-all ${
                    isUnsupported
                      ? 'bg-[#F9F7F4] border-[#D1CBC0] opacity-85'
                      : isActive
                      ? 'bg-white border-[#121212] shadow-[2px_2px_0px_#1E7E34]'
                      : 'bg-white border-[#D1CBC0]'
                  }`}
                >
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                    <div className="space-y-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-black text-xs uppercase tracking-wider text-[#121212]">
                          {provider.name}
                        </span>
                        {isActive && (
                          <NeoBadge variant="success" size="sm">
                            <CheckCircle2 className="w-3 h-3" /> ACTIVE
                          </NeoBadge>
                        )}
                        {provider.status === 'DISABLED' && (
                          <NeoBadge variant="neutral" size="sm">
                            DISABLED
                          </NeoBadge>
                        )}
                        {isPending && (
                          <NeoBadge variant="accent" size="sm">
                            PENDING ACTIVATION
                          </NeoBadge>
                        )}
                        {isUnsupported && (
                          <NeoBadge variant="warning" size="sm">
                            <AlertCircle className="w-3 h-3" /> UNSUPPORTED ON THIS MECHANISM
                          </NeoBadge>
                        )}
                      </div>

                      {provider.target ? (
                        <div className="text-xs text-[#4B5563] flex flex-wrap items-center gap-2">
                          <span className="font-bold text-[11px] text-[#6B7280]">Target:</span>
                          <code className="font-mono text-xs text-[#121212] bg-[#EBE7DE] px-1.5 py-0.5 border border-[#D1CBC0]">
                            {provider.target}
                          </code>
                          <span className="text-[#9CA3AF]">|</span>
                          <span className="text-[11px] text-[#6B7280]">
                            {provider.domains?.length || 0} hostnames covered
                          </span>
                        </div>
                      ) : (
                        <div className="text-xs text-[#92400E] font-medium">
                          {provider.warning || 'Provider does not support canonical DNS CNAME rewriting.'}
                        </div>
                      )}
                    </div>

                    <div className="flex items-center gap-3 shrink-0">
                      {provider.docUrl && (
                        <a
                          href={provider.docUrl}
                          target="_blank"
                          rel="noopener noreferrer"
                          title="View official technical documentation"
                          className="text-xs font-semibold text-[#B38B6D] hover:text-[#121212] flex items-center gap-1 border-b border-[#B38B6D]"
                        >
                          <span>Docs</span>
                          <ExternalLink className="w-3 h-3" />
                        </a>
                      )}

                      <button
                        type="button"
                        disabled={isUnsupported || togglingProvider === provider.id}
                        onClick={() => handleToggleProvider(provider.id, !provider.enabled)}
                        className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer border-2 border-[#121212] transition-colors duration-200 ease-in-out focus:outline-none ${
                          isUnsupported
                            ? 'bg-[#E5E0D8] cursor-not-allowed opacity-60'
                            : provider.enabled
                            ? 'bg-[#1E7E34]'
                            : 'bg-[#D1CBC0]'
                        }`}
                        aria-pressed={provider.enabled}
                        aria-label={`Toggle SafeSearch for ${provider.name}`}
                      >
                        <span
                          className={`pointer-events-none inline-block h-4 w-4 transform border border-[#121212] bg-white shadow-sm transition duration-200 ease-in-out ${
                            provider.enabled ? 'translate-x-5' : 'translate-x-0.5'
                          } mt-0.5`}
                        />
                      </button>
                    </div>
                  </div>

                  {/* Diagnostic Warning Note */}
                  {provider.warning && isUnsupported && (
                    <div className="mt-2.5 text-[11px] text-[#92400E] bg-[#FEF3C7] p-2.5 border border-[#D1CBC0] leading-relaxed">
                      <span className="font-bold">Provider Architecture Limitation: </span>
                      {provider.warning}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      </NeoCard>

      {/* Engine & Performance Controls (Planned) */}
      <NeoCard
        title="Performance & In-Memory Cache"
        subtitle="Resolver caching and bootstrap resolution settings"
        badge={
          <NeoBadge variant="neutral" size="sm">
            PLANNED (MILESTONE 3)
          </NeoBadge>
        }
      >
        <div className="space-y-4 opacity-70">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label htmlFor="cache-size-input" className="block text-xs font-bold uppercase tracking-wider text-[#121212] mb-1">
                In-Memory Cache Size
              </label>
              <input
                id="cache-size-input"
                name="cacheSize"
                aria-label="In-Memory Cache Size"
                type="text"
                value="4,096 entries (LRU)"
                disabled
                className="w-full text-xs font-mono p-2 border-2 border-[#D1CBC0] bg-[#EBE7DE] cursor-not-allowed"
              />
              <span className="text-[10px] text-[#6B7280] mt-0.5 block">
                Caches positive and negative DNS replies within TTL limits.
              </span>
            </div>

            <div>
              <label htmlFor="bootstrap-dns-input" className="block text-xs font-bold uppercase tracking-wider text-[#121212] mb-1">
                Bootstrap DNS Resolver
              </label>
              <input
                id="bootstrap-dns-input"
                name="bootstrapDns"
                aria-label="Bootstrap DNS Resolver"
                type="text"
                value="1.1.1.1 (Cloudflare Anycast)"
                disabled
                className="w-full text-xs font-mono p-2 border-2 border-[#D1CBC0] bg-[#EBE7DE] cursor-not-allowed"
              />
              <span className="text-[10px] text-[#6B7280] mt-0.5 block">
                Used strictly to resolve upstream DoH hostname without causing circular DNS loops.
              </span>
            </div>
          </div>
        </div>
      </NeoCard>

      {/* Start with Windows Settings */}
      <NeoCard
        title="Windows Login Startup"
        subtitle="User-level login launch configuration"
        badge={
          startupStatus?.enabled ? (
            <NeoBadge variant="success" size="sm">
              <CheckCircle2 className="w-3 h-3" />
              ENABLED
            </NeoBadge>
          ) : (
            <NeoBadge variant="neutral" size="sm">
              DISABLED
            </NeoBadge>
          )
        }
      >
        <div className="space-y-4">
          <div className="flex items-center justify-between p-3 bg-white neo-border">
            <div className="space-y-0.5">
              <div className="text-xs font-bold text-[#121212]">
                Start BlockAds Desktop on User Login
              </div>
              <p className="text-[11px] text-[#6B7280]">
                Registers application launch under Current User Run key (HKCU). Does not require administrator rights.
              </p>
              {startupStatus?.message && (
                <p className="text-[10px] text-[#1B4F72] font-mono mt-1">
                  {startupStatus.message}
                </p>
              )}
            </div>

            <label htmlFor="toggle-startup-windows" className="relative inline-flex items-center cursor-pointer ml-3">
              <input
                id="toggle-startup-windows"
                aria-label="Start BlockAds Desktop on User Login"
                type="checkbox"
                checked={!!startupStatus?.enabled}
                disabled={isTogglingStartup || !onToggleStartup}
                onChange={(e) => handleToggleStartup(e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-[#EBE7DE] border-2 border-[#121212] peer-checked:bg-[#1E7E34] transition-colors relative">
                <div
                  className={`w-4 h-4 bg-white border border-[#121212] transition-transform duration-75 absolute top-0.5 left-0.5 ${
                    startupStatus?.enabled ? 'translate-x-5' : 'translate-x-0'
                  }`}
                />
              </div>
            </label>
          </div>

          {startupError && (
            <div className="p-2.5 bg-[#FEECEB] border border-[#D9381E] text-xs text-[#D9381E] flex items-center gap-2">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{startupError}</span>
            </div>
          )}

          <div className="p-3 bg-[#EBF5FB] neo-border text-xs text-[#1B4F72] leading-relaxed">
            <span className="font-bold">Scope Disclaimer: </span>
            Start with Windows launches the desktop application interface. Full system-wide DNS adapter interception and background Windows Service mode will be delivered in Milestones 05 and 06.
          </div>
        </div>
      </NeoCard>

      {/* Application Data & Portable Mode Diagnostics */}
      <NeoCard
        title="Application Data & Storage Isolation"
        subtitle="Active SQLite directory and portable environment contract"
        badge={
          appInfo?.isPortable ? (
            <NeoBadge variant="accent" size="sm">
              PORTABLE MODE
            </NeoBadge>
          ) : (
            <NeoBadge variant="neutral" size="sm">
              INSTALLED MODE
            </NeoBadge>
          )
        }
      >
        <div className="space-y-3">
          <div>
            <label htmlFor="data-dir-display" className="block text-xs font-bold uppercase tracking-wider text-[#121212] mb-1">
              Active Data Directory
            </label>
            <input
              id="data-dir-display"
              type="text"
              readOnly
              value={appInfo?.dataDir || 'Resolving data directory...'}
              className="w-full text-xs font-mono p-2 border-2 border-[#121212] bg-[#F5F1E8]"
            />
            <span className="text-[10px] text-[#6B7280] mt-1 block">
              Contains local SQLite database (<code>blockads.db</code>), WAL logs, filter snapshots, and SafeSearch settings.
            </span>
          </div>

          <div className="text-xs text-[#4B5563] leading-relaxed p-3 bg-white neo-border">
            {appInfo?.isPortable ? (
              <span>
                <strong>Portable Mode Active:</strong> All user state is isolated inside this folder's <code>data/</code> subdirectory. You can safely copy or move this entire directory to any writable location or USB drive.
              </span>
            ) : (
              <span>
                <strong>Installed Mode Active:</strong> Persistent data is safely stored under your Windows user profile (<code>%LOCALAPPDATA%\BlockAds Desktop</code>). Application updates and uninstallation preserve your database and rules by default.
              </span>
            )}
          </div>
        </div>
      </NeoCard>

      {/* Windows System Integration (Planned Milestones) */}
      <NeoCard
        title="Windows System Integration & Daemon"
        subtitle="Network adapter configuration and background service"
        badge={
          <NeoBadge variant="neutral" size="sm">
            PLANNED (MILESTONES 5 & 6)
          </NeoBadge>
        }
      >
        <div className="space-y-3 opacity-70">
          <div className="flex items-center justify-between p-2.5 bg-[#F5F1E8] border border-[#D1CBC0]">
            <div>
              <div className="text-xs font-bold text-[#121212]">
                Automatic Network Adapter DNS Redirection
              </div>
              <div className="text-[11px] text-[#6B7280]">
                Points active Windows network adapters to 127.0.0.1 on start with crash rollback.
              </div>
            </div>
            <NeoBadge variant="neutral" size="sm">
              MILESTONE 5
            </NeoBadge>
          </div>

          <div className="flex items-center justify-between p-2.5 bg-[#F5F1E8] border border-[#D1CBC0]">
            <div>
              <div className="text-xs font-bold text-[#121212]">
                Windows Service Integration
              </div>
              <div className="text-[11px] text-[#6B7280]">
                Runs the Go DNS engine as a background service communicating with UI via Named Pipes.
              </div>
            </div>
            <NeoBadge variant="neutral" size="sm">
              MILESTONE 6
            </NeoBadge>
          </div>
        </div>
      </NeoCard>
    </div>
  );
};
