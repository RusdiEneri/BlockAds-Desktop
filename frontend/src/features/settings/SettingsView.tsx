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
        <div className="flex items-center gap-2">
          <span className="text-[#33FF00] font-['VT323',monospace] text-3xl font-bold glow-green">
            {'>'} CONFIGURATION // UPSTREAM & SYSTEM PREFERENCES
          </span>
        </div>
        <p className="text-xs text-[#D2B48C] font-mono mt-0.5">
          Encrypted DNS Transport, Cache Allocation & Windows System Preferences
        </p>
      </div>

      {/* Scope Disclaimer */}
      <div className="bg-[#2B200A] border-2 border-[#5E430D] rounded-lg p-3.5 flex items-start gap-3 shadow-[0_0_12px_rgba(255,176,0,0.15)]">
        <Info className="w-5 h-5 text-[#FFB000] shrink-0 mt-0.5" />
        <div className="text-xs text-[#E8E0D0] font-mono leading-relaxed">
          <span className="font-bold text-[#FFB000]">OPERATIONAL BOUNDARY DISCIPLINE: </span>
          Only upstream DoH parameters are actively validated by the Go backend in this release. System adapter hooks and daemon autostart switches are disabled and clearly tagged with their planned milestones to guarantee zero unintended network mutations.
        </div>
      </div>

      {/* Upstream DoH Section (Active) */}
      <NeoCard
        title="ENCRYPTED DNS UPSTREAM (DOH)"
        subtitle="RFC 8484 DNS-over-HTTPS resolver endpoint"
        badge={
          <NeoBadge variant="success" size="sm">
            ACTIVE CONTRACT
          </NeoBadge>
        }
      >
        <form onSubmit={handleSaveUpstream} className="space-y-4">
          <div>
            <label htmlFor="doh-endpoint-input" className="block text-xs font-mono font-bold uppercase tracking-wider text-[#D2B48C] mb-1">
              DoH Endpoint URL <span className="text-[#FF5252]">*</span>
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
                className="flex-1 text-xs font-mono p-2.5 rounded-lg border-2 border-[#373E4F] bg-[#14161E] text-[#33FF00] focus:outline-none focus:border-[#33FF00]"
                required
              />
              <NeoButton
                type="submit"
                variant="primary"
                size="sm"
                isLoading={isSaving}
              >
                SAVE UPSTREAM
              </NeoButton>
            </div>
            <p className="text-[11px] text-[#A0AAB8] font-mono mt-1.5">
              Must use encrypted HTTPS scheme and specify a path (typically <code className="font-mono text-[#33FF00]">/dns-query</code>).
            </p>
          </div>

          {statusMessage && (
            <div
              className={`p-3 rounded-lg border-2 text-xs flex items-center gap-2 font-mono ${
                statusMessage.type === 'success'
                  ? 'bg-[#102B15] text-[#33FF00] border-[#1E5C2A]'
                  : 'bg-[#2D1212] text-[#FF5252] border-[#612323]'
              }`}
            >
              {statusMessage.type === 'success' ? (
                <CheckCircle2 className="w-4 h-4 shrink-0 text-[#33FF00]" />
              ) : (
                <AlertCircle className="w-4 h-4 shrink-0 text-[#FF5252]" />
              )}
              <span>{statusMessage.text}</span>
            </div>
          )}

          {/* Upstream Presets */}
          <div className="pt-2 border-t-2 border-[#2B313E]">
            <div className="text-[11px] font-mono font-bold uppercase tracking-wider text-[#D2B48C] mb-2">
              Tested Public Resolvers:
            </div>
            <div className="flex flex-wrap gap-2">
              {presets.map((preset) => (
                <button
                  key={preset.name}
                  type="button"
                  onClick={() => setUpstreamUrl(preset.url)}
                  className={`px-3 py-1 text-xs font-mono font-medium rounded-lg border transition-all cursor-pointer ${
                    upstreamUrl === preset.url
                      ? 'bg-[#33FF00] text-[#14161E] border-[#29CC00] shadow-[0_0_8px_rgba(51,255,0,0.3)]'
                      : 'bg-[#181B23] text-[#A0AAB8] border-[#3F4657] hover:text-[#E8E0D0] hover:border-[#D2B48C]'
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
          <div className="bg-[#151820] p-3 border border-[#D2B48C]/30 rounded text-xs text-[#D2B48C] font-mono flex items-start gap-2.5">
            <Info className="w-4 h-4 text-[#FFB000] shrink-0 mt-0.5" />
            <div className="leading-relaxed">
              <span className="font-bold text-[#33FF00]">&gt; DNS-Only Policy Enforcement: </span>
              SafeSearch rewrites DNS queries for supported search and video hostnames to their official canonical safe search targets (e.g.{' '}
              <code className="font-mono text-[#33FF00] bg-[#1B1E26] px-1 border border-[#33FF00]/30 rounded">forcesafesearch.google.com</code>).
              It strictly affects DNS resolution and does NOT decrypt, inspect, or intercept HTTPS content, nor does it replace operating system, account, or browser-level controls.
            </div>
          </div>

          {safeSearchError && (
            <div className="p-3 bg-[#2D1212] border-2 border-[#802020] text-xs text-[#FF4D4D] rounded-lg flex items-center gap-2 font-mono">
              <AlertCircle className="w-4 h-4 shrink-0 text-[#FF4D4D]" />
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
                  className={`p-3.5 border rounded transition-all ${
                    isUnsupported
                      ? 'bg-[#151820] border-[#3F4657]/30 opacity-75'
                      : isActive
                      ? 'bg-[#1B1E26] border-[#33FF00]/40 shadow-[0_0_10px_rgba(51,255,0,0.15)]'
                      : 'bg-[#1B1E26] border-[#3F4657]'
                  }`}
                >
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 font-mono">
                    <div className="space-y-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-bold text-xs uppercase tracking-wider text-[#E8E0D0]">
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
                        <div className="text-xs text-[#A0AAB8] flex flex-wrap items-center gap-2">
                          <span className="font-bold text-[11px] text-[#D2B48C]">Target:</span>
                          <code className="font-mono text-xs text-[#33FF00] bg-[#151820] px-1.5 py-0.5 border border-[#33FF00]/30 rounded">
                            {provider.target}
                          </code>
                          <span className="text-[#A0AAB8]/50">|</span>
                          <span className="text-[11px] text-[#A0AAB8]">
                            {provider.domains?.length || 0} hostnames covered
                          </span>
                        </div>
                      ) : (
                        <div className="text-xs text-[#FFB000] font-mono">
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
                          className="text-xs font-semibold text-[#D2B48C] hover:text-[#33FF00] flex items-center gap-1 font-mono"
                        >
                          <span>Docs</span>
                          <ExternalLink className="w-3 h-3" />
                        </a>
                      )}

                      <button
                        type="button"
                        disabled={isUnsupported || togglingProvider === provider.id}
                        onClick={() => handleToggleProvider(provider.id, !provider.enabled)}
                        className={`relative inline-flex h-5 w-10 shrink-0 cursor-pointer border rounded-full transition-colors duration-200 ease-in-out focus-visible:outline-2 focus-visible:outline-[#33FF00] ${
                          isUnsupported
                            ? 'bg-[#151820] border-[#3F4657]/30 opacity-50 cursor-not-allowed'
                            : provider.enabled
                            ? 'bg-[#33FF00] border-[#33FF00]'
                            : 'bg-[#151820] border-[#3F4657]'
                        }`}
                        aria-pressed={provider.enabled}
                        aria-label={`Toggle SafeSearch for ${provider.name}`}
                      >
                        <span
                          className={`pointer-events-none inline-block h-3.5 w-3.5 transform rounded-full transition duration-200 ease-in-out ${
                            provider.enabled ? 'translate-x-5 bg-[#14161E]' : 'translate-x-0.5 bg-[#D2B48C]'
                          } mt-0.5`}
                        />
                      </button>
                    </div>
                  </div>

                  {/* Diagnostic Warning Note */}
                  {provider.warning && isUnsupported && (
                    <div className="mt-2.5 text-[11px] text-[#FFB000] bg-[#2A2318] p-2.5 border border-[#FFB000]/30 rounded font-mono leading-relaxed">
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
        <div className="space-y-4 opacity-75 font-mono">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label htmlFor="cache-size-input" className="block text-xs font-bold uppercase tracking-wider text-[#D2B48C] mb-1">
                &gt; In-Memory Cache Size
              </label>
              <input
                id="cache-size-input"
                name="cacheSize"
                aria-label="In-Memory Cache Size"
                type="text"
                value="4,096 entries (LRU)"
                disabled
                className="w-full text-xs font-mono p-2 border border-[#3F4657] bg-[#151820] text-[#A0AAB8] rounded cursor-not-allowed"
              />
              <span className="text-[10px] text-[#A0AAB8] mt-0.5 block">
                Caches positive and negative DNS replies within TTL limits.
              </span>
            </div>

            <div>
              <label htmlFor="bootstrap-dns-input" className="block text-xs font-bold uppercase tracking-wider text-[#D2B48C] mb-1">
                &gt; Bootstrap DNS Resolver
              </label>
              <input
                id="bootstrap-dns-input"
                name="bootstrapDns"
                aria-label="Bootstrap DNS Resolver"
                type="text"
                value={
                  settings?.bootstrapDns
                    ? `${settings.bootstrapDns} (${
                        settings.bootstrapDns === '8.8.8.8'
                          ? 'Google'
                          : settings.bootstrapDns === '1.1.1.1'
                          ? 'Cloudflare'
                          : settings.bootstrapDns === '9.9.9.9'
                          ? 'Quad9'
                          : 'Anycast'
                      } Anycast)`
                    : '8.8.8.8 (Google Anycast)'
                }
                disabled
                className="w-full text-xs font-mono p-2 border border-[#3F4657] bg-[#151820] text-[#A0AAB8] rounded cursor-not-allowed"
              />
              <span className="text-[10px] text-[#A0AAB8] mt-0.5 block">
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
        <div className="space-y-4 font-mono">
          <div className="flex items-center justify-between p-3 bg-[#1B1E26] border border-[#3F4657] rounded">
            <div className="space-y-0.5">
              <div className="text-xs font-bold text-[#E8E0D0]">
                &gt; Start BlockAds Desktop on User Login
              </div>
              <p className="text-[11px] text-[#A0AAB8]">
                Registers application launch under Current User Run key (HKCU). Does not require administrator rights.
              </p>
              {startupStatus?.message && (
                <p className="text-[10px] text-[#33FF00] font-mono mt-1">
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
              <div className="w-8 h-4 bg-[#151820] border border-[#3F4657] rounded-full peer-checked:bg-[#33FF00] transition-colors relative">
                <div
                  className={`w-3 h-3 bg-[#D2B48C] peer-checked:bg-[#14161E] rounded-full transition-transform duration-75 absolute top-0.5 left-0.5 ${
                    startupStatus?.enabled ? 'translate-x-3.5' : 'translate-x-0'
                  }`}
                />
              </div>
            </label>
          </div>

          {startupError && (
            <div className="p-2.5 bg-[#2D1212] border-2 border-[#802020] rounded-lg text-xs text-[#FF4D4D] flex items-center gap-2 font-mono">
              <AlertCircle className="w-4 h-4 shrink-0 text-[#FF4D4D]" />
              <span>{startupError}</span>
            </div>
          )}

          <div className="p-3 bg-[#151820] border border-[#33FF00]/30 rounded text-xs text-[#D2B48C] leading-relaxed">
            <span className="font-bold text-[#33FF00]">&gt; Scope Disclaimer: </span>
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
        <div className="space-y-3 font-mono">
          <div>
            <label htmlFor="data-dir-display" className="block text-xs font-bold uppercase tracking-wider text-[#D2B48C] mb-1">
              &gt; Active Data Directory
            </label>
            <input
              id="data-dir-display"
              type="text"
              readOnly
              value={appInfo?.dataDir || 'Resolving data directory...'}
              className="w-full text-xs font-mono p-2 border border-[#33FF00]/30 bg-[#151820] text-[#33FF00] rounded"
            />
            <span className="text-[10px] text-[#A0AAB8] mt-1 block">
              Contains local SQLite database (<code>blockads.db</code>), WAL logs, filter snapshots, and SafeSearch settings.
            </span>
          </div>

          <div className="text-xs text-[#E8E0D0] leading-relaxed p-3 bg-[#1B1E26] border border-[#3F4657] rounded">
            {appInfo?.isPortable ? (
              <span>
                <strong className="text-[#33FF00]">Portable Mode Active:</strong> All user state is isolated inside this folder's <code>data/</code> subdirectory. You can safely copy or move this entire directory to any writable location or USB drive.
              </span>
            ) : (
              <span>
                <strong className="text-[#33FF00]">Installed Mode Active:</strong> Persistent data is safely stored under your Windows user profile (<code>%LOCALAPPDATA%\BlockAds Desktop</code>). Application updates and uninstallation preserve your database and rules by default.
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
        <div className="space-y-3 opacity-75 font-mono">
          <div className="flex items-center justify-between p-2.5 bg-[#1B1E26] border border-[#3F4657] rounded">
            <div>
              <div className="text-xs font-bold text-[#E8E0D0]">
                &gt; Automatic Network Adapter DNS Redirection
              </div>
              <div className="text-[11px] text-[#A0AAB8]">
                Points active Windows network adapters to 127.0.0.1 on start with crash rollback.
              </div>
            </div>
            <NeoBadge variant="neutral" size="sm">
              MILESTONE 5
            </NeoBadge>
          </div>

          <div className="flex items-center justify-between p-2.5 bg-[#1B1E26] border border-[#3F4657] rounded">
            <div>
              <div className="text-xs font-bold text-[#E8E0D0]">
                &gt; Windows Service Integration
              </div>
              <div className="text-[11px] text-[#A0AAB8]">
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
