import React, { useState, useMemo } from 'react';
import {
  Search,
  Filter,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  RotateCcw,
  Info,
} from 'lucide-react';
import { NeoCard } from '../../components/NeoCard';
import { NeoButton } from '../../components/NeoButton';
import { NeoBadge } from '../../components/NeoBadge';
import { models } from '../../../wailsjs/go/models';

interface QueryLogViewProps {
  logs: models.QueryLogEntry[];
}

export const QueryLogView: React.FC<QueryLogViewProps> = ({ logs }) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<'ALL' | 'ALLOWED' | 'BLOCKED' | 'FAILED'>('ALL');
  const [typeFilter, setTypeFilter] = useState<string>('ALL');

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
          <h1 className="text-xl font-black uppercase tracking-wider text-[#121212]">
            DNS Query Log
          </h1>
          <p className="text-xs text-[#4B5563] mt-0.5">
            Real-Time Resolution Inspection, Interception Reason & Latency
          </p>
        </div>
        <NeoBadge variant="neutral">
          <Info className="w-3.5 h-3.5" />
          SAMPLE DATA STREAM
        </NeoBadge>
      </div>

      {/* Honest Demo Notice Banner */}
      <div className="bg-[#FEF3C7] neo-border p-3.5 flex items-start gap-3">
        <Info className="w-5 h-5 text-[#92400E] shrink-0 mt-0.5" />
        <div className="text-xs text-[#92400E]">
          <span className="font-bold">Simulated Resolution Records: </span>
          The queries below demonstrate the filtering schema, rule matches, and search UI. In Milestone 2, this stream will be connected directly to the Go DNS proxy on <code className="bg-[#FDE68A] px-1 py-0.5 border border-[#121212] font-mono text-[11px]">127.0.0.1:53</code>.
        </div>
      </div>

      {/* Search and Filters Toolbar */}
      <div className="bg-white neo-border neo-shadow-md p-4 space-y-3">
        <div className="flex flex-col md:flex-row items-stretch md:items-center gap-3">
          {/* Domain Search */}
          <div className="relative flex-1">
            <Search className="w-4 h-4 text-[#6B7280] absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              id="query-search-input"
              name="querySearch"
              aria-label="Search queries by domain"
              type="text"
              placeholder="Search by domain (e.g. google, telemetry, github)..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-9 pr-3 py-2 text-xs border-2 border-[#121212] focus:outline-none focus:shadow-[2px_2px_0px_#B38B6D] transition-shadow"
            />
            {searchTerm && (
              <button
                onClick={() => setSearchTerm('')}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-xs font-bold text-[#6B7280] hover:text-[#121212]"
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
              Reset Filters
            </NeoButton>
          )}
        </div>

        {/* Filter Pills */}
        <div className="flex flex-wrap items-center justify-between gap-3 pt-2 border-t border-[#EBE7DE]">
          {/* Status Tabs */}
          <div className="flex items-center gap-1.5 flex-wrap">
            <span className="text-[11px] font-bold uppercase tracking-wider text-[#4B5563] mr-1">
              Status:
            </span>
            {(['ALL', 'ALLOWED', 'BLOCKED', 'FAILED'] as const).map((status) => (
              <button
                key={status}
                onClick={() => setStatusFilter(status)}
                className={`px-2.5 py-1 text-[11px] font-bold uppercase border-2 transition-all ${
                  statusFilter === status
                    ? 'bg-[#121212] text-[#F5F1E8] border-[#121212] shadow-[2px_2px_0px_#B38B6D]'
                    : 'bg-[#F5F1E8] text-[#4B5563] border-[#121212] hover:bg-[#FFFFFF]'
                }`}
              >
                {status}
              </button>
            ))}
          </div>

          {/* Type Filter */}
          <div className="flex items-center gap-1.5 flex-wrap">
            <span className="text-[11px] font-bold uppercase tracking-wider text-[#4B5563] mr-1">
              Query Type:
            </span>
            {uniqueTypes.map((type) => (
              <button
                key={type}
                onClick={() => setTypeFilter(type)}
                className={`px-2 py-0.5 text-[10px] font-mono font-bold uppercase border transition-all ${
                  typeFilter === type
                    ? 'bg-[#B38B6D] text-white border-[#121212]'
                    : 'bg-white text-[#4B5563] border-[#D1CBC0] hover:border-[#121212]'
                }`}
              >
                {type}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Results Count */}
      <div className="flex items-center justify-between text-xs text-[#6B7280] font-medium px-1">
        <span>
          Showing <strong className="text-[#121212]">{filteredLogs.length}</strong> of {logs.length} logged queries
        </span>
        <span className="font-mono text-[11px]">Format: RFC 1035 wire mock</span>
      </div>

      {/* Query Table */}
      <NeoCard noPadding>
        {filteredLogs.length === 0 ? (
          <div className="p-8 text-center space-y-2">
            <Filter className="w-8 h-8 text-[#9CA3AF] mx-auto" />
            <h4 className="text-xs font-bold uppercase tracking-wider text-[#121212]">
              No queries match your filter
            </h4>
            <p className="text-xs text-[#6B7280]">
              Try searching for a different domain or reset your status filter.
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse text-xs">
              <thead>
                <tr className="bg-[#F5F1E8] neo-border-b font-bold uppercase tracking-wider text-[#4B5563]">
                  <th className="py-2.5 px-4">Timestamp</th>
                  <th className="py-2.5 px-4">Domain</th>
                  <th className="py-2.5 px-4">Type</th>
                  <th className="py-2.5 px-4">Client</th>
                  <th className="py-2.5 px-4">Latency</th>
                  <th className="py-2.5 px-4">Rule Matched</th>
                  <th className="py-2.5 px-4">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#EBE7DE]">
                {filteredLogs.map((log) => (
                  <tr
                    key={log.id}
                    className="hover:bg-[#FDFBF7] transition-colors"
                  >
                    <td className="py-2.5 px-4 font-mono text-[#6B7280] whitespace-nowrap">
                      {log.timestamp}
                    </td>
                    <td className="py-2.5 px-4 font-mono font-bold text-[#121212] select-text">
                      {log.domain}
                    </td>
                    <td className="py-2.5 px-4 font-mono text-[11px] text-[#4B5563]">
                      <span className="bg-[#EBE7DE] px-1.5 py-0.5 border border-[#121212]">
                        {log.queryType}
                      </span>
                    </td>
                    <td className="py-2.5 px-4 font-mono text-[#6B7280]">
                      {log.clientIp}
                    </td>
                    <td className="py-2.5 px-4 font-mono text-[#6B7280]">
                      {log.upstreamLatencyMs} ms
                    </td>
                    <td className="py-2.5 px-4 text-[#4B5563]">
                      <div className="font-medium text-[#121212]">{log.filterRule}</div>
                      <div className="text-[10px] text-[#6B7280] truncate max-w-xs">{log.reason}</div>
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
