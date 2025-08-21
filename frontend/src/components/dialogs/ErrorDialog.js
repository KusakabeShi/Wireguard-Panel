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
    
    // If it's a string, return it directly
    if (typeof error === 'string') return error;
    
    // If it has a message property
    if (error.message) return error.message;
    
    // If it's an object, try to stringify it
    if (typeof error === 'object') {
      try {
        return JSON.stringify(error, null, 2);
      } catch (e) {
        return 'Unable to parse error details';
      }
    }
    
    return String(error);
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
              fontFamily: 'inherit'
            }}
          >
            {getErrorMessage(error)}
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