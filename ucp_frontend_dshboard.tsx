import React, { useState, useEffect, useRef } from 'react';
import { Send, Link2, Globe, Shield, Zap, MessageSquare, Users, Settings, Menu, X, Check, AlertCircle, Activity, Database, Lock, Signal, Wifi, WifiOff, Plus, Search, Filter, MoreVertical, Download, Upload, ChevronRight, Bell, User, LogOut, Power, RefreshCw } from 'lucide-react';

// IP: Julius Cameron Hill - Universal Communication Protocol Dashboard
// Enterprise Production Build

const API_BASE = 'http://localhost:8080/api';
const WS_URL = 'ws://localhost:8080/ws';

const themes = {
  slate: {
    primary: 'from-slate-600 to-slate-800',
    secondary: 'from-slate-700 to-slate-900',
    accent: 'slate-500',
    bg: 'from-slate-950 via-slate-900 to-slate-950'
  },
  blue: {
    primary: 'from-blue-600 to-blue-800',
    secondary: 'from-blue-700 to-blue-900',
    accent: 'blue-500',
    bg: 'from-slate-950 via-blue-950 to-slate-950'
  },
  emerald: {
    primary: 'from-emerald-600 to-emerald-800',
    secondary: 'from-emerald-700 to-emerald-900',
    accent: 'emerald-500',
    bg: 'from-slate-950 via-emerald-950 to-slate-950'
  },
  violet: {
    primary: 'from-violet-600 to-violet-800',
    secondary: 'from-violet-700 to-violet-900',
    accent: 'violet-500',
    bg: 'from-slate-950 via-violet-950 to-slate-950'
  },
  amber: {
    primary: 'from-amber-600 to-amber-800',
    secondary: 'from-amber-700 to-amber-900',
    accent: 'amber-500',
    bg: 'from-slate-950 via-amber-950 to-slate-950'
  },
  rose: {
    primary: 'from-rose-600 to-rose-800',
    secondary: 'from-rose-700 to-rose-900',
    accent: 'rose-500',
    bg: 'from-slate-950 via-rose-950 to-slate-950'
  }
};

const platformConfig = {
  native: {
    name: 'Native Protocol',
    icon: Signal,
    color: 'blue-500',
    bgGradient: 'from-blue-500/10 to-cyan-500/10'
  },
  telegram: {
    name: 'Telegram',
    icon: Send,
    color: 'sky-400',
    bgGradient: 'from-sky-500/10 to-blue-500/10'
  },
  discord: {
    name: 'Discord',
    icon: MessageSquare,
    color: 'indigo-400',
    bgGradient: 'from-indigo-500/10 to-purple-500/10'
  },
  whatsapp: {
    name: 'WhatsApp',
    icon: MessageSquare,
    color: 'green-400',
    bgGradient: 'from-green-500/10 to-emerald-500/10'
  },
  meet: {
    name: 'Google Meet',
    icon: Users,
    color: 'yellow-400',
    bgGradient: 'from-yellow-500/10 to-orange-500/10'
  },
  zoom: {
    name: 'Zoom',
    icon: Users,
    color: 'blue-500',
    bgGradient: 'from-blue-600/10 to-indigo-600/10'
  },
  messenger: {
    name: 'Messenger',
    icon: MessageSquare,
    color: 'pink-400',
    bgGradient: 'from-pink-500/10 to-purple-500/10'
  }
};

export default function UCPEnterpriseDashboard() {
  const [user, setUser] = useState(null);
  const [messages, setMessages] = useState([]);
  const [bridges, setBridges] = useState({});
  const [activeChat, setActiveChat] = useState(null);
  const [messageInput, setMessageInput] = useState('');
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [activeView, setActiveView] = useState('messages');
  const [linkPlatform, setLinkPlatform] = useState('');
  const [linkAccount, setLinkAccount] = useState('');
  const [notification, setNotification] = useState(null);
  const [currentTheme, setCurrentTheme] = useState('slate');
  const [searchQuery, setSearchQuery] = useState('');
  const [showThemeMenu, setShowThemeMenu] = useState(false);
  
  const ws = useRef(null);
  const messagesEndRef = useRef(null);
  const theme = themes[currentTheme];

  useEffect(() => {
    initializeUser();
    loadBridges();
    const interval = setInterval(loadBridges, 3000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (user) {
      connectWebSocket();
      loadMessages();
    }
    return () => {
      if (ws.current) ws.current.close();
    };
  }, [user]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const initializeUser = async () => {
    try {
      const res = await fetch(`${API_BASE}/users`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: 'enterprise-user' })
      });
      const data = await res.json();
      setUser(data);
      showNotification('System connected', 'success');
    } catch (err) {
      showNotification('Connection failed', 'error');
    }
  };

  const connectWebSocket = () => {
    ws.current = new WebSocket(`${WS_URL}?user_id=${user.universal_id}`);
    
    ws.current.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      setMessages(prev => [...prev, msg]);
      showNotification(`New message from ${platformConfig[msg.platform]?.name}`, 'info');
    };

    ws.current.onerror = () => {
      showNotification('Connection lost - reconnecting', 'error');
    };
  };

  const loadMessages = async () => {
    try {
      const res = await fetch(`${API_BASE}/messages?user_id=${user.universal_id}`);
      const data = await res.json();
      setMessages(data || []);
    } catch (err) {
      console.error('Failed to load messages:', err);
    }
  };

  const loadBridges = async () => {
    try {
      const res = await fetch(`${API_BASE}/bridges`);
      const data = await res.json();
      setBridges(data || {});
    } catch (err) {
      console.error('Failed to load bridges:', err);
    }
  };

  const sendMessage = async () => {
    if (!messageInput.trim() || !user) return;

    const msg = {
      platform: activeChat || 'native',
      from_user_id: user.universal_id,
      to_user_id: 'broadcast',
      content: messageInput
    };

    try {
      await fetch(`${API_BASE}/messages/send`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(msg)
      });
      setMessageInput('');
      showNotification('Message sent', 'success');
    } catch (err) {
      showNotification('Failed to send message', 'error');
    }
  };

  const linkAccountToPlatform = async () => {
    if (!linkPlatform || !linkAccount || !user) return;

    try {
      await fetch(`${API_BASE}/accounts/link`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          user_id: user.universal_id,
          platform: linkPlatform,
          account_id: linkAccount
        })
      });
      setLinkPlatform('');
      setLinkAccount('');
      showNotification(`${platformConfig[linkPlatform]?.name} linked`, 'success');
      initializeUser();
    } catch (err) {
      showNotification('Failed to link account', 'error');
    }
  };

  const simulateIncomingMessage = async (platform) => {
    if (!user) return;
    
    try {
      await fetch(`${API_BASE}/simulate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          platform,
          from_user_id: 'demo-user',
          to_user_id: user.universal_id,
          content: `Test message from ${platformConfig[platform]?.name}`
        })
      });
    } catch (err) {
      console.error('Simulation failed:', err);
    }
  };

  const showNotification = (message, type) => {
    setNotification({ message, type });
    setTimeout(() => setNotification(null), 3000);
  };

  const platformList = Object.keys(platformConfig);
  const connectedPlatforms = user?.linked_accounts ? Object.keys(user.linked_accounts) : [];
  const connectedBridges = Object.values(bridges).filter(b => b.connected).length;
  const totalMessages = messages.length;

  return (
    <div className={`h-screen w-full bg-gradient-to-br ${theme.bg} flex overflow-hidden relative`}>
      
      {/* Background Grid Pattern */}
      <div className="absolute inset-0 bg-[linear-gradient(rgba(255,255,255,.02)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,.02)_1px,transparent_1px)] bg-[size:50px_50px] [mask-image:radial-gradient(ellipse_80%_50%_at_50%_50%,black,transparent)]" />
      
      {/* Sidebar */}
      <div className={`${sidebarOpen ? 'w-80' : 'w-0'} transition-all duration-300 bg-black/40 backdrop-blur-2xl border-r border-white/5 flex flex-col overflow-hidden relative z-10`}>
        <div className="p-6 border-b border-white/5">
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-3">
              <div className={`w-10 h-10 rounded-xl bg-gradient-to-br ${theme.primary} flex items-center justify-center`}>
                <Shield className="w-6 h-6 text-white" />
              </div>
              <div>
                <h1 className="text-xl font-bold text-white">UCP Enterprise</h1>
                <p className="text-xs text-white/40">Universal Protocol</p>
              </div>
            </div>
            <button onClick={() => setSidebarOpen(false)} className="lg:hidden text-white/40 hover:text-white/80 transition-colors">
              <X className="w-5 h-5" />
            </button>
          </div>
          
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/30" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search messages..."
              className="w-full bg-white/5 border border-white/10 rounded-lg pl-10 pr-4 py-2.5 text-sm text-white placeholder-white/30 focus:outline-none focus:border-white/20 transition-colors"
            />
          </div>
        </div>

        <nav className="flex-1 p-4 space-y-1 overflow-y-auto">
          <button
            onClick={() => setActiveView('messages')}
            className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
              activeView === 'messages'
                ? `bg-gradient-to-r ${theme.primary} text-white shadow-lg`
                : 'text-white/60 hover:bg-white/5 hover:text-white'
            }`}
          >
            <MessageSquare className="w-5 h-5" />
            <span className="font-medium">Messages</span>
            {totalMessages > 0 && (
              <span className="ml-auto text-xs bg-white/20 px-2 py-0.5 rounded-full">
                {totalMessages}
              </span>
            )}
          </button>

          <button
            onClick={() => setActiveView('bridges')}
            className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
              activeView === 'bridges'
                ? `bg-gradient-to-r ${theme.primary} text-white shadow-lg`
                : 'text-white/60 hover:bg-white/5 hover:text-white'
            }`}
          >
            <Globe className="w-5 h-5" />
            <span className="font-medium">Bridge Network</span>
            <span className={`ml-auto text-xs bg-${theme.accent}/20 text-${theme.accent} px-2 py-0.5 rounded-full`}>
              {connectedBridges}
            </span>
          </button>

          <button
            onClick={() => setActiveView('accounts')}
            className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
              activeView === 'accounts'
                ? `bg-gradient-to-r ${theme.primary} text-white shadow-lg`
                : 'text-white/60 hover:bg-white/5 hover:text-white'
            }`}
          >
            <Link2 className="w-5 h-5" />
            <span className="font-medium">Integrations</span>
            {connectedPlatforms.length > 0 && (
              <Check className="ml-auto w-4 h-4 text-green-400" />
            )}
          </button>

          <button
            onClick={() => setActiveView('analytics')}
            className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
              activeView === 'analytics'
                ? `bg-gradient-to-r ${theme.primary} text-white shadow-lg`
                : 'text-white/60 hover:bg-white/5 hover:text-white'
            }`}
          >
            <Activity className="w-5 h-5" />
            <span className="font-medium">Analytics</span>
          </button>

          <button
            onClick={() => setActiveView('security')}
            className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
              activeView === 'security'
                ? `bg-gradient-to-r ${theme.primary} text-white shadow-lg`
                : 'text-white/60 hover:bg-white/5 hover:text-white'
            }`}
          >
            <Lock className="w-5 h-5" />
            <span className="font-medium">Security</span>
          </button>

          <button
            onClick={() => setActiveView('settings')}
            className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
              activeView === 'settings'
                ? `bg-gradient-to-r ${theme.primary} text-white shadow-lg`
                : 'text-white/60 hover:bg-white/5 hover:text-white'
            }`}
          >
            <Settings className="w-5 h-5" />
            <span className="font-medium">Settings</span>
          </button>
        </nav>

        <div className="p-4 border-t border-white/5 space-y-3">
          <div className={`bg-gradient-to-r ${theme.secondary} rounded-lg p-4 border border-white/10`}>
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm font-medium text-white">System Status</span>
              {user && ws.current?.readyState === 1 ? (
                <Wifi className="w-4 h-4 text-green-400" />
              ) : (
                <WifiOff className="w-4 h-4 text-red-400" />
              )}
            </div>
            <div className="text-xs text-white/60">
              {user && ws.current?.readyState === 1 ? 'Connected' : 'Connecting...'}
            </div>
          </div>
          
          <div className="text-xs text-white/30 text-center">
            © 2026 Julius Cameron Hill
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col relative z-10">
        
        {/* Top Bar */}
        <div className="h-16 bg-black/30 backdrop-blur-2xl border-b border-white/5 flex items-center justify-between px-6">
          <div className="flex items-center gap-4">
            {!sidebarOpen && (
              <button onClick={() => setSidebarOpen(true)} className="text-white/40 hover:text-white/80 transition-colors">
                <Menu className="w-6 h-6" />
              </button>
            )}
            <h2 className="text-xl font-semibold text-white capitalize">{activeView}</h2>
            {activeChat && (
              <div className="flex items-center gap-2 text-sm text-white/60">
                <ChevronRight className="w-4 h-4" />
                <span>{platformConfig[activeChat]?.name}</span>
              </div>
            )}
          </div>
          
          <div className="flex items-center gap-4">
            <div className="relative">
              <button 
                onClick={() => setShowThemeMenu(!showThemeMenu)}
                className="text-white/60 hover:text-white transition-colors"
              >
                <Settings className="w-5 h-5" />
              </button>
              
              {showThemeMenu && (
                <div className="absolute right-0 top-full mt-2 bg-black/90 backdrop-blur-2xl border border-white/10 rounded-xl p-3 w-48 shadow-2xl">
                  <div className="text-xs font-medium text-white/60 mb-2">Color Theme</div>
                  <div className="grid grid-cols-3 gap-2">
                    {Object.keys(themes).map(t => (
                      <button
                        key={t}
                        onClick={() => {
                          setCurrentTheme(t);
                          setShowThemeMenu(false);
                        }}
                        className={`h-8 rounded-lg bg-gradient-to-br ${themes[t].primary} border-2 transition-all ${
                          currentTheme === t ? 'border-white scale-110' : 'border-transparent hover:scale-105'
                        }`}
                      />
                    ))}
                  </div>
                </div>
              )}
            </div>
            
            <button className="text-white/60 hover:text-white transition-colors relative">
              <Bell className="w-5 h-5" />
              <span className="absolute -top-1 -right-1 w-2 h-2 bg-red-500 rounded-full" />
            </button>
            
            <div className={`w-10 h-10 rounded-lg bg-gradient-to-br ${theme.primary} flex items-center justify-center text-white font-bold text-sm shadow-lg`}>
              {user?.universal_id?.slice(-2).toUpperCase() || 'U'}
            </div>
          </div>
        </div>

        {/* View Content */}
        <div className="flex-1 overflow-hidden">
          
          {/* Messages View */}
          {activeView === 'messages' && (
            <div className="h-full flex">
              
              {/* Platform Selector */}
              <div className="w-20 bg-black/20 backdrop-blur-2xl border-r border-white/5 p-3 space-y-3 overflow-y-auto">
                {platformList.map(platform => {
                  const PlatformIcon = platformConfig[platform].icon;
                  const isActive = activeChat === platform;
                  
                  return (
                    <button
                      key={platform}
                      onClick={() => setActiveChat(platform)}
                      className={`w-14 h-14 rounded-xl flex items-center justify-center transition-all relative group ${
                        isActive
                          ? `bg-gradient-to-br ${theme.primary} scale-110 shadow-lg`
                          : 'bg-white/5 hover:bg-white/10'
                      }`}
                      title={platformConfig[platform].name}
                    >
                      <PlatformIcon className={`w-6 h-6 ${isActive ? 'text-white' : `text-${platformConfig[platform].color}`}`} />
                      
                      {bridges[platform]?.connected && (
                        <span className="absolute -top-1 -right-1 w-3 h-3 bg-green-500 rounded-full border-2 border-black/50" />
                      )}
                      
                      <div className="absolute left-full ml-2 px-2 py-1 bg-black/90 text-white text-xs rounded opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap pointer-events-none">
                        {platformConfig[platform].name}
                      </div>
                    </button>
                  );
                })}
              </div>

              {/* Messages Area */}
              <div className="flex-1 flex flex-col">
                <div className="flex-1 overflow-y-auto p-6 space-y-4">
                  {messages
                    .filter(m => !activeChat || m.platform === activeChat)
                    .filter(m => !searchQuery || m.content.toLowerCase().includes(searchQuery.toLowerCase()))
                    .map((msg, i) => {
                      const PlatformIcon = platformConfig[msg.platform].icon;
                      const isSent = msg.from_user_id === user?.universal_id;
                      
                      return (
                        <div
                          key={i}
                          className={`flex ${isSent ? 'justify-end' : 'justify-start'}`}
                        >
                          <div
                            className={`max-w-md rounded-xl p-4 backdrop-blur-xl border shadow-lg ${
                              isSent
                                ? `bg-gradient-to-br ${theme.primary} border-white/20`
                                : 'bg-black/40 border-white/10'
                            }`}
                          >
                            <div className="flex items-center gap-2 mb-2">
                              <PlatformIcon className={`w-4 h-4 text-${platformConfig[msg.platform].color}`} />
                              <span className="text-xs text-white/60 font-medium">
                                {platformConfig[msg.platform].name}
                              </span>
                              <span className="text-xs text-white/40 ml-auto">
                                {new Date(msg.timestamp).toLocaleTimeString()}
                              </span>
                            </div>
                            <p className="text-white text-sm leading-relaxed">{msg.content}</p>
                            {msg.encrypted && (
                              <div className="mt-2 flex items-center gap-1 text-xs text-green-400">
                                <Lock className="w-3 h-3" />
                                <span>End-to-end encrypted</span>
                              </div>
                            )}
                          </div>
                        </div>
                      );
                    })}
                  <div ref={messagesEndRef} />
                </div>

                {/* Message Input */}
                <div className="p-6 bg-black/30 backdrop-blur-2xl border-t border-white/5">
                  <div className="flex gap-3">
                    <input
                      type="text"
                      value={messageInput}
                      onChange={(e) => setMessageInput(e.target.value)}
                      onKeyPress={(e) => e.key === 'Enter' && sendMessage()}
                      placeholder={`Send via ${platformConfig[activeChat]?.name || 'Native Protocol'}...`}
                      className="flex-1 bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-white placeholder-white/30 focus:outline-none focus:border-white/20 transition-colors"
                    />
                    <button
                      onClick={sendMessage}
                      className={`px-6 py-3 bg-gradient-to-r ${theme.primary} rounded-xl text-white font-medium hover:shadow-lg transition-all flex items-center gap-2`}
                    >
                      <Send className="w-5 h-5" />
                      Send
                    </button>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Bridges View */}
          {activeView === 'bridges' && (
            <div className="h-full overflow-y-auto p-6">
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                {platformList.map(platform => {
                  const bridge = bridges[platform];
                  const isConnected = bridge?.connected;
                  const PlatformIcon = platformConfig[platform].icon;
                  
                  return (
                    <div
                      key={platform}
                      className={`bg-gradient-to-br ${platformConfig[platform].bgGradient} backdrop-blur-xl border border-white/10 rounded-xl p-6 hover:scale-105 transition-all shadow-lg`}
                    >
                      <div className="flex items-start justify-between mb-4">
                        <div className="flex items-center gap-3">
                          <div className={`w-12 h-12 rounded-xl bg-${platformConfig[platform].color}/20 border border-${platformConfig[platform].color}/30 flex items-center justify-center`}>
                            <PlatformIcon className={`w-6 h-6 text-${platformConfig[platform].color}`} />
                          </div>
                          <div>
                            <h3 className="text-base font-bold text-white">{platformConfig[platform].name}</h3>
                            <div className="flex items-center gap-2 mt-1">
                              <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-400' : 'bg-gray-500'}`} />
                              <span className="text-xs text-white/60">
                                {isConnected ? 'Online' : 'Offline'}
                              </span>
                            </div>
                          </div>
                        </div>
                        <button className="text-white/40 hover:text-white/80 transition-colors">
                          <MoreVertical className="w-5 h-5" />
                        </button>
                      </div>
                      
                      <div className="space-y-3 mb-4">
                        <div className="flex justify-between items-center">
                          <span className="text-xs text-white/50">Messages</span>
                          <span className="text-sm text-white font-semibold">{bridge?.msg_count || 0}</span>
                        </div>
                        <div className="flex justify-between items-center">
                          <span className="text-xs text-white/50">Last Sync</span>
                          <span className="text-xs text-white/70">
                            {bridge?.last_sync ? new Date(bridge.last_sync).toLocaleTimeString() : 'Never'}
                          </span>
                        </div>
                        <div className="w-full bg-white/10 rounded-full h-1.5">
                          <div 
                            className={`h-full rounded-full bg-${platformConfig[platform].color} transition-all`}
                            style={{ width: isConnected ? '100%' : '0%' }}
                          />
                        </div>
                      </div>
                      
                      <button
                        onClick={() => simulateIncomingMessage(platform)}
                        className="w-full bg-white/10 hover:bg-white/20 border border-white/20 rounded-lg px-4 py-2.5 text-white text-sm font-medium transition-all flex items-center justify-center gap-2"
                      >
                        <Zap className="w-4 h-4" />
                        Test Connection
                      </button>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {/* Integrations View */}
          {activeView === 'accounts' && (
            <div className="h-full overflow-y-auto p-6">
              <div className="max-w-4xl mx-auto space-y-6">
                
                <div className="bg-black/30 backdrop-blur-2xl border border-white/10 rounded-xl p-6">
                  <h3 className="text-xl font-bold text-white mb-6 flex items-center gap-2">
                    <Plus className="w-5 h-5" />
                    Add Integration
                  </h3>
                  <div className="grid md:grid-cols-2 gap-4">
                    <select
                      value={linkPlatform}
                      onChange={(e) => setLinkPlatform(e.target.value)}
                      className="bg-white/5 border border-white/10 rounded-lg px-4 py-3 text-white focus:outline-none focus:border-white/20"
                    >
                      <option value="">Select Platform</option>
                      {platformList.map(p => (
                        <option key={p} value={p}>{platformConfig[p].name}</option>
                      ))}
                    </select>
                    
                    <input
                      type="text"
                      value={linkAccount}
                      onChange={(e) => setLinkAccount(e.target.value)}
                      placeholder="Account ID"
                      className="bg-white/5 border border-white/10 rounded-lg px-4 py-3 text-white placeholder-white/30 focus:outline-none focus:border-white/20"
                    />
                  </div>
                  
                  <button
                    onClick={linkAccountToPlatform}
                    disabled={!linkPlatform || !linkAccount}
                    className={`mt-4 w-full bg-gradient-to-r ${theme.primary} rounded-lg px-6 py-3 text-white font-medium transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2`}
                  >
                    <Link2 className="w-5 h-5" />
                    Connect Account
                  </button>
                </div>

                <div className="bg-black/30 backdrop-blur-2xl border border-white/10 rounded-xl p-6">
                  <h3 className="text-xl font-bold text-white mb-6">Active Integrations</h3>
                  <div className="grid gap-4">
                    {connectedPlatforms.length > 0 ? (
                      connectedPlatforms.map(platform => {
                        const PlatformIcon = platformConfig[platform].icon;
                        return (
                          <div key={platform} className="flex items-center justify-between bg-white/5 rounded-xl p-4 border border-white/10">
                            <div className="flex items-center gap-4">
                              <div className={`w-12 h-12 rounded-lg bg-${platformConfig[platform].color}/20 border border-${platformConfig[platform].color}/30 flex items-center justify-center`}>
                                <PlatformIcon className={`w-6 h-6 text-${platformConfig[platform].color}`} />
                              </div>
                              <div>
                                <div className="text-white font-semibold">{platformConfig[platform].name}</div>
                                <div className="text-sm text-white/60">{user.linked_accounts[platform]}</div>
                              </div>
                            </div>
                            <Check className="w-6 h-6 text-green-400" />
                          </div>
                        );
                      })
                    ) : (
                      <div className="text-center text-white/60 py-12">
                        <Database className="w-12 h-12 mx-auto mb-3 opacity-30" />
                        <p>No integrations configured</p>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Analytics View */}
          {activeView === 'analytics' && (
            <div className="h-full overflow-y-auto p-6">
              <div className="max-w-6xl mx-auto grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-6">
                <div className={`bg-gradient-to-br ${theme.secondary} backdrop-blur-2xl border border-white/10 rounded-xl p-6`}>
                  <div className="flex items-center justify-between mb-2">
                    <MessageSquare className="w-8 h-8 text-blue-400" />
                    <span className="text-xs text-white/60">Total</span>
                  </div>
                  <div className="text-3xl font-bold text-white mb-1">{totalMessages}</div>
                  <div className="text-sm text-white/60">Messages</div>
                </div>
                
                <div className={`bg-gradient-to-br ${theme.secondary} backdrop-blur-2xl border border-white/10 rounded-xl p-6`}>
                  <div className="flex items-center justify-between mb-2">
                    <Globe className="w-8 h-8 text-green-400" />
                    <span className="text-xs text-white/60">Active</span>
                  </div>
                  <div className="text-3xl font-bold text-white mb-1">{connectedBridges}</div>
                  <div className="text-sm text-white/60">Bridges</div>
                </div>
                
                <div className={`bg-gradient-to-br ${theme.secondary} backdrop-blur-2xl border border-white/10 rounded-xl p-6`}>
                  <div className="flex items-center justify-between mb-2">
                    <Link2 className="w-8 h-8 text-purple-400" />
                    <span className="text-xs text-white/60">Linked</span>
                  </div>
                  <div className="text-3xl font-bold text-white mb-1">{connectedPlatforms.length}</div>
                  <div className="text-sm text-white/60">Accounts</div>
                </div>
                
                <div className={`bg-gradient-to-br ${theme.secondary} backdrop-blur-2xl border border-white/10 rounded-xl p-6`}>
                  <div className="flex items-center justify-between mb-2">
                    <Activity className="w-8 h-8 text-orange-400" />
                    <span className="text-xs text-white/60">Status</span>
                  </div>
                  <div className="text-3xl font-bold text-white mb-1">100%</div>
                  <div className="text-sm text-white/60">Uptime</div>
                </div>
              </div>
              
              <div className="max-w-6xl mx-auto bg-black/30 backdrop-blur-2xl border border-white/10 rounded-xl p-6">
                <h3 className="text-xl font-bold text-white mb-4">Platform Activity</h3>
                <div className="space-y-4">
                  {platformList.map(platform => {
                    const PlatformIcon = platformConfig[platform].icon;
                    const msgCount = messages.filter(m => m.platform === platform).length;
                    const percentage = totalMessages > 0 ? (msgCount / totalMessages) * 100 : 0;
                    
                    return (
                      <div key={platform}>
                        <div className="flex items-center justify-between mb-2">
                          <div className="flex items-center gap-2">
                            <PlatformIcon className={`w-4 h-4 text-${platformConfig[platform].color}`} />
                            <span className="text-sm text-white">{platformConfig[platform].name}</span>
                          </div>
                          <span className="text-sm text-white/60">{msgCount} messages</span>
                        </div>
                        <div className="w-full bg-white/10 rounded-full h-2">
                          <div 
                            className={`h-full rounded-full bg-${platformConfig[platform].color} transition-all`}
                            style={{ width: `${percentage}%` }}
                          />
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          )}

          {/* Security View */}
          {activeView === 'security' && (
            <div className="h-full overflow-y-auto p-6">
              <div className="max-w-4xl mx-auto space-y-6">
                <div className="bg-black/30 backdrop-blur-2xl border border-white/10 rounded-xl p-6">
                  <div className="flex items-start gap-4">
                    <Shield className="w-12 h-12 text-blue-400 flex-shrink-0" />
                    <div className="flex-1">
                      <h3 className="text-2xl font-bold text-white mb-2">End-to-End Encryption</h3>
                      <p className="text-white/60 leading-relaxed">
                        All messages transmitted through the native protocol are secured with military-grade Ed25519 signatures 
                        and NaCl box encryption. Your private keys remain on your device and are never transmitted.
                      </p>
                    </div>
                  </div>
                </div>

                <div className="bg-black/30 backdrop-blur-2xl border border-white/10 rounded-xl p-6">
                  <h3 className="text-xl font-bold text-white mb-4 flex items-center gap-2">
                    <Lock className="w-5 h-5" />
                    Encryption Keys
                  </h3>
                  <div className="space-y-4">
                    <div>
                      <div className="text-sm text-white/60 mb-2">Public Key</div>
                      <div className="text-white font-mono text-xs bg-white/5 rounded-lg px-4 py-3 break-all border border-white/10">
                        {user?.public_key || 'Generating...'}
                      </div>
                    </div>
                    <div>
                      <div className="text-sm text-white/60 mb-2">Universal ID</div>
                      <div className="text-white font-mono bg-white/5 rounded-lg px-4 py-3 border border-white/10">
                        {user?.universal_id || 'Initializing...'}
                      </div>
                    </div>
                  </div>
                </div>

                <div className="grid md:grid-cols-2 gap-6">
                  <div className="bg-black/30 backdrop-blur-2xl border border-white/10 rounded-xl p-6">
                    <div className="flex items-center gap-3 mb-4">
                      <Database className="w-6 h-6 text-green-400" />
                      <h4 className="text-lg font-bold text-white">Data Privacy</h4>
                    </div>
                    <p className="text-sm text-white/60">
                      Message data is stored encrypted at rest. Decryption only occurs on authorized devices with valid keys.
                    </p>
                  </div>
                  
                  <div className="bg-black/30 backdrop-blur-2xl border border-white/10 rounded-xl p-6">
                    <div className="flex items-center gap-3 mb-4">
                      <Activity className="w-6 h-6 text-orange-400" />
                      <h4 className="text-lg font-bold text-white">Audit Log</h4>
                    </div>
                    <p className="text-sm text-white/60">
                      All system access and message routing is logged with cryptographic signatures for full audit trails.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Settings View */}
          {activeView === 'settings' && (
            <div className="h-full overflow-y-auto p-6">
              <div className="max-w-4xl mx-auto space-y-6">
                <div className="bg-black/30 backdrop-blur-2xl border border-white/10 rounded-xl p-6">
                  <h3 className="text-xl font-bold text-white mb-6">System Configuration</h3>
                  <div className="space-y-4">
                    <div className="flex items-center justify-between py-3 border-b border-white/10">
                      <div>
                        <div className="text-white font-medium">Auto-connect Bridges</div>
                        <div className="text-sm text-white/60">Automatically connect to available bridges on startup</div>
                      </div>
                      <button className={`w-12 h-6 rounded-full transition-colors ${theme.accent === 'slate' ? 'bg-slate-500' : `bg-${theme.accent}`}`}>
                        <div className="w-5 h-5 bg-white rounded-full shadow-lg ml-auto mr-0.5" />
                      </button>
                    </div>
                    
                    <div className="flex items-center justify-between py-3 border-b border-white/10">
                      <div>
                        <div className="text-white font-medium">Message Notifications</div>
                        <div className="text-sm text-white/60">Show desktop notifications for new messages</div>
                      </div>
                      <button className={`w-12 h-6 rounded-full transition-colors ${theme.accent === 'slate' ? 'bg-slate-500' : `bg-${theme.accent}`}`}>
                        <div className="w-5 h-5 bg-white rounded-full shadow-lg ml-auto mr-0.5" />
                      </button>
                    </div>
                    
                    <div className="flex items-center justify-between py-3">
                      <div>
                        <div className="text-white font-medium">E2E Encryption</div>
                        <div className="text-sm text-white/60">Always encrypt messages via native protocol</div>
                      </div>
                      <button className={`w-12 h-6 rounded-full transition-colors ${theme.accent === 'slate' ? 'bg-slate-500' : `bg-${theme.accent}`}`}>
                        <div className="w-5 h-5 bg-white rounded-full shadow-lg ml-auto mr-0.5" />
                      </button>
                    </div>
                  </div>
                </div>

                <div className="bg-black/30 backdrop-blur-2xl border border-white/10 rounded-xl p-6">
                  <h3 className="text-xl font-bold text-white mb-6">API Endpoints</h3>
                  <div className="space-y-3">
                    <div className="flex items-center gap-2">
                      <Database className="w-4 h-4 text-blue-400" />
                      <span className="text-sm text-white/60">HTTP:</span>
                      <code className="text-sm text-white font-mono">{API_BASE}</code>
                    </div>
                    <div className="flex items-center gap-2">
                      <Zap className="w-4 h-4 text-green-400" />
                      <span className="text-sm text-white/60">WebSocket:</span>
                      <code className="text-sm text-white font-mono">{WS_URL}</code>
                    </div>
                  </div>
                </div>

                <div className="text-center text-xs text-white/30 pt-6">
                  Universal Communication Protocol v1.0.0 | © 2026 Julius Cameron Hill
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Notification Toast */}
      {notification && (
        <div className="fixed top-6 right-6 z-50 animate-in slide-in-from-top-5">
          <div className={`bg-black/95 backdrop-blur-2xl border rounded-xl px-6 py-4 flex items-center gap-3 shadow-2xl min-w-[300px] ${
            notification.type === 'success' ? 'border-green-500/50' :
            notification.type === 'error' ? 'border-red-500/50' :
            'border-blue-500/50'
          }`}>
            {notification.type === 'success' && <Check className="w-5 h-5 text-green-400 flex-shrink-0" />}
            {notification.type === 'error' && <AlertCircle className="w-5 h-5 text-red-400 flex-shrink-0" />}
            {notification.type === 'info' && <Bell className="w-5 h-5 text-blue-400 flex-shrink-0" />}
            <span className="text-white font-medium">{notification.message}</span>
          </div>
        </div>
      )}
    </div>
  );
}