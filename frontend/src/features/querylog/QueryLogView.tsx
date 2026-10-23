import React, { useState, useMemo } from 'react';
import {
  Search,
  Filter,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  RotateCcw,
  ShieldCheck,
  RefreshCw,
  Zap,
  Play,
} from 'lucide-react';
import { NeoCard } from '../../components/NeoCard';
import { NeoButton } from '../../components/NeoButton';
import { NeoBadge } from '../../components/NeoBadge';
import { models } from '../../../wailsjs/go/models';

interface QueryLogViewProps {
  logs: models.QueryLogEntry[];
  onSimulateQuery?: (domain: string, queryType?: string) => Promise<void>;
  onRefresh?: () => Promise<void>;
}

export const QueryLogView: React.FC<QueryLogViewProps> = ({
  logs,
  onSimulateQuery,
  onRefresh,
}) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<'ALL' | 'ALLOWED' | 'BLOCKED' | 'FAILED'>('ALL');
  const [typeFilter, setTypeFilter] = useState<string>('ALL');
  const [testDomain, setTestDomain] = useState('');
  const [isTesting, setIsTesting] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);

  const formatTime = (ts: string) => {
    if (!ts) return '';
    if (ts.includes('T')) {
      try {
        const d = new Date(ts);
        if (!isNaN(d.getTime())) {
          return d.toLocaleTimeString([], { hour12: false, minute: '2-digit', second: '2-digit' });
        }
      } catch {
        return ts;
      }
    }
    return ts;
  };

  const handleTest = async (domain: string) => {
    if (!onSimulateQuery || !domain.trim()) return;
    try {
      setIsTesting(true);
      await onSimulateQuery(domain.trim(), 'A');
    } finally {
      setIsTesting(false);
    }
  };

  const handleRefresh = async () => {
    if (!onRefresh) return;
    try {
      setIsRefreshing(true);
      await onRefresh();
    } finally {
      setIsRefreshing(false);
    }
  };

  const filteredLogs = useMemo(() => {
    return logs.filter((log) => {
      const matchesSearch =
        searchTerm.trim() === '' ||
        log.domain.toLowerCase().includes(searchTerm.toLowerCase()) ||
        log.filterRule.toLowerCase().includes(searchTerm.toLowerCase());

      const matchesStatus =
        statusFilter === 'ALL' || log.status === statusFilter;

      const matchesType =
        typeFilter === 'ALL' || log.queryType === typeFilter;

      return matchesSearch && matchesStatus && matchesType;
    });
  }, [logs, searchTerm, statusFilter, typeFilter]);

  const uniqueTypes = useMemo(() => {
    const types = new Set(logs.map((l) => l.queryType));
    return ['ALL', ...Array.from(types)];
  }, [logs]);

  return (
    <div className="space-y-5">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <span className="text-[#33FF00] font-['VT323',monospace] text-3xl font-bold glow-green">
              {'>'} TELEMETRY TAPE // DNS LOG STREAM
            </span>
          </div>
          <p className="text-xs text-[#D2B48C] font-mono mt-0.5">
            Real-Time Resolution Inspection, Interception Reason & Latency
          </p>
        </div>
        <div className="flex items-center gap-2">
          {onRefresh && (
            <button
              onClick={handleRefresh}
              disabled={isRefreshing}
              title="Refresh logs from SQLite"
              className="p-1.5 rounded-lg border-2 border-[#3F485C] bg-[#222632] hover:bg-[#2B303D] text-[#E8E0D0] hover:text-[#33FF00] transition-all cursor-pointer shadow-[0_2px_6px_rgba(0,0,0,0.3)]"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${isRefreshing ? 'animate-spin text-[#33FF00]' : ''}`} />
            </button>
          )}
          <NeoBadge variant="success" size="sm">
            <span className="w-1.5 h-1.5 rounded-full bg-current mr-1 inline-block animate-ping" />
            ENGINE STREAM ({logs.length} RECORDS)
          </NeoBadge>
        </div>
      </div>

      {/* Live Engine Stream Active Banner */}
      <div className="bg-[#102B15] border-2 border-[#1E5C2A] rounded-lg p-3.5 flex items-start gap-3 shadow-[0_0_12px_rgba(51,255,0,0.1)]">
        <ShieldCheck className="w-5 h-5 text-[#33FF00] shrink-0 mt-0.5" />
        <div className="text-xs text-[#E8E0D0] font-mono leading-relaxed">
          <span className="font-bold text-[#33FF00]">LIVE DNS PROXY CONNECTED: </span>
          Resolution requests intercepted on <code className="bg-[#0C1E0F] text-[#33FF00] px-1.5 py-0.5 rounded border border-[#1E5C2A] font-mono text-[11px]">127.0.0.1:53</code> are processed and recorded in real time. Filter list decisions, rule matches, and upstream DoH latencies are saved to SQLite.
        </div>
      </div>

      {/* Quick Query Test / Lookup Bar */}
      {onSimulateQuery && (
        <div className="bg-[#222632] border-2 border-[#3F485C] rounded-lg shadow-md p-3 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-mono font-bold uppercase text-[11px] text-[#D2B48C] flex items-center gap-1.5">
              <Zap className="w-3.5 h-3.5 text-[#33FF00]" />
              INJECT PROBE:
            </span>
            <button
              onClick={() => handleTest('adservice.google.com')}
              disabled={isTesting}
              className="px-2.5 py-1 rounded bg-[#2D1212] border border-[#612323] font-mono font-bold text-[#FF5252] hover:bg-[#401818] transition-colors cursor-pointer"
            >
              adservice.google.com [Ad]
            </button>
            <button
              onClick={() => handleTest('doubleclick.net')}
              disabled={isTesting}
              className="px-2.5 py-1 rounded bg-[#2D1212] border border-[#612323] font-mono font-bold text-[#FF5252] hover:bg-[#401818] transition-colors cursor-pointer"
            >
              doubleclick.net [Tracker]
            </button>
            <button
              onClick={() => handleTest('github.com')}
              disabled={isTesting}
              className="px-2.5 py-1 rounded bg-[#102D15] border border-[#1F5E2B] font-mono font-bold text-[#33FF00] hover:bg-[#153D1C] transition-colors cursor-pointer"
            >
              github.com [Clean]
            </button>
          </div>

          <div className="flex items-center gap-1.5 w-full sm:w-auto">
            <input
              type="text"
              placeholder="> test lookup domain..."
              value={testDomain}
              onChange={(e) => setTestDomain(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && testDomain.trim()) {
                  handleTest(testDomain.trim());
                  setTestDomain('');
                }
              }}
              className="px-2.5 py-1 rounded border-2 border-[#2F3647] font-mono text-xs w-full sm:w-48 focus:outline-none focus:border-[#33FF00] bg-[#0E1015] text-[#33FF00]"
            />
            <button
              onClick={() => {
                if (testDomain.trim()) {
                  handleTest(testDomain.trim());
                  setTestDomain('');
                }
              }}
              disabled={!testDomain.trim() || isTesting}
              className="px-3 py-1 bg-[#33FF00] text-[#14161E] font-mono font-bold rounded hover:bg-[#2EE600] transition-colors cursor-pointer shrink-0 disabled:opacity-50"
            >
              <Play className="w-3 h-3 fill-current" />
            </button>
          </div>
        </div>
      )}

      {/* Search and Filters Toolbar */}
      <div className="bg-[#222632] border-2 border-[#3F485C] rounded-lg shadow-md p-4 space-y-3">
        <div className="flex flex-col md:flex-row items-stretch md:items-center gap-3">
          {/* Domain Search */}
          <div className="relative flex-1">
            <Search className="w-4 h-4 text-[#A0AAB8] absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              id="query-search-input"
              name="querySearch"
              aria-label="Search queries by domain"
              type="text"
              placeholder="> Filter by domain or rule (google, tracker, malware)..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-9 pr-3 py-2 text-xs font-mono rounded-lg border-2 border-[#3F4657] bg-[#14161E] text-[#33FF00] focus:outline-none focus:border-[#33FF00] transition-all"
            />
            {searchTerm && (
              <button
                onClick={() => setSearchTerm('')}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-xs font-mono font-bold text-[#A0AAB8] hover:text-[#33FF00]"
              >
                ✕
              </button>
            )}
          </div>

          {/* Reset Action */}
          {(searchTerm || statusFilter !== 'ALL' || typeFilter !== 'ALL') && (
            <NeoButton
              variant="outline"
              size="sm"
              onClick={() => {
                setSearchTerm('');
                setStatusFilter('ALL');
                setTypeFilter('ALL');
              }}
            >
              <RotateCcw className="w-3.5 h-3.5" />
              RESET FILTERS
            </NeoButton>
          )}
        </div>

        {/* Filter Pills */}
        <div className="flex flex-wrap items-center justify-between gap-3 pt-2 border-t-2 border-[#2B313E]">
          {/* Status Tabs */}
          <div className="flex items-center gap-1.5 flex-wrap">
            <span className="text-[11px] font-mono font-bold uppercase tracking-wider text-[#D2B48C] mr-1">
              STATUS:
            </span>
            {(['ALL', 'ALLOWED', 'BLOCKED', 'FAILED'] as const).map((status) => (
              <button
                key={status}
                onClick={() => setStatusFilter(status)}
                className={`px-2.5 py-1 text-[11px] font-mono font-bold uppercase rounded border transition-all cursor-pointer ${
                  statusFilter === status
                    ? 'bg-[#33FF00] text-[#14161E] border-[#29CC00] shadow-[0_0_8px_rgba(51,255,0,0.3)]'
                    : 'bg-[#181B23] text-[#A0AAB8] border-[#3F4657] hover:text-[#E8E0D0] hover:border-[#D2B48C]'
                }`}
              >
                {status}
              </button>
            ))}
          </div>

          {/* Type Filter */}
          <div className="flex items-center gap-1.5 flex-wrap">
            <span className="text-[11px] font-mono font-bold uppercase tracking-wider text-[#D2B48C] mr-1">
              TYPE:
            </span>
            {uniqueTypes.map((type) => (
              <button
                key={type}
                onClick={() => setTypeFilter(type)}
                className={`px-2 py-0.5 text-[10px] font-mono font-bold uppercase rounded border transition-all cursor-pointer ${
                  typeFilter === type
                    ? 'bg-[#D2B48C] text-[#14161E] border-[#B89B77]'
                    : 'bg-[#181B23] text-[#A0AAB8] border-[#3F4657] hover:text-[#E8E0D0]'
                }`}
              >
                {type}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Results Count */}
      <div className="flex items-center justify-between text-xs text-[#A0AAB8] font-mono px-1">
        <span>
          Showing <strong className="text-[#33FF00]">{filteredLogs.length}</strong> of {logs.length} logged queries
        </span>
        <span className="font-mono text-[11px] text-[#D2B48C]">Auto-Sync Active (1.5s)</span>
      </div>

      {/* Query Table */}
      <NeoCard noPadding>
        {filteredLogs.length === 0 ? (
          <div className="p-8 text-center space-y-2">
            <Filter className="w-8 h-8 text-[#555E70] mx-auto" />
            <h4 className="text-xs font-mono font-bold uppercase tracking-wider text-[#E8E0D0]">
              NO MATCHING TAPE RECORDS FOUND
            </h4>
            <p className="text-xs text-[#A0AAB8] font-mono">
              Try searching for a different domain or reset your status filter.
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse text-xs">
              <thead>
                <tr className="bg-[#181B23] border-b-2 border-[#2E3442] font-mono font-bold uppercase tracking-wider text-[#D2B48C]">
                  <th className="py-2.5 px-4">Timestamp</th>
                  <th className="py-2.5 px-4">Domain</th>
                  <th className="py-2.5 px-4">Type</th>
                  <th className="py-2.5 px-4">Client</th>
                  <th className="py-2.5 px-4">Latency</th>
                  <th className="py-2.5 px-4">Rule Matched</th>
                  <th className="py-2.5 px-4">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#252A36]">
                {filteredLogs.map((log) => (
                  <tr
                    key={log.id}
                    className="hover:bg-[#1E232F] transition-colors"
                  >
                    <td className="py-2.5 px-4 font-mono text-[#A0AAB8] whitespace-nowrap">
                      {formatTime(log.timestamp)}
                    </td>
                    <td className="py-2.5 px-4 font-mono font-bold text-[#E8E0D0] select-text">
                      {log.domain}
                    </td>
                    <td className="py-2.5 px-4 font-mono text-[11px] text-[#D2B48C]">
                      <span className="bg-[#14161E] px-1.5 py-0.5 rounded border border-[#333A4A]">
                        {log.queryType}
                      </span>
                    </td>
                    <td className="py-2.5 px-4 font-mono text-[#A0AAB8]">
                      {log.clientIp}
                    </td>
                    <td className="py-2.5 px-4 font-mono text-[#D2B48C]">
                      {log.upstreamLatencyMs} ms
                    </td>
                    <td className="py-2.5 px-4 text-[#A0AAB8]">
                      <div className="font-mono text-[#E8E0D0]">{log.filterRule || 'Passthrough'}</div>
                      <div className="text-[10px] font-mono text-[#A0AAB8] truncate max-w-xs">{log.reason || 'Resolved via engine'}</div>
                    </td>
                    <td className="py-2.5 px-4 whitespace-nowrap">
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
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </NeoCard>
    </div>
  );
};
