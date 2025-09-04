import React from 'react';
import { Box } from '@mui/material';
import ClientItem from './ClientItem';

const ClientList = ({ 
  clients,
  clientsState,
  previousClientsState,
  lastUpdateTime,
  previousUpdateTime,
  trafficDisplayMode,
  onTrafficModeToggle,
  expandedClients,
  onToggleExpanded,
  onEdit,
  onDelete,
  onToggle,
  interfaceId,
  serverId
}) => {
  if (!clients || clients.length === 0) {
    return null;
  }

  return (
    <Box sx={{ p: 2 }}>
      {clients.map((client) => (
        <ClientItem
          key={client.id}
          client={client}
          clientState={clientsState[client.id] || null}
          previousClientState={previousClientsState[client.id] || null}
          lastUpdateTime={lastUpdateTime}
          previousUpdateTime={previousUpdateTime}
          trafficDisplayMode={trafficDisplayMode}
          onTrafficModeToggle={onTrafficModeToggle}
          expanded={expandedClients.has(client.id)}
          onToggleExpanded={onToggleExpanded}
          onEdit={onEdit}
          onDelete={onDelete}
          onToggle={onToggle}
          interfaceId={interfaceId}
          serverId={serverId}
        />
      ))}
    </Box>
  );
};

export default ClientList;