import React from 'react';
import { Box, Typography, IconButton, useTheme } from '@mui/material';
import { Settings as SettingsIcon } from '@mui/icons-material';

const InterfaceHeader = ({ interface_, onEdit }) => {
  const theme = useTheme();
  
  if (!interface_) return null;

  return (
    <Box 
      sx={{ 
        p: 2, 
        backgroundColor: theme.palette.background.paper,
        borderBottom: `1px solid ${theme.palette.divider}`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between'
      }}
    >
      <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
        {interface_.ifname} {interface_.endpoint}:{interface_.port}
      </Typography>
      <IconButton onClick={() => onEdit(interface_)} size="small">
        <SettingsIcon />
      </IconButton>
    </Box>
  );
};

export default InterfaceHeader;