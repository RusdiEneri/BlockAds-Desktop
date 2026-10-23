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
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-xs">
      <div className="bg-[#1B1E26] border-2 border-[#33FF00]/40 rounded-lg shadow-2xl w-full max-w-3xl max-h-[90vh] flex flex-col overflow-hidden">
        {/* Modal Header */}
        <div className="p-4 bg-[#232733] border-b border-[#33FF00]/30 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <BookOpen className="w-5 h-5 text-[#33FF00]" />
            <h2 className="text-xl font-bold uppercase tracking-wider text-[#33FF00] font-['VT323'] glow-green">
              Curated Filter Catalog
            </h2>
          </div>
          <button
            onClick={onClose}
            aria-label="Close modal"
            className="p-1.5 border border-[#33FF00]/40 bg-[#151820] text-[#D2B48C] hover:text-[#33FF00] hover:border-[#33FF00] rounded transition-all cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Search & Category Filter Controls */}
        <div className="p-4 space-y-3 bg-[#1F232D] border-b border-[#3F4657]">
          <div className="relative">
            <Search className="w-4 h-4 absolute left-3 top-3 text-[#A0AAB8]" />
            <input
              type="text"
              placeholder="Search catalog by name, maintainer, or description..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-3 py-2 text-xs border border-[#33FF00]/30 bg-[#151820] text-[#E8E0D0] placeholder-[#A0AAB8] rounded font-mono focus:border-[#33FF00] focus:outline-hidden"
            />
          </div>

          <div className="flex flex-wrap gap-1.5 items-center">
            <span className="text-[11px] font-bold uppercase tracking-wider text-[#A0AAB8] font-mono mr-1 flex items-center gap-1">
              <Filter className="w-3 h-3" /> Category:
            </span>
            {categories.map((cat) => (
              <button
                key={cat}
                onClick={() => setSelectedCategory(cat)}
                className={`px-2.5 py-1 text-[11px] font-mono font-bold uppercase tracking-wider border rounded transition-all cursor-pointer ${
                  selectedCategory === cat
                    ? 'bg-[#33FF00] text-[#14161E] border-[#29CC00] font-bold'
                    : 'bg-[#151820] text-[#D2B48C] border-[#3F4657] hover:border-[#33FF00]/50 hover:text-[#33FF00]'
                }`}
              >
                {cat}
              </button>
            ))}
          </div>

          {errorMsg && (
            <div className="p-2.5 bg-[#2B1B1D] border border-[#CC0000] text-xs text-[#FF8080] font-mono rounded">
              {errorMsg}
            </div>
          )}
        </div>

        {/* Scrollable Catalog Grid */}
        <div className="p-4 overflow-y-auto space-y-3 flex-1">
          {filteredItems.length === 0 ? (
            <div className="p-8 text-center text-xs text-[#A0AAB8] font-mono bg-[#151820] border border-[#3F4657] rounded">
              No verified filter lists match your search criteria.
            </div>
          ) : (
            filteredItems.map((item) => (
              <div
                key={item.id}
                className="p-3.5 bg-[#232733] border border-[#3F4657] hover:border-[#33FF00]/40 rounded flex flex-col sm:flex-row sm:items-center justify-between gap-3 transition-colors"
              >
                <div className="space-y-1.5 flex-1">
                  <div className="flex items-center flex-wrap gap-2">
                    <span className="text-sm font-bold text-[#E8E0D0] font-mono">{item.name}</span>
                    <NeoBadge variant="neutral" size="sm">
                      {item.format.toUpperCase()}
                    </NeoBadge>
                    <NeoBadge variant="accent" size="sm">
                      {item.category}
                    </NeoBadge>
                  </div>

                  <p className="text-xs text-[#A0AAB8] leading-relaxed font-mono">{item.description}</p>

                  <div className="flex items-center gap-4 text-[11px] text-[#A0AAB8] font-mono pt-0.5">
                    <span>
                      <strong className="text-[#D2B48C]">Maintainer:</strong> {item.maintainer}
                    </span>
                    <span>
                      <strong className="text-[#D2B48C]">License:</strong> {item.license}
                    </span>
                    {item.homepage && (
                      <a
                        href={item.homepage}
                        target="_blank"
                        rel="noreferrer"
                        className="text-[#33FF00] hover:underline flex items-center gap-0.5"
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
                          className="text-[11px] font-bold text-[#33FF00] hover:underline font-mono"
                        >
                          View
                        </button>
                      )}
                    </div>
                  ) : (
                    <button
                      onClick={() => handleSubscribeClick(item.id)}
                      disabled={subscribingId === item.id}
                      className="px-3.5 py-1.5 bg-[#33FF00] text-[#14161E] text-xs font-bold font-mono rounded hover:brightness-110 active:translate-y-[1px] disabled:opacity-50 transition-all flex items-center gap-1.5 cursor-pointer"
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
        <div className="p-3 bg-[#232733] border-t border-[#33FF00]/30 flex items-center justify-between text-xs text-[#A0AAB8] font-mono">
          <span>
            Showing {filteredItems.length} curated list{filteredItems.length !== 1 ? 's' : ''}
          </span>
          <button
            onClick={onClose}
            className="px-4 py-1.5 border border-[#3F4657] bg-[#151820] hover:border-[#D2B48C] text-xs font-mono font-bold text-[#D2B48C] rounded cursor-pointer"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
