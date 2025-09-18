import React from 'react';
import { Box, Typography, IconButton, useTheme, useMediaQuery } from '@mui/material';
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
  children,
  sx = {}
}) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  
  return (
    <Box 
      sx={{ 
        flexGrow: 1, 
        height: '100%',
        overflowY: 'auto',
        backgroundColor: theme.palette.background.background,
        paddingTop: isMobile ? theme.spacing(2) : theme.spacing(3),
        paddingBottom: isMobile ? theme.spacing(2) : theme.spacing(3),
        paddingLeft: isMobile ? theme.spacing(2) : theme.spacing(3),
        paddingRight: isMobile ? theme.spacing(2) : theme.spacing(3),
        ...sx
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

