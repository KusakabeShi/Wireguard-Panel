import React from 'react';
import { Box } from '@mui/material';
import ClientItem from './ClientItem';

const ClientList = ({ 
  clients,
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