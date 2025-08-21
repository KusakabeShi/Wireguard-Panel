import React from 'react';
import { Box, IconButton, CircularProgress } from '@mui/material';
import { Add as AddIcon } from '@mui/icons-material';
import ServerItem from './ServerItem';

const ServerList = ({ 
  servers,
  expandedServers,
  expandedClients,
  loading,
  onToggleServerExpanded,
  onToggleClientExpanded,
  onAddServer,
  onEditServer,
  onDeleteServer,
  onToggleServer,
  onAddClient,
  onEditClient,
  onDeleteClient,
  onToggleClient,
  interfaceId
}) => {
  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 2 }}>
      {servers.map((server) => (
        <ServerItem
          key={server.id}
          server={server}
          expanded={expandedServers.has(server.id)}
          expandedClients={expandedClients}
          onToggleExpanded={onToggleServerExpanded}
          onToggleClientExpanded={onToggleClientExpanded}
          onEdit={onEditServer}
          onDelete={onDeleteServer}
          onToggle={onToggleServer}
          onAddClient={onAddClient}
          onEditClient={onEditClient}
          onDeleteClient={onDeleteClient}
          onToggleClient={onToggleClient}
          interfaceId={interfaceId}
        />
      ))}
      
      {/* Add Server Button */}
      <Box sx={{ textAlign: 'center', mt: 2 }}>
        <IconButton 
          onClick={onAddServer}
          sx={{ 
            backgroundColor: '#1976d2',
            color: 'white',
            width: 56,
            height: 56,
            '&:hover': {
              backgroundColor: '#1565c0',
            }
          }}
        >
          <AddIcon sx={{ fontSize: 32 }} />
        </IconButton>
      </Box>
    </Box>
  );
};

export default ServerList;