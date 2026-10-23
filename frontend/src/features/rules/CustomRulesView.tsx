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
          <div className="flex items-center gap-2">
            <span className="text-[#33FF00] font-['VT323',monospace] text-3xl font-bold glow-green">
              {'>'} HARDWIRED RULES // DOMAIN OVERRIDES
            </span>
          </div>
          <p className="text-xs text-[#D2B48C] font-mono mt-0.5">
            Explicit Domain Allowlists & Blocklists (Overrides Upstream Filters)
          </p>
        </div>
        <NeoButton variant="primary" size="sm" onClick={handleOpenAddModal}>
          <Plus className="w-4 h-4 text-[#14161E]" />
          ADD CUSTOM RULE
        </NeoButton>
      </div>

      {/* Active Rules Banner */}
      <div className="bg-[#102B15] border-2 border-[#1E5C2A] rounded-lg p-3.5 flex items-start gap-3 shadow-[0_0_12px_rgba(51,255,0,0.1)]">
        <ShieldCheck className="w-5 h-5 text-[#33FF00] shrink-0 mt-0.5" />
        <div className="text-xs text-[#E8E0D0] font-mono leading-relaxed">
          <span className="font-bold text-[#33FF00]">ACTIVE PERSISTENT OVERRIDES: </span>
          Custom allow and block rules are stored in local SQLite persistence and evaluated in real-time by the DNS filtering engine.
        </div>
      </div>

      {/* Add / Edit Rule Dialog Modal */}
      {isModalOpen && (
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="modal-title"
          className="fixed inset-0 z-50 bg-black/75 flex items-center justify-center p-4"
        >
          <div className="bg-[#222632] border-2 border-[#3F485C] rounded-lg shadow-2xl max-w-md w-full p-5 space-y-4 relative">
            <div className="flex items-center justify-between border-b-2 border-[#323846] pb-3">
              <h2
                id="modal-title"
                className="text-base font-['VT323',monospace] font-bold uppercase tracking-wider text-[#33FF00] glow-green"
              >
                {editingRule ? '> EDIT CUSTOM RULE' : '> ADD CUSTOM RULE'}
              </h2>
              <button
                onClick={() => setIsModalOpen(false)}
                className="text-xs font-mono font-bold text-[#A0AAB8] hover:text-[#33FF00] px-2 py-1 cursor-pointer"
                aria-label="Close dialog"
              >
                ✕
              </button>
            </div>

            {formError && (
              <div className="bg-[#2D1212] border-2 border-[#612323] rounded-md p-2.5 text-xs text-[#FF5252] flex items-center gap-2 font-mono">
                <AlertCircle className="w-4 h-4 shrink-0" />
                <span>{formError}</span>
              </div>
            )}

            <form onSubmit={handleSubmitRule} className="space-y-4">
              <div>
                <label htmlFor="rule-domain-input" className="block text-xs font-mono font-bold uppercase tracking-wider text-[#E8E0D0] mb-1">
                  Domain Name <span className="text-[#FF4D4D]">*</span>
                </label>
                <input
                  id="rule-domain-input"
                  name="domain"
                  aria-label="Domain Name"
                  type="text"
                  placeholder="> e.g. analytics.tracker.com"
                  value={domain}
                  onChange={(e) => setDomain(e.target.value)}
                  disabled={Boolean(editingRule)}
                  className={`w-full text-xs font-mono p-2.5 rounded-lg border-2 border-[#3F4657] focus:outline-none focus:border-[#33FF00] ${
                    editingRule ? 'bg-[#181B23] text-[#A0AAB8] cursor-not-allowed' : 'bg-[#12141A] text-[#E8E0D0]'
                  }`}
                  required
                />
                <p className="text-[11px] text-[#A0AAB8] font-mono mt-1">
                  {editingRule
                    ? 'Domain name cannot be changed once created. Delete and recreate if needed.'
                    : 'Enter a naked hostname without http:// or paths.'}
                </p>
              </div>

              <div>
                <label className="block text-xs font-mono font-bold uppercase tracking-wider text-[#E8E0D0] mb-1">
                  Action Policy
                </label>
                <div className="grid grid-cols-2 gap-2 font-mono">
                  <button
                    type="button"
                    onClick={() => setAction('BLOCK')}
                    className={`py-2 text-xs font-bold uppercase rounded-lg border-2 transition-all cursor-pointer ${
                      action === 'BLOCK'
                        ? 'bg-[#2D1212] text-[#FF4D4D] border-[#802020] shadow-[0_0_8px_rgba(255,77,77,0.2)]'
                        : 'bg-[#151820] text-[#A0AAB8] border-[#3F4657] hover:border-[#FF4D4D]'
                    }`}
                  >
                    Block (0.0.0.0)
                  </button>
                  <button
                    type="button"
                    onClick={() => setAction('ALLOW')}
                    className={`py-2 text-xs font-bold uppercase rounded-lg border-2 transition-all cursor-pointer ${
                      action === 'ALLOW'
                        ? 'bg-[#102B15] text-[#33FF00] border-[#1C5927] shadow-[0_0_8px_rgba(51,255,0,0.2)]'
                        : 'bg-[#151820] text-[#A0AAB8] border-[#3F4657] hover:border-[#33FF00]'
                    }`}
                  >
                    Allow (Bypass)
                  </button>
                </div>
              </div>

              <div>
                <label htmlFor="rule-notes-input" className="block text-xs font-mono font-bold uppercase tracking-wider text-[#E8E0D0] mb-1">
                  Notes (Optional)
                </label>
                <input
                  id="rule-notes-input"
                  name="notes"
                  aria-label="Rule Notes"
                  type="text"
                  placeholder="> e.g. Whitelist internal NAS server"
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  className="w-full text-xs font-mono p-2.5 rounded-lg border-2 border-[#3F4657] bg-[#12141A] text-[#E8E0D0] focus:outline-none focus:border-[#33FF00]"
                />
              </div>

              {editingRule && (
                <div className="flex items-center gap-2 pt-1 font-mono">
                  <input
                    type="checkbox"
                    id="rule-enabled-chk"
                    checked={enabled}
                    onChange={(e) => setEnabled(e.target.checked)}
                    className="w-4 h-4 border-2 border-[#3F4657] bg-[#12141A] accent-[#33FF00] rounded"
                  />
                  <label
                    htmlFor="rule-enabled-chk"
                    className="text-xs font-bold text-[#E8E0D0] select-none cursor-pointer"
                  >
                    Rule Active &amp; Enforced
                  </label>
                </div>
              )}

              <div className="flex items-center justify-end gap-2 pt-3 border-t-2 border-[#323846]">
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
      )}      {/* Filter Tabs */}
      <div className="flex items-center justify-between flex-wrap gap-2">
        <div className="flex items-center gap-1.5">
          {(['ALL', 'BLOCK', 'ALLOW'] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setFilterAction(tab)}
              className={`px-3 py-1.5 text-xs font-mono font-bold uppercase border rounded transition-all cursor-pointer ${
                filterAction === tab
                  ? 'bg-[#33FF00] text-[#14161E] border-[#29CC00] shadow-[0_0_8px_rgba(51,255,0,0.3)]'
                  : 'bg-[#151820] text-[#D2B48C] border-[#3F4657] hover:text-[#33FF00] hover:border-[#33FF00]/40'
              }`}
            >
              {tab === 'ALL' ? `All Rules (${rules.length})` : tab}
            </button>
          ))}
        </div>
        <span className="text-xs text-[#A0AAB8] font-mono">
          Total active rules: <strong className="text-[#33FF00]">{filteredRules.length}</strong>
        </span>
      </div>

      {/* Rules Table */}
      <NeoCard noPadding>
        {filteredRules.length === 0 ? (
          <div className="p-8 text-center space-y-2">
            <FileCode className="w-8 h-8 text-[#A0AAB8] mx-auto" />
            <h4 className="text-sm font-bold uppercase tracking-wider text-[#D2B48C] font-mono">
              &gt; No custom rules found
            </h4>
            <p className="text-xs text-[#A0AAB8] font-mono">
              Click "Add Custom Rule" to create a domain blocklist or allowlist entry.
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse text-xs font-mono">
              <thead>
                <tr className="bg-[#151820] border-b-2 border-[#2E3442] font-bold uppercase tracking-wider text-[#D2B48C]">
                  <th className="py-2.5 px-4">Status</th>
                  <th className="py-2.5 px-4">Domain</th>
                  <th className="py-2.5 px-4">Action</th>
                  <th className="py-2.5 px-4">Notes</th>
                  <th className="py-2.5 px-4">Created</th>
                  <th className="py-2.5 px-4 text-right">Manage</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#3F4657]/30">
                {filteredRules.map((rule) => (
                  <tr
                    key={rule.id}
                    className={`transition-colors ${
                      rule.enabled ? 'hover:bg-[#232733]/60' : 'bg-[#151820]/40 opacity-70'
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
                        <div className="w-8 h-4 bg-[#151820] border border-[#3F4657] rounded-full peer-checked:bg-[#33FF00] transition-colors relative">
                          <div
                            className={`w-3 h-3 bg-[#D2B48C] peer-checked:bg-[#14161E] rounded-full transition-transform duration-75 absolute top-0.5 left-0.5 ${
                              rule.enabled ? 'translate-x-3.5' : 'translate-x-0'
                            }`}
                          />
                        </div>
                      </label>
                    </td>
                    <td className="py-2.5 px-4 font-mono font-bold text-[#E8E0D0]">
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
                    <td className="py-2.5 px-4 text-[#A0AAB8]">
                      {rule.notes || 'None'}
                    </td>
                    <td className="py-2.5 px-4 font-mono text-[#A0AAB8] text-[11px]">
                      {rule.createdAt}
                    </td>
                    <td className="py-2.5 px-4 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          onClick={() => handleOpenEditModal(rule)}
                          className="p-1 text-[#D2B48C] hover:text-[#33FF00] hover:bg-[#151820] border border-transparent hover:border-[#33FF00]/40 rounded transition-all cursor-pointer"
                          title="Edit rule"
                        >
                          <Edit2 className="w-3.5 h-3.5" />
                        </button>
                        <button
                          onClick={() => onDeleteRule(rule.id)}
                          className="p-1 text-[#FF4D4D] hover:text-[#FF7777] hover:bg-[#2B1B1D] border border-transparent hover:border-[#FF4D4D]/40 rounded transition-all cursor-pointer"
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
