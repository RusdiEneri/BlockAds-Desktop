import React from 'react';

interface NeoCardProps {
  children: React.ReactNode;
  title?: string;
  subtitle?: string;
  badge?: React.ReactNode;
  headerAction?: React.ReactNode;
  className?: string;
  noPadding?: boolean;
}

export const NeoCard: React.FC<NeoCardProps> = ({
  children,
  title,
  subtitle,
  badge,
  headerAction,
  className = '',
  noPadding = false,
}) => {
  const hasHeader = Boolean(title || badge || headerAction);

  return (
    <div
      className={`bg-[#242833] border-2 border-[#3F4657] rounded-lg shadow-[0_4px_16px_rgba(0,0,0,0.45)] overflow-hidden ${className}`}
    >
      {hasHeader && (
        <div className="flex items-center justify-between px-4 py-2.5 border-b-2 border-[#323846] bg-[#1B1E27] relative">
          {/* Subtle cockpit screw accents on faceplate */}
          <div className="absolute top-2 left-2 w-1.5 h-1.5 rounded-full bg-[#4A5162] opacity-60" />
          <div className="absolute top-2 right-2 w-1.5 h-1.5 rounded-full bg-[#4A5162] opacity-60" />

          <div className="pl-3">
            <div className="flex items-center gap-2">
              {title && (
                <h3 className="font-bold uppercase tracking-wider text-[#E8E0D0] font-['VT323',monospace] text-base">
                  <span className="text-[#33FF00] mr-1.5 font-mono text-xs">{'>'}</span>
                  {title}
                </h3>
              )}
              {badge}
            </div>
            {subtitle && (
              <p className="text-[11px] text-[#A0AAB8] font-mono mt-0.5">
                {subtitle}
              </p>
            )}
          </div>
          {headerAction && <div className="pr-3">{headerAction}</div>}
        </div>
      )}
      <div className={noPadding ? '' : 'p-4'}>{children}</div>
    </div>
  );
};
