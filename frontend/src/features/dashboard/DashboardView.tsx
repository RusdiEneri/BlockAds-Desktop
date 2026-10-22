import React from 'react';
import {
  ShieldAlert,
  Server,
  Activity,
  CheckCircle2,
  XCircle,
  Clock,
  ArrowRight,
  ShieldCheck,
  AlertTriangle,
} from 'lucide-react';
import { NeoCard } from '../../components/NeoCard';
import { NeoButton } from '../../components/NeoButton';
import { NeoBadge } from '../../components/NeoBadge';
import { models } from '../../../wailsjs/go/models';

interface DashboardViewProps {
  protectionStatus: models.ProtectionStatus | null;
  sampleLogs: models.QueryLogEntry[];
  filterLists: models.FilterList[];
  onNavigate: (tab: 'querylog' | 'filters' | 'settings' | 'diagnostics') => void;
}

export const DashboardView: React.FC<DashboardViewProps> = ({
  protectionStatus,
  sampleLogs,
  filterLists,
  onNavigate,
}) => {
  const recentLogs = sampleLogs.slice(0, 5);

  const totalRules = filterLists
    .filter((f) => f.enabled)
    .reduce((acc, f) => acc + f.ruleCount, 0);

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-xl font-black uppercase tracking-wider text-[#121212]">
          Dashboard Overview
        </h1>
        <p className="text-xs text-[#4B5563] mt-0.5">
          System DNS Filtering Telemetry & Operational State
        </p>
      </div>

      {/* Main Protection Status Banner */}
      <div className="bg-[#FFFFFF] neo-border neo-shadow-md p-5 border-l-8 border-l-[#D97706]">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div className="flex items-start gap-4">
            <div className="p-3 bg-[#FEF3C7] border-2 border-[#121212] shrink-0">
              <ShieldAlert className="w-6 h-6 text-[#92400E]" />
            </div>
            <div>
              <div className="flex items-center gap-2 flex-wrap">
                <span className="font-extrabold text-sm uppercase tracking-wide text-[#121212]">
                  DNS Filtering Inactive (Milestone 1)
                </span>
                <NeoBadge variant="warning" size="sm">
                  STANDBY MODE
                </NeoBadge>
              </div>
              <p className="text-xs text-[#4B5563] mt-1 max-w-2xl leading-relaxed">
                {protectionStatus?.message ||
                  'The core Go DNS proxy is in development. In accordance with Milestone 1 safety rules, your Windows network adapter DNS settings have NOT been altered.'}
              </p>
              <div className="flex items-center gap-4 mt-3 text-[11px] font-mono text-[#4B5563] flex-wrap">
                <span className="flex items-center gap-1.5 bg-[#F5F1E8] px-2 py-1 border border-[#121212]">
                  <Server className="w-3.5 h-3.5 text-[#B38B6D]" />
                  <span>Upstream: {protectionStatus?.upstreamUrl || 'https://dns.caliph.dev/dns-query'}</span>
                </span>
                <span className="flex items-center gap-1.5 bg-[#F5F1E8] px-2 py-1 border border-[#121212]">
                  <Activity className="w-3.5 h-3.5 text-[#6B7280]" />
                  <span>Listener: {protectionStatus?.listenAddr || '127.0.0.1:53 Standby'}</span>
                </span>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <NeoButton
              variant="secondary"
              size="sm"
              onClick={() => onNavigate('diagnostics')}
            >
              Run Audit
            </NeoButton>
            <NeoButton
              variant="primary"
              size="sm"
              onClick={() => onNavigate('settings')}
            >
              Configure Upstream
            </NeoButton>
          </div>
        </div>
      </div>

      {/* Metrics Row with Explicit Demo Badges */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <NeoCard
          title="Total Queries"
          badge={
            <NeoBadge variant="neutral" size="sm">
              SAMPLE
            </NeoBadge>
          }
        >
          <div className="flex items-baseline justify-between mt-1">
            <div className="text-2xl font-black tracking-tight text-[#121212]">
              1,428
            </div>
            <Clock className="w-4 h-4 text-[#6B7280]" />
          </div>
          <p className="text-[11px] text-[#6B7280] mt-1 font-medium">
            Simulated throughput over 24h
          </p>
        </NeoCard>

        <NeoCard
          title="Blocked Queries"
          badge={
            <NeoBadge variant="blocked" size="sm">
              SAMPLE
            </NeoBadge>
          }
        >
          <div className="flex items-baseline justify-between mt-1">
            <div className="text-2xl font-black tracking-tight text-[#D9381E]">
              412
            </div>
            <span className="text-xs font-bold text-[#D9381E] bg-[#FEECEB] px-1.5 py-0.5 border border-[#121212]">
              28.8%
            </span>
          </div>
          <p className="text-[11px] text-[#6B7280] mt-1 font-medium">
            Ads, trackers & telemetry blocked
          </p>
        </NeoCard>

        <NeoCard
          title="Allowed Queries"
          badge={
            <NeoBadge variant="success" size="sm">
              SAMPLE
            </NeoBadge>
          }
        >
          <div className="flex items-baseline justify-between mt-1">
            <div className="text-2xl font-black tracking-tight text-[#1E7E34]">
              1,016
            </div>
            <span className="text-xs font-bold text-[#1E7E34] bg-[#EBFBEE] px-1.5 py-0.5 border border-[#121212]">
              71.2%
            </span>
          </div>
          <p className="text-[11px] text-[#6B7280] mt-1 font-medium">
            Permitted and forwarded to DoH
          </p>
        </NeoCard>

        <NeoCard
          title="Active Rules"
          badge={
            <NeoBadge variant="accent" size="sm">
              CATALOG
            </NeoBadge>
          }
        >
          <div className="flex items-baseline justify-between mt-1">
            <div className="text-2xl font-black tracking-tight text-[#B38B6D]">
              {totalRules.toLocaleString()}
            </div>
            <ShieldCheck className="w-4 h-4 text-[#B38B6D]" />
          </div>
          <p className="text-[11px] text-[#6B7280] mt-1 font-medium">
            Across {filterLists.filter((f) => f.enabled).length} enabled filter lists
          </p>
        </NeoCard>
      </div>

      {/* Recent Queries Activity */}
      <NeoCard
        title="Recent Query Activity"
        subtitle="Illustrative log stream of intercepted DNS resolution events"
        badge={
          <NeoBadge variant="neutral" size="sm">
            SAMPLE LOG STREAM
          </NeoBadge>
        }
        headerAction={
          <button
            onClick={() => onNavigate('querylog')}
            className="flex items-center gap-1 text-xs font-bold uppercase tracking-wider text-[#121212] hover:text-[#B38B6D] transition-colors"
          >
            <span>View Full Log</span>
            <ArrowRight className="w-3.5 h-3.5" />
          </button>
        }
        noPadding
      >
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse text-xs">
            <thead>
              <tr className="bg-[#F5F1E8] neo-border-b font-bold uppercase tracking-wider text-[#4B5563]">
                <th className="py-2.5 px-4">Time</th>
                <th className="py-2.5 px-4">Domain</th>
                <th className="py-2.5 px-4">Type</th>
                <th className="py-2.5 px-4">Status</th>
                <th className="py-2.5 px-4">Latency</th>
                <th className="py-2.5 px-4">Rule Matched</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#EBE7DE]">
              {recentLogs.map((log) => (
                <tr key={log.id} className="hover:bg-[#FDFBF7] transition-colors">
                  <td className="py-2.5 px-4 font-mono text-[#6B7280]">
                    {log.timestamp}
                  </td>
                  <td className="py-2.5 px-4 font-mono font-bold text-[#121212]">
                    {log.domain}
                  </td>
                  <td className="py-2.5 px-4 font-mono text-[11px] text-[#4B5563]">
                    {log.queryType}
                  </td>
                  <td className="py-2.5 px-4">
                    {log.status === 'BLOCKED' && (
                      <NeoBadge variant="blocked" size="sm">
                        <XCircle className="w-3 h-3" />
                        BLOCKED
                      </NeoBadge>
                    )}
                    {log.status === 'ALLOWED' && (
                      <NeoBadge variant="success" size="sm">
                        <CheckCircle2 className="w-3 h-3" />
                        ALLOWED
                      </NeoBadge>
                    )}
                    {log.status === 'FAILED' && (
                      <NeoBadge variant="warning" size="sm">
                        <AlertTriangle className="w-3 h-3" />
                        FAILED
                      </NeoBadge>
                    )}
                  </td>
                  <td className="py-2.5 px-4 font-mono text-[#6B7280]">
                    {log.upstreamLatencyMs} ms
                  </td>
                  <td className="py-2.5 px-4 text-[#4B5563] truncate max-w-xs">
                    {log.filterRule}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </NeoCard>

      {/* Architecture & Milestone Transparency Box */}
      <div className="bg-[#EBE7DE] neo-border p-4 text-xs">
        <div className="flex items-center gap-2 mb-1.5 font-bold uppercase tracking-wider text-[#121212]">
          <AlertTriangle className="w-4 h-4 text-[#B38B6D]" />
          <span>Safety & Scope Invariants (Milestone 1)</span>
        </div>
        <p className="text-[#4B5563] leading-relaxed">
          BlockAds Desktop adheres to a strict phased rollout. In Milestone 1, the desktop UI, design system, and backend IPC contracts are implemented and verified. No packet interception drivers, Windows Services, or network adapter changes are installed. Real DNS resolution will begin in Milestone 2.
        </p>
      </div>
    </div>
  );
};
