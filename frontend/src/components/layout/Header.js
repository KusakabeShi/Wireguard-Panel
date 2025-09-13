import React from 'react';
import { AppBar, Toolbar, Typography, IconButton, Box, useTheme } from '@mui/material';
import { Settings as SettingsIcon, Logout as LogoutIcon } from '@mui/icons-material';
import ThemeModeToggle from './ThemeModeToggle';

const Header = ({ onSettingsClick, onLogoutClick }) => {
  const theme = useTheme();
  
  return (
    <AppBar position="static" sx={{ backgroundColor: theme.palette.primary.main, height: 64 }}>
      <Toolbar sx={{ minHeight: 64 }}>
        <Typography variant="h6" component="div" sx={{ flexGrow: 1, fontWeight: 'bold' }}>
          { document.title }
        </Typography>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <ThemeModeToggle />
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