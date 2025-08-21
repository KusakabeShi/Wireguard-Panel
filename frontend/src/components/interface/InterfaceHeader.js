import React from 'react';
import { Box, Typography, IconButton } from '@mui/material';
import { Settings as SettingsIcon } from '@mui/icons-material';

const InterfaceHeader = ({ interface_, onEdit }) => {
  if (!interface_) return null;

  return (
    <Box 
      sx={{ 
        p: 2, 
        backgroundColor: '#f5f5f5',
        borderBottom: '1px solid #e0e0e0',
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