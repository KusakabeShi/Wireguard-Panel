import React, { useState, useEffect } from 'react';
import { Box, CssBaseline, CircularProgress, useMediaQuery, useTheme } from '@mui/material';
import { AuthProvider, useAuth } from './context/AuthContext';
import { ThemeProvider } from './context/ThemeContext';
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
import stateManager from './utils/stateManager';


function AppContent() {
  const { isAuthenticated, isLoading } = useAuth();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  const [interfaces, setInterfaces] = useState([]);
  const [selectedInterface, setSelectedInterface] = useState(null);
  const [loginDialogOpen, setLoginDialogOpen] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(() => !isMobile);
  
  // Dialog states
  const [interfaceDialog, setInterfaceDialog] = useState({ open: false, interface: null });
  const [serverDialog, setServerDialog] = useState({ open: false, server: null, interface: null });
  const [clientDialog, setClientDialog] = useState({ open: false, client: null, server: null, interface: null });
  const [errorDialog, setErrorDialog] = useState({ open: false, error: null, title: 'Error' });
  const [settingsDialogOpen, setSettingsDialogOpen] = useState(false);
  const [warningShown, setWarningShown] = useState(false);

  useEffect(() => {
    if (isAuthenticated) {
      loadInterfaces();
    }
  }, [isAuthenticated]);

  useEffect(() => {
    setSidebarOpen(isMobile ? false : true);
  }, [isMobile]);

  // Sync interface selection once stateManager is initialized
  useEffect(() => {
    const syncInterfaceSelection = () => {
      if (interfaces.length > 0 && stateManager.initialized) {
        const savedInterfaceId = stateManager.getSelectedInterfaceId();
        const savedInterface = interfaces.find(i => i.id === savedInterfaceId);
        
        if (savedInterface && (!selectedInterface || selectedInterface.id !== savedInterface.id)) {
          setSelectedInterface(savedInterface);
        } else if (selectedInterface && !savedInterfaceId) {
          // Save current selection if none was saved
          stateManager.setSelectedInterfaceId(selectedInterface.id);
        }
      }
    };

    // Use the event system
    stateManager.onInitialized(syncInterfaceSelection);
  }, [interfaces, selectedInterface]);

  useEffect(() => {
    if (!isAuthenticated && !isLoading) {
      setLoginDialogOpen(true);
    } else {
      setLoginDialogOpen(false);
    }
  }, [isAuthenticated, isLoading]);

  // Update document title from injected runtime config
  useEffect(() => {
    if (window.WG_PANEL_TITLE) {
      document.title = window.WG_PANEL_TITLE;
    } else {
      document.title = 'Wireguard Server Panel (React Dev)';
    }
  }, []);

  // Show warning message from server at startup (show only once until refresh)
  useEffect(() => {
    if ( !warningShown && window.INIT_WARNING_MESSAGE) {
      const warningMessage = window.INIT_WARNING_MESSAGE.trim();
      if (warningMessage) {
        setErrorDialog({ 
          open: true, 
          error: warningMessage, 
          title: 'System Configuration Warning' 
        });
        setWarningShown(true);
      }
    }
  }, [isAuthenticated, warningShown]);

  const showError = (error, title = 'Error') => {
    setErrorDialog({ open: true, error, title });
  };

  const loadInterfaces = async () => {
    try {
      const interfaces = await apiService.getInterfaces();
      setInterfaces(interfaces);
      
      // Clean up all UI state for interfaces that no longer exist
      if (stateManager.initialized && interfaces.length > 0) {
        try {
          // Use comprehensive cleanup that validates interface IDs
          stateManager.cleanupAllUIStateForDeletedInterfaces(interfaces);
        } catch (error) {
          console.warn('Failed to cleanup UI state for deleted interfaces:', error);
        }
      }
      
      // Try to restore selected interface from state manager
      let savedInterface = null;
      if (stateManager.initialized) {
        const savedInterfaceId = stateManager.getSelectedInterfaceId();
        savedInterface = interfaces.find(i => i.id === savedInterfaceId);
      }
      
      if (savedInterface) {
        setSelectedInterface(savedInterface);
      } else if (!selectedInterface && interfaces.length > 0) {
        // Select first interface if none selected and save to state if possible
        setSelectedInterface(interfaces[0]);
        if (stateManager.initialized) {
          stateManager.setSelectedInterfaceId(interfaces[0].id);
        }
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
    // Always use the current interface data from the interfaces list
    const currentInterface = interfaces.find(i => i.id === interface_.id) || interface_;
    setInterfaceDialog({ open: true, interface: currentInterface });
  };

  const handleSaveInterface = async (interfaceData) => {
    let interfaceId;
    
    if (interfaceDialog.interface) {
      // Edit existing
      interfaceId = interfaceDialog.interface.id;
      await apiService.updateInterface(interfaceId, interfaceData);
    } else {
      // Create new
      const newInterface = await apiService.createInterface(interfaceData);
      interfaceId = newInterface.id;
      // Enable the newly created interface
      await apiService.setInterfaceEnabled(interfaceId, true);
    }
    
    // Load updated interfaces
    const updatedInterfaces = await apiService.getInterfaces();
    setInterfaces(updatedInterfaces);
    
    // Update selected interface to reflect changes if it was the one edited
    if (selectedInterface && selectedInterface.id === interfaceId) {
      const updatedInterface = updatedInterfaces.find(i => i.id === interfaceId);
      if (updatedInterface) {
        setSelectedInterface(updatedInterface);
      }
    }
    
    // Select first interface if none selected and interfaces exist
    if (!selectedInterface && updatedInterfaces.length > 0) {
      setSelectedInterface(updatedInterfaces[0]);
    }
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

  const handleToggleInterface = async (interfaceId, enabled) => {
    try {
      await apiService.setInterfaceEnabled(interfaceId, enabled);
      const updatedInterfaces = await apiService.getInterfaces();
      setInterfaces(updatedInterfaces);
      
      // Update the interface in the dialog if it's the same one
      if (interfaceDialog.interface && interfaceDialog.interface.id === interfaceId) {
        const updatedInterface = updatedInterfaces.find(i => i.id === interfaceId);
        if (updatedInterface) {
          setInterfaceDialog(prev => ({
            ...prev,
            interface: updatedInterface
          }));
        }
      }
    } catch (error) {
      console.error('Failed to toggle interface:', error);
      showError(error, 'Failed to Toggle Interface');
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
      const newServer = await apiService.createServer(interface_.id, serverData);
      // Enable the newly created server
      await apiService.setServerEnabled(interface_.id, newServer.id, true);
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
      const newClient = await apiService.createClient(interface_.id, server.id, clientData);
      // Enable the newly created client
      await apiService.setClientEnabled(interface_.id, server.id, newClient.id, true);
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

  const closeSidebar = () => setSidebarOpen(false);
  const toggleSidebar = () => setSidebarOpen(prev => !prev);

  const handleInterfaceSelect = (interface_) => {
    setSelectedInterface(interface_);
    if (stateManager.initialized) {
      stateManager.setSelectedInterfaceId(interface_.id);
    }
    if (isMobile) {
      closeSidebar();
    }
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
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh' ,minWidth: '600px',}}>
      <Header 
        onSettingsClick={handleSettingsClick} 
        onLogoutClick={handleLogoutClick}
        showMenuTrigger={isMobile}
        onMenuClick={toggleSidebar}
      />
      
      <Box 
        sx={{ 
          display: 'flex', 
          flexGrow: 1, 
          width: '100%',
          maxWidth: isMobile ? '100%' : '1280px',
          margin: '0 auto',
          position: 'relative'
        }}
      >
        <Sidebar
          interfaces={interfaces}
          selectedInterface={selectedInterface}
          onInterfaceSelect={handleInterfaceSelect}
          onAddInterface={handleAddInterface}
          isOpen={sidebarOpen}
          onToggle={toggleSidebar}
          onClose={closeSidebar}
          isMobile={isMobile}
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
        suppressFocusTrap={errorDialog.open}
      />

      <InterfaceDialog
        open={interfaceDialog.open}
        onClose={() => setInterfaceDialog({ open: false, interface: null })}
        onSave={handleSaveInterface}
        onDelete={handleDeleteInterface}
        onToggleEnable={handleToggleInterface}
        interface_={interfaceDialog.interface}
      />

      <ServerDialog
        open={serverDialog.open}
        onClose={() => setServerDialog({ open: false, server: null, interface: null })}
        onSave={handleSaveServer}
        onDelete={handleDeleteServer}
        wgvrf={selectedInterface?.vrfName}
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
    <ThemeProvider>
      <CssBaseline />
      <AuthProvider>
        <AppContent />
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
