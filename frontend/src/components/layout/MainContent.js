import React from 'react';
import { Box, Typography, IconButton, useTheme } from '@mui/material';
import { Add as AddIcon } from '@mui/icons-material';

const EmptyState = ({ onAddInterface }) => {
  const theme = useTheme();
  
  return (
    <Box 
      sx={{ 
        display: 'flex', 
        flexDirection: 'column',
        alignItems: 'center', 
        justifyContent: 'center',
        height: '100%',
        color: theme.palette.text.secondary
      }}
    >
      <Typography variant="h6" sx={{ mb: 2 }}>
        No interface selected
      </Typography>
      <IconButton 
        onClick={onAddInterface}
        sx={{ 
          backgroundColor: theme.palette.primary.main,
          color: 'white',
          width: 56,
          height: 56,
          '&:hover': {
            backgroundColor: theme.palette.primary.dark,
          }
        }}
      >
        <AddIcon sx={{ fontSize: 32 }} />
      </IconButton>
    </Box>
  );
};

const MainContent = ({ 
  selectedInterface, 
  onAddInterface,
  children 
}) => {
  const theme = useTheme();
  
  return (
    <Box 
      sx={{ 
        flexGrow: 1, 
        height: 'calc(100vh - 64px)',
        overflow: 'auto',
        backgroundColor: theme.palette.background.default
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