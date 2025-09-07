import React from 'react';
import { 
  Box, 
  Typography, 
  IconButton, 
  Switch,
  Collapse,
  useTheme
} from '@mui/material';
import { 
  Edit as EditIcon, 
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon
} from '@mui/icons-material';
import ClientList from '../client/ClientList';

const ServerItem = ({ 
  server, 
  clientsState,
  previousClientsState,
  lastUpdateTime,
  previousUpdateTime,
  trafficDisplayMode,
  onTrafficModeToggle,
  expanded,
  expandedClients,
  onToggleExpanded,
  onToggleClientExpanded,
  onEdit,
  onDelete,
  onToggle,
  onAddClient,
  onEditClient,
  onDeleteClient,
  onToggleClient,
  interfaceId,
  interfaceInfo
}) => {
  const theme = useTheme();
  
  const getNetworkDisplay = (server) => {
    const networks = [];
    if (server.ipv4?.enabled && server.ipv4?.network) {
      networks.push(server.ipv4.network);
    }
    if (server.ipv6?.enabled && server.ipv6?.network) {
      networks.push(server.ipv6.network);
    }
    return networks.join(', ');
  };

  return (
    <Box sx={{ mb: 1 }}>
      {/* Server Row */}
      <Box 
        sx={{ 
          display: 'flex',
          alignItems: 'center',
          p: 1,
          backgroundColor: theme.palette.custom.server.background,
          color: 'white',
          borderRadius: '4px 4px 0 0'
        }}
      >
        <Box sx={{ flexGrow: 1 }}>
          <Typography variant="subtitle1" sx={{ fontWeight: 'bold' }}>
            {server.name} {getNetworkDisplay(server)}
          </Typography>
        </Box>
        
        <Switch
          checked={server.enabled}
          onChange={(e) => onToggle(server, e.target.checked)}
          sx={{
            '& .MuiSwitch-switchBase.Mui-checked': {
              color: 'white',
            },
            '& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track': {
              backgroundColor: 'rgba(255,255,255,0.3)',
            },
          }}
        />
        
        <IconButton 
          onClick={() => onEdit(server)}
          sx={{ color: 'white', ml: 1 }}
          size="small"
        >
          <EditIcon />
        </IconButton>
        
        <IconButton 
          onClick={() => onToggleExpanded(server.id)}
          sx={{ color: 'white', ml: 1 }}
          size="small"
        >
          {expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
        </IconButton>
      </Box>

      {/* Clients Section */}
      <Collapse in={expanded}>
        <Box sx={{ backgroundColor: theme.palette.custom.expanded.background, borderRadius: '0 0 4px 4px' }}>
          <ClientList
            clients={server.clients || []}
            clientsState={clientsState[server.id] || {}}
            previousClientsState={previousClientsState[server.id] || {}}
            lastUpdateTime={lastUpdateTime}
            previousUpdateTime={previousUpdateTime}
            trafficDisplayMode={trafficDisplayMode}
            onTrafficModeToggle={onTrafficModeToggle}
            expandedClients={expandedClients}
            onToggleExpanded={onToggleClientExpanded}
            onEdit={(client) => onEditClient(server, client)}
            onDelete={(client) => onDeleteClient(server, client)}
            onToggle={(client, enabled) => onToggleClient(server, client, enabled)}
            interfaceId={interfaceId}
            serverId={server.id}
            onAddClient={() => onAddClient(server)}
            interfaceInfo={interfaceInfo}
            serverInfo={server}
          />
        </Box>
      </Collapse>
    </Box>
  );
};

export default ServerItem;