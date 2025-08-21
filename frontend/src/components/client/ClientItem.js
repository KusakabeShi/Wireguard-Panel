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
  Circle as StatusIcon
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

        
        <Box sx={{ flexGrow: 1 }}>
          <Typography variant="body1" sx={{ display: 'flex', alignItems: 'center', fontWeight: 'bold' }}>
            <span>{client.name}</span>
            <span style={{ fontSize: '0.875rem', color: 'rgb(255, 255, 255)' }}>
              {
                getTrafficText()
              }
            </span>
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', mr: 2 }}>
          <StatusIcon 
            sx={{ 
              color: isActive ? ' #4caf50' : ' #f44336', 
              fontSize: 12 
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