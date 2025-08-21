import React, { useState, useEffect } from 'react';
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableContainer, 
  TableRow,
  TextField,
  IconButton,
  Box,
  Paper,
  Snackbar,
  Alert
} from '@mui/material';
import { ContentCopy as CopyIcon, QrCode as QrCodeIcon } from '@mui/icons-material';
import { formatBytes, formatLastHandshake } from '../../utils/formatUtils';
import apiService from '../../services/apiService';
import QRCodeDialog from '../dialogs/QRCodeDialog';

const ClientDetails = ({ client, clientState, interfaceId, serverId }) => {
  const [config, setConfig] = useState('');
  const [loadingConfig, setLoadingConfig] = useState(false);
  const [qrDialogOpen, setQrDialogOpen] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    loadConfig();
  }, [client.id]);

  const loadConfig = async () => {
    setLoadingConfig(true);
    try {
      const configText = await apiService.getClientConfig(interfaceId, serverId, client.id);
      setConfig(configText);
    } catch (error) {
      console.error('Failed to load client config:', error);
      setConfig('Failed to load configuration');
      setError(error.message || 'Failed to load configuration');
    } finally {
      setLoadingConfig(false);
    }
  };

  const handleCopyConfig = () => {
    navigator.clipboard.writeText(config);
  };

  const handleShowQRCode = () => {
    setQrDialogOpen(true);
  };

  const getTrafficDisplay = () => {
    if (!clientState || (!clientState.transferTx && !clientState.transferRx)) {
      return 'No data';
    }
    
    const tx = formatBytes(clientState.transferTx || 0);
    const rx = formatBytes(clientState.transferRx || 0);
    return `Tx: ${tx}, Rx: ${rx}`;
  };

  return (
    <Box>
      <TableContainer component={Paper} elevation={0}>
        <Table size="small">
          <TableBody>
            <TableRow>
              <TableCell sx={{ fontWeight: 'bold', width: 150 }}>IP:</TableCell>
              <TableCell>{client.ip || 'null'}</TableCell>
            </TableRow>
            <TableRow>
              <TableCell sx={{ fontWeight: 'bold' }}>IPv6:</TableCell>
              <TableCell>{client.ipv6 || 'null'}</TableCell>
            </TableRow>
            <TableRow>
              <TableCell sx={{ fontWeight: 'bold' }}>Transferred:</TableCell>
              <TableCell>{getTrafficDisplay()}</TableCell>
            </TableRow>
            <TableRow>
              <TableCell sx={{ fontWeight: 'bold' }}>Last handshake:</TableCell>
              <TableCell>{formatLastHandshake(clientState?.latestHandshake)}</TableCell>
            </TableRow>
            <TableRow>
              <TableCell sx={{ fontWeight: 'bold' }}>Endpoint:</TableCell>
              <TableCell>{clientState?.endpoint || 'Not connected'}</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </TableContainer>

      <Box sx={{ mt: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
          <Box sx={{ fontWeight: 'bold', mr: 1 }}>WireGuard config:</Box>
          <IconButton 
            onClick={handleCopyConfig}
            size="small"
            disabled={loadingConfig || !config}
            title="Copy to clipboard"
          >
            <CopyIcon fontSize="small" />
          </IconButton>
          <IconButton 
            onClick={handleShowQRCode}
            size="small"
            disabled={loadingConfig || !config}
            title="Show QR code"
          >
            <QrCodeIcon fontSize="small" />
          </IconButton>
        </Box>
        <TextField
          multiline
          rows={8}
          fullWidth
          value={loadingConfig ? 'Loading...' : config}
          variant="outlined"
          size="small"
          InputProps={{
            readOnly: true,
            style: { 
              fontFamily: 'monospace',
              fontSize: '0.875rem'
            }
          }}
        />
      </Box>

      <QRCodeDialog
        open={qrDialogOpen}
        onClose={() => setQrDialogOpen(false)}
        config={config}
        clientName={client.name}
      />

      <Snackbar
        open={!!error}
        autoHideDuration={6000}
        onClose={() => setError(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={() => setError(null)} severity="error" sx={{ width: '100%' }}>
          {error}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default ClientDetails;