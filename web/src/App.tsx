import { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Toaster } from 'react-hot-toast';
import { Layout } from './components/layout/Layout';
import { ChatPage } from './pages/ChatPage';
import { ForestPage } from './pages/ForestPage';
import { SessionsPage } from './pages/SessionsPage';
import { ListenersPage } from './pages/ListenersPage';
import { SettingsPage } from './pages/SettingsPage';
import { useSettingsStore } from './stores/settingsStore';

function App() {
  const loadSettings = useSettingsStore((state) => state.loadSettings);
  const isLoaded = useSettingsStore((state) => state.isLoaded);

  useEffect(() => {
    // Load settings once on app startup
    if (!isLoaded) {
      loadSettings();
    }
  }, [loadSettings, isLoaded]);

  return (
    <BrowserRouter>
      <Toaster
        position="top-right"
        toastOptions={{
          duration: 3000,
          style: {
            background: '#1a1f3a',
            color: '#e2e8f0',
            border: '1px solid #252b4a',
          },
          success: {
            iconTheme: {
              primary: '#10b981',
              secondary: '#1a1f3a',
            },
          },
          error: {
            iconTheme: {
              primary: '#ef4444',
              secondary: '#1a1f3a',
            },
          },
        }}
      />
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Navigate to="/chat" replace />} />
          <Route path="chat" element={<ChatPage />} />
          <Route path="forest" element={<ForestPage />} />
          <Route path="sessions" element={<SessionsPage />} />
          <Route path="listeners" element={<ListenersPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
