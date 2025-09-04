import React, { useState, useEffect } from 'react';
import { Box, Divider, Snackbar, Alert } from '@mui/material';
import InterfaceHeader from './InterfaceHeader';
import ServerList from '../server/ServerList';
import apiService from '../../services/apiService';
import { getTrafficDisplayMode, TRAFFIC_DISPLAY_MODES } from '../../utils/formatUtils';

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
  const [clientsState, setClientsState] = useState({});
  const [previousClientsState, setPreviousClientsState] = useState({});
  const [lastUpdateTime, setLastUpdateTime] = useState(null);
  const [previousUpdateTime, setPreviousUpdateTime] = useState(null);
  const [trafficDisplayMode, setTrafficDisplayMode] = useState(getTrafficDisplayMode());
  const [expandedServers, setExpandedServers] = useState(new Set());
  const [expandedClients, setExpandedClients] = useState(new Set());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    if (interface_) {
      loadServers();
      loadClientsState();
    }
  }, [interface_, interface_?.lastModified]);

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
      
      const serversWithClients = await Promise.all(
        servers.map(async (server) => {
          try {
            const clients = await apiService.getServerClients(interface_.id, server.id);
            return { ...server, clients };
          } catch (error) {
            console.error(`Failed to load clients for server ${server.id}:`, error);
            return { ...server, clients: [] };
          }
        })
      );
      
      setServers(serversWithClients);
      
      // Expand all servers by default
      const serverIds = new Set(serversWithClients.map(server => server.id));
      setExpandedServers(serverIds);
    } catch (error) {
      console.error('Failed to load servers:', error);
      setError(error.message || 'Failed to load servers');
      setServers([]);
    } finally {
      setLoading(false);
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
    // Clear previous states when switching modes to avoid incorrect calculations
    setPreviousClientsState({});
    setLastUpdateTime(null);
    setPreviousUpdateTime(null);
  };

  const handleToggleServerExpanded = (serverId) => {
    setExpandedServers(prev => {
      const newSet = new Set(prev);
      if (newSet.has(serverId)) {
        newSet.delete(serverId);
      } else {
        newSet.add(serverId);
      }
      return newSet;
    });
  };

  const handleToggleClientExpanded = (clientId) => {
    setExpandedClients(prev => {
      const newSet = new Set(prev);
      if (newSet.has(clientId)) {
        newSet.delete(clientId);
      } else {
        newSet.add(clientId);
      }
      return newSet;
    });
  };

  const handleServerToggle = async (server, enabled) => {
    try {
      await onToggleServer(server, enabled);
      await loadServers(); // Reload to get updated state
    } catch (error) {
      console.error('Failed to toggle server:', error);
      setError(error.message || 'Failed to toggle server');
    }
  };

  const handleClientToggle = async (server, client, enabled) => {
    try {
      await onToggleClient(server, client, enabled);
      await loadServers(); // Reload to get updated state
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
        clientsState={clientsState}
        previousClientsState={previousClientsState}
        lastUpdateTime={lastUpdateTime}
        previousUpdateTime={previousUpdateTime}
        trafficDisplayMode={trafficDisplayMode}
        onTrafficModeToggle={handleTrafficModeToggle}
        expandedServers={expandedServers}
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