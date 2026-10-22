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
  const variantStyles: Record<BadgeVariant, string> = {
    success: 'bg-[#EBFBEE] text-[#1E7E34] border-[#121212]',
    blocked: 'bg-[#FEECEB] text-[#D9381E] border-[#121212]',
    warning: 'bg-[#FEF3C7] text-[#92400E] border-[#121212]',
    neutral: 'bg-[#EBE7DE] text-[#4B5563] border-[#121212]',
    accent: 'bg-[#F2ECE4] text-[#B38B6D] border-[#121212]',
  };

  const sizeStyles = {
    sm: 'text-[11px] px-1.5 py-0.5 tracking-wider font-semibold',
    md: 'text-[12px] px-2.5 py-1 tracking-wide font-bold',
  };

  return (
    <span
      role="status"
      className={`inline-flex items-center gap-1.5 border-2 uppercase select-none rounded-[2px] ${variantStyles[variant]} ${sizeStyles[size]} ${className}`}
    >
      {children}
    </span>
  );
};
