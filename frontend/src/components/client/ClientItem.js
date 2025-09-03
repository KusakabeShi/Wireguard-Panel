import React, { useState, useEffect } from 'react';
import { 
  Box, 
  Typography, 
  IconButton, 
  Switch,
  Collapse
} from '@mui/material';
import { 
  Edit as EditIcon, 
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  Circle as CircleIcon
} from '@mui/icons-material';
import ClientDetails from './ClientDetails';
import { formatBytes, formatLastHandshake, isClientActive } from '../../utils/formatUtils';
import apiService from '../../services/apiService';

const ClientItem = ({ 
  client, 
  expanded,
  onToggleExpanded,
  onEdit,
  onDelete,
  onToggle,
  interfaceId,
  serverId
}) => {
  const [clientState, setClientState] = useState(null);

  useEffect(() => {
    // Always load client state to show traffic stats and status
    loadClientState();
    
    // Set up interval to update client status every 5 seconds
    const interval = setInterval(() => {
      loadClientState();
    }, 5000);
    
    // Cleanup interval on unmount or when client.id changes
    return () => clearInterval(interval);
  }, [client.id]);

  const loadClientState = async () => {
    try {
      const state = await apiService.getClientState(interfaceId, serverId, client.id);
      setClientState(state);
    } catch (error) {
      console.error('Failed to load client state:', error);
      setClientState(null);
    }
  };

  const getTrafficText = () => {
    // Always show traffic stats, even if null (treat as 0)
    const tx = formatBytes(clientState?.transferTx || 0);
    const rx = formatBytes(clientState?.transferRx || 0);
    return ` ↑ ${tx} ↓ ${rx}`;
  };

  const isActive = isClientActive(clientState?.latestHandshake);

  return (
    <Box sx={{ mb: 1 }}>
      {/* Client Row */}
        <Box 
          sx={{ 
            display: 'flex',
            alignItems: 'center',
            p: 2,
            backgroundColor: 'rgb(51, 109, 43)',
            color: 'white',
            borderRadius: '4px 4px 0 0'
          }}
        >
          <Box sx={{ flexGrow: 1, display: 'flex', alignItems: 'center' }}>
            <Typography variant="body1" sx={{ fontWeight: 'bold' }}>
          {client.name}
            </Typography>
          </Box>
          <Box sx={{ mr: 2, display: 'flex', alignItems: 'center' }}>
            <Typography variant="body2" sx={{ color: 'rgb(255, 255, 255)', mr: 1 }}>
          {getTrafficText()}
            </Typography>
            <Box
              sx={{
                width: 16,
                height: 16,
                borderRadius: '50%',
                backgroundColor: isActive ? '#4caf50' : '#f44336',
                border: '2px solid #d3d3d3',
                filter: 'drop-shadow(0 0 1px rgba(128,128,128,0.8))'
              }}
            />
          </Box>
          <Switch
            checked={client.enabled}
            onChange={(e) => onToggle(client, e.target.checked)}
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
            onClick={() => onEdit(client)}
            sx={{ color: 'white', ml: 1 }}
            size="small"
          >
            <EditIcon fontSize="small" />
          </IconButton>
          
          <IconButton 
            onClick={() => onToggleExpanded(client.id)}
            sx={{ color: 'white', ml: 1 }}
            size="small"
          >
            {expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
          </IconButton>
        </Box>

        {/* Client Details */}
      <Collapse in={expanded}>
        <Box sx={{ backgroundColor: '#f9f9f9', p: 2, borderLeft: '4px solid #4caf50' }}>
          <ClientDetails 
            client={client}
            clientState={clientState}
            interfaceId={interfaceId}
            serverId={serverId}
          />
        </Box>
      </Collapse>
    </Box>
  );
};

export default ClientItem;