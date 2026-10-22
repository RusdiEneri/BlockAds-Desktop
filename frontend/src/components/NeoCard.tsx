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
      className={`bg-white neo-border neo-shadow-md rounded-[2px] overflow-hidden ${className}`}
    >
      {hasHeader && (
        <div className="flex items-center justify-between px-4 py-3 neo-border-b bg-[#FDFBF7]">
          <div>
            <div className="flex items-center gap-2">
              {title && (
                <h3 className="text-xs font-bold uppercase tracking-wider text-[#121212]">
                  {title}
                </h3>
              )}
              {badge}
            </div>
            {subtitle && (
              <p className="text-[11px] text-[#6B7280] font-medium mt-0.5">
                {subtitle}
              </p>
            )}
          </div>
          {headerAction && <div>{headerAction}</div>}
        </div>
      )}
      <div className={noPadding ? '' : 'p-4'}>{children}</div>
    </div>
  );
};
