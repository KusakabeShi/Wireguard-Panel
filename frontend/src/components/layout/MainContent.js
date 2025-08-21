import React from 'react';
import { Box, Typography, IconButton } from '@mui/material';
import { Add as AddIcon } from '@mui/icons-material';

const EmptyState = ({ onAddInterface }) => (
  <Box 
    sx={{ 
      display: 'flex', 
      flexDirection: 'column',
      alignItems: 'center', 
      justifyContent: 'center',
      height: '100%',
      color: '#666'
    }}
  >
    <Typography variant="h6" sx={{ mb: 2 }}>
      No interface selected
    </Typography>
    <IconButton 
      onClick={onAddInterface}
      sx={{ 
        backgroundColor: '#1976d2',
        color: 'white',
        width: 56,
        height: 56,
        '&:hover': {
          backgroundColor: '#1565c0',
        }
      }}
    >
      <AddIcon sx={{ fontSize: 32 }} />
    </IconButton>
  </Box>
);

const MainContent = ({ 
  selectedInterface, 
  onAddInterface,
  children 
}) => {
  return (
    <Box 
      sx={{ 
        flexGrow: 1, 
        height: 'calc(100vh - 64px)',
        overflow: 'auto',
        backgroundColor: '#ffffff'
      }}
    >
      {!selectedInterface ? (
        <EmptyState onAddInterface={onAddInterface} />
      ) : (
        children
      )}
    </Box>
  );
};

export default MainContent;