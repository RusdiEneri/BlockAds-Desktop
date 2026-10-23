import React from 'react';

export type BadgeVariant = 'success' | 'blocked' | 'warning' | 'neutral' | 'accent';

interface NeoBadgeProps {
  children: React.ReactNode;
  variant?: BadgeVariant;
  size?: 'sm' | 'md';
  className?: string;
}

export const NeoBadge: React.FC<NeoBadgeProps> = ({
  children,
  variant = 'neutral',
  size = 'md',
  className = '',
}) => {
  const variantStyles: Record<BadgeVariant, { container: string; led: string }> = {
    success: {
      container: 'bg-[#0E2614] text-[#33FF00] border-[#1C5927] shadow-[0_0_8px_rgba(51,255,0,0.2)]',
      led: 'bg-[#33FF00] shadow-[0_0_6px_#33FF00]',
    },
    blocked: {
      container: 'bg-[#290E0E] text-[#FF4D4D] border-[#5E1A1A] shadow-[0_0_8px_rgba(255,77,77,0.2)]',
      led: 'bg-[#CC0000] shadow-[0_0_6px_#FF4D4D]',
    },
    warning: {
      container: 'bg-[#2B1F0A] text-[#FFB000] border-[#614510] shadow-[0_0_8px_rgba(255,176,0,0.2)]',
      led: 'bg-[#FFB000] shadow-[0_0_6px_#FFB000]',
    },
    neutral: {
      container: 'bg-[#1E222B] text-[#A0AAB8] border-[#3F4657]',
      led: 'bg-[#A0AAB8]',
    },
    accent: {
      container: 'bg-[#2B241A] text-[#D2B48C] border-[#5E4B31] shadow-[0_0_8px_rgba(210,180,140,0.15)]',
      led: 'bg-[#D2B48C] shadow-[0_0_6px_#D2B48C]',
    },
  };

  const sizeStyles = {
    sm: 'text-[10px] px-2 py-0.5 tracking-wider font-mono font-bold',
    md: 'text-[11px] px-2.5 py-1 tracking-wider font-mono font-bold',
  };

  const style = variantStyles[variant];

  return (
    <span
      role="status"
      className={`inline-flex items-center gap-1.5 border rounded-md uppercase select-none ${style.container} ${sizeStyles[size]} ${className}`}
    >
      <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${style.led}`} />
      <span>{children}</span>
    </span>
  );
};
