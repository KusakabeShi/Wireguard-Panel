import React, { useState, useEffect } from 'react';
import { Box, Divider, Snackbar, Alert } from '@mui/material';
import InterfaceHeader from './InterfaceHeader';
import ServerList from '../server/ServerList';
import apiService from '../../services/apiService';

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
  const [expandedServers, setExpandedServers] = useState(new Set());
  const [expandedClients, setExpandedClients] = useState(new Set());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    if (interface_) {
      loadServers();
    }
  }, [interface_, interface_?.lastModified]);

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