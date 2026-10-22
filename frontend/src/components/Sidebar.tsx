import React from 'react';
import {
  LayoutDashboard,
  Activity,
  Shield,
  FileCode,
  Settings,
  Terminal,
  ShieldAlert,
} from 'lucide-react';
import { NeoBadge } from './NeoBadge';

export type NavTab =
  | 'dashboard'
  | 'querylog'
  | 'filters'
  | 'rules'
  | 'settings'
  | 'diagnostics';

interface SidebarProps {
  currentTab: NavTab;
  onSelectTab: (tab: NavTab) => void;
  appVersion: string;
}

export const Sidebar: React.FC<SidebarProps> = ({
  currentTab,
  onSelectTab,
  appVersion,
}) => {
  const navItems = [
    { id: 'dashboard' as NavTab, label: 'Dashboard', icon: LayoutDashboard },
    { id: 'querylog' as NavTab, label: 'Query Log', icon: Activity },
    { id: 'filters' as NavTab, label: 'Filter Lists', icon: Shield },
    { id: 'rules' as NavTab, label: 'Custom Rules', icon: FileCode },
    { id: 'settings' as NavTab, label: 'Settings', icon: Settings },
    { id: 'diagnostics' as NavTab, label: 'Diagnostics', icon: Terminal },
  ];

  return (
    <aside className="w-64 bg-[#121212] text-[#F5F1E8] flex flex-col h-screen neo-border-r select-none shrink-0">
      {/* Brand Header */}
      <div className="p-4 neo-border-b border-[#2A2A2A] bg-[#161616]">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 bg-[#B38B6D] border-2 border-[#F5F1E8] flex items-center justify-center font-black text-black text-sm shadow-[2px_2px_0px_#F5F1E8]">
            <Shield className="w-4 h-4 text-black fill-current" />
          </div>
          <div>
            <div className="flex items-center gap-1.5">
              <span className="font-extrabold text-sm tracking-wider text-[#F5F1E8]">
                BLOCKADS
              </span>
              <span className="text-[10px] font-bold bg-[#B38B6D] text-black px-1 py-0.2 rounded-[1px]">
                DESKTOP
              </span>
            </div>
            <p className="text-[10px] text-[#9CA3AF] tracking-wide font-mono mt-0.5">
              WINDOWS DNS FILTER
            </p>
          </div>
        </div>
      </div>

      {/* Operational State Pill */}
      <div className="px-4 py-3 neo-border-b border-[#2A2A2A] bg-[#1A1A1A]">
        <div className="flex items-center justify-between mb-1.5">
          <span className="text-[10px] font-bold uppercase tracking-wider text-[#9CA3AF]">
            Filter Engine
          </span>
          <NeoBadge variant="warning" size="sm">
            MILESTONE 1
          </NeoBadge>
        </div>
        <div className="flex items-center gap-2 text-xs font-bold text-[#F5F1E8]">
          <ShieldAlert className="w-4 h-4 text-[#D97706]" />
          <span>Protection Inactive</span>
        </div>
        <p className="text-[10px] text-[#6B7280] mt-1 leading-tight">
          System network adapters are untouched.
        </p>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = currentTab === item.id;

          return (
            <button
              key={item.id}
              onClick={() => onSelectTab(item.id)}
              className={`w-full flex items-center gap-3 px-3 py-2.5 text-xs font-bold uppercase tracking-wider transition-all duration-75 text-left border-2 ${
                isActive
                  ? 'bg-[#F5F1E8] text-[#121212] border-[#F5F1E8] shadow-[2px_2px_0px_#B38B6D]'
                  : 'bg-transparent text-[#9CA3AF] border-transparent hover:text-[#F5F1E8] hover:bg-[#1E1E1E]'
              }`}
            >
              <Icon className={`w-4 h-4 ${isActive ? 'text-[#121212]' : 'text-[#6B7280]'}`} />
              <span>{item.label}</span>
            </button>
          );
        })}
      </nav>

      {/* Footer Info */}
      <div className="p-3 neo-border-t border-[#2A2A2A] bg-[#161616] text-[11px] text-[#9CA3AF]">
        <div className="flex items-center justify-between font-mono">
          <span>{appVersion || 'v0.1.0-alpha'}</span>
          <span className="text-[#6B7280]">Go + Wails v2</span>
        </div>
        <p className="text-[10px] text-[#6B7280] mt-1">
          Open-Source Windows Privacy
        </p>
      </div>
    </aside>
  );
};
