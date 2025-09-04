import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box
} from '@mui/material';
import ErrorDialog from './ErrorDialog';

const ClientDialog = ({ 
  open, 
  onClose, 
  onSave, 
  onDelete,
  client,
  title 
}) => {
  const [formData, setFormData] = useState({
    name: '',
    dns: '',
    ip: '',
    ipv6: '',
    privateKey: '',
    publicKey: '',
    presharedKey: '',
    keepalive: ''
  });
  const [loading, setLoading] = useState(false);
  const [errorDialog, setErrorDialog] = useState({ open: false, error: null, title: 'Error' });

  const isEdit = Boolean(client);

  useEffect(() => {
    if (client) {
      const dns = Array.isArray(client.dns) ? client.dns.join(', ') : '';
      
      setFormData({
        name: client.name || '',
        dns: dns,
        ip: client.ip || '',
        ipv6: client.ipv6 || '',
        privateKey: '',
        publicKey: client.publicKey || '',
        presharedKey: '',
        keepalive: client.keepalive?.toString() || ''
      });
    } else {
      setFormData({
        name: '',
        dns: '',
        ip: 'auto',
        ipv6: 'auto',
        privateKey: '',
        publicKey: '',
        presharedKey: '',
        keepalive: ''
      });
    }
  }, [client, open]);

  const handleChange = (field) => (event) => {
    setFormData(prev => ({
      ...prev,
      [field]: event.target.value
    }));
  };

  const handleSave = async () => {
    setLoading(true);

    try {
      const data = {
        name: formData.name
      };

      // DNS
      if (formData.dns) {
        data.dns = formData.dns.split(',').map(s => s.trim()).filter(s => s);
      }

      // IP addresses
      data.ip =  formData.ip || null;
      data.ipv6 = formData.ipv6 || null;

      // Keys
      if (formData.privateKey) {
        data.privateKey = formData.privateKey;
      } else if (formData.publicKey && !isEdit) {
        data.publicKey = formData.publicKey;
      }

      if (formData.presharedKey) {
        data.presharedKey = formData.presharedKey;
      }

      // Keepalive
      if (formData.keepalive) {
        data.keepalive = parseInt(formData.keepalive) || null;
      }

      await onSave(data);
      onClose();
    } catch (err) {
      setErrorDialog({ 
        open: true, 
        error: err.message || 'Failed to save client', 
        title: 'Save Failed' 
      });
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!client) return;
    
    setLoading(true);

    try {
      await onDelete(client.id);
      onClose();
    } catch (err) {
      setErrorDialog({ 
        open: true, 
        error: err.message || 'Failed to delete client', 
        title: 'Delete Failed' 
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Dialog 
        open={open} 
        onClose={onClose}
        maxWidth="sm"
        fullWidth
        PaperProps={{
          sx: { borderRadius: 2 }
        }}
      >
      <DialogTitle>{title || (isEdit ? 'Edit Client' : 'New Client')}</DialogTitle>
      
      <DialogContent>
        
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
          <TextField
            label="Name"
            value={formData.name}
            onChange={handleChange('name')}
            required
            fullWidth
            variant="outlined"
          />
          
          <TextField
            label="DNS"
            value={formData.dns}
            onChange={handleChange('dns')}
            fullWidth
            variant="outlined"
            placeholder="8.8.8.8, 1.1.1.1"
            helperText="Leave empty to inherit from server"
          />
          
          <TextField
            label="IP"
            value={formData.ip}
            onChange={handleChange('ip')}
            fullWidth
            variant="outlined"
            placeholder={isEdit ? "Current IP address" : "auto"}
            helperText={"Use 'auto' for automatic assignment"}
          />
          
          <TextField
            label="IPv6"
            value={formData.ipv6}
            onChange={handleChange('ipv6')}
            fullWidth
            variant="outlined"
            placeholder={isEdit ? "Current IPv6 address" : "auto"}
            helperText={"Use 'auto' for automatic assignment"}
          />
          
          <TextField
            label="Private key"
            value={formData.privateKey}
            onChange={handleChange('privateKey')}
            fullWidth
            variant="outlined"
            placeholder={isEdit ? "Leave empty to keep current key" : "Leave empty to generate new key"}
          />
          
          {!isEdit && (
            <TextField
              label="Public key"
              value={formData.publicKey}
              onChange={handleChange('publicKey')}
              fullWidth
              variant="outlined"
              helperText="Only used if Private Key is not provided"
            />
          )}
          
          <TextField
            label="Preshared Key"
            value={formData.presharedKey}
            onChange={handleChange('presharedKey')}
            fullWidth
            variant="outlined"
            placeholder={isEdit ? "Leave empty to keep current" : "Leave empty for none"}
          />
          
          <TextField
            label="Keepalive"
            type="number"
            value={formData.keepalive}
            onChange={handleChange('keepalive')}
            fullWidth
            variant="outlined"
            helperText="PersistentKeepalive interval in seconds"
          />
        </Box>
      </DialogContent>
      
      <DialogActions sx={{ px: 3, pb: 3 }}>
        {isEdit && (
          <Button 
            onClick={handleDelete}
            color="error"
            disabled={loading}
          >
            DELETE
          </Button>
        )}
        <Box sx={{ flexGrow: 1 }} />
        <Button 
          onClick={onClose} 
          disabled={loading}
        >
          CANCEL
        </Button>
        <Button 
          onClick={handleSave}
          variant="contained"
          disabled={loading}
        >
          SAVE
        </Button>
      </DialogActions>
      </Dialog>

      <ErrorDialog
        open={errorDialog.open}
        onClose={() => setErrorDialog({ open: false, error: null, title: 'Error' })}
        error={errorDialog.error}
        title={errorDialog.title}
      />
    </>
  );
};

export default ClientDialog;