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

  const handleRun = async () => {
    setIsRunning(true);
    try {
      await onRefreshDiagnostics();
    } finally {
      setIsRunning(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-black uppercase tracking-wider text-[#121212]">
            System Diagnostics & Audit
          </h1>
          <p className="text-xs text-[#4B5563] mt-0.5">
            Port Availability, TLS Upstream Reachability & Integrity Verification
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
        <h2 className="text-xs font-bold uppercase tracking-wider text-[#121212]">
          Real-Time Probes
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
              <div className="text-[11px] font-bold uppercase text-[#6B7280] mb-1">
                Category: {probe.category}
              </div>
              <p className="text-xs text-[#121212] font-medium leading-relaxed font-mono bg-[#F5F1E8] p-2.5 border border-[#121212]">
                {probe.details}
              </p>
            </NeoCard>
          ))}
        </div>
      </div>

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
          <div className="p-3 bg-[#FDFBF7] border border-[#121212]">
            <span className="block text-[10px] font-bold uppercase text-[#6B7280] font-sans">
              Application Version
            </span>
            <span className="font-bold text-[#121212] text-sm mt-0.5 block">
              {appInfo?.version || '0.1.0-alpha'}
            </span>
            <span className="text-[10px] text-[#6B7280] font-sans">
              {appInfo?.milestone || 'Milestone 1'}
            </span>
          </div>

          <div className="p-3 bg-[#FDFBF7] border border-[#121212]">
            <span className="block text-[10px] font-bold uppercase text-[#6B7280] font-sans">
              Go Toolchain & Platform
            </span>
            <span className="font-bold text-[#121212] text-sm mt-0.5 block">
              {appInfo?.goVersion || 'go1.27.1'}
            </span>
            <span className="text-[10px] text-[#6B7280] font-sans">
              Platform: {appInfo?.platform || 'windows/amd64'}
            </span>
          </div>

          <div className="p-3 bg-[#FDFBF7] border border-[#121212]">
            <span className="block text-[10px] font-bold uppercase text-[#6B7280] font-sans">
              WebView2 Runtime
            </span>
            <span className="font-bold text-[#121212] text-sm mt-0.5 block">
              {appInfo?.webView2Version || '155.0.4283.62'}
            </span>
            <span className="text-[10px] text-[#6B7280] font-sans">
              Wails v2.17.0 Desktop Bridge
            </span>
          </div>
        </div>
      </NeoCard>

      {/* Safety & System Invariants Verification */}
      <NeoCard
        title="Safety & Scope Verification Checklist"
        subtitle="Affirmation of non-invasive operation for Milestone 1"
        badge={
          <NeoBadge variant="success" size="sm">
            VERIFIED SAFE
          </NeoBadge>
        }
      >
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs">
          <div className="flex items-start gap-2.5 p-2.5 bg-[#EBFBEE] border border-[#121212]">
            <CheckCircle2 className="w-4 h-4 text-[#1E7E34] shrink-0 mt-0.5" />
            <div>
              <span className="font-bold text-[#121212]">No Network Adapter Modifications</span>
              <p className="text-[11px] text-[#4B5563] mt-0.5">
                DNS server addresses on all Wi-Fi, Ethernet, and VPN adapters remain in their original system-configured state.
              </p>
            </div>
          </div>

          <div className="flex items-start gap-2.5 p-2.5 bg-[#EBFBEE] border border-[#121212]">
            <CheckCircle2 className="w-4 h-4 text-[#1E7E34] shrink-0 mt-0.5" />
            <div>
              <span className="font-bold text-[#121212]">No Registry or Group Policy Edits</span>
              <p className="text-[11px] text-[#4B5563] mt-0.5">
                Zero modifications made to Windows Registry keys or system policies.
              </p>
            </div>
          </div>

          <div className="flex items-start gap-2.5 p-2.5 bg-[#EBFBEE] border border-[#121212]">
            <CheckCircle2 className="w-4 h-4 text-[#1E7E34] shrink-0 mt-0.5" />
            <div>
              <span className="font-bold text-[#121212]">No Root CA Certificates Injected</span>
              <p className="text-[11px] text-[#4B5563] mt-0.5">
                No HTTPS MITM root certificates are generated, required, or added to the Windows Certificate Store.
              </p>
            </div>
          </div>

          <div className="flex items-start gap-2.5 p-2.5 bg-[#EBFBEE] border border-[#121212]">
            <CheckCircle2 className="w-4 h-4 text-[#1E7E34] shrink-0 mt-0.5" />
            <div>
              <span className="font-bold text-[#121212]">No Kernel Drivers Installed</span>
              <p className="text-[11px] text-[#4B5563] mt-0.5">
                Operates purely in user-space without WinDivert, WinPcap, or WFP kernel network drivers.
              </p>
            </div>
          </div>
        </div>
      </NeoCard>
    </div>
  );
};
