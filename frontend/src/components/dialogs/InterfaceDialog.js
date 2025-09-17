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
import apiService from '../../services/apiService';
import ErrorDialog from './ErrorDialog';

const InterfaceDialog = ({ 
  open, 
  onClose, 
  onSave, 
  onDelete,
  onToggleEnable,
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
  const [loading, setLoading] = useState(false);
  const [isEnabled, setIsEnabled] = useState(false);
  const [serviceConfig, setServiceConfig] = useState(null);
  const [validationError, setValidationError] = useState('');
  const [errorDialog, setErrorDialog] = useState({ open: false, error: null, title: 'Error' });

  const isEdit = Boolean(interface_);

  // Fetch service configuration when dialog opens
  useEffect(() => {
    if (open && !serviceConfig) {
      apiService.getServiceConfig()
        .then(config => setServiceConfig(config))
        .catch(err => console.error('Failed to fetch service config:', err));
    }
  }, [open, serviceConfig]);

  useEffect(() => {
    if (open && interface_) {
      // Always use the current interface data when dialog opens
      setFormData({
        ifname: interface_.ifname || '',
        endpoint: interface_.endpoint || '',
        port: interface_.port?.toString() || '',
        mtu: interface_.mtu?.toString() || '',
        privateKey: '',
        vrfName: interface_.vrfName || '',
        fwMark: interface_.fwMark || ''
      });
      setIsEnabled(interface_.enabled || false);
      setValidationError('');
    } else if (open && !interface_) {
      // New interface
      setFormData({
        ifname: serviceConfig?.wgIfPrefix,
        endpoint: window.location.hostname,
        port: ( Math.floor(Math.random() * 10000)+50000).toString(),
        mtu: '1420',
        privateKey: '',
        vrfName: '',
        fwMark: ''
      });
      setIsEnabled(false);
      setValidationError('');
    }
  }, [interface_, open]);

  const validateInterfaceName = (name) => {
    if (!serviceConfig || !serviceConfig.wgIfPrefix) return '';
    
    if (name && !name.startsWith(serviceConfig.wgIfPrefix)) {
      return `Interface name must start with "${serviceConfig.wgIfPrefix}"`;
    } else if (name && name.length > 15) {// len > 15
      return 'Interface name must be 15 characters or less';
    } else if (name && !/^[a-zA-Z0-9_-]*$/.test(name)) {// contains invalid charactor, only 0-9a-zA-Z_-
      return 'Interface name must contain only 0-9a-zA-Z_-';
    }
    return '';
  };

  const handleChange = (field) => (event) => {
    const value = event.target.value;
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));

    // Validate interface name if it's the ifname field
    if (field === 'ifname') {
      setValidationError(validateInterfaceName(value));
    }
  };

  const handleSave = async () => {
    setLoading(true);

    // Check validation first
    const nameValidationError = validateInterfaceName(formData.ifname);
    if (nameValidationError) {
      setValidationError(nameValidationError);
      setLoading(false);
      return;
    }

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
      setErrorDialog({ 
        open: true, 
        error: err.message || 'Failed to save interface', 
        title: 'Save Failed' 
      });
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!interface_) return;
    
    setLoading(true);

    try {
      await onDelete(interface_.id);
      onClose();
    } catch (err) {
      setErrorDialog({ 
        open: true, 
        error: err.message || 'Failed to delete interface', 
        title: 'Delete Failed' 
      });
    } finally {
      setLoading(false);
    }
  };

  const handleToggleEnable = async () => {
    if (!interface_) return;
    
    setLoading(true);

    try {
      await onToggleEnable(interface_.id, !isEnabled);
      setIsEnabled(!isEnabled);
    } catch (err) {
      setErrorDialog({ 
        open: true, 
        error: err.message || 'Failed to update interface status', 
        title: 'Toggle Failed' 
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
        <DialogTitle>{title || (isEdit ? 'Edit Interface' : 'New Interface')}</DialogTitle>
        
        <DialogContent>        
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            <TextField
              label="Name"
              value={formData.ifname}
              onChange={handleChange('ifname')}
              required
              fullWidth
              variant="outlined"
              error={!!validationError}
              helperText={validationError || ""}
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
          {isEdit && (
            <Button 
              onClick={handleToggleEnable}
              color={isEnabled ? "warning" : "success"}
              disabled={loading}
              sx={{ ml: 1 }}
            >
              {isEnabled ? "DISABLE" : "ENABLE"}
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

export default InterfaceDialog;