import React, { useState } from 'react';
import {
  Layers,
  Clock,
  ExternalLink,
  ShieldCheck,
  RefreshCw,
  Plus,
  Trash2,
  AlertCircle,
  CheckCircle2,
  FileText,
  BookOpen,
  Search,
} from 'lucide-react';
import { NeoCard } from '../../components/NeoCard';
import { NeoButton } from '../../components/NeoButton';
import { NeoBadge } from '../../components/NeoBadge';
import { FilterCatalogModal } from './FilterCatalogModal';
import { AddFilterModal } from './AddFilterModal';
import { models } from '../../../wailsjs/go/models';

interface FilterListsViewProps {
  filterLists: models.FilterList[];
  catalog: models.CatalogFilterItem[];
  onToggleFilter: (id: string, enabled: boolean) => Promise<void> | void;
  onRefreshAll?: () => Promise<void>;
  onRefreshSingle?: (id: string) => Promise<void>;
  onAddSubscription?: (name: string, url: string, format: string, category: string, description: string) => Promise<void>;
  onSubscribeCatalog?: (catalogId: string) => Promise<void>;
  onDeleteSubscription?: (id: string) => Promise<void>;
}

export const FilterListsView: React.FC<FilterListsViewProps> = ({
  filterLists,
  catalog,
  onToggleFilter,
  onRefreshAll,
  onRefreshSingle,
  onAddSubscription,
  onSubscribeCatalog,
  onDeleteSubscription,
}) => {
  const [isRefreshingAll, setIsRefreshingAll] = useState(false);
  const [refreshingId, setRefreshingId] = useState<string | null>(null);
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [isCatalogOpen, setIsCatalogOpen] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [highlightedId, setHighlightedId] = useState<string | null>(null);
  const [listSearch, setListSearch] = useState('');

  // Calculate statistics
  const enabledLists = filterLists.filter((f) => f.enabled);
  const activeRules = enabledLists
    .filter((f) => !!f.contentChecksum)
    .reduce((acc, f) => acc + (f.ruleCount || 0), 0);

  const handleRefreshAllClick = async () => {
    if (!onRefreshAll || isRefreshingAll) return;
    try {
      setIsRefreshingAll(true);
      await onRefreshAll();
    } catch (err: any) {
      console.error('Refresh all failed:', err);
    } finally {
      setIsRefreshingAll(false);
    }
  };

  const handleRefreshSingleClick = async (id: string) => {
    if (!onRefreshSingle || refreshingId) return;
    try {
      setRefreshingId(id);
      await onRefreshSingle(id);
    } catch (err: any) {
      console.error(`Refresh ${id} failed:`, err);
    } finally {
      setRefreshingId(null);
    }
  };

  const handleDeleteClick = async (id: string, name: string) => {
    if (!onDeleteSubscription) return;
    if (confirm(`Are you sure you want to delete filter subscription "${name}"?`)) {
      try {
        setDeletingId(id);
        await onDeleteSubscription(id);
      } catch (err) {
        console.error('Failed to delete subscription:', err);
      } finally {
        setDeletingId(null);
      }
    }
  };

  const handleNavigateToSubscription = (id: string) => {
    setHighlightedId(id);
    const element = document.getElementById(`filter-card-${id}`);
    if (element) {
      element.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
    setTimeout(() => {
      setHighlightedId(null);
    }, 3500);
  };

  const displayedLists = filterLists.filter((f) => {
    if (!listSearch.trim()) return true;
    const q = listSearch.toLowerCase();
    return (
      f.name.toLowerCase().includes(q) ||
      (f.category && f.category.toLowerCase().includes(q)) ||
      (f.description && f.description.toLowerCase().includes(q)) ||
      f.sourceUrl.toLowerCase().includes(q)
    );
  });

  return (
    <div className="space-y-5">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <span className="text-[#33FF00] font-['VT323',monospace] text-3xl font-bold glow-green">
              {'>'} CASSETTE BANK // FILTER LIST MANAGER
            </span>
          </div>
          <p className="text-xs text-[#D2B48C] font-mono mt-0.5">
            Curated catalog, manual URL additions, deduplication, and atomic hot-reload
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {onRefreshAll && (
            <button
              onClick={handleRefreshAllClick}
              disabled={isRefreshingAll}
              className="px-3 py-1.5 rounded-lg border-2 border-[#3F485C] bg-[#222632] text-[#E8E0D0] hover:text-[#33FF00] hover:border-[#33FF00] text-xs font-mono font-bold flex items-center gap-1.5 transition-all disabled:opacity-50 cursor-pointer shadow-sm"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${isRefreshingAll ? 'animate-spin text-[#33FF00]' : ''}`} />
              <span>{isRefreshingAll ? 'UPDATING ALL...' : 'REFRESH ALL'}</span>
            </button>
          )}

          <button
            onClick={() => setIsCatalogOpen(true)}
            className="px-3 py-1.5 rounded-lg border-2 border-[#3F485C] bg-[#222632] text-[#D2B48C] hover:border-[#D2B48C] text-xs font-mono font-bold flex items-center gap-1.5 transition-all cursor-pointer shadow-sm"
          >
            <BookOpen className="w-3.5 h-3.5 text-[#D2B48C]" />
            <span>CATALOG ({catalog.length})</span>
          </button>

          {onAddSubscription && (
            <button
              onClick={() => setIsAddModalOpen(true)}
              className="px-3 py-1.5 rounded-lg border-2 border-[#29CC00] bg-[#33FF00] text-[#14161E] text-xs font-mono font-bold flex items-center gap-1.5 hover:bg-[#2EE600] transition-all cursor-pointer shadow-[0_0_8px_rgba(51,255,0,0.3)]"
            >
              <Plus className="w-3.5 h-3.5 text-[#14161E]" />
              <span>ADD URL</span>
            </button>
          )}
        </div>
      </div>

      {/* Architecture & Security Notice */}
      <div className="bg-[#10202F] border-2 border-[#1E4366] rounded-lg p-3.5 flex items-start gap-3 shadow-[0_0_12px_rgba(30,67,102,0.3)]">
        <ShieldCheck className="w-5 h-5 text-[#85C1E9] shrink-0 mt-0.5" />
        <div className="text-xs text-[#D6EAF8] font-mono leading-relaxed">
          <span className="font-bold text-[#85C1E9]">PERSISTENT STORAGE & ATOMIC HOT-RELOAD: </span>
          Filter subscriptions are stored in SQLite and downloaded with strict HTTPS verification, SSRF blocking (private & loopback IPs forbidden), duplicate URL protection, and 25MB response caps. Updates use HTTP 304 Not Modified headers, publishing candidate trie matchers atomically without DNS downtime.
        </div>
      </div>

      {/* Summary Metrics */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="bg-[#222632] border-2 border-[#3F485C] rounded-lg p-3 shadow-md">
          <div className="text-[11px] font-mono uppercase tracking-wider text-[#D2B48C]">
            Configured Subscriptions
          </div>
          <div className="text-2xl font-['VT323',monospace] font-bold text-[#E8E0D0] mt-0.5">
            {enabledLists.length} of {filterLists.length} Enabled
          </div>
        </div>

        <div className="bg-[#222632] border-2 border-[#3F485C] rounded-lg p-3 shadow-md">
          <div className="text-[11px] font-mono uppercase tracking-wider text-[#D2B48C]">
            Active Rules in Matcher
          </div>
          <div className="text-2xl font-['VT323',monospace] font-bold text-[#33FF00] glow-green mt-0.5">
            {activeRules.toLocaleString()} Domains
          </div>
        </div>

        <div className="bg-[#222632] border-2 border-[#3F485C] rounded-lg p-3 shadow-md">
          <div className="text-[11px] font-mono uppercase tracking-wider text-[#D2B48C]">
            Scheduler Sync
          </div>
          <div className="text-2xl font-['VT323',monospace] font-bold text-[#FFB000] glow-amber mt-0.5">
            24h Auto-Sync (±10% Jitter)
          </div>
        </div>
      </div>

      {/* Filter Subscriptions List Section */}
      <div className="space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <h2 className="text-sm font-mono font-bold uppercase tracking-wider text-[#E8E0D0] flex items-center gap-2">
            <Layers className="w-4 h-4 text-[#33FF00]" />
            <span>Active Subscriptions ({filterLists.length})</span>
          </h2>

          <div className="relative w-full sm:w-64">
            <Search className="w-3.5 h-3.5 absolute left-2.5 top-2.5 text-[#A0AAB8]" />
            <input
              type="text"
              placeholder="> Search subscriptions..."
              value={listSearch}
              onChange={(e) => setListSearch(e.target.value)}
              className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg border-2 border-[#3F4657] bg-[#12141A] text-[#33FF00] placeholder-[#A0AAB8] font-mono focus:border-[#33FF00] focus:outline-none"
            />
          </div>
        </div>

        {displayedLists.length === 0 ? (
          <div className="p-8 text-center text-xs text-[#A0AAB8] bg-[#151820] border-2 border-[#3F4657] rounded-lg space-y-3 font-mono">
            <p>&gt; No filter subscriptions match your filter query.</p>
            <NeoButton
              variant="primary"
              size="sm"
              onClick={() => setIsCatalogOpen(true)}
            >
              Open Curated Catalog
            </NeoButton>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {displayedLists.map((list) => {
              const isUpdating = refreshingId === list.id;
              const isFailed = list.lastStatus === 'failed';
              const isHighlighted = highlightedId === list.id;

              return (
                <div
                  key={list.id}
                  id={`filter-card-${list.id}`}
                  className={`transition-all duration-300 rounded-lg ${
                    isHighlighted ? 'ring-2 ring-[#FF4D4D] bg-[#22171A]' : ''
                  }`}
                >
                  <NeoCard
                    title={list.name}
                    subtitle={list.category || 'General'}
                    badge={
                      <div className="flex items-center gap-1.5">
                        <NeoBadge variant="neutral" size="sm">
                          {list.format ? list.format.toUpperCase() : 'HOSTS'}
                        </NeoBadge>

                        {isFailed ? (
                          <NeoBadge variant="blocked" size="sm">
                            <AlertCircle className="w-3 h-3" />
                            FAILED
                          </NeoBadge>
                        ) : list.contentChecksum ? (
                          <NeoBadge variant="success" size="sm">
                            <CheckCircle2 className="w-3 h-3" />
                            ACTIVE
                          </NeoBadge>
                        ) : (
                          <NeoBadge variant="warning" size="sm">
                            PENDING
                          </NeoBadge>
                        )}
                      </div>
                    }
                    headerAction={
                      <div className="flex items-center gap-2">
                        {onRefreshSingle && (
                          <button
                            onClick={() => handleRefreshSingleClick(list.id)}
                            disabled={isUpdating || !list.enabled}
                            title="Update list now via conditional HTTP"
                            className="p-1 hover:bg-[#1E232F] text-[#D2B48C] hover:text-[#33FF00] border border-transparent hover:border-[#33FF00]/40 rounded transition-all disabled:opacity-40 cursor-pointer"
                          >
                            <RefreshCw className={`w-3.5 h-3.5 ${isUpdating ? 'animate-spin' : ''}`} />
                          </button>
                        )}

                        <label
                          htmlFor={`toggle-filter-${list.id}`}
                          className="relative inline-flex items-center cursor-pointer"
                        >
                          <input
                            id={`toggle-filter-${list.id}`}
                            aria-label={`Toggle ${list.name}`}
                            type="checkbox"
                            checked={list.enabled}
                            onChange={(e) => onToggleFilter(list.id, e.target.checked)}
                            className="sr-only peer"
                          />
                          <div className="w-11 h-6 bg-[#151820] border-2 border-[#3F4657] peer-checked:bg-[#33FF00] rounded-full transition-colors relative">
                            <div
                              className={`w-4 h-4 bg-[#D2B48C] peer-checked:bg-[#14161E] rounded-full transition-transform duration-75 absolute top-0.5 left-0.5 ${
                                list.enabled ? 'translate-x-5' : 'translate-x-0'
                              }`}
                            />
                          </div>
                        </label>

                        {onDeleteSubscription && (
                          <button
                            onClick={() => handleDeleteClick(list.id, list.name)}
                            disabled={deletingId === list.id}
                            title="Delete subscription"
                            className="p-1 hover:bg-[#2B1B1D] text-[#FF4D4D] hover:text-[#FF7777] border border-transparent hover:border-[#FF4D4D]/40 rounded transition-all cursor-pointer"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    }
                  >
                    <p className="text-xs text-[#E8E0D0] leading-relaxed mb-3 font-mono">
                      {list.description || 'Custom filter list subscription.'}
                    </p>

                    {/* Status & Error diagnostic banner */}
                    {isFailed && list.lastError && (
                      <div className="mb-3 p-2.5 bg-[#2B1414] border border-[#802020] text-[11px] text-[#FF4D4D] rounded flex items-start gap-1.5 font-mono">
                        <AlertCircle className="w-3.5 h-3.5 shrink-0 mt-0.5" />
                        <span>{list.lastError}</span>
                      </div>
                    )}

                    <div className="space-y-1.5 pt-2.5 border-t border-[#3F4657]/40 text-[11px] font-mono">
                      <div className="flex items-center justify-between">
                        <span className="font-bold text-[#A0AAB8]">Active Rules:</span>
                        <span className="font-bold text-[#33FF00]">
                          {list.ruleCount ? list.ruleCount.toLocaleString() : 0} rules
                        </span>
                      </div>

                      <div className="flex items-center justify-between">
                        <span className="font-bold text-[#A0AAB8]">Source URL:</span>
                        <a
                          href={list.sourceUrl}
                          target="_blank"
                          rel="noopener noreferrer"
                          title={list.sourceUrl}
                          className="text-[#D2B48C] hover:text-[#33FF00] hover:underline truncate max-w-[200px] flex items-center gap-1"
                        >
                          <span>{list.sourceUrl}</span>
                          <ExternalLink className="w-2.5 h-2.5 shrink-0" />
                        </a>
                      </div>

                      <div className="flex items-center justify-between">
                        <span className="font-bold text-[#A0AAB8]">ETag / Checksum:</span>
                        <span className="text-[#A0AAB8] truncate max-w-[200px]">
                          {list.etag ? list.etag : list.contentChecksum ? list.contentChecksum.substring(0, 12) + '...' : 'None'}
                        </span>
                      </div>

                      <div className="flex items-center justify-between">
                        <span className="font-bold text-[#A0AAB8]">Last Updated:</span>
                        <span className="text-[#A0AAB8] flex items-center gap-1">
                          <Clock className="w-3 h-3" />
                          <span>{list.lastUpdated ? new Date(list.lastUpdated).toLocaleDateString() : 'Never'}</span>
                        </span>
                      </div>
                    </div>
                  </NeoCard>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Catalog Modal */}
      <FilterCatalogModal
        isOpen={isCatalogOpen}
        onClose={() => setIsCatalogOpen(false)}
        catalog={catalog}
        onSubscribe={async (catalogId) => {
          if (onSubscribeCatalog) {
            await onSubscribeCatalog(catalogId);
          }
        }}
        onNavigateToSubscription={handleNavigateToSubscription}
      />

      {/* Add Subscription Modal */}
      <AddFilterModal
        isOpen={isAddModalOpen}
        onClose={() => setIsAddModalOpen(false)}
        onAdd={async (name, url, format, category, description) => {
          if (onAddSubscription) {
            await onAddSubscription(name, url, format, category, description);
          }
        }}
        onNavigateToSubscription={handleNavigateToSubscription}
      />
    </div>
  );
};
