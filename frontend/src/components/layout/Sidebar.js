import React from 'react';
import { 
  Box, 
  List, 
  ListItem, 
  ListItemButton, 
  ListItemText, 
  Typography, 
  IconButton,
  Divider,
  useTheme
} from '@mui/material';
import { Menu as MenuIcon, Add as AddIcon, Circle as CircleIcon } from '@mui/icons-material';

const Sidebar = ({ 
  interfaces, 
  selectedInterface, 
  onInterfaceSelect, 
  onAddInterface,
  isOpen,
  onToggle
}) => {
  const theme = useTheme();
  
  return (
    <Box 
      sx={{ 
        width: isOpen ? 200 : 60, 
        height: 'calc(100vh - 64px)',
        borderRight: `2px solid ${theme.palette.divider}`,
        backgroundColor: theme.palette.background.sidebar,
        display: 'flex',
        flexDirection: 'column',
        transition: 'width 0.3s ease',
        overflow: 'hidden'
      }}
    >
      <Box 
        sx={{ 
          p: 2, 
          borderBottom: `1px solid ${theme.palette.divider}`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: isOpen ? 'space-between' : 'center',
          minHeight: 56
        }}
      >
        {isOpen && (
          <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
            Interfaces
          </Typography>
        )}
        <IconButton size="small" onClick={onToggle}>
          <MenuIcon />
        </IconButton>
      </Box>
      
      <List sx={{ flexGrow: 1, p: 0 }}>
        {interfaces.map((interface_) => (
          <ListItem key={interface_.id} disablePadding>
            <ListItemButton
              selected={selectedInterface?.id === interface_.id}
              onClick={() => onInterfaceSelect(interface_)}
              sx={{
                justifyContent: isOpen ? 'initial' : 'center',
                px: isOpen ? 2 : 1,
                '&.Mui-selected': {
                  backgroundColor: theme.palette.mode === 'dark' ? 'rgba(25, 118, 210, 0.2)' : '#e3f2fd',
                  borderRight: `3px solid ${theme.palette.primary.main}`,
                },
                '&:hover': {
                  backgroundColor: theme.palette.action.hover,
                }
              }}
              title={!isOpen ? interface_.ifname : ''}
            >
              {isOpen ? (
                <>
                  <ListItemText 
                    primary={interface_.ifname}
                    sx={{ 
                      '& .MuiTypography-root': { 
                        fontWeight: selectedInterface?.id === interface_.id ? 'bold' : 'normal' 
                      }
                    }}
                  />
                  <CircleIcon
                    sx={{
                      fontSize: 12,
                      color: interface_.enabled ? '#4caf50' : '#f44336',
                      ml: 1,
                      filter: 'drop-shadow(0 0 1px rgba(128,128,128,0.8))'
                    }}
                  />
                </>
              ) : (
                <CircleIcon
                  sx={{
                    fontSize: 16,
                    color: interface_.enabled ? '#4caf50' : '#f44336',
                    filter: 'drop-shadow(0 0 1px rgba(128,128,128,0.8))'
                  }}
                />
              )}
            </ListItemButton>
          </ListItem>
        ))}
      </List>

      <Divider />
      <Box sx={{ p: 1, textAlign: 'center' }}>
        <IconButton 
          onClick={onAddInterface}
          sx={{ 
            backgroundColor: theme.palette.primary.main,
            color: 'white',
            '&:hover': {
              backgroundColor: theme.palette.primary.dark,
            }
          }}
          title={!isOpen ? 'Add Interface' : ''}
        >
          <AddIcon />
        </IconButton>
      </Box>
    </Box>
  );
};

export default Sidebar;