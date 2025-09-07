import React from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Box,
  Alert,
  AlertTitle
} from '@mui/material';
import { Error as ErrorIcon } from '@mui/icons-material';

const ErrorDialog = ({ open, onClose, error, title = "Error" }) => {
  const getErrorMessage = (error) => {
    if (!error) return 'An unknown error occurred';
    
    let errorStr;
    
    // If it's a string, use it directly
    if (typeof error === 'string') {
      errorStr = error;
    }
    // If it has a message property
    else if (error.message) {
      errorStr = error.message;
    }
    // If it's an object, try to stringify it
    else if (typeof error === 'object') {
      try {
        errorStr = JSON.stringify(error, null, 2);
      } catch (e) {
        errorStr = 'Unable to parse error details';
      }
    } else {
      errorStr = String(error);
    }
    
    return errorStr;
  };

  const formatHierarchicalError = (errorMessage) => {
    // Split by :-> to get error hierarchy
    const parts = errorMessage.split(':->').map(part => part.trim());
    
    if (parts.length === 1) {
      // No hierarchy, return as is
      return errorMessage;
    }
    
    // Format with hierarchy using └> symbol
    return parts.map((part, index) => {
      if (index === 0) {
        return part;
      } else {
        const indent = '  '.repeat(index - 1); // Two spaces per level
        return `${indent}└> ${part}`;
      }
    }).join('\n');
  };

  const handleClose = () => {
    onClose();
  };

  return (
    <Dialog 
      open={open} 
      onClose={handleClose}
      maxWidth="sm"
      fullWidth
      sx={{ zIndex: 2000 }}
    >
      <DialogTitle sx={{ 
        display: 'flex', 
        alignItems: 'center', 
        gap: 1,
        pb: 1
      }}>
        <ErrorIcon color="error" />
        {title}
      </DialogTitle>
      
      <DialogContent>
        <Alert severity="error" sx={{ mb: 2 }}>
          <AlertTitle>Operation Failed</AlertTitle>
          <Typography 
            variant="body2" 
            component="pre" 
            sx={{ 
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-word',
              fontFamily: 'monospace',
              lineHeight: 1.6
            }}
          >
            {formatHierarchicalError(getErrorMessage(error))}
          </Typography>
        </Alert>
        
        <Typography variant="body2" color="text.secondary">
          Please try again or contact support if the problem persists.
        </Typography>
      </DialogContent>
      
      <DialogActions>
        <Button onClick={handleClose} variant="contained">
          OK
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default ErrorDialog;