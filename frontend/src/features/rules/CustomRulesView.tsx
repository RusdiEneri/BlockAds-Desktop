import React, { useState, useEffect } from 'react';
import {
  FileCode,
  Plus,
  Trash2,
  Edit2,
  CheckCircle2,
  XCircle,
  AlertCircle,
  Info,
  ShieldCheck,
} from 'lucide-react';
import { NeoCard } from '../../components/NeoCard';
import { NeoButton } from '../../components/NeoButton';
import { NeoBadge } from '../../components/NeoBadge';
import { models } from '../../../wailsjs/go/models';

interface CustomRulesViewProps {
  rules: models.CustomRule[];
  onAddRule: (domain: string, action: string, notes: string) => Promise<void>;
  onUpdateRule: (
    id: string,
    action: string,
    notes: string,
    enabled: boolean
  ) => Promise<void>;
  onToggleRule: (id: string, enabled: boolean) => Promise<void>;
  onDeleteRule: (id: string) => Promise<void>;
}

export const CustomRulesView: React.FC<CustomRulesViewProps> = ({
  rules,
  onAddRule,
  onUpdateRule,
  onToggleRule,
  onDeleteRule,
}) => {
  const [filterAction, setFilterAction] = useState<'ALL' | 'BLOCK' | 'ALLOW'>('ALL');
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<models.CustomRule | null>(null);

  // Form states
  const [domain, setDomain] = useState('');
  const [action, setAction] = useState<'BLOCK' | 'ALLOW'>('BLOCK');
  const [notes, setNotes] = useState('');
  const [enabled, setEnabled] = useState(true);
  const [formError, setFormError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Handle ESC key to close modal
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isModalOpen) {
        setIsModalOpen(false);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isModalOpen]);

  const validateDomainInput = (val: string): string | null => {
    const trimmed = val.trim();
    if (!trimmed) return 'Domain cannot be empty';
    if (trimmed.includes('://')) return 'Domain must not include protocol (e.g., http:// or https://)';
    if (trimmed.includes('/')) return 'Domain must not contain URL path slashes';
    if (trimmed.includes(' ')) return 'Domain must not contain spaces';
    if (trimmed.includes('*')) return 'Wildcards (*) are not supported in standard domain rules';
    if (!trimmed.includes('.') && trimmed !== 'localhost') {
      return 'Domain must include a valid top-level domain (e.g., example.com)';
    }
    return null;
  };

  const handleOpenAddModal = () => {
    setEditingRule(null);
    setDomain('');
    setAction('BLOCK');
    setNotes('');
    setEnabled(true);
    setFormError(null);
    setIsModalOpen(true);
  };

  const handleOpenEditModal = (rule: models.CustomRule) => {
    setEditingRule(rule);
    setDomain(rule.domain);
    setAction(rule.action as 'BLOCK' | 'ALLOW');
    setNotes(rule.notes || '');
    setEnabled(rule.enabled);
    setFormError(null);
    setIsModalOpen(true);
  };

  const handleSubmitRule = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);
    setIsSubmitting(true);

    try {
      if (editingRule) {
        await onUpdateRule(editingRule.id, action, notes.trim(), enabled);
      } else {
        const validationError = validateDomainInput(domain);
        if (validationError) {
          setFormError(validationError);
          setIsSubmitting(false);
          return;
        }
        await onAddRule(domain.trim(), action, notes.trim());
      }
      setIsModalOpen(false);
    } catch (err: any) {
      setFormError(err?.message || String(err));
    } finally {
      setIsSubmitting(false);
    }
  };

  const filteredRules = rules.filter((r) => {
    if (filterAction === 'ALL') return true;
    return r.action === filterAction;
  });

  return (
    <div className="space-y-5">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-black uppercase tracking-wider text-[#121212]">
            Custom Domain Rules
          </h1>
          <p className="text-xs text-[#4B5563] mt-0.5">
            Explicit Domain Allowlists & Blocklists (Overrides Upstream Filters)
          </p>
        </div>
        <NeoButton variant="primary" size="sm" onClick={handleOpenAddModal}>
          <Plus className="w-4 h-4" />
          Add Custom Rule
        </NeoButton>
      </div>

      {/* Active Rules Banner */}
      <div className="bg-[#EBFBEE] neo-border p-3.5 flex items-start gap-3">
        <ShieldCheck className="w-5 h-5 text-[#1E7E34] shrink-0 mt-0.5" />
        <div className="text-xs text-[#1E7E34] leading-relaxed">
          <span className="font-bold">Active Persistent Rules: </span>
          Custom allow and block rules are stored in local SQLite persistence and evaluated in real-time by the DNS filtering engine.
        </div>
      </div>

      {/* Add / Edit Rule Dialog Modal */}
      {isModalOpen && (
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="modal-title"
          className="fixed inset-0 z-50 bg-black/60 flex items-center justify-center p-4"
        >
          <div className="bg-white neo-border neo-shadow-lg max-w-md w-full p-5 space-y-4">
            <div className="flex items-center justify-between border-b-2 border-[#121212] pb-3">
              <h2
                id="modal-title"
                className="text-xs font-bold uppercase tracking-wider text-[#121212]"
              >
                {editingRule ? 'Edit Custom Domain Rule' : 'Add Custom Domain Rule'}
              </h2>
              <button
                onClick={() => setIsModalOpen(false)}
                className="text-xs font-bold text-[#6B7280] hover:text-[#121212] px-2 py-1"
                aria-label="Close dialog"
              >
                ✕
              </button>
            </div>

            {formError && (
              <div className="bg-[#FEECEB] border-2 border-[#D9381E] p-2.5 text-xs text-[#D9381E] flex items-center gap-2 font-medium">
                <AlertCircle className="w-4 h-4 shrink-0" />
                <span>{formError}</span>
              </div>
            )}

            <form onSubmit={handleSubmitRule} className="space-y-4">
              <div>
                <label htmlFor="rule-domain-input" className="block text-xs font-bold uppercase tracking-wider text-[#121212] mb-1">
                  Domain Name <span className="text-[#D9381E]">*</span>
                </label>
                <input
                  id="rule-domain-input"
                  name="domain"
                  aria-label="Domain Name"
                  type="text"
                  placeholder="e.g. analytics.tracker.com"
                  value={domain}
                  onChange={(e) => setDomain(e.target.value)}
                  disabled={Boolean(editingRule)}
                  className={`w-full text-xs font-mono p-2.5 border-2 border-[#121212] focus:outline-none focus:shadow-[2px_2px_0px_#B38B6D] ${
                    editingRule ? 'bg-[#EBE7DE] text-[#6B7280] cursor-not-allowed' : 'bg-white'
                  }`}
                  required
                />
                <p className="text-[11px] text-[#6B7280] mt-1">
                  {editingRule
                    ? 'Domain name cannot be changed once created. Delete and recreate if needed.'
                    : 'Enter a naked hostname without http:// or paths.'}
                </p>
              </div>

              <div>
                <label className="block text-xs font-bold uppercase tracking-wider text-[#121212] mb-1">
                  Action Policy
                </label>
                <div className="grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    onClick={() => setAction('BLOCK')}
                    className={`py-2 text-xs font-bold uppercase border-2 transition-all ${
                      action === 'BLOCK'
                        ? 'bg-[#FEECEB] text-[#D9381E] border-[#D9381E] shadow-[2px_2px_0px_#D9381E]'
                        : 'bg-white text-[#4B5563] border-[#121212]'
                    }`}
                  >
                    Block (0.0.0.0)
                  </button>
                  <button
                    type="button"
                    onClick={() => setAction('ALLOW')}
                    className={`py-2 text-xs font-bold uppercase border-2 transition-all ${
                      action === 'ALLOW'
                        ? 'bg-[#EBFBEE] text-[#1E7E34] border-[#1E7E34] shadow-[2px_2px_0px_#1E7E34]'
                        : 'bg-white text-[#4B5563] border-[#121212]'
                    }`}
                  >
                    Allow (Bypass)
                  </button>
                </div>
              </div>

              <div>
                <label htmlFor="rule-notes-input" className="block text-xs font-bold uppercase tracking-wider text-[#121212] mb-1">
                  Notes (Optional)
                </label>
                <input
                  id="rule-notes-input"
                  name="notes"
                  aria-label="Rule Notes"
                  type="text"
                  placeholder="e.g. Whitelist internal NAS server"
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  className="w-full text-xs p-2 border-2 border-[#121212] focus:outline-none focus:shadow-[2px_2px_0px_#B38B6D]"
                />
              </div>

              {editingRule && (
                <div className="flex items-center gap-2 pt-1">
                  <input
                    type="checkbox"
                    id="rule-enabled-chk"
                    checked={enabled}
                    onChange={(e) => setEnabled(e.target.checked)}
                    className="w-4 h-4 border-2 border-[#121212] accent-[#121212]"
                  />
                  <label
                    htmlFor="rule-enabled-chk"
                    className="text-xs font-bold text-[#121212] select-none cursor-pointer"
                  >
                    Rule Active & Enforced
                  </label>
                </div>
              )}

              <div className="flex items-center justify-end gap-2 pt-2 border-t border-[#EBE7DE]">
                <NeoButton
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => setIsModalOpen(false)}
                >
                  Cancel
                </NeoButton>
                <NeoButton
                  type="submit"
                  variant="primary"
                  size="sm"
                  isLoading={isSubmitting}
                >
                  {editingRule ? 'Update Rule' : 'Save Rule'}
                </NeoButton>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Filter Tabs */}
      <div className="flex items-center justify-between flex-wrap gap-2">
        <div className="flex items-center gap-1.5">
          {(['ALL', 'BLOCK', 'ALLOW'] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setFilterAction(tab)}
              className={`px-3 py-1.5 text-xs font-bold uppercase border-2 transition-all ${
                filterAction === tab
                  ? 'bg-[#121212] text-[#F5F1E8] border-[#121212] shadow-[2px_2px_0px_#B38B6D]'
                  : 'bg-white text-[#4B5563] border-[#121212] hover:bg-[#F5F1E8]'
              }`}
            >
              {tab === 'ALL' ? `All Rules (${rules.length})` : tab}
            </button>
          ))}
        </div>
        <span className="text-xs text-[#6B7280]">
          Total active rules: <strong className="text-[#121212]">{filteredRules.length}</strong>
        </span>
      </div>

      {/* Rules Table */}
      <NeoCard noPadding>
        {filteredRules.length === 0 ? (
          <div className="p-8 text-center space-y-2">
            <FileCode className="w-8 h-8 text-[#9CA3AF] mx-auto" />
            <h4 className="text-xs font-bold uppercase tracking-wider text-[#121212]">
              No custom rules found
            </h4>
            <p className="text-xs text-[#6B7280]">
              Click "Add Custom Rule" to create a domain blocklist or allowlist entry.
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse text-xs">
              <thead>
                <tr className="bg-[#F5F1E8] neo-border-b font-bold uppercase tracking-wider text-[#4B5563]">
                  <th className="py-2.5 px-4">Status</th>
                  <th className="py-2.5 px-4">Domain</th>
                  <th className="py-2.5 px-4">Action</th>
                  <th className="py-2.5 px-4">Notes</th>
                  <th className="py-2.5 px-4">Created</th>
                  <th className="py-2.5 px-4 text-right">Manage</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#EBE7DE]">
                {filteredRules.map((rule) => (
                  <tr
                    key={rule.id}
                    className={`transition-colors ${
                      rule.enabled ? 'hover:bg-[#FDFBF7]' : 'bg-[#F9F7F2] opacity-75'
                    }`}
                  >
                    <td className="py-2.5 px-4">
                      <label htmlFor={`rule-toggle-${rule.id}`} className="relative inline-flex items-center cursor-pointer select-none">
                        <input
                          id={`rule-toggle-${rule.id}`}
                          name={`rule-toggle-${rule.id}`}
                          aria-label={`Toggle rule for ${rule.domain}`}
                          type="checkbox"
                          checked={rule.enabled}
                          onChange={(e) => onToggleRule(rule.id, e.target.checked)}
                          className="sr-only peer"
                        />
                        <div className="w-9 h-5 bg-[#EBE7DE] border-2 border-[#121212] peer-checked:bg-[#1E7E34] transition-colors relative">
                          <div
                            className={`w-3 h-3 bg-white border border-[#121212] transition-transform duration-75 absolute top-0.5 left-0.5 ${
                              rule.enabled ? 'translate-x-4' : 'translate-x-0'
                            }`}
                          />
                        </div>
                      </label>
                    </td>
                    <td className="py-2.5 px-4 font-mono font-bold text-[#121212]">
                      {rule.domain}
                    </td>
                    <td className="py-2.5 px-4">
                      {rule.action === 'BLOCK' ? (
                        <NeoBadge variant="blocked" size="sm">
                          <XCircle className="w-3 h-3" />
                          BLOCK
                        </NeoBadge>
                      ) : (
                        <NeoBadge variant="success" size="sm">
                          <CheckCircle2 className="w-3 h-3" />
                          ALLOW
                        </NeoBadge>
                      )}
                    </td>
                    <td className="py-2.5 px-4 text-[#4B5563]">
                      {rule.notes || 'None'}
                    </td>
                    <td className="py-2.5 px-4 font-mono text-[#6B7280] text-[11px]">
                      {rule.createdAt}
                    </td>
                    <td className="py-2.5 px-4 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          onClick={() => handleOpenEditModal(rule)}
                          className="p-1 text-[#4B5563] hover:text-[#121212] hover:bg-[#EBE7DE] border border-transparent hover:border-[#121212] transition-all"
                          title="Edit rule"
                        >
                          <Edit2 className="w-3.5 h-3.5" />
                        </button>
                        <button
                          onClick={() => onDeleteRule(rule.id)}
                          className="p-1 text-[#D9381E] hover:bg-[#FEECEB] border border-transparent hover:border-[#D9381E] transition-all"
                          title="Delete rule"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
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
