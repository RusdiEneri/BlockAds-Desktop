import React, { useState } from 'react';
import {
  X,
  Search,
  BookOpen,
  CheckCircle2,
  ExternalLink,
  ShieldCheck,
  Plus,
  Filter,
} from 'lucide-react';
import { NeoCard } from '../../components/NeoCard';
import { NeoBadge } from '../../components/NeoBadge';
import { models } from '../../../wailsjs/go/models';

interface FilterCatalogModalProps {
  isOpen: boolean;
  onClose: () => void;
  catalog: models.CatalogFilterItem[];
  onSubscribe: (catalogId: string) => Promise<void>;
  onNavigateToSubscription: (subscriptionId: string) => void;
}

export const FilterCatalogModal: React.FC<FilterCatalogModalProps> = ({
  isOpen,
  onClose,
  catalog,
  onSubscribe,
  onNavigateToSubscription,
}) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('All');
  const [subscribingId, setSubscribingId] = useState<string | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  if (!isOpen) return null;

  const categories = ['All', 'Advertising', 'Privacy & Tracking', 'Security & Malware', 'General Protection'];

  const filteredItems = catalog.filter((item) => {
    const matchesCategory =
      selectedCategory === 'All' || item.category.toLowerCase().includes(selectedCategory.toLowerCase());
    const matchesSearch =
      searchQuery === '' ||
      item.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      item.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
      item.maintainer.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesCategory && matchesSearch;
  });

  const handleSubscribeClick = async (catalogId: string) => {
    setErrorMsg(null);
    setSubscribingId(catalogId);
    try {
      await onSubscribe(catalogId);
    } catch (err: any) {
      setErrorMsg(err?.message || 'Failed to subscribe to filter list');
    } finally {
      setSubscribingId(null);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
      <div className="bg-[#F5F1E8] neo-border neo-shadow-lg w-full max-w-3xl max-h-[90vh] flex flex-col overflow-hidden">
        {/* Modal Header */}
        <div className="p-4 bg-white neo-border-b flex items-center justify-between">
          <div className="flex items-center gap-2">
            <BookOpen className="w-5 h-5 text-[#121212]" />
            <h2 className="text-base font-black uppercase tracking-wider text-[#121212]">
              Curated Filter Catalog
            </h2>
          </div>
          <button
            onClick={onClose}
            aria-label="Close modal"
            className="p-1 border border-[#121212] bg-[#F5F1E8] hover:bg-white transition-all"
          >
            <X className="w-4 h-4 text-[#121212]" />
          </button>
        </div>

        {/* Search & Category Filter Controls */}
        <div className="p-4 space-y-3 bg-white/70 neo-border-b">
          <div className="relative">
            <Search className="w-4 h-4 absolute left-3 top-3 text-[#6B7280]" />
            <input
              type="text"
              placeholder="Search catalog by name, maintainer, or description..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-3 py-2 text-xs border-2 border-[#121212] bg-white font-medium focus:outline-hidden"
            />
          </div>

          <div className="flex flex-wrap gap-1.5 items-center">
            <span className="text-[11px] font-bold uppercase tracking-wider text-[#6B7280] mr-1 flex items-center gap-1">
              <Filter className="w-3 h-3" /> Category:
            </span>
            {categories.map((cat) => (
              <button
                key={cat}
                onClick={() => setSelectedCategory(cat)}
                className={`px-2.5 py-1 text-[11px] font-bold uppercase tracking-wider border border-[#121212] transition-all ${
                  selectedCategory === cat
                    ? 'bg-[#121212] text-[#F5F1E8]'
                    : 'bg-white text-[#121212] hover:bg-[#EBE7DE]'
                }`}
              >
                {cat}
              </button>
            ))}
          </div>

          {errorMsg && (
            <div className="p-2 bg-[#FEECEB] border border-[#D9381E] text-xs text-[#D9381E] font-medium">
              {errorMsg}
            </div>
          )}
        </div>

        {/* Scrollable Catalog Grid */}
        <div className="p-4 overflow-y-auto space-y-3 flex-1">
          {filteredItems.length === 0 ? (
            <div className="p-8 text-center text-xs text-[#6B7280] bg-white neo-border">
              No verified filter lists match your search criteria.
            </div>
          ) : (
            filteredItems.map((item) => (
              <div
                key={item.id}
                className="p-3.5 bg-white neo-border neo-shadow-sm flex flex-col sm:flex-row sm:items-center justify-between gap-3"
              >
                <div className="space-y-1.5 flex-1">
                  <div className="flex items-center flex-wrap gap-2">
                    <span className="text-sm font-black text-[#121212]">{item.name}</span>
                    <NeoBadge variant="neutral" size="sm">
                      {item.format.toUpperCase()}
                    </NeoBadge>
                    <NeoBadge variant="accent" size="sm">
                      {item.category}
                    </NeoBadge>
                  </div>

                  <p className="text-xs text-[#4B5563] leading-relaxed">{item.description}</p>

                  <div className="flex items-center gap-4 text-[11px] text-[#6B7280] pt-0.5">
                    <span>
                      <strong className="text-[#121212]">Maintainer:</strong> {item.maintainer}
                    </span>
                    <span>
                      <strong className="text-[#121212]">License:</strong> {item.license}
                    </span>
                    {item.homepage && (
                      <a
                        href={item.homepage}
                        target="_blank"
                        rel="noreferrer"
                        className="text-[#2E86C1] hover:underline flex items-center gap-0.5"
                      >
                        <span>Homepage</span>
                        <ExternalLink className="w-2.5 h-2.5" />
                      </a>
                    )}
                  </div>
                </div>

                <div className="shrink-0 flex sm:flex-col items-end justify-center gap-2">
                  {item.isSubscribed ? (
                    <div className="flex items-center gap-1.5">
                      <NeoBadge variant="success" size="sm">
                        <CheckCircle2 className="w-3 h-3" />
                        SUBSCRIBED
                      </NeoBadge>
                      {item.subscribedId && (
                        <button
                          onClick={() => {
                            onClose();
                            onNavigateToSubscription(item.subscribedId!);
                          }}
                          className="text-[11px] font-bold text-[#2E86C1] hover:underline"
                        >
                          View
                        </button>
                      )}
                    </div>
                  ) : (
                    <button
                      onClick={() => handleSubscribeClick(item.id)}
                      disabled={subscribingId === item.id}
                      className="px-3.5 py-1.5 bg-[#121212] text-[#F5F1E8] text-xs font-bold neo-border hover:bg-[#2C2C2C] disabled:opacity-50 transition-all flex items-center gap-1.5"
                    >
                      <Plus className="w-3 h-3" />
                      <span>{subscribingId === item.id ? 'Adding...' : 'Subscribe'}</span>
                    </button>
                  )}
                </div>
              </div>
            ))
          )}
        </div>

        {/* Modal Footer */}
        <div className="p-3 bg-white neo-border-t flex items-center justify-between text-xs text-[#6B7280]">
          <span>
            Showing {filteredItems.length} curated list{filteredItems.length !== 1 ? 's' : ''}
          </span>
          <button
            onClick={onClose}
            className="px-4 py-1.5 border border-[#121212] bg-[#F5F1E8] hover:bg-white text-xs font-bold text-[#121212]"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
