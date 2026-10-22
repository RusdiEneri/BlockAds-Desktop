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
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
      <div className="bg-[#F5F1E8] neo-border neo-shadow-lg w-full max-w-xl max-h-[90vh] flex flex-col overflow-hidden">
        {/* Modal Header */}
        <div className="p-4 bg-white neo-border-b flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Plus className="w-5 h-5 text-[#121212]" />
            <h2 className="text-base font-black uppercase tracking-wider text-[#121212]">
              Add Filter Subscription from URL
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

        {/* Content Body */}
        <form onSubmit={handleConfirmSubmit} className="p-5 space-y-4 overflow-y-auto flex-1">
          {/* URL Input with Inspect Button */}
          <div>
            <label htmlFor="filter-url" className="block text-xs font-bold uppercase tracking-wider text-[#121212] mb-1">
              Source URL (HTTPS Required)
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
                className="flex-1 px-3 py-2 text-xs border-2 border-[#121212] bg-white font-mono focus:outline-hidden"
              />
              <button
                type="button"
                onClick={handleInspectURL}
                disabled={isPreviewing || !url.trim()}
                className="px-4 py-2 bg-white border-2 border-[#121212] neo-shadow-sm text-xs font-bold text-[#121212] hover:bg-[#F5F1E8] disabled:opacity-50 transition-all flex items-center gap-1.5"
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
            <span className="text-[10px] text-[#6B7280] mt-1 block">
              The engine will validate RFC 9110 HTTPS, enforce SSRF protection, and extract metadata.
            </span>
          </div>

          {/* Error Message */}
          {errorMsg && (
            <div className="p-3 bg-[#FEECEB] border-2 border-[#D9381E] text-xs text-[#D9381E] flex items-start gap-2">
              <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
              <span>{errorMsg}</span>
            </div>
          )}

          {/* Duplicate Detection Warning */}
          {preview?.isAlreadySubscribed && (
            <div className="p-3.5 bg-[#FEF9E7] border-2 border-[#B38B6D] space-y-2">
              <div className="flex items-center gap-2 text-xs font-bold text-[#7D5A38]">
                <ShieldAlert className="w-4 h-4 text-[#B38B6D]" />
                <span>Duplicate Subscription Detected</span>
              </div>
              <p className="text-xs text-[#5D4037] leading-relaxed">
                This exact source URL is already subscribed in your database as{' '}
                <strong>"{preview.existingSubscription}"</strong>.
              </p>
              {preview.existingId && (
                <button
                  type="button"
                  onClick={() => {
                    onClose();
                    onNavigateToSubscription(preview.existingId || '');
                  }}
                  className="px-3 py-1 bg-white border border-[#B38B6D] text-xs font-bold text-[#7D5A38] hover:bg-[#F5F1E8]"
                >
                  Locate Existing Subscription
                </button>
              )}
            </div>
          )}

          {/* Preview / Metadata Result Fields */}
          {preview && !preview.isAlreadySubscribed && (
            <div className="p-4 bg-white neo-border space-y-3.5">
              <div className="flex items-center justify-between">
                <span className="text-xs font-black uppercase tracking-wider text-[#121212]">
                  Extracted Metadata & Verification
                </span>
                <NeoBadge variant="success" size="sm">
                  <CheckCircle2 className="w-3 h-3" />
                  VERIFIED HTTPS SOURCE
                </NeoBadge>
              </div>

              <div className="grid grid-cols-2 gap-3 text-xs">
                <div>
                  <label htmlFor="preview-name" className="block text-[11px] font-bold text-[#6B7280] mb-1">
                    Display Name
                  </label>
                  <input
                    id="preview-name"
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    className="w-full px-2.5 py-1.5 border border-[#121212] text-xs font-bold"
                  />
                </div>

                <div>
                  <label htmlFor="preview-format" className="block text-[11px] font-bold text-[#6B7280] mb-1">
                    Detected Format
                  </label>
                  <select
                    id="preview-format"
                    value={format}
                    onChange={(e) => setFormat(e.target.value)}
                    className="w-full px-2.5 py-1.5 border border-[#121212] text-xs font-bold bg-white"
                  >
                    <option value="hosts">Hosts File (hosts)</option>
                    <option value="domains">Domain List (domains)</option>
                    <option value="adblock">Adblock Syntax (adblock)</option>
                  </select>
                </div>
              </div>

              <div>
                <label htmlFor="preview-category" className="block text-[11px] font-bold text-[#6B7280] mb-1">
                  Category
                </label>
                <input
                  id="preview-category"
                  type="text"
                  value={category}
                  onChange={(e) => setCategory(e.target.value)}
                  className="w-full px-2.5 py-1.5 border border-[#121212] text-xs"
                />
              </div>

              <div>
                <label htmlFor="preview-description" className="block text-[11px] font-bold text-[#6B7280] mb-1">
                  Description
                </label>
                <textarea
                  id="preview-description"
                  rows={2}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="w-full px-2.5 py-1.5 border border-[#121212] text-xs resize-none"
                />
              </div>

              <div className="text-[11px] text-[#6B7280] flex items-center justify-between pt-1 border-t border-[#EBE7DE]">
                <span>
                  <strong>Candidate Rule Count:</strong> ~{preview.ruleCount.toLocaleString()} domains
                </span>
                {preview.homepage && (
                  <span className="truncate max-w-[200px]">
                    <strong>Source:</strong> {preview.homepage}
                  </span>
                )}
              </div>
            </div>
          )}

          {/* Form Action Buttons */}
          <div className="pt-2 flex items-center justify-end gap-2 border-t border-[#D1CBC0]">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-[#121212] bg-[#F5F1E8] hover:bg-white text-xs font-bold text-[#121212]"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting || !name.trim() || preview?.isAlreadySubscribed}
              className="px-5 py-2 bg-[#121212] text-[#F5F1E8] neo-border text-xs font-bold hover:bg-[#2C2C2C] disabled:opacity-50 transition-all flex items-center gap-1.5"
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
