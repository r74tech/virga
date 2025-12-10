export interface Settings {
  connection: ConnectionSettings;
  ui: UISettings;
  session: SessionSettings;
  notification: NotificationSettings;
  terminal: TerminalSettings;
  security: SecuritySettings;
  forestMap: ForestMapSettings;
}

export interface ConnectionSettings {
  apiEndpoint: string;
  websocketEndpoint: string;
  environment: 'development' | 'staging' | 'production';
  authToken?: string;
  timeout: number;
  retryAttempts: number;
  retryDelay: number;
}

export interface UISettings {
  theme: 'light' | 'dark' | 'auto';
  language: 'ja' | 'en';
  timezone: string;
  dateFormat: string;
  timeFormat: '12h' | '24h';
  animations: boolean;
  compactMode: boolean;
}

export interface SessionSettings {
  defaultBeaconInterval: number;
  jitter: number;
  inactiveTimeout: number;
  chatTimeout: number;
  autoReconnect: boolean;
  maxReconnectAttempts: number;
  saveCommandHistory: boolean;
}

export interface NotificationSettings {
  newSessionAlert: boolean;
  sessionDisconnectAlert: boolean;
  commandCompleteAlert: boolean;
  errorAlert: boolean;
  browserNotifications: boolean;
  soundEnabled: boolean;
  soundVolume: number;
}

export interface TerminalSettings {
  fontFamily: string;
  fontSize: number;
  colorScheme: string;
  historySize: number;
  scrollback: number;
  cursorBlink: boolean;
  lineNumbers: boolean;
  autocomplete: boolean;
}

export interface SecuritySettings {
  sessionTimeout: number;
  autoLogout: boolean;
  require2FA: boolean;
  rememberPassword: boolean;
  auditLog: boolean;
  logCommands: boolean;
}

export interface ForestMapSettings {
  layout: 'cola' | 'cose' | 'grid' | 'circle' | 'breadthfirst' | 'concentric';
  nodeSpacing: number;
  edgeLength: number;
  animationDuration: number;
  showLabels: boolean;
  showEdgeLabels: boolean;
  autoCenter: boolean;
  autoRefresh: boolean;
  refreshInterval: number;
}

export const DEFAULT_SETTINGS: Settings = {
  connection: {
    apiEndpoint: 'http://localhost:8080/api',
    websocketEndpoint: 'ws://localhost:8080/ws',
    environment: 'development',
    timeout: 30000,
    retryAttempts: 3,
    retryDelay: 1000,
  },
  ui: {
    theme: 'dark',
    language: 'ja',
    timezone: 'Asia/Tokyo',
    dateFormat: 'YYYY/MM/DD',
    timeFormat: '24h',
    animations: true,
    compactMode: false,
  },
  session: {
    defaultBeaconInterval: 60,
    jitter: 20,
    inactiveTimeout: 30,
    chatTimeout: 5,
    autoReconnect: true,
    maxReconnectAttempts: 5,
    saveCommandHistory: true,
  },
  notification: {
    newSessionAlert: true,
    sessionDisconnectAlert: true,
    commandCompleteAlert: false,
    errorAlert: true,
    browserNotifications: false,
    soundEnabled: false,
    soundVolume: 50,
  },
  terminal: {
    fontFamily: 'JetBrains Mono, monospace',
    fontSize: 14,
    colorScheme: 'default',
    historySize: 1000,
    scrollback: 10000,
    cursorBlink: true,
    lineNumbers: false,
    autocomplete: true,
  },
  security: {
    sessionTimeout: 60,
    autoLogout: true,
    require2FA: false,
    rememberPassword: false,
    auditLog: true,
    logCommands: true,
  },
  forestMap: {
    layout: 'cola',
    nodeSpacing: 100,
    edgeLength: 150,
    animationDuration: 500,
    showLabels: true,
    showEdgeLabels: false,
    autoCenter: true,
    autoRefresh: false,
    refreshInterval: 30,
  },
};
