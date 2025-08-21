import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  CircularProgress
} from '@mui/material';
import QRCode from 'qrcode';

const QRCodeDialog = ({ open, onClose, config, clientName }) => {
  const [qrCodeDataUrl, setQrCodeDataUrl] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (open && config) {
      generateQRCode();
    }
  }, [open, config]);

  const generateQRCode = async () => {
    if (!config) return;
    
    setLoading(true);
    setError('');
    
    try {
      const dataUrl = await QRCode.toDataURL(config, {
        width: 300,
        margin: 2,
        color: {
          dark: '#000000',
          light: '#FFFFFF'
        }
      });
      setQrCodeDataUrl(dataUrl);
    } catch (err) {
      console.error('Failed to generate QR code:', err);
      setError('Failed to generate QR code');
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setQrCodeDataUrl('');
    setError('');
    onClose();
  };

  return (
    <Dialog 
      open={open} 
      onClose={handleClose}
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle>
        WireGuard QR Code
        {clientName && (
          <Typography variant="subtitle2" color="text.secondary">
            {clientName}
          </Typography>
        )}
      </DialogTitle>
      
      <DialogContent>
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            py: 2
          }}
        >
          {loading && (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <CircularProgress size={24} />
              <Typography>Generating QR code...</Typography>
            </Box>
          )}
          
          {error && (
            <Typography color="error" align="center">
              {error}
            </Typography>
          )}
          
          {qrCodeDataUrl && !loading && !error && (
            <>
              <Box
                component="img"
                src={qrCodeDataUrl}
                alt="WireGuard Configuration QR Code"
                sx={{
                  maxWidth: '100%',
                  height: 'auto',
                  border: '1px solid #e0e0e0',
                  borderRadius: 1,
                  mb: 2
                }}
              />
              <Typography 
                variant="body2" 
                color="text.secondary" 
                align="center"
              >
                Scan this QR code with your WireGuard mobile app to import the configuration.
              </Typography>
            </>
          )}
        </Box>
      </DialogContent>
      
      <DialogActions>
        <Button onClick={handleClose}>
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default QRCodeDialog;