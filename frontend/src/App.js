import React, { useState, useEffect } from 'react';
import { Box, CssBaseline, ThemeProvider, createTheme, CircularProgress } from '@mui/material';
import { AuthProvider, useAuth } from './context/AuthContext';
import './App.css';

import Header from './components/layout/Header';
import Sidebar from './components/layout/Sidebar';
import MainContent from './components/layout/MainContent';
import LoginDialog from './components/layout/LoginDialog';
import InterfaceView from './components/interface/InterfaceView';

import InterfaceDialog from './components/dialogs/InterfaceDialog';
import ServerDialog from './components/dialogs/ServerDialog';
import ClientDialog from './components/dialogs/ClientDialog';
import ErrorDialog from './components/dialogs/ErrorDialog';
import SettingsDialog from './components/dialogs/SettingsDialog';

import apiService from './services/apiService';

const theme = createTheme({
  palette: {
    primary: {
      main: '#1976d2',
    },
    secondary: {
      main: '#d32f2f',
    },
  },
});

function AppContent() {
  const { isAuthenticated, isLoading } = useAuth();
  const [interfaces, setInterfaces] = useState([]);
  const [selectedInterface, setSelectedInterface] = useState(null);
  const [loginDialogOpen, setLoginDialogOpen] = useState(false);
  
  // Dialog states
  const [interfaceDialog, setInterfaceDialog] = useState({ open: false, interface: null });
  const [serverDialog, setServerDialog] = useState({ open: false, server: null, interface: null });
  const [clientDialog, setClientDialog] = useState({ open: false, client: null, server: null, interface: null });
  const [errorDialog, setErrorDialog] = useState({ open: false, error: null, title: 'Error' });
  const [settingsDialogOpen, setSettingsDialogOpen] = useState(false);

  useEffect(() => {
    if (isAuthenticated) {
      loadInterfaces();
    }
  }, [isAuthenticated]);

  useEffect(() => {
    if (!isAuthenticated && !isLoading) {
      setLoginDialogOpen(true);
    } else {
      setLoginDialogOpen(false);
    }
  }, [isAuthenticated, isLoading]);

  const showError = (error, title = 'Error') => {
    setErrorDialog({ open: true, error, title });
  };

  const loadInterfaces = async () => {
    try {
      const interfaces = await apiService.getInterfaces();
      setInterfaces(interfaces);
      
      // Select first interface if none selected
      if (!selectedInterface && interfaces.length > 0) {
        setSelectedInterface(interfaces[0]);
      }
    } catch (error) {
      if (error.message === 'Authentication required') {
        setLoginDialogOpen(true);
      } else {
        console.error('Failed to load interfaces:', error);
        showError(error, 'Failed to Load Interfaces');
      }
    }
  };

  // Interface handlers
  const handleAddInterface = () => {
    setInterfaceDialog({ open: true, interface: null });
  };

  const handleEditInterface = (interface_) => {
    setInterfaceDialog({ open: true, interface: interface_ });
  };

  const handleSaveInterface = async (interfaceData) => {
    if (interfaceDialog.interface) {
      // Edit existing
      await apiService.updateInterface(interfaceDialog.interface.id, interfaceData);
    } else {
      // Create new
      await apiService.createInterface(interfaceData);
    }
    await loadInterfaces();
  };

  const handleDeleteInterface = async (interfaceId) => {
    try {
      await apiService.deleteInterface(interfaceId);
      await loadInterfaces();
      
      // If deleted interface was selected, clear selection
      if (selectedInterface?.id === interfaceId) {
        setSelectedInterface(null);
      }
    } catch (error) {
      console.error('Failed to delete interface:', error);
      showError(error, 'Failed to Delete Interface');
    }
  };

  // Server handlers
  const handleAddServer = (interface_) => {
    setServerDialog({ open: true, server: null, interface: interface_ });
  };

  const handleEditServer = (server) => {
    setServerDialog({ open: true, server: server, interface: selectedInterface });
  };

  const handleSaveServer = async (serverData) => {
    const interface_ = serverDialog.interface;
    
    if (serverDialog.server) {
      // Edit existing
      await apiService.updateServer(interface_.id, serverDialog.server.id, serverData);
    } else {
      // Create new
      await apiService.createServer(interface_.id, serverData);
    }
    // Trigger refresh of interface view data
    
    setSelectedInterface(prev => ({ ...prev, lastModified: Date.now() }));
  };

  const handleDeleteServer = async (serverId) => {
    try {
      const interface_ = serverDialog.interface;
      await apiService.deleteServer(interface_.id, serverId);
      // Trigger refresh of interface view data
      setSelectedInterface(prev => ({ ...prev, lastModified: Date.now() }));
    } catch (error) {
      console.error('Failed to delete server:', error);
      showError(error, 'Failed to Delete Server');
    }
  };

  const handleToggleServer = async (server, enabled) => {
    try {
      await apiService.setServerEnabled(selectedInterface.id, server.id, enabled);
    } catch (error) {
      console.error('Failed to toggle server:', error);
      showError(error, 'Failed to Toggle Server');
    }
  };

  // Client handlers
  const handleAddClient = (server) => {
    setClientDialog({ 
      open: true, 
      client: null, 
      server: server, 
      interface: selectedInterface 
    });
  };

  const handleEditClient = (server, client) => {
    setClientDialog({ 
      open: true, 
      client: client, 
      server: server, 
      interface: selectedInterface 
    });
  };

  const handleSaveClient = async (clientData) => {
    const { interface: interface_, server } = clientDialog;
    
    if (clientDialog.client) {
      // Edit existing
      await apiService.updateClient(interface_.id, server.id, clientDialog.client.id, clientData);
    } else {
      // Create new
      await apiService.createClient(interface_.id, server.id, clientData);
    }
    // Trigger refresh of interface view data
    setSelectedInterface(prev => ({ ...prev, lastModified: Date.now() }));
  };

  const handleDeleteClient = async (clientId) => {
    try {
      const { interface: interface_, server } = clientDialog;
      await apiService.deleteClient(interface_.id, server.id, clientId);
      // Trigger refresh of interface view data
      setSelectedInterface(prev => ({ ...prev, lastModified: Date.now() }));
    } catch (error) {
      console.error('Failed to delete client:', error);
      showError(error, 'Failed to Delete Client');
    }
  };

  const handleToggleClient = async (server, client, enabled) => {
    try {
      await apiService.setClientEnabled(selectedInterface.id, server.id, client.id, enabled);
    } catch (error) {
      console.error('Failed to toggle client:', error);
      showError(error, 'Failed to Toggle Client');
    }
  };

  const handleSettingsClick = () => {
    setSettingsDialogOpen(true);
  };

  const handleLogoutClick = async () => {
    try {
      await apiService.logout();
      // The AuthContext will handle the auth state change
      window.location.reload(); // Refresh to reset app state
    } catch (error) {
      console.error('Logout failed:', error);
      showError(error, 'Failed to Logout');
    }
  };

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
      <Header onSettingsClick={handleSettingsClick} onLogoutClick={handleLogoutClick} />
      
      <Box sx={{ display: 'flex', flexGrow: 1 }}>
        <Sidebar
          interfaces={interfaces}
          selectedInterface={selectedInterface}
          onInterfaceSelect={setSelectedInterface}
          onAddInterface={handleAddInterface}
        />
        
        <MainContent
          selectedInterface={selectedInterface}
          onAddInterface={handleAddInterface}
        >
          {selectedInterface && (
            <InterfaceView
              interface_={selectedInterface}
              onEditInterface={handleEditInterface}
              onAddServer={handleAddServer}
              onEditServer={handleEditServer}
              onDeleteServer={handleDeleteServer}
              onToggleServer={handleToggleServer}
              onAddClient={handleAddClient}
              onEditClient={handleEditClient}
              onDeleteClient={handleDeleteClient}
              onToggleClient={handleToggleClient}
            />
          )}
        </MainContent>
      </Box>

      {/* Dialogs */}
      <LoginDialog 
        open={loginDialogOpen} 
        onClose={() => setLoginDialogOpen(false)} 
      />

      <InterfaceDialog
        open={interfaceDialog.open}
        onClose={() => setInterfaceDialog({ open: false, interface: null })}
        onSave={handleSaveInterface}
        onDelete={handleDeleteInterface}
        interface_={interfaceDialog.interface}
      />

      <ServerDialog
        open={serverDialog.open}
        onClose={() => setServerDialog({ open: false, server: null, interface: null })}
        onSave={handleSaveServer}
        onDelete={handleDeleteServer}
        server={serverDialog.server}
      />

      <ClientDialog
        open={clientDialog.open}
        onClose={() => setClientDialog({ open: false, client: null, server: null, interface: null })}
        onSave={handleSaveClient}
        onDelete={handleDeleteClient}
        client={clientDialog.client}
      />

      <ErrorDialog
        open={errorDialog.open}
        onClose={() => setErrorDialog({ open: false, error: null, title: 'Error' })}
        error={errorDialog.error}
        title={errorDialog.title}
      />

      <SettingsDialog
        open={settingsDialogOpen}
        onClose={() => setSettingsDialogOpen(false)}
        onSave={() => {
          // Settings saved successfully - could show a notification here
        }}
      />
    </Box>
  );
}

function App() {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <AuthProvider>
        <AppContent />
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
