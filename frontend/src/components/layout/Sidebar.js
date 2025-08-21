import React from 'react';
import { 
  Box, 
  List, 
  ListItem, 
  ListItemButton, 
  ListItemText, 
  Typography, 
  IconButton,
  Divider
} from '@mui/material';
import { Menu as MenuIcon, Add as AddIcon } from '@mui/icons-material';

const Sidebar = ({ 
  interfaces, 
  selectedInterface, 
  onInterfaceSelect, 
  onAddInterface 
}) => {
  return (
    <Box 
      sx={{ 
        width: 200, 
        height: 'calc(100vh - 64px)',
        borderRight: '2px solid #e0e0e0',
        backgroundColor: '#fafafa',
        display: 'flex',
        flexDirection: 'column'
      }}
    >
      <Box 
        sx={{ 
          p: 2, 
          borderBottom: '1px solid #e0e0e0',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between'
        }}
      >
        <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
          Interfaces
        </Typography>
        <IconButton size="small">
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
                '&.Mui-selected': {
                  backgroundColor: '#e3f2fd',
                  borderRight: '3px solid #1976d2',
                },
                '&:hover': {
                  backgroundColor: '#f5f5f5',
                }
              }}
            >
              <ListItemText 
                primary={interface_.ifname}
                sx={{ 
                  '& .MuiTypography-root': { 
                    fontWeight: selectedInterface?.id === interface_.id ? 'bold' : 'normal' 
                  }
                }}
              />
            </ListItemButton>
          </ListItem>
        ))}
      </List>

      <Divider />
      <Box sx={{ p: 1, textAlign: 'center' }}>
        <IconButton 
          onClick={onAddInterface}
          sx={{ 
            backgroundColor: '#1976d2',
            color: 'white',
            '&:hover': {
              backgroundColor: '#1565c0',
            }
          }}
        >
          <AddIcon />
        </IconButton>
      </Box>
    </Box>
  );
};

export default Sidebar;