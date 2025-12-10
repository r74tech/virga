import { MessageSquare, Network, Users, Radio, Settings } from 'lucide-react';
import { NavLink } from 'react-router-dom';

const navItems = [
  { to: '/chat', icon: MessageSquare, label: 'Chat' },
  { to: '/forest', icon: Network, label: 'Forest Map' },
  { to: '/sessions', icon: Users, label: 'Sessions' },
  { to: '/listeners', icon: Radio, label: 'Listeners' },
  { to: '/settings', icon: Settings, label: 'Settings' },
];

export const Sidebar = () => {
  return (
    <aside className="w-64 bg-dark-bg-secondary border-r border-primary-900/30 flex flex-col">
      <nav className="flex-1 p-4">
        <ul className="space-y-2">
          {navItems.map((item) => {
            const Icon = item.icon;
            return (
              <li key={item.to}>
                <NavLink
                  to={item.to}
                  className={({ isActive }) =>
                    `flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
                      isActive
                        ? 'bg-primary-600 text-white'
                        : 'text-dark-text-secondary hover:bg-dark-bg-tertiary hover:text-dark-text-primary'
                    }`
                  }
                >
                  <Icon className="w-5 h-5" />
                  <span className="font-medium">{item.label}</span>
                </NavLink>
              </li>
            );
          })}
        </ul>
      </nav>

      <div className="p-4 border-t border-primary-900/30">
        <div className="text-xs text-dark-text-secondary">
          <div>Virga C2 v1.0.0</div>
          <div className="mt-1">© 2025 Virga</div>
        </div>
      </div>
    </aside>
  );
};
