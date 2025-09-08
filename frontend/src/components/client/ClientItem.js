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
  ExpandLess as ExpandLessIcon,
  Circle as CircleIcon
} from '@mui/icons-material';
import ClientDetails from './ClientDetails';
import { 
  formatBytes, 
  formatLastHandshake, 
  isClientActive, 
  formatTransferRate, 
  calculateTransferRate,
  TRAFFIC_DISPLAY_MODES,
  setTrafficDisplayMode
} from '../../utils/formatUtils';

const ClientItem = ({ 
  client, 
  clientState,
  previousClientState,
  lastUpdateTime,
  previousUpdateTime,
  trafficDisplayMode,
  onTrafficModeToggle,
  expanded,
  onToggleExpanded,
  onEdit,
  onDelete,
  onToggle,
  interfaceId,
  serverId,
  interfaceInfo,
  serverInfo
}) => {
  const theme = useTheme();

  const getTrafficText = () => {
    if (trafficDisplayMode === TRAFFIC_DISPLAY_MODES.RATE) {
      // Calculate transfer rates
      if (!clientState || !previousClientState || !lastUpdateTime || !previousUpdateTime) {
        return ' ↑ 0 Bps ↓ 0 Bps';
      }

      const txRate = calculateTransferRate(
        clientState.transferTx || 0,
        previousClientState.transferTx || 0,
        lastUpdateTime,
        previousUpdateTime
      );
      
      const rxRate = calculateTransferRate(
        clientState.transferRx || 0,
        previousClientState.transferRx || 0,
        lastUpdateTime,
        previousUpdateTime
      );

      return ` ↑ ${formatTransferRate(txRate)} ↓ ${formatTransferRate(rxRate)}`;
    } else {
      // Show total traffic (original behavior)
      const tx = formatBytes(clientState?.transferTx || 0);
      const rx = formatBytes(clientState?.transferRx || 0);
      return ` ↑ ${tx} ↓ ${rx}`;
    }
  };

  const handleTrafficTextClick = () => {
    // Save new mode to cookie and call parent toggle handler
    const newMode = trafficDisplayMode === TRAFFIC_DISPLAY_MODES.TOTAL 
      ? TRAFFIC_DISPLAY_MODES.RATE 
      : TRAFFIC_DISPLAY_MODES.TOTAL;
    
    setTrafficDisplayMode(newMode);
    onTrafficModeToggle();
  };

  const isActive = isClientActive(lastUpdateTime,clientState?.latestHandshake);

  return (
    <Box sx={{ mb: 0.3 }}>
      {/* Client Row */}
        <Box 
          sx={{ 
            display: 'flex',
            alignItems: 'center',
            p: 0.5,
            backgroundColor: theme.palette.custom.client.background,
            color: 'white',
            borderRadius: '4px 4px 0 0'
          }}
        >
          <Box sx={{ ml: 1,flexGrow: 1, display: 'flex', alignItems: 'center' }}>
            <Typography variant="body1" sx={{ fontWeight: 'bold' }}>
          {client.name}
            </Typography>
            <Box sx={{ ml: 2, display: 'flex', flexDirection: 'column' }}>
              {client.ip && (
                <Typography variant="body2" sx={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: '0.75rem' }}>
                  {client.ip}
                </Typography>
              )}
              {client.ipv6 && (
                <Typography variant="body2" sx={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: '0.75rem' }}>
                  {client.ipv6}
                </Typography>
              )}
            </Box>
          </Box>
          <Box sx={{ mr: 2, display: 'flex', alignItems: 'center' }}>
            <Typography 
              variant="body2" 
              sx={{ 
                color: 'rgb(255, 255, 255)', 
                mr: 1, 
                cursor: 'pointer',
                '&:hover': {
                  textDecoration: 'underline',
                  opacity: 0.8
                }
              }}
              onClick={handleTrafficTextClick}
              title={`Click to switch between total traffic and transfer rate mode. Currently showing: ${trafficDisplayMode === TRAFFIC_DISPLAY_MODES.RATE ? 'Transfer Rate' : 'Total Traffic'}`}
            >
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
        <Box sx={{ backgroundColor: theme.palette.custom.clientDetails.background, p: 2, borderLeft: '4px solid #4caf50' }}>
          {expanded && (
            <ClientDetails 
              client={client}
              clientState={clientState}
              lastUpdateTime={lastUpdateTime}
              interfaceId={interfaceId}
              serverId={serverId}
              interfaceInfo={interfaceInfo}
              serverInfo={serverInfo}
              visible={true}
            />
          )}
        </Box>
      </Collapse>
    </Box>
  );
};

export default ClientItem;