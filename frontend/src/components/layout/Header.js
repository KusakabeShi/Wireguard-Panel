import React from 'react';
import { AppBar, Toolbar, Typography, IconButton, Box } from '@mui/material';
import { Settings as SettingsIcon, Logout as LogoutIcon } from '@mui/icons-material';

const Header = ({ onSettingsClick, onLogoutClick }) => {
  return (
    <AppBar position="static" sx={{ backgroundColor: '#1976d2', height: 64 }}>
      <Toolbar sx={{ minHeight: 64 }}>
        <Typography variant="h6" component="div" sx={{ flexGrow: 1, fontWeight: 'bold' }}>
          WG-Panel
        </Typography>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <IconButton 
            color="inherit" 
            onClick={onSettingsClick}
            sx={{ color: 'white' }}
            title="Settings"
          >
            <SettingsIcon />
          </IconButton>
          <IconButton 
            color="inherit" 
            onClick={onLogoutClick}
            sx={{ color: 'white' }}
            title="Logout"
          >
            <LogoutIcon />
          </IconButton>
        </Box>
      </Toolbar>
    </AppBar>
  );
};

export default Header;