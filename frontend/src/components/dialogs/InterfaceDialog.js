import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  Alert
} from '@mui/material';

const InterfaceDialog = ({ 
  open, 
  onClose, 
  onSave, 
  onDelete,
  interface_,
  title 
}) => {
  const [formData, setFormData] = useState({
    ifname: '',
    endpoint: '',
    port: '',
    mtu: '',
    privateKey: '',
    vrfName: '',
    fwMark: ''
  });
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const isEdit = Boolean(interface_);

  useEffect(() => {
    if (interface_) {
      setFormData({
        ifname: interface_.ifname || '',
        endpoint: interface_.endpoint || '',
        port: interface_.port?.toString() || '',
        mtu: interface_.mtu?.toString() || '',
        privateKey: '',
        vrfName: interface_.vrfName || '',
        fwMark: interface_.fwMark || ''
      });
    } else {
      setFormData({
        ifname: '',
        endpoint: '',
        port: '',
        mtu: '1420',
        privateKey: '',
        vrfName: '',
        fwMark: ''
      });
    }
    setError('');
  }, [interface_, open]);

  const handleChange = (field) => (event) => {
    setFormData(prev => ({
      ...prev,
      [field]: event.target.value
    }));
  };

  const handleSave = async () => {
    setError('');
    setLoading(true);

    try {
      const data = {
        ifname: formData.ifname,
        endpoint: formData.endpoint,
        port: parseInt(formData.port) || 51820,
        mtu: parseInt(formData.mtu) || 1420,
        vrfName: formData.vrfName || null,
        fwMark: formData.fwMark || null
      };

      if (formData.privateKey) {
        data.privateKey = formData.privateKey;
      }

      await onSave(data);
      onClose();
    } catch (err) {
      setError(err.message || 'Failed to save interface');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!interface_) return;
    
    setError('');
    setLoading(true);

    try {
      await onDelete(interface_.id);
      onClose();
    } catch (err) {
      setError(err.message || 'Failed to delete interface');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog 
      open={open} 
      onClose={onClose}
      maxWidth="sm"
      fullWidth
      PaperProps={{
        sx: { borderRadius: 2 }
      }}
    >
      <DialogTitle>{title || (isEdit ? 'Edit Interface' : 'New Interface')}</DialogTitle>
      
      <DialogContent>
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
          <TextField
            label="Name"
            value={formData.ifname}
            onChange={handleChange('ifname')}
            required
            fullWidth
            variant="outlined"
          />
          
          <TextField
            label="Endpoint"
            value={formData.endpoint}
            onChange={handleChange('endpoint')}
            required
            fullWidth
            variant="outlined"
          />
          
          <TextField
            label="Port"
            type="number"
            value={formData.port}
            onChange={handleChange('port')}
            required
            fullWidth
            variant="outlined"
          />
          
          <TextField
            label="MTU"
            type="number"
            value={formData.mtu}
            onChange={handleChange('mtu')}
            fullWidth
            variant="outlined"
          />
          
          <TextField
            label="Private Key"
            value={formData.privateKey}
            onChange={handleChange('privateKey')}
            fullWidth
            variant="outlined"
            placeholder={isEdit ? "Leave empty to keep current key" : "Leave empty to generate new key"}
          />
          
          <TextField
            label="VRF Name"
            value={formData.vrfName}
            onChange={handleChange('vrfName')}
            fullWidth
            variant="outlined"
          />
          
          <TextField
            label="FWMark"
            value={formData.fwMark}
            onChange={handleChange('fwMark')}
            fullWidth
            variant="outlined"
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
  );
};

export default InterfaceDialog;