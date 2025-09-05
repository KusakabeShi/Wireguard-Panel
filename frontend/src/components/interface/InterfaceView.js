import React, { useState, useEffect } from 'react';
import { Box, Divider, Snackbar, Alert } from '@mui/material';
import InterfaceHeader from './InterfaceHeader';
import ServerList from '../server/ServerList';
import apiService from '../../services/apiService';
import { getTrafficDisplayMode, TRAFFIC_DISPLAY_MODES } from '../../utils/formatUtils';
import stateManager from '../../utils/stateManager';

const InterfaceView = ({ 
  interface_, 
  onEditInterface, 
  onAddServer,
  onEditServer,
  onDeleteServer,
  onToggleServer,
  onAddClient,
  onEditClient,
  onDeleteClient,
  onToggleClient
}) => {
  const [servers, setServers] = useState([]);
  const [serverClients, setServerClients] = useState({}); // Store clients per server
  const [clientsLoaded, setClientsLoaded] = useState(new Set()); // Track which servers have loaded clients
  const [clientsState, setClientsState] = useState({});
  const [previousClientsState, setPreviousClientsState] = useState({});
  const [lastUpdateTime, setLastUpdateTime] = useState(null);
  const [previousUpdateTime, setPreviousUpdateTime] = useState(null);
  const [trafficDisplayMode, setTrafficDisplayMode] = useState(getTrafficDisplayMode());
  const [collapsedServers, setCollapsedServers] = useState(new Set());
  const [expandedClients, setExpandedClients] = useState(new Set());
  const [stateInitialized, setStateInitialized] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [interfaceInfo, setInterfaceInfo] = useState(null);

  // Listen for stateManager initialization
  useEffect(() => {
    const syncState = () => {
      if (!stateInitialized) {
        setCollapsedServers(stateManager.getCollapsedServers());
        setExpandedClients(stateManager.getExpandedClients());
        setStateInitialized(true);
      }
    };

    // Use the event system instead of polling
    stateManager.onInitialized(syncState);
  }, [stateInitialized]);

  useEffect(() => {
    if (interface_) {
      loadServers();
      loadClientsState();
      loadInterfaceInfo();
    }
  }, [interface_, interface_?.lastModified]);

  const loadInterfaceInfo = async () => {
    if (!interface_) return;
    
    try {
      const serviceConfig = await apiService.getServiceConfig();
      console.log('ServiceConfig response:', serviceConfig);
      console.log('wgIfPrefix from API:', serviceConfig.wgIfPrefix);
      setInterfaceInfo({ 
        ...interface_, 
        wgIfPrefix: serviceConfig.wgIfPrefix 
      });
    } catch (error) {
      console.error('Failed to load service config:', error);
      // Fallback to interface without wgIfPrefix
      setInterfaceInfo(interface_);
    }
  };

  // Load clients for expanded servers when servers are loaded
  useEffect(() => {
    if (servers.length > 0 && stateInitialized) {
      console.log('Loading clients for expanded servers:', servers.map(s => s.id), 'collapsed:', Array.from(collapsedServers));
      servers.forEach(server => {
        if (!collapsedServers.has(server.id)) {
          console.log('Loading clients for server:', server.id);
          loadServerClients(server.id);
        }
      });
    }
  }, [servers, collapsedServers, stateInitialized]);


  useEffect(() => {
    if (!interface_) return;

    // Set up interval based on display mode
    const refreshInterval = trafficDisplayMode === TRAFFIC_DISPLAY_MODES.RATE ? 1000 : 5000;
    
    const interval = setInterval(() => {
      loadClientsState();
    }, refreshInterval);

    // Cleanup interval on unmount or when interface/mode changes
    return () => clearInterval(interval);
  }, [interface_?.id, trafficDisplayMode]);

  const loadServers = async () => {
    if (!interface_) return;
    
    setLoading(true);
    try {
      const servers = await apiService.getServers(interface_.id);
      setServers(servers);
      // Clear clientsLoaded state so that client lists will be reloaded for expanded servers
      setClientsLoaded(new Set());
    } catch (error) {
      console.error('Failed to load servers:', error);
      setError(error.message || 'Failed to load servers');
      setServers([]);
    } finally {
      setLoading(false);
    }
  };

  // Load clients for a specific server when needed
  const loadServerClients = async (serverId) => {
    if (!interface_ || clientsLoaded.has(serverId)) {
      console.log('Skipping client load for server:', serverId, 'interface:', !!interface_, 'already loaded:', clientsLoaded.has(serverId));
      return;
    }
    
    console.log('Loading clients for server:', serverId);
    
    try {
      const clients = await apiService.getServerClients(interface_.id, serverId);
      console.log('Loaded', clients.length, 'clients for server:', serverId);
      setServerClients(prev => ({
        ...prev,
        [serverId]: clients
      }));
      setClientsLoaded(prev => new Set(prev).add(serverId));
    } catch (error) {
      console.error(`Failed to load clients for server ${serverId}:`, error);
      setServerClients(prev => ({
        ...prev,
        [serverId]: []
      }));
      setClientsLoaded(prev => new Set(prev).add(serverId));
    }
  };

  const loadClientsState = async () => {
    if (!interface_) return;
    
    try {
      const clientsState = await apiService.getInterfaceClientsState(interface_.id);
      const clientsStateData = clientsState.state
      const currentTime = clientsState.timestamp
      
      setClientsState(prevClientsState => {
        if (Object.keys(prevClientsState).length > 0) {
          setPreviousClientsState(prevClientsState);
        }
        return clientsStateData || {};
      });
      
      setLastUpdateTime(prevLastUpdateTime => {
        if (prevLastUpdateTime) {
          setPreviousUpdateTime(prevLastUpdateTime);
        }
        return currentTime;
      });
    } catch (error) {
      console.error('Failed to load client states:', error);
      // Don't show error to user for state updates, just reset to empty
      setClientsState({});
    }
  };

  const handleTrafficModeToggle = () => {
    const newMode = trafficDisplayMode === TRAFFIC_DISPLAY_MODES.TOTAL 
      ? TRAFFIC_DISPLAY_MODES.RATE 
      : TRAFFIC_DISPLAY_MODES.TOTAL;
    
    setTrafficDisplayMode(newMode);
    // Clear previous states when switching modes to avoid incorrect rate calculations
    // Note: Keep lastUpdateTime for proper handshake time display
    setPreviousClientsState({});
    setPreviousUpdateTime(null);
  };

  const handleToggleServerExpanded = (serverId) => {
    setCollapsedServers(prevCollapsed => {
      const isCurrentlyExpanded = !prevCollapsed.has(serverId);
      const newCollapsed = isCurrentlyExpanded; // If expanded, we want to collapse it
      
      // Remove the direct loadServerClients call - let useEffect handle it
      
      if (stateManager.initialized) {
        stateManager.setServerCollapsed(serverId, newCollapsed);
      }
      
      const newCollapsedSet = new Set(prevCollapsed);
      if (newCollapsed) {
        newCollapsedSet.add(serverId);
      } else {
        newCollapsedSet.delete(serverId);
      }
      return newCollapsedSet;
    });
  };

  const handleToggleClientExpanded = (interfaceId, serverId, clientId) => {
    const compositeKey = `${interfaceId}_${serverId}_${clientId}`;
    const isCurrentlyExpanded = expandedClients.has(compositeKey);
    
    if (stateManager.initialized) {
      stateManager.setClientExpanded(interfaceId, serverId, clientId, !isCurrentlyExpanded);
      setExpandedClients(new Set(stateManager.getExpandedClients()));
    } else {
      // Local fallback when not initialized
      setExpandedClients(prev => {
        const newSet = new Set(prev);
        if (isCurrentlyExpanded) {
          newSet.delete(compositeKey);
        } else {
          newSet.add(compositeKey);
        }
        return newSet;
      });
    }
  };

  const handleServerToggle = async (server, enabled) => {
    try {
      await onToggleServer(server, enabled);
      await loadServers(); // Reload to get updated state
      // If server is expanded, reload its clients too
      if (!collapsedServers.has(server.id)) {
        setClientsLoaded(prev => {
          const newSet = new Set(prev);
          newSet.delete(server.id);
          return newSet;
        });
        loadServerClients(server.id);
      }
    } catch (error) {
      console.error('Failed to toggle server:', error);
      setError(error.message || 'Failed to toggle server');
    }
  };

  const handleClientToggle = async (server, client, enabled) => {
    try {
      await onToggleClient(server, client, enabled);
      await loadServers(); // Reload to get updated state
      // If server is expanded, reload its clients too
      if (!collapsedServers.has(server.id)) {
        setClientsLoaded(prev => {
          const newSet = new Set(prev);
          newSet.delete(server.id);
          return newSet;
        });
        loadServerClients(server.id);
      }
    } catch (error) {
      console.error('Failed to toggle client:', error);
      setError(error.message || 'Failed to toggle client');
    }
  };

  if (!interface_) {
    return null;
  }

  return (
    <Box>
      <InterfaceHeader 
        interface_={interface_} 
        onEdit={onEditInterface}
      />
      <Divider />
      <ServerList
        servers={servers}
        serverClients={serverClients}
        clientsState={clientsState}
        previousClientsState={previousClientsState}
        lastUpdateTime={lastUpdateTime}
        previousUpdateTime={previousUpdateTime}
        trafficDisplayMode={trafficDisplayMode}
        onTrafficModeToggle={handleTrafficModeToggle}
        collapsedServers={collapsedServers}
        expandedClients={expandedClients}
        loading={loading}
        onToggleServerExpanded={handleToggleServerExpanded}
        onToggleClientExpanded={handleToggleClientExpanded}
        onAddServer={() => onAddServer(interface_)}
        onEditServer={onEditServer}
        onDeleteServer={onDeleteServer}
        onToggleServer={handleServerToggle}
        onAddClient={onAddClient}
        onEditClient={onEditClient}
        onDeleteClient={onDeleteClient}
        onToggleClient={handleClientToggle}
        interfaceId={interface_.id}
        interfaceInfo={interfaceInfo}
      />
      
      <Snackbar
        open={!!error}
        autoHideDuration={6000}
        onClose={() => setError(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={() => setError(null)} severity="error" sx={{ width: '100%' }}>
          {error}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default InterfaceView;