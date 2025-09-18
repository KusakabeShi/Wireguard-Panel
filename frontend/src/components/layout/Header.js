import React, { useState } from 'react';
import { 
  AppBar, 
  Toolbar, 
  Typography, 
  IconButton, 
  Box, 
  Menu, 
  MenuItem,
  ListItemIcon,
  ListItemText,
  useMediaQuery,
  useTheme 
} from '@mui/material';
import { 
  Settings as SettingsIcon, 
  Logout as LogoutIcon,
  Menu as MenuIcon,
  MoreVert as MoreVertIcon
} from '@mui/icons-material';
import ThemeModeToggle from './ThemeModeToggle';

const Header = ({ 
  onSettingsClick, 
  onLogoutClick, 
  onMenuClick = () => {}, 
  showMenuTrigger = false 
}) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  const [menuAnchorEl, setMenuAnchorEl] = useState(null);

  const handleMenuOpen = (event) => {
    setMenuAnchorEl(event.currentTarget);
  };

  const handleMenuClose = () => {
    setMenuAnchorEl(null);
  };
  
  return (
    <AppBar position="static" sx={{ backgroundColor: theme.palette.primary.main, height: 64 }}>
      <Toolbar sx={{ minHeight: 64, px: { xs: 1, sm: 2 } }}>
        {showMenuTrigger && (
          <IconButton 
            color="inherit" 
            onClick={onMenuClick}
            sx={{ color: 'white', mr: 1 }}
            title="Open navigation"
          >
            <MenuIcon />
          </IconButton>
        )}
        <Typography 
          variant="h6" 
          component="div" 
          sx={{ 
            flexGrow: 1, 
            fontWeight: 'bold',
            minWidth: 0,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap'
          }}
        >
          { document.title }
        </Typography>
        {isMobile ? (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <ThemeModeToggle />
            <IconButton 
              color="inherit" 
              onClick={handleMenuOpen}
              sx={{ color: 'white' }}
              title="More options"
            >
              <MoreVertIcon />
            </IconButton>
            <Menu
              anchorEl={menuAnchorEl}
              open={Boolean(menuAnchorEl)}
              onClose={handleMenuClose}
              anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
              transformOrigin={{ vertical: 'top', horizontal: 'right' }}
            >
              <MenuItem
                onClick={() => {
                  handleMenuClose();
                  onSettingsClick();
                }}
              >
                <ListItemIcon>
                  <SettingsIcon fontSize="small" />
                </ListItemIcon>
                <ListItemText primary="Settings" />
              </MenuItem>
              <MenuItem
                onClick={() => {
                  handleMenuClose();
                  onLogoutClick();
                }}
              >
                <ListItemIcon>
                  <LogoutIcon fontSize="small" />
                </ListItemIcon>
                <ListItemText primary="Logout" />
              </MenuItem>
            </Menu>
          </Box>
        ) : (
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
        )}
      </Toolbar>
    </AppBar>
  );
};

export default Header;
