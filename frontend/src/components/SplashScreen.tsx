import React, { useState, useEffect } from 'react';
import { Shield, CheckCircle2, AlertTriangle, RefreshCw, Disc } from 'lucide-react';
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
    'MOUNTING CASSETTE DATA BANK (SQLITE)...',
    'READING MAGNETIC CACHE & DNS REWRITES...',
    'SYNCHRONIZING FILTER SUBSCRIPTIONS...',
    'CALIBRATING SAFESEARCH DNS RULES...',
    'ENGAGING PORT 53 LOOPBACK HEAD...',
  ];

  useEffect(() => {
    if (isReady && !error) {
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
      className={`fixed inset-0 z-50 flex items-center justify-center bg-[#14171E] p-6 transition-opacity duration-300 ${
        fadingOut ? 'opacity-0 pointer-events-none' : 'opacity-100'
      }`}
    >
      {/* Scanline texture */}
      <div className="absolute inset-0 crt-scanlines opacity-40 pointer-events-none" />

      <div className="w-full max-w-md bg-[#222632] border-2 border-[#3F485C] rounded-lg shadow-[0_6px_24px_rgba(0,0,0,0.6)] p-6 space-y-6 relative overflow-hidden">
        {/* Faceplate Corner Screws */}
        <div className="absolute top-2.5 left-2.5 w-1.5 h-1.5 rounded-full bg-[#4A5162] opacity-50" />
        <div className="absolute top-2.5 right-2.5 w-1.5 h-1.5 rounded-full bg-[#4A5162] opacity-50" />
        <div className="absolute bottom-2.5 left-2.5 w-1.5 h-1.5 rounded-full bg-[#4A5162] opacity-50" />
        <div className="absolute bottom-2.5 right-2.5 w-1.5 h-1.5 rounded-full bg-[#4A5162] opacity-50" />

        {/* Branding Header */}
        <div className="flex items-center gap-3.5 border-b-2 border-[#32394A] pb-4">
          <div className="w-12 h-12 rounded-lg bg-[#14161E] border-2 border-[#4A5369] flex items-center justify-center shadow-inner">
            <Disc className="w-7 h-7 text-[#33FF00] animate-reel" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="font-['VT323',monospace] text-3xl font-bold tracking-wider text-[#33FF00] glow-green leading-none">
                BLOCKADS MK-84
              </h1>
              <span className="text-[10px] font-mono font-bold bg-[#D2B48C] text-[#14161E] px-1.5 py-0.5 rounded">
                BOOT
              </span>
            </div>
            <p className="text-[11px] font-mono text-[#D2B48C] tracking-wide mt-1">
              CASSETTE FUTURISM DNS TERMINAL • {version}
            </p>
          </div>
        </div>

        {/* Error State or Initialization Progress */}
        {error ? (
          <div className="p-4 bg-[#2A0E0E] rounded-lg border-2 border-[#661A1A] space-y-3">
            <div className="flex items-center gap-2 text-xs font-mono font-bold text-[#FF5252]">
              <AlertTriangle className="w-4 h-4 shrink-0 text-[#FF5252]" />
              <span>BOOT HEAD INITIALIZATION FAILED</span>
            </div>
            <p className="text-xs text-[#E8E0D0] font-mono leading-relaxed bg-[#14161E] p-2.5 rounded border border-[#4A1F1F]">
              {error}
            </p>
            {onRetry && (
              <button
                onClick={onRetry}
                className="w-full py-2 bg-[#CC0000] text-white rounded font-mono text-xs font-bold uppercase hover:bg-[#E60000] transition-all flex items-center justify-center gap-1.5 cursor-pointer"
              >
                <RefreshCw className="w-3.5 h-3.5" />
                <span>RETRY CASSETTE BOOT</span>
              </button>
            )}
          </div>
        ) : (
          <div className="space-y-4 font-mono">
            <div className="space-y-2 bg-[#14161E] p-3 rounded-lg border border-[#2B313E] shadow-inner text-xs">
              {stages.map((stage, idx) => {
                const isCurrent = idx === stageIndex;
                const isDone = isReady || idx < stageIndex;

                return (
                  <div
                    key={stage}
                    className={`flex items-center gap-2 transition-opacity duration-200 ${
                      isDone
                        ? 'text-[#33FF00] font-bold'
                        : isCurrent
                        ? 'text-[#E8E0D0] font-black'
                        : 'text-[#555E72] opacity-50'
                    }`}
                  >
                    {isDone ? (
                      <span className="text-[#33FF00] font-bold text-xs">[OK]</span>
                    ) : isCurrent ? (
                      <span className="text-[#FFB000] font-bold text-xs animate-cursor">{'>'}</span>
                    ) : (
                      <span className="text-[#555E72] text-xs">[..]</span>
                    )}
                    <span>{stage}</span>
                  </div>
                );
              })}
            </div>

            <div className="pt-2 border-t border-[#2F3543] flex items-center justify-between text-[11px] text-[#A0AAB8]">
              <span className="font-mono">ROM 84K • ZERO CGO</span>
              <span className="font-mono text-[#33FF00] glow-green">READYING ENGINE...</span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
