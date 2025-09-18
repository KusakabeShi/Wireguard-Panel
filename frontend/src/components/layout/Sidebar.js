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
  Drawer,
  useTheme
} from '@mui/material';
import { Menu as MenuIcon, Add as AddIcon, Circle as CircleIcon } from '@mui/icons-material';

const Sidebar = ({ 
  interfaces, 
  selectedInterface, 
  onInterfaceSelect, 
  onAddInterface,
  isOpen,
  onToggle,
  onClose,
  isMobile = false
}) => {
  const theme = useTheme();
  const content = (
    <Box 
      sx={{ 
        width: isMobile ? '100%' : isOpen ? 240 : 72, 
        height: '100%',
        borderRight: `2px solid ${theme.palette.divider}`,
        backgroundColor: theme.palette.background.sidebar,
        display: 'flex',
        flexDirection: 'column',
        transition: isMobile ? 'none' : 'width 0.3s ease',
        overflow: 'hidden'
      }}
    >
      <Box 
        sx={{ 
          p: 2, 
          borderBottom: `1px solid ${theme.palette.divider}`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: isOpen ? 'flex-start' : 'center',
          gap: isOpen ? 1 : 0,
          minHeight: 56
        }}
      >
        <IconButton size="small" onClick={onToggle}>
          <MenuIcon />
        </IconButton>
        {isOpen && (
          <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
            Interfaces
          </Typography>
        )}
      </Box>
      
      <List sx={{ flexGrow: 1, p: 0 }}>
        {interfaces.sort((a, b) => a.ifname.localeCompare(b.ifname)).map((interface_) => (
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

  if (isMobile) {
    return (
      <Drawer
        variant="temporary"
        open={isOpen}
        onClose={onClose || onToggle}
        ModalProps={{ keepMounted: true }}
        sx={{
          '& .MuiDrawer-paper': {
            width: 'min(70vw, 280px)',
            backgroundColor: theme.palette.background.sidebar,
            borderRight: `2px solid ${theme.palette.divider}`
          }
        }}
      >
        {content}
      </Drawer>
    );
  }

  return content;
};

export default Sidebar;

