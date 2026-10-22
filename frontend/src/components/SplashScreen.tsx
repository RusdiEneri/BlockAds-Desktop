import React, { useState, useEffect } from 'react';
import { Shield, CheckCircle2, AlertTriangle, RefreshCw } from 'lucide-react';
import { NeoBadge } from './NeoBadge';

interface SplashScreenProps {
  version?: string;
  isReady: boolean;
  error?: string | null;
  onRetry?: () => void;
  onDismiss: () => void;
}

export const SplashScreen: React.FC<SplashScreenProps> = ({
  version = 'v0.4.0',
  isReady,
  error,
  onRetry,
  onDismiss,
}) => {
  const [stageIndex, setStageIndex] = useState(0);
  const [fadingOut, setFadingOut] = useState(false);

  const stages = [
    'Connecting to local SQLite database...',
    'Restoring DNS cache and custom rewrites...',
    'Reconciling active filter subscriptions...',
    'Synchronizing SafeSearch DNS policies...',
    'Initializing local loopback DNS engine...',
  ];

  useEffect(() => {
    if (isReady && !error) {
      // Allow user to see final initialization check briefly, then fade out
      const timer = setTimeout(() => {
        setFadingOut(true);
        const dismissTimer = setTimeout(onDismiss, 350);
        return () => clearTimeout(dismissTimer);
      }, 450);
      return () => clearTimeout(timer);
    }

    if (!isReady && !error) {
      const interval = setInterval(() => {
        setStageIndex((prev) => (prev < stages.length - 1 ? prev + 1 : prev));
      }, 120);
      return () => clearInterval(interval);
    }
  }, [isReady, error, onDismiss, stages.length]);

  return (
    <div
      className={`fixed inset-0 z-50 flex items-center justify-center bg-[#F5F1E8] p-6 transition-opacity duration-300 ${
        fadingOut ? 'opacity-0 pointer-events-none' : 'opacity-100'
      }`}
    >
      <div className="w-full max-w-md bg-white neo-border neo-shadow-lg p-6 space-y-6">
        {/* Branding Header */}
        <div className="flex items-center gap-3 border-b-2 border-[#121212] pb-4">
          <div className="w-12 h-12 bg-[#121212] flex items-center justify-center neo-border">
            <Shield className="w-7 h-7 text-[#F5F1E8]" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-xl font-black uppercase tracking-wider text-[#121212]">
                BlockAds
              </h1>
              <NeoBadge variant="accent" size="sm">
                DESKTOP
              </NeoBadge>
            </div>
            <p className="text-[11px] font-bold text-[#6B7280] tracking-wide">
              Windows DNS Filtering Engine • {version}
            </p>
          </div>
        </div>

        {/* Error State or Initialization Progress */}
        {error ? (
          <div className="p-4 bg-[#FEECEB] border-2 border-[#D9381E] space-y-3">
            <div className="flex items-center gap-2 text-xs font-bold text-[#D9381E]">
              <AlertTriangle className="w-4 h-4 shrink-0" />
              <span>Startup Initialization Failed</span>
            </div>
            <p className="text-xs text-[#4B5563] font-mono leading-relaxed bg-white p-2 border border-[#121212]">
              {error}
            </p>
            {onRetry && (
              <button
                onClick={onRetry}
                className="w-full py-2 bg-[#121212] text-[#F5F1E8] neo-border text-xs font-bold uppercase hover:bg-[#2C2C2C] transition-all flex items-center justify-center gap-1.5"
              >
                <RefreshCw className="w-3.5 h-3.5" />
                <span>Retry Engine Initialization</span>
              </button>
            )}
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-2">
              {stages.map((stage, idx) => {
                const isCurrent = idx === stageIndex;
                const isDone = isReady || idx < stageIndex;

                return (
                  <div
                    key={stage}
                    className={`flex items-center gap-2 text-xs transition-opacity duration-200 ${
                      isDone
                        ? 'text-[#1E7E34] font-bold'
                        : isCurrent
                        ? 'text-[#121212] font-black'
                        : 'text-[#9CA3AF] opacity-50'
                    }`}
                  >
                    {isDone ? (
                      <CheckCircle2 className="w-3.5 h-3.5 text-[#1E7E34] shrink-0" />
                    ) : isCurrent ? (
                      <div className="w-3.5 h-3.5 border-2 border-[#121212] border-t-transparent animate-spin shrink-0" />
                    ) : (
                      <div className="w-3.5 h-3.5 border border-[#9CA3AF] shrink-0" />
                    )}
                    <span>{stage}</span>
                  </div>
                );
              })}
            </div>

            <div className="pt-2 border-t border-[#EBE7DE] flex items-center justify-between text-[11px] text-[#6B7280]">
              <span>Pure Go Core • Zero CGO</span>
              <span className="font-mono text-[#1E7E34]">Readying DNS...</span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
