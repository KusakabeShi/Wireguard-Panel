import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  Alert,
  Typography,
  Divider,
  Tab,
  Tabs,
  Paper
} from '@mui/material';
import apiService from '../../services/apiService';

const TabPanel = ({ children, value, index, ...other }) => (
  <div
    role="tabpanel"
    hidden={value !== index}
    id={`settings-tabpanel-${index}`}
    aria-labelledby={`settings-tab-${index}`}
    {...other}
  >
    {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
  </div>
);

const SettingsDialog = ({ open, onClose, onSave }) => {
  const [tabValue, setTabValue] = useState(0);
  const [formData, setFormData] = useState({
    // Server settings (read-only display)
    wireguardConfigPath: '',
    user: '',
    listenIP: '',
    listenPort: '',
    siteUrlPrefix: '',
    apiPrefix: '',
    // Password change
    currentPassword: '',
    newPassword: '',
    confirmPassword: ''
  });
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState('');

  useEffect(() => {
    if (open) {
      loadServiceConfig();
      // Reset password fields when dialog opens
      setFormData(prev => ({
        ...prev,
        currentPassword: '',
        newPassword: '',
        confirmPassword: ''
      }));
      setError('');
      setSuccess('');
    }
  }, [open]);

  const loadServiceConfig = async () => {
    setLoading(true);
    try {
      const config = await apiService.getServiceConfig();
      setFormData(prev => ({
        ...prev,
        wireguardConfigPath: config.wireguardConfigPath || '',
        user: config.user || '',
        listenIP: config.listenIP || '',
        listenPort: config.listenPort?.toString() || '',
        siteUrlPrefix: config.siteUrlPrefix || '',
        apiPrefix: config.apiPrefix || ''
      }));
    } catch (error) {
      console.error('Failed to load service config:', error);
      setError(error.message || 'Failed to load service configuration');
    } finally {
      setLoading(false);
    }
  };

  const handlePasswordChange = (field, value) => {
    setFormData(prev => ({
      ...prev,
      [field]: value
    }));
    // Clear error when user starts typing
    if (error) setError('');
    if (success) setSuccess('');
  };

  const handleTabChange = (event, newValue) => {
    setTabValue(newValue);
    setError('');
    setSuccess('');
  };

  const handleSavePassword = async () => {
    setError('');
    setSuccess('');
    
    // Validation
    if (!formData.currentPassword) {
      setError('Current password is required');
      return;
    }
    
    if (!formData.newPassword) {
      setError('New password is required');
      return;
    }
    
    if (formData.newPassword !== formData.confirmPassword) {
      setError('Passwords do not match');
      return;
    }
    
    if (formData.newPassword.length < 6) {
      setError('Password must be at least 6 characters long');
      return;
    }

    setLoading(true);
    
    try {
      await apiService.updatePassword(formData.currentPassword, formData.newPassword);
      
      setSuccess('Password updated successfully!');
      setFormData(prev => ({
        ...prev,
        currentPassword: '',
        newPassword: '',
        confirmPassword: ''
      }));
      
      // Call onSave if provided
      if (onSave) {
        onSave();
      }
    } catch (error) {
      console.error('Failed to update password:', error);
      setError(error.message || 'Failed to update password');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog 
      open={open} 
      onClose={onClose}
      maxWidth="md"
      fullWidth
      PaperProps={{
        sx: { borderRadius: 2 }
      }}
    >
      <DialogTitle>Panel Settings</DialogTitle>
      
      <DialogContent sx={{ p: 0 }}>
        <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tabs value={tabValue} onChange={handleTabChange} aria-label="settings tabs">
            <Tab label="Server Information" />
            <Tab label="Change Password" />
          </Tabs>
        </Box>

        <TabPanel value={tabValue} index={0}>
          <Typography variant="h6" gutterBottom>
            Server Configuration
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
            Read-only server configuration information
          </Typography>
          
          {error && tabValue === 0 && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          )}
          
          <Box sx={{ display: 'grid', gap: 2 }}>
            <TextField
              label="WireGuard Config Path"
              value={formData.wireguardConfigPath}
              InputProps={{ readOnly: true }}
              fullWidth
              variant="outlined"
            />
            
            <TextField
              label="Username"
              value={formData.user}
              InputProps={{ readOnly: true }}
              fullWidth
              variant="outlined"
            />
            
            <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
              <TextField
                label="Listen IP"
                value={formData.listenIP}
                InputProps={{ readOnly: true }}
                variant="outlined"
              />
              
              <TextField
                label="Listen Port"
                value={formData.listenPort}
                InputProps={{ readOnly: true }}
                variant="outlined"
              />
            </Box>
            
            <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
              <TextField
                label="Site URL Prefix"
                value={formData.siteUrlPrefix}
                InputProps={{ readOnly: true }}
                variant="outlined"
              />
              
              <TextField
                label="API Prefix"
                value={formData.apiPrefix}
                InputProps={{ readOnly: true }}
                variant="outlined"
              />
            </Box>
          </Box>
        </TabPanel>

        <TabPanel value={tabValue} index={1}>
          <Typography variant="h6" gutterBottom>
            Change Login Password
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
            Update your login password. Changes will take effect immediately.
          </Typography>
          
          {error && tabValue === 1 && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          )}
          
          {success && (
            <Alert severity="success" sx={{ mb: 2 }}>
              {success}
            </Alert>
          )}
          
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, maxWidth: 400 }}>
            <TextField
              label="Current Password"
              type="password"
              value={formData.currentPassword}
              onChange={(e) => handlePasswordChange('currentPassword', e.target.value)}
              fullWidth
              variant="outlined"
              disabled={loading}
              autoComplete="current-password"
            />
            
            <TextField
              label="New Password"
              type="password"
              value={formData.newPassword}
              onChange={(e) => handlePasswordChange('newPassword', e.target.value)}
              fullWidth
              variant="outlined"
              disabled={loading}
              autoComplete="new-password"
            />
            
            <TextField
              label="Confirm New Password"
              type="password"
              value={formData.confirmPassword}
              onChange={(e) => handlePasswordChange('confirmPassword', e.target.value)}
              fullWidth
              variant="outlined"
              disabled={loading}
              autoComplete="new-password"
            />
            
            <Button
              variant="contained"
              onClick={handleSavePassword}
              disabled={loading || !formData.currentPassword || !formData.newPassword || !formData.confirmPassword}
              sx={{ alignSelf: 'flex-start', mt: 1 }}
            >
              {loading ? 'Updating...' : 'Update Password'}
            </Button>
          </Box>
        </TabPanel>
      </DialogContent>
      
      <DialogActions sx={{ px: 3, pb: 3 }}>
        <Button onClick={onClose}>
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default SettingsDialog;