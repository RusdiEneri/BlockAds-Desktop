import React, { useState } from 'react';
import {
  X,
  Search,
  Plus,
  AlertTriangle,
  CheckCircle2,
  FileText,
  ShieldAlert,
  Loader2,
  ExternalLink,
} from 'lucide-react';
import { NeoBadge } from '../../components/NeoBadge';
import { ipc } from '../../lib/ipc';
import { models } from '../../../wailsjs/go/models';

interface AddFilterModalProps {
  isOpen: boolean;
  onClose: () => void;
  onAdd: (name: string, url: string, format: string, category: string, description: string) => Promise<void>;
  onNavigateToSubscription: (id: string) => void;
}

export const AddFilterModal: React.FC<AddFilterModalProps> = ({
  isOpen,
  onClose,
  onAdd,
  onNavigateToSubscription,
}) => {
  const [url, setUrl] = useState('');
  const [name, setName] = useState('');
  const [format, setFormat] = useState('hosts');
  const [category, setCategory] = useState('Custom');
  const [description, setDescription] = useState('');

  const [preview, setPreview] = useState<models.FilterPreviewResult | null>(null);
  const [isPreviewing, setIsPreviewing] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  if (!isOpen) return null;

  const handleInspectURL = async () => {
    const trimmed = url.trim();
    if (!trimmed) {
      setErrorMsg('Please enter a source URL.');
      return;
    }
    if (!trimmed.toLowerCase().startsWith('https://')) {
      setErrorMsg('Source URL must use secure HTTPS protocol (RFC 9110).');
      return;
    }

    setErrorMsg(null);
    setIsPreviewing(true);
    setPreview(null);

    try {
      const res = await ipc.previewFilterURL(trimmed);
      setPreview(res);
      setName(res.detectedName || '');
      setFormat(res.format || 'hosts');
      setDescription(res.description || 'Custom user subscription');
    } catch (err: any) {
      setErrorMsg(err?.message || 'Failed to inspect remote filter URL.');
    } finally {
      setIsPreviewing(false);
    }
  };

  const handleConfirmSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);

    const trimmedName = name.trim();
    const trimmedUrl = (preview?.normalizedUrl || url).trim();

    if (!trimmedName) {
      setErrorMsg('Subscription name is required.');
      return;
    }
    if (!trimmedUrl) {
      setErrorMsg('Source URL is required.');
      return;
    }

    try {
      setIsSubmitting(true);
      await onAdd(trimmedName, trimmedUrl, format, category, description);
      onClose();
    } catch (err: any) {
      setErrorMsg(err?.message || 'Failed to add filter subscription.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-xs">
      <div className="bg-[#1B1E26] border-2 border-[#33FF00]/40 rounded-lg shadow-2xl w-full max-w-xl max-h-[90vh] flex flex-col overflow-hidden">
        {/* Modal Header */}
        <div className="p-4 bg-[#232733] border-b border-[#33FF00]/30 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Plus className="w-5 h-5 text-[#33FF00]" />
            <h2 className="text-xl font-bold uppercase tracking-wider text-[#33FF00] font-['VT323'] glow-green">
              Add Filter Subscription from URL
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

        {/* Content Body */}
        <form onSubmit={handleConfirmSubmit} className="p-5 space-y-4 overflow-y-auto flex-1 font-mono">
          {/* URL Input with Inspect Button */}
          <div>
            <label htmlFor="filter-url" className="block text-xs font-bold uppercase tracking-wider text-[#D2B48C] mb-1 font-mono">
              &gt; Source URL (HTTPS Required)
            </label>
            <div className="flex gap-2">
              <input
                id="filter-url"
                type="text"
                placeholder="https://example.com/hosts.txt"
                value={url}
                onChange={(e) => {
                  setUrl(e.target.value);
                  setPreview(null);
                }}
                className="flex-1 px-3 py-2 text-xs border border-[#33FF00]/30 bg-[#151820] text-[#E8E0D0] placeholder-[#A0AAB8] rounded font-mono focus:border-[#33FF00] focus:outline-hidden"
              />
              <button
                type="button"
                onClick={handleInspectURL}
                disabled={isPreviewing || !url.trim()}
                className="px-4 py-2 bg-[#232733] border border-[#33FF00]/40 rounded text-xs font-mono font-bold text-[#33FF00] hover:bg-[#33FF00] hover:text-[#14161E] disabled:opacity-50 transition-all flex items-center gap-1.5 cursor-pointer"
              >
                {isPreviewing ? (
                  <>
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    <span>Checking...</span>
                  </>
                ) : (
                  <>
                    <Search className="w-3.5 h-3.5" />
                    <span>Inspect</span>
                  </>
                )}
              </button>
            </div>
            <span className="text-[10px] text-[#A0AAB8] mt-1 block font-mono">
              The engine will validate RFC 9110 HTTPS, enforce SSRF protection, and extract metadata.
            </span>
          </div>

          {/* Error Message */}
          {errorMsg && (
            <div className="p-3 bg-[#2D1212] border-2 border-[#802020] rounded-lg text-xs text-[#FF4D4D] flex items-start gap-2 font-mono">
              <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5 text-[#FF4D4D]" />
              <span>{errorMsg}</span>
            </div>
          )}

          {/* Duplicate Detection Warning */}
          {preview?.isAlreadySubscribed && (
            <div className="p-3.5 bg-[#2A2318] border border-[#FFB000] rounded space-y-2 font-mono">
              <div className="flex items-center gap-2 text-xs font-bold text-[#FFB000]">
                <ShieldAlert className="w-4 h-4 text-[#FFB000]" />
                <span>Duplicate Subscription Detected</span>
              </div>
              <p className="text-xs text-[#D2B48C] leading-relaxed">
                This exact source URL is already subscribed in your database as{' '}
                <strong className="text-[#33FF00]">"{preview.existingSubscription}"</strong>.
              </p>
              {preview.existingId && (
                <button
                  type="button"
                  onClick={() => {
                    onClose();
                    onNavigateToSubscription(preview.existingId || '');
                  }}
                  className="px-3 py-1 bg-[#151820] border border-[#FFB000] text-xs font-bold text-[#FFB000] hover:bg-[#FFB000] hover:text-[#14161E] rounded cursor-pointer"
                >
                  Locate Existing Subscription
                </button>
              )}
            </div>
          )}

          {/* Preview / Metadata Result Fields */}
          {preview && !preview.isAlreadySubscribed && (
            <div className="p-4 bg-[#232733] border border-[#33FF00]/30 rounded space-y-3.5">
              <div className="flex items-center justify-between">
                <span className="text-xs font-bold uppercase tracking-wider text-[#33FF00] font-mono">
                  &gt; Extracted Metadata & Verification
                </span>
                <NeoBadge variant="success" size="sm">
                  <CheckCircle2 className="w-3 h-3" />
                  VERIFIED HTTPS SOURCE
                </NeoBadge>
              </div>

              <div className="grid grid-cols-2 gap-3 text-xs">
                <div>
                  <label htmlFor="preview-name" className="block text-[11px] font-bold text-[#A0AAB8] mb-1 font-mono">
                    Display Name
                  </label>
                  <input
                    id="preview-name"
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    className="w-full px-2.5 py-1.5 border border-[#3F4657] bg-[#151820] text-[#E8E0D0] text-xs font-bold rounded font-mono focus:border-[#33FF00] focus:outline-hidden"
                  />
                </div>

                <div>
                  <label htmlFor="preview-format" className="block text-[11px] font-bold text-[#A0AAB8] mb-1 font-mono">
                    Detected Format
                  </label>
                  <select
                    id="preview-format"
                    value={format}
                    onChange={(e) => setFormat(e.target.value)}
                    className="w-full px-2.5 py-1.5 border border-[#3F4657] bg-[#151820] text-[#E8E0D0] text-xs font-bold rounded font-mono focus:border-[#33FF00] focus:outline-hidden"
                  >
                    <option value="hosts">Hosts File (hosts)</option>
                    <option value="domains">Domain List (domains)</option>
                    <option value="adblock">Adblock Syntax (adblock)</option>
                  </select>
                </div>
              </div>

              <div>
                <label htmlFor="preview-category" className="block text-[11px] font-bold text-[#A0AAB8] mb-1 font-mono">
                  Category
                </label>
                <input
                  id="preview-category"
                  type="text"
                  value={category}
                  onChange={(e) => setCategory(e.target.value)}
                  className="w-full px-2.5 py-1.5 border border-[#3F4657] bg-[#151820] text-[#E8E0D0] text-xs rounded font-mono focus:border-[#33FF00] focus:outline-hidden"
                />
              </div>

              <div>
                <label htmlFor="preview-description" className="block text-[11px] font-bold text-[#A0AAB8] mb-1 font-mono">
                  Description
                </label>
                <textarea
                  id="preview-description"
                  rows={2}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="w-full px-2.5 py-1.5 border border-[#3F4657] bg-[#151820] text-[#E8E0D0] text-xs resize-none rounded font-mono focus:border-[#33FF00] focus:outline-hidden"
                />
              </div>

              <div className="text-[11px] text-[#A0AAB8] flex items-center justify-between pt-1 border-t border-[#3F4657] font-mono">
                <span>
                  <strong className="text-[#D2B48C]">Candidate Rule Count:</strong> ~{preview.ruleCount.toLocaleString()} domains
                </span>
                {preview.homepage && (
                  <span className="truncate max-w-[200px]">
                    <strong className="text-[#D2B48C]">Source:</strong> {preview.homepage}
                  </span>
                )}
              </div>
            </div>
          )}

          {/* Form Action Buttons */}
          <div className="pt-2 flex items-center justify-end gap-2 border-t border-[#3F4657]">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-[#3F4657] bg-[#151820] hover:border-[#D2B48C] text-xs font-mono font-bold text-[#D2B48C] rounded cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting || !name.trim() || preview?.isAlreadySubscribed}
              className="px-5 py-2 bg-[#33FF00] text-[#14161E] rounded-lg text-xs font-mono font-bold hover:brightness-110 active:translate-y-[1px] disabled:opacity-50 transition-all flex items-center gap-1.5 cursor-pointer"
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  <span>Adding & Compiling...</span>
                </>
              ) : (
                <>
                  <Plus className="w-3.5 h-3.5" />
                  <span>Add Subscription</span>
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
