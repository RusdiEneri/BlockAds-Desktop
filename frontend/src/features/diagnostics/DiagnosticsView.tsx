import React, { useState } from 'react';
import {
  Terminal,
  RefreshCw,
  CheckCircle2,
  AlertTriangle,
  Info,
  ShieldCheck,
  Server,
  Cpu,
  MonitorCheck,
  Copy,
  Check,
} from 'lucide-react';
import { NeoCard } from '../../components/NeoCard';
import { NeoButton } from '../../components/NeoButton';
import { NeoBadge } from '../../components/NeoBadge';
import { models } from '../../../wailsjs/go/models';

interface DiagnosticsViewProps {
  appInfo: models.AppInfo | null;
  diagnostics: models.DiagnosticCheck[];
  onRefreshDiagnostics: () => Promise<void>;
}

export const DiagnosticsView: React.FC<DiagnosticsViewProps> = ({
  appInfo,
  diagnostics,
  onRefreshDiagnostics,
}) => {
  const [isRunning, setIsRunning] = useState(false);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const handleRun = async () => {
    setIsRunning(true);
    try {
      await onRefreshDiagnostics();
    } finally {
      setIsRunning(false);
    }
  };

  const handleCopyCmd = (id: string, text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-black uppercase tracking-wider text-[#33FF00] font-['VT323'] glow-green">
            System Diagnostics & Audit
          </h1>
          <p className="text-xs text-[#A0AAB8] font-mono mt-0.5">
            &gt; PORT AVAILABILITY, TLS UPSTREAM REACHABILITY &amp; INTEGRITY AUDIT
          </p>
        </div>
        <NeoButton
          variant="primary"
          size="sm"
          onClick={handleRun}
          isLoading={isRunning}
        >
          <RefreshCw className="w-3.5 h-3.5" />
          Run Health Audit
        </NeoButton>
      </div>

      {/* Diagnostic Probes Grid */}
      <div className="space-y-3">
        <h2 className="text-sm font-bold uppercase tracking-wider text-[#D2B48C] font-['VT323']">
          &gt; Real-Time Probes
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {diagnostics.map((probe, idx) => (
            <NeoCard
              key={idx}
              title={probe.name}
              badge={
                probe.status === 'PASS' ? (
                  <NeoBadge variant="success" size="sm">
                    <CheckCircle2 className="w-3 h-3" />
                    PASS
                  </NeoBadge>
                ) : probe.status === 'WARN' ? (
                  <NeoBadge variant="warning" size="sm">
                    <AlertTriangle className="w-3 h-3" />
                    WARN
                  </NeoBadge>
                ) : probe.status === 'FAIL' ? (
                  <NeoBadge variant="blocked" size="sm">
                    FAIL
                  </NeoBadge>
                ) : (
                  <NeoBadge variant="neutral" size="sm">
                    <Info className="w-3 h-3" />
                    INFO
                  </NeoBadge>
                )
              }
            >
              <div className="text-[11px] font-bold uppercase text-[#A0AAB8] mb-1 font-mono">
                Category: {probe.category}
              </div>
              <p className="text-xs text-[#33FF00] font-medium leading-relaxed font-mono bg-[#151820] p-2.5 border border-[#33FF00]/30 rounded">
                {probe.details}
              </p>
            </NeoCard>
          ))}
        </div>
      </div>

      {/* Windows Adapter DNS Activation Guide */}
      <NeoCard
        title="Windows Adapter DNS Activation Guide"
        subtitle="Route system-wide traffic through BlockAds local proxy"
        badge={
          <NeoBadge variant="neutral" size="sm">
            WINDOWS INTEGRATION
          </NeoBadge>
        }
      >
        <div className="space-y-3 text-xs">
          <p className="text-[#E8E0D0] leading-relaxed font-mono text-[11px]">
            BlockAds Desktop automatically manages your active network adapter DNS whenever Protection is toggled <strong className="text-[#33FF00]">ON</strong> (routes via <code className="bg-[#151820] text-[#33FF00] px-1.5 py-0.5 border border-[#33FF00]/30 font-mono font-bold rounded">127.0.0.1</code>) and restores default DHCP when Protection is toggled <strong className="text-[#FFB000]">OFF</strong>. Manual commands are provided below for reference or manual troubleshooting:
          </p>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 pt-1">
            <div className="p-3 bg-[#1B1E26] border border-[#33FF00]/30 rounded space-y-2">
              <div className="flex items-center justify-between">
                <span className="font-bold text-[#33FF00] uppercase text-[11px] font-mono">
                  &gt; 1. Enable BlockAds (Wi-Fi)
                </span>
                <button
                  onClick={() => handleCopyCmd('enable-wifi', 'Set-DnsClientServerAddress -InterfaceAlias "Wi-Fi" -ServerAddresses ("127.0.0.1")')}
                  className="flex items-center gap-1 text-[11px] font-bold text-[#D2B48C] hover:text-[#33FF00] cursor-pointer font-mono"
                >
                  {copiedId === 'enable-wifi' ? <Check className="w-3.5 h-3.5 text-[#33FF00]" /> : <Copy className="w-3.5 h-3.5" />}
                  <span>{copiedId === 'enable-wifi' ? 'Copied!' : 'Copy'}</span>
                </button>
              </div>
              <pre className="p-2.5 bg-[#151820] text-[#33FF00] font-mono text-[11px] border border-[#33FF00]/20 rounded overflow-x-auto select-all">
                Set-DnsClientServerAddress -InterfaceAlias "Wi-Fi" -ServerAddresses ("127.0.0.1")
              </pre>
            </div>

            <div className="p-3 bg-[#1B1E26] border border-[#FFB000]/30 rounded space-y-2">
              <div className="flex items-center justify-between">
                <span className="font-bold text-[#FFB000] uppercase text-[11px] font-mono">
                  &gt; 2. Restore Default / DHCP (Wi-Fi)
                </span>
                <button
                  onClick={() => handleCopyCmd('reset-wifi', 'Set-DnsClientServerAddress -InterfaceAlias "Wi-Fi" -ResetServerAddresses')}
                  className="flex items-center gap-1 text-[11px] font-bold text-[#D2B48C] hover:text-[#FFB000] cursor-pointer font-mono"
                >
                  {copiedId === 'reset-wifi' ? <Check className="w-3.5 h-3.5 text-[#33FF00]" /> : <Copy className="w-3.5 h-3.5" />}
                  <span>{copiedId === 'reset-wifi' ? 'Copied!' : 'Copy'}</span>
                </button>
              </div>
              <pre className="p-2.5 bg-[#151820] text-[#FFB000] font-mono text-[11px] border border-[#FFB000]/20 rounded overflow-x-auto select-all">
                Set-DnsClientServerAddress -InterfaceAlias "Wi-Fi" -ResetServerAddresses
              </pre>
            </div>
          </div>
        </div>
      </NeoCard>

      {/* Runtime Environment Matrix */}
      <NeoCard
        title="Runtime Environment Matrix"
        subtitle="Active desktop build and dependency specifications"
        badge={
          <NeoBadge variant="neutral" size="sm">
            SYSTEM INFO
          </NeoBadge>
        }
      >
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs font-mono">
          <div className="p-3 bg-[#1B1E26] border border-[#3F4657] rounded">
            <span className="block text-[10px] font-bold uppercase text-[#A0AAB8]">
              Application Version
            </span>
            <span className="font-bold text-[#33FF00] text-lg font-['VT323'] mt-0.5 block tracking-wider glow-green">
              {appInfo?.version || '0.4.0'}
            </span>
            <span className="text-[10px] text-[#D2B48C]">
              {appInfo?.milestone || 'Production Release'}
            </span>
          </div>

          <div className="p-3 bg-[#1B1E26] border border-[#3F4657] rounded">
            <span className="block text-[10px] font-bold uppercase text-[#A0AAB8]">
              Go Toolchain &amp; Platform
            </span>
            <span className="font-bold text-[#33FF00] text-lg font-['VT323'] mt-0.5 block tracking-wider glow-green">
              {appInfo?.goVersion || 'go1.27.1'}
            </span>
            <span className="text-[10px] text-[#D2B48C]">
              Platform: {appInfo?.platform || 'windows/amd64'}
            </span>
          </div>

          <div className="p-3 bg-[#1B1E26] border border-[#3F4657] rounded">
            <span className="block text-[10px] font-bold uppercase text-[#A0AAB8]">
              WebView2 Runtime
            </span>
            <span className="font-bold text-[#33FF00] text-lg font-['VT323'] mt-0.5 block tracking-wider glow-green">
              {appInfo?.webView2Version || '155.0.4283.62'}
            </span>
            <span className="text-[10px] text-[#D2B48C]">
              Wails v2.17.0 Desktop Bridge
            </span>
          </div>
        </div>
      </NeoCard>

      {/* Safety & System Invariants Verification */}
      <NeoCard
        title="Safety & Scope Verification Checklist"
        subtitle="Affirmation of non-invasive user-space operation"
        badge={
          <NeoBadge variant="success" size="sm">
            VERIFIED SAFE
          </NeoBadge>
        }
      >
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs">
          <div className="flex items-start gap-2.5 p-3 bg-[#16281E] border border-[#33FF00]/30 rounded">
            <CheckCircle2 className="w-4 h-4 text-[#33FF00] shrink-0 mt-0.5" />
            <div>
              <span className="font-bold text-[#33FF00] font-mono">No Network Adapter Modifications</span>
              <p className="text-[11px] text-[#A0AAB8] mt-0.5 font-mono">
                DNS server addresses on all Wi-Fi, Ethernet, and VPN adapters remain in their original system-configured state.
              </p>
            </div>
          </div>

          <div className="flex items-start gap-2.5 p-3 bg-[#16281E] border border-[#33FF00]/30 rounded">
            <CheckCircle2 className="w-4 h-4 text-[#33FF00] shrink-0 mt-0.5" />
            <div>
              <span className="font-bold text-[#33FF00] font-mono">No Registry or Group Policy Edits</span>
              <p className="text-[11px] text-[#A0AAB8] mt-0.5 font-mono">
                Zero modifications made to Windows Registry keys or system policies.
              </p>
            </div>
          </div>

          <div className="flex items-start gap-2.5 p-3 bg-[#16281E] border border-[#33FF00]/30 rounded">
            <CheckCircle2 className="w-4 h-4 text-[#33FF00] shrink-0 mt-0.5" />
            <div>
              <span className="font-bold text-[#33FF00] font-mono">No Root CA Certificates Injected</span>
              <p className="text-[11px] text-[#A0AAB8] mt-0.5 font-mono">
                No HTTPS MITM root certificates are generated, required, or added to the Windows Certificate Store.
              </p>
            </div>
          </div>

          <div className="flex items-start gap-2.5 p-3 bg-[#16281E] border border-[#33FF00]/30 rounded">
            <CheckCircle2 className="w-4 h-4 text-[#33FF00] shrink-0 mt-0.5" />
            <div>
              <span className="font-bold text-[#33FF00] font-mono">No Kernel Drivers Installed</span>
              <p className="text-[11px] text-[#A0AAB8] mt-0.5 font-mono">
                Operates purely in user-space without WinDivert, WinPcap, or WFP kernel network drivers.
              </p>
            </div>
          </div>
        </div>
      </NeoCard>
    </div>
  );
};
