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
import { ContentCopy as CopyIcon, QrCode as QrCodeIcon, Download as DownloadIcon } from '@mui/icons-material';
import { formatBytes, formatLastHandshake } from '../../utils/formatUtils';
import apiService from '../../services/apiService';
import QRCodeDialog from '../dialogs/QRCodeDialog';

const ClientDetails = ({ client, clientState, lastUpdateTime, interfaceId, serverId, interfaceInfo, serverInfo, visible = true }) => {
  const [config, setConfig] = useState('');
  const [loadingConfig, setLoadingConfig] = useState(false);
  const [configLoaded, setConfigLoaded] = useState(false);
  const [qrDialogOpen, setQrDialogOpen] = useState(false);
  const [error, setError] = useState(null);
  const [copySuccess, setCopySuccess] = useState(false);

  // Only load config when component becomes visible
  useEffect(() => {
    if (visible && !configLoaded) {
      loadConfig();
    }
  }, [visible, client.id, configLoaded]);


  const loadConfig = async () => {
    setLoadingConfig(true);
    try {
      const configText = await apiService.getClientConfig(interfaceId, serverId, client.id);
      setConfig(configText);
      setConfigLoaded(true);
    } catch (error) {
      console.error('Failed to load client config:', error);
      setConfig('Failed to load configuration');
      setConfigLoaded(true); // Mark as loaded even on error to prevent retry
      setError(error.message || 'Failed to load configuration');
    } finally {
      setLoadingConfig(false);
    }
  };

  const handleCopyConfig = async () => {
    try {
      // Check if clipboard API is available
      if (navigator.clipboard && navigator.clipboard.writeText) {
        await navigator.clipboard.writeText(config);
        setCopySuccess(true);
        setTimeout(() => setCopySuccess(false), 2000);
      } else {
        // Fallback for browsers without clipboard API
        const textArea = document.createElement('textarea');
        textArea.value = config;
        textArea.style.position = 'fixed';
        textArea.style.left = '-999999px';
        textArea.style.top = '-999999px';
        document.body.appendChild(textArea);
        textArea.focus();
        textArea.select();
        
        try {
          document.execCommand('copy');
          setCopySuccess(true);
          setTimeout(() => setCopySuccess(false), 2000);
        } catch (err) {
          console.error('Fallback copy failed:', err);
          setError('Failed to copy to clipboard. Please copy manually.');
        } finally {
          document.body.removeChild(textArea);
        }
      }
    } catch (err) {
      console.error('Copy to clipboard failed:', err);
      setError('Failed to copy to clipboard. Please copy manually.');
    }
  };

  const handleDownloadConfig = () => {
    if (!config || !interfaceInfo || !serverInfo) return;


    // Strip wgIfPrefix from interface name
    let interfaceName = interfaceInfo.ifname || 'interface';
    
    if (interfaceInfo.wgIfPrefix && interfaceName.startsWith(interfaceInfo.wgIfPrefix)) {
      interfaceName = interfaceName.substring(interfaceInfo.wgIfPrefix.length);
    } 

    // Generate filename: ifname-servername-clientname.conf
    const fileName = `${interfaceName}-${serverInfo.name || 'server'}-${client.name || 'client'}.conf`;
    
    // Create blob and download
    const blob = new Blob([config], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = fileName;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
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
              <TableCell>{formatLastHandshake(lastUpdateTime, clientState?.latestHandshake)}</TableCell>
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
            onClick={handleDownloadConfig}
            size="small"
            disabled={loadingConfig || !config || !interfaceInfo || !serverInfo}
            title="Download config file"
          >
            <DownloadIcon fontSize="small" />
          </IconButton>
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

      <Snackbar
        open={copySuccess}
        autoHideDuration={2000}
        onClose={() => setCopySuccess(false)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert onClose={() => setCopySuccess(false)} severity="success" sx={{ width: '100%' }}>
          Configuration copied to clipboard!
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default ClientDetails;