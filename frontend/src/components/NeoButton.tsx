import React from 'react';

export type ButtonVariant = 'primary' | 'secondary' | 'accent' | 'destructive' | 'outline';

interface NeoButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: 'sm' | 'md' | 'lg';
  isLoading?: boolean;
}

export const NeoButton: React.FC<NeoButtonProps> = ({
  children,
  variant = 'primary',
  size = 'md',
  isLoading = false,
  disabled,
  className = '',
  ...props
}) => {
  const baseStyles =
    'inline-flex items-center justify-center font-bold uppercase tracking-wider rounded-lg border-2 select-none transition-all duration-100 cursor-pointer focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#33FF00] disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none active:translate-y-[1px] active:shadow-none';

  const sizeStyles = {
    sm: 'text-xs px-3 py-1.5 h-8 gap-1.5 font-mono',
    md: 'text-xs px-4 py-2 h-10 gap-2 font-mono',
    lg: 'text-sm px-5 py-2.5 h-12 gap-2.5 font-mono text-base',
  };

  const variantStyles: Record<ButtonVariant, string> = {
    primary:
      'bg-[#33FF00] text-[#14161D] border-[#29CC00] hover:bg-[#2EE600] shadow-[0_2px_8px_rgba(51,255,0,0.3)]',
    secondary:
      'bg-[#2B303D] text-[#E8E0D0] border-[#444C5E] hover:bg-[#353C4D] hover:border-[#D2B48C] shadow-[0_2px_6px_rgba(0,0,0,0.3)]',
    accent:
      'bg-[#FFB000] text-[#14161D] border-[#E09B00] hover:bg-[#EAA200] shadow-[0_2px_8px_rgba(255,176,0,0.3)]',
    destructive:
      'bg-[#CC0000] text-[#FFFFFF] border-[#B30000] hover:bg-[#B80000] shadow-[0_2px_8px_rgba(204,0,0,0.35)]',
    outline:
      'bg-transparent text-[#D2B48C] hover:bg-[#2B303D] border-[#555E72]',
  };

  return (
    <button
      disabled={disabled || isLoading}
      className={`${baseStyles} ${sizeStyles[size]} ${variantStyles[variant]} ${className}`}
      {...props}
    >
      {isLoading ? (
        <>
          <span className="inline-block w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin" />
          <span>PROCESSING...</span>
        </>
      ) : (
        children
      )}
    </button>
  );
};
