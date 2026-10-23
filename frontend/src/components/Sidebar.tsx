import React from 'react';
import {
  LayoutDashboard,
  Activity,
  Shield,
  FileCode,
  Settings,
  Terminal,
  ShieldAlert,
  ShieldCheck,
  Power,
  Disc,
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
  isProtected?: boolean;
  onToggleProtection?: (enabled: boolean) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
  currentTab,
  onSelectTab,
  appVersion,
  isProtected = true,
  onToggleProtection,
}) => {
  const navItems = [
    { id: 'dashboard' as NavTab, label: '01. DASHBOARD', icon: LayoutDashboard },
    { id: 'querylog' as NavTab, label: '02. QUERY LOG', icon: Activity },
    { id: 'filters' as NavTab, label: '03. FILTER LISTS', icon: Shield },
    { id: 'rules' as NavTab, label: '04. CUSTOM RULES', icon: FileCode },
    { id: 'settings' as NavTab, label: '05. SETTINGS', icon: Settings },
    { id: 'diagnostics' as NavTab, label: '06. DIAGNOSTICS', icon: Terminal },
  ];

  return (
    <aside className="w-64 bg-[#191C24] text-[#E8E0D0] flex flex-col h-screen border-r-2 border-[#373E4F] select-none shrink-0 relative">
      {/* Brand Header with Animated Cassette Spools */}
      <div className="p-4 border-b-2 border-[#2F3543] bg-[#14161E]">
        <div className="flex items-center gap-2.5">
          <div className="w-9 h-9 rounded-lg bg-[#242936] border-2 border-[#4A5369] flex items-center justify-center text-[#33FF00] shadow-[inset_0_2px_4px_rgba(0,0,0,0.5)]">
            <Disc className={`w-5 h-5 ${isProtected ? 'animate-reel text-[#33FF00]' : 'text-[#6C768D]'}`} />
          </div>
          <div>
            <div className="flex items-center gap-1.5">
              <span className="font-['VT323',monospace] text-2xl font-bold tracking-wider text-[#33FF00] glow-green leading-none">
                BLOCKADS
              </span>
              <span className="text-[10px] font-mono font-bold bg-[#D2B48C] text-[#14161E] px-1 py-0.2 rounded">
                MK-84
              </span>
            </div>
            <p className="text-[10px] text-[#A0AAB8] tracking-wider font-mono mt-0.5">
              ANALOG-DIGITAL DNS
            </p>
          </div>
        </div>

        {/* Cassette Tape Deck Micro Window */}
        <div className="mt-3 bg-[#0E1015] border-2 border-[#303644] rounded-lg p-2 relative overflow-hidden shadow-inner">
          <div className="flex items-center justify-between px-2">
            {/* Left Spool */}
            <div className="flex flex-col items-center">
              <div
                className={`w-6 h-6 rounded-full border-2 border-[#4A5368] bg-[#1E232F] flex items-center justify-center relative ${
                  isProtected ? 'animate-reel' : ''
                }`}
              >
                <div className="w-2 h-2 rounded-full bg-[#0E1015]" />
                <div className="absolute w-full h-0.5 bg-[#4A5368]" />
                <div className="absolute h-full w-0.5 bg-[#4A5368]" />
              </div>
            </div>

            {/* Tape Header Label */}
            <div className="text-center">
              <div className="text-[9px] font-mono uppercase tracking-widest text-[#D2B48C]">
                {isProtected ? 'TAPE 01 // SIDE A' : 'TAPE PAUSED'}
              </div>
              <div className="w-16 h-1 bg-[#CC0000] mx-auto my-1 rounded-full opacity-80" />
              <div className="text-[8px] font-mono text-[#A0AAB8]">
                {isProtected ? 'PLAYING 4.75 CM/S' : 'MOTOR STOPPED'}
              </div>
            </div>

            {/* Right Spool */}
            <div className="flex flex-col items-center">
              <div
                className={`w-6 h-6 rounded-full border-2 border-[#4A5368] bg-[#1E232F] flex items-center justify-center relative ${
                  isProtected ? 'animate-reel-reverse' : ''
                }`}
              >
                <div className="w-2 h-2 rounded-full bg-[#0E1015]" />
                <div className="absolute w-full h-0.5 bg-[#4A5368]" />
                <div className="absolute h-full w-0.5 bg-[#4A5368]" />
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Operational State Panel */}
      <div className="px-4 py-3 border-b-2 border-[#2F3543] bg-[#181B23]">
        <div className="flex items-center justify-between mb-1.5">
          <span className="text-[10px] font-bold uppercase tracking-wider text-[#A0AAB8] font-mono">
            HEAD OPERATION
          </span>
          <NeoBadge variant={isProtected ? 'success' : 'warning'} size="sm">
            {isProtected ? 'REC / RUN' : 'STANDBY'}
          </NeoBadge>
        </div>
        <div className="flex items-center justify-between gap-2 mt-1">
          <div className="flex items-center gap-2 text-xs font-bold text-[#E8E0D0] font-mono">
            {isProtected ? (
              <ShieldCheck className="w-4 h-4 text-[#33FF00]" />
            ) : (
              <ShieldAlert className="w-4 h-4 text-[#FFB000]" />
            )}
            <span>{isProtected ? 'LOOPBACK ACTIVE' : 'SHIELD MUTED'}</span>
          </div>
          {onToggleProtection && (
            <button
              onClick={() => onToggleProtection(!isProtected)}
              title={isProtected ? 'Pause Engine' : 'Engage Engine'}
              className={`p-1.5 rounded border text-xs font-mono font-bold flex items-center gap-1 transition-all cursor-pointer focus-visible:outline-2 focus-visible:outline-[#33FF00] ${
                isProtected
                  ? 'bg-[#103816] text-[#33FF00] border-[#1C5927] hover:bg-[#154A1E]'
                  : 'bg-[#38260A] text-[#FFB000] border-[#664610] hover:bg-[#4D330E]'
              }`}
            >
              <Power className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
        <p className="text-[10px] text-[#A0AAB8] font-mono mt-1 leading-tight">
          {isProtected ? 'Intercepting 127.0.0.1:53' : 'Bypassing filter engine'}
        </p>
      </div>

      {/* Navigation Buttons */}
      <nav className="flex-1 px-3 py-4 space-y-1.5 overflow-y-auto">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = currentTab === item.id;

          return (
            <button
              key={item.id}
              onClick={() => onSelectTab(item.id)}
              className={`w-full flex items-center gap-3 px-3 py-2.5 text-xs font-mono font-bold uppercase tracking-wider transition-all duration-100 text-left rounded-lg border-2 cursor-pointer focus-visible:outline-2 focus-visible:outline-[#33FF00] ${
                isActive
                  ? 'bg-[#242A36] text-[#33FF00] border-[#3F485C] shadow-[0_2px_8px_rgba(51,255,0,0.2)] glow-green'
                  : 'bg-transparent text-[#A0AAB8] border-transparent hover:text-[#D2B48C] hover:bg-[#1E232E]'
              }`}
            >
              <Icon className={`w-4 h-4 ${isActive ? 'text-[#33FF00]' : 'text-[#6C768D]'}`} />
              <span>{item.label}</span>
            </button>
          );
        })}
      </nav>

      {/* Footer Info / Tape Counter */}
      <div className="p-3 border-t-2 border-[#2F3543] bg-[#14161E] text-[11px] text-[#A0AAB8] font-mono">
        <div className="flex items-center justify-between">
          <span className="text-[#D2B48C] font-bold">CTR 0428.1</span>
          <span className="text-[#6C768D]">{appVersion || 'v0.1.0-alpha'}</span>
        </div>
        <p className="text-[10px] text-[#6C768D] mt-0.5">
          CASSETTE FUTURISM OS
        </p>
      </div>
    </aside>
  );
};
