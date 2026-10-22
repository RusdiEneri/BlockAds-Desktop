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
          <h1 className="text-xl font-black uppercase tracking-wider text-[#121212]">
            Filter-List Manager & Scheduler
          </h1>
          <p className="text-xs text-[#4B5563] mt-0.5">
            Curated catalog, manual URL additions, deduplication, and atomic hot-reload
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {onRefreshAll && (
            <button
              onClick={handleRefreshAllClick}
              disabled={isRefreshingAll}
              className="px-3 py-1.5 bg-white neo-border neo-shadow-sm text-xs font-bold text-[#121212] flex items-center gap-1.5 hover:bg-[#F5F1E8] transition-all disabled:opacity-50"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${isRefreshingAll ? 'animate-spin' : ''}`} />
              <span>{isRefreshingAll ? 'Updating All...' : 'Refresh All'}</span>
            </button>
          )}

          <button
            onClick={() => setIsCatalogOpen(true)}
            className="px-3 py-1.5 bg-white neo-border neo-shadow-sm text-xs font-bold text-[#121212] flex items-center gap-1.5 hover:bg-[#F5F1E8] transition-all"
          >
            <BookOpen className="w-3.5 h-3.5 text-[#2E86C1]" />
            <span>Browse Catalog ({catalog.length})</span>
          </button>

          {onAddSubscription && (
            <button
              onClick={() => setIsAddModalOpen(true)}
              className="px-3 py-1.5 bg-[#121212] text-[#F5F1E8] neo-border neo-shadow-sm text-xs font-bold flex items-center gap-1.5 hover:bg-[#2C2C2C] transition-all"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>Add from URL</span>
            </button>
          )}
        </div>
      </div>

      {/* Architecture & Security Notice */}
      <div className="bg-[#EBF5FB] neo-border p-3.5 flex items-start gap-3">
        <ShieldCheck className="w-5 h-5 text-[#2E86C1] shrink-0 mt-0.5" />
        <div className="text-xs text-[#1B4F72] leading-relaxed">
          <span className="font-bold">Persistent Storage & Atomic Matcher Hot-Reload: </span>
          Filter subscriptions are stored in SQLite and downloaded with strict HTTPS verification, SSRF blocking (private & loopback IPs forbidden), duplicate URL protection, and 25MB response caps. Updates use HTTP 304 Not Modified headers, publishing candidate trie matchers atomically without DNS downtime.
        </div>
      </div>

      {/* Summary Metrics */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="bg-white neo-border neo-shadow-sm p-3">
          <div className="text-[11px] font-bold uppercase tracking-wider text-[#6B7280]">
            Configured Subscriptions
          </div>
          <div className="text-xl font-black text-[#121212] mt-0.5">
            {enabledLists.length} of {filterLists.length} Enabled
          </div>
        </div>

        <div className="bg-white neo-border neo-shadow-sm p-3">
          <div className="text-[11px] font-bold uppercase tracking-wider text-[#6B7280]">
            Active Rules in Matcher
          </div>
          <div className="text-xl font-black text-[#1E7E34] mt-0.5">
            {activeRules.toLocaleString()} Domains
          </div>
        </div>

        <div className="bg-white neo-border neo-shadow-sm p-3">
          <div className="text-[11px] font-bold uppercase tracking-wider text-[#6B7280]">
            Scheduler Sync
          </div>
          <div className="text-xl font-black text-[#B38B6D] mt-0.5">
            24h Auto-Sync (±10% Jitter)
          </div>
        </div>
      </div>

      {/* Filter Subscriptions List Section */}
      <div className="space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <h2 className="text-sm font-black uppercase tracking-wider text-[#121212] flex items-center gap-2">
            <Layers className="w-4 h-4" />
            <span>Active Subscriptions ({filterLists.length})</span>
          </h2>

          <div className="relative w-full sm:w-64">
            <Search className="w-3.5 h-3.5 absolute left-2.5 top-2.5 text-[#6B7280]" />
            <input
              type="text"
              placeholder="Search subscriptions..."
              value={listSearch}
              onChange={(e) => setListSearch(e.target.value)}
              className="w-full pl-8 pr-3 py-1.5 text-xs border border-[#121212] bg-white font-medium focus:outline-hidden"
            />
          </div>
        </div>

        {displayedLists.length === 0 ? (
          <div className="p-8 text-center text-xs text-[#6B7280] bg-white neo-border space-y-2">
            <p>No filter subscriptions match your filter.</p>
            <button
              onClick={() => setIsCatalogOpen(true)}
              className="px-3 py-1.5 bg-[#121212] text-[#F5F1E8] text-xs font-bold neo-border"
            >
              Open Curated Catalog
            </button>
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
                  className={`transition-all duration-300 ${
                    isHighlighted ? 'ring-4 ring-[#D9381E] bg-[#FEF9E7]' : ''
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
                            className="p-1 hover:bg-[#EBE7DE] text-[#121212] border border-transparent hover:border-[#121212] transition-all disabled:opacity-40"
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
                          <div className="w-11 h-6 bg-[#EBE7DE] border-2 border-[#121212] peer-checked:bg-[#1E7E34] transition-colors relative">
                            <div
                              className={`w-4 h-4 bg-white border border-[#121212] transition-transform duration-75 absolute top-0.5 left-0.5 ${
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
                            className="p-1 hover:bg-[#FEECEB] text-[#D9381E] border border-transparent hover:border-[#D9381E] transition-all"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    }
                  >
                    <p className="text-xs text-[#4B5563] leading-relaxed mb-3">
                      {list.description || 'Custom filter list subscription.'}
                    </p>

                    {/* Status & Error diagnostic banner */}
                    {isFailed && list.lastError && (
                      <div className="mb-3 p-2 bg-[#FEECEB] border border-[#D9381E] text-[11px] text-[#D9381E] flex items-start gap-1.5">
                        <AlertCircle className="w-3.5 h-3.5 shrink-0 mt-0.5" />
                        <span className="font-mono">{list.lastError}</span>
                      </div>
                    )}

                    <div className="space-y-1.5 pt-2.5 border-t border-[#EBE7DE] text-[11px]">
                      <div className="flex items-center justify-between">
                        <span className="font-bold text-[#6B7280]">Active Rules:</span>
                        <span className="font-mono font-bold text-[#121212]">
                          {list.ruleCount ? list.ruleCount.toLocaleString() : 0} rules
                        </span>
                      </div>

                      <div className="flex items-center justify-between">
                        <span className="font-bold text-[#6B7280]">Source URL:</span>
                        <a
                          href={list.sourceUrl}
                          target="_blank"
                          rel="noopener noreferrer"
                          title={list.sourceUrl}
                          className="font-mono text-[#2E86C1] hover:underline truncate max-w-[200px] flex items-center gap-1"
                        >
                          <span>{list.sourceUrl}</span>
                          <ExternalLink className="w-2.5 h-2.5 shrink-0" />
                        </a>
                      </div>

                      <div className="flex items-center justify-between">
                        <span className="font-bold text-[#6B7280]">ETag / Checksum:</span>
                        <span className="font-mono text-[#6B7280] truncate max-w-[200px]">
                          {list.etag ? list.etag : list.contentChecksum ? list.contentChecksum.substring(0, 12) + '...' : 'None'}
                        </span>
                      </div>

                      <div className="flex items-center justify-between">
                        <span className="font-bold text-[#6B7280]">Last Updated:</span>
                        <span className="font-mono text-[#6B7280] flex items-center gap-1">
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
