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
    'inline-flex items-center justify-center font-bold uppercase tracking-wider border-2 border-[#121212] select-none transition-all duration-75 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#B38B6D] disabled:opacity-50 disabled:cursor-not-allowed disabled:transform-none disabled:shadow-none';

  const sizeStyles = {
    sm: 'text-xs px-2.5 py-1.5 h-8 gap-1.5',
    md: 'text-xs px-4 py-2 h-10 gap-2',
    lg: 'text-sm px-5 py-2.5 h-12 gap-2.5',
  };

  const variantStyles: Record<ButtonVariant, string> = {
    primary:
      'bg-[#121212] text-[#F5F1E8] hover:bg-[#262626] neo-shadow-hover',
    secondary:
      'bg-[#FFFFFF] text-[#121212] hover:bg-[#F5F1E8] neo-shadow-hover',
    accent:
      'bg-[#B38B6D] text-[#FFFFFF] hover:bg-[#9C775A] neo-shadow-hover',
    destructive:
      'bg-[#D9381E] text-[#FFFFFF] hover:bg-[#B92B15] neo-shadow-hover',
    outline:
      'bg-transparent text-[#121212] hover:bg-[#EBE7DE] border-[#121212]',
  };

  return (
    <button
      disabled={disabled || isLoading}
      className={`${baseStyles} ${sizeStyles[size]} ${variantStyles[variant]} ${className}`}
      {...props}
    >
      {isLoading ? (
        <>
          <span className="inline-block w-3.5 h-3.5 border-2 border-current border-t-transparent animate-spin" />
          <span>Processing...</span>
        </>
      ) : (
        children
      )}
    </button>
  );
};
