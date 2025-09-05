import React, { useState, useEffect, useRef } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Box,
  Alert,
  Checkbox,
  FormControlLabel,
  Typography,
  Divider
} from '@mui/material';
import ErrorDialog from './ErrorDialog';
import apiService from '../../services/apiService';

const ServerDialog = ({ 
  open, 
  onClose, 
  onSave, 
  onDelete,
  server,
  title 
}) => {
  const [formData, setFormData] = useState({
    name: '',
    dns: '',
    ipv4: {
      enabled: false,
      network: '',
      pseudoBridgeMasterInterface: '',
      pseudoBridgeMasterInterfaceEnabled: false,
      routedNetworks: '',
      routedNetworksFirewall: false,
      snat: {
        enabled: false,
        snatIpNet: '',
        snatExcludedNetwork: '',
        roamingMasterInterface: '',
        roamingMasterInterfaceEnabled: false
      }
    },
    ipv6: {
      enabled: false,
      network: '',
      pseudoBridgeMasterInterface: '',
      pseudoBridgeMasterInterfaceEnabled: false,
      routedNetworks: '',
      routedNetworksFirewall: false,
      snat: {
        enabled: false,
        snatIpNet: '',
        snatExcludedNetwork: '',
        roamingMasterInterface: '',
        roamingMasterInterfaceEnabled: false,
        roamingPseudoBridge: false
      }
    }
  });
  const [warnings, setWarnings] = useState([]);
  const [loading, setLoading] = useState(false);
  const [errorDialog, setErrorDialog] = useState({ open: false, error: null, title: 'Error' });
  const [validationErrors, setValidationErrors] = useState({
    ipv4: { snatRoamingInterface: '', snatIpNet: '' },
    ipv6: { snatRoamingInterface: '', snatIpNet: '' }
  });
  const [validationSuccess, setValidationSuccess] = useState({
    ipv4: { snatIpNet: '' },
    ipv6: { snatIpNet: '' }
  });
  const validationTimeouts = useRef({
    ipv4: null,
    ipv6: null
  });

  const isEdit = Boolean(server);

  useEffect(() => {
    if (server) {
      const dns = Array.isArray(server.dns) ? server.dns.join(', ') : '';
      
      setFormData({
        name: server.name || '',
        dns: dns,
        ipv4: {
          enabled: server.ipv4?.enabled || false,
          network: server.ipv4?.network || '',
          pseudoBridgeMasterInterface: server.ipv4?.pseudoBridgeMasterInterface || '',
          pseudoBridgeMasterInterfaceEnabled: Boolean(server.ipv4?.pseudoBridgeMasterInterface),
          routedNetworks: Array.isArray(server.ipv4?.routedNetworks) 
            ? server.ipv4.routedNetworks.join('\n') 
            : '',
          routedNetworksFirewall: server.ipv4?.routedNetworksFirewall || false,
          snat: {
            enabled: server.ipv4?.snat?.enabled || false,
            snatIpNet: server.ipv4?.snat?.snatIpNet || '',
            snatExcludedNetwork: server.ipv4?.snat?.snatExcludedNetwork || '',
            roamingMasterInterface: server.ipv4?.snat?.roamingMasterInterface || '',
            roamingMasterInterfaceEnabled: Boolean(server.ipv4?.snat?.roamingMasterInterface)
          }
        },
        ipv6: {
          enabled: server.ipv6?.enabled || false,
          network: server.ipv6?.network || '',
          pseudoBridgeMasterInterface: server.ipv6?.pseudoBridgeMasterInterface || '',
          pseudoBridgeMasterInterfaceEnabled: Boolean(server.ipv6?.pseudoBridgeMasterInterface),
          routedNetworks: Array.isArray(server.ipv6?.routedNetworks) 
            ? server.ipv6.routedNetworks.join('\n') 
            : '',
          routedNetworksFirewall: server.ipv6?.routedNetworksFirewall || false,
          snat: {
            enabled: server.ipv6?.snat?.enabled || false,
            snatIpNet: server.ipv6?.snat?.snatIpNet || '',
            snatExcludedNetwork: server.ipv6?.snat?.snatExcludedNetwork || '',
            roamingMasterInterface: server.ipv6?.snat?.roamingMasterInterface || '',
            roamingMasterInterfaceEnabled: Boolean(server.ipv6?.snat?.roamingMasterInterface),
            roamingPseudoBridge: server.ipv6?.snat?.roamingPseudoBridge || false
          }
        }
      });
    } else {
      setFormData({
        name: '',
        dns: '',
        ipv4: {
          enabled: true,
          network: '',
          pseudoBridgeMasterInterface: '',
          pseudoBridgeMasterInterfaceEnabled: false,
          routedNetworks: '',
          routedNetworksFirewall: false,
          snat: {
            enabled: false,
            snatIpNet: '',
            snatExcludedNetwork: '',
            roamingMasterInterface: '',
            roamingMasterInterfaceEnabled: false
          }
        },
        ipv6: {
          enabled: false,
          network: '',
          pseudoBridgeMasterInterface: '',
          pseudoBridgeMasterInterfaceEnabled: false,
          routedNetworks: '',
          routedNetworksFirewall: false,
          snat: {
            enabled: false,
            snatIpNet: '',
            snatExcludedNetwork: '',
            roamingMasterInterface: '',
            roamingMasterInterfaceEnabled: false,
            roamingPseudoBridge: false
          }
        }
      });
    }
    setWarnings([]);
    setValidationErrors({
      ipv4: { snatRoamingInterface: '', snatIpNet: '' },
      ipv6: { snatRoamingInterface: '', snatIpNet: '' }
    });
    setValidationSuccess({
      ipv4: { snatIpNet: '' },
      ipv6: { snatIpNet: '' }
    });
    
    // Clear any pending validation timeouts when dialog closes
    Object.values(validationTimeouts.current).forEach(timeout => {
      if (timeout) {
        clearTimeout(timeout);
      }
    });
    validationTimeouts.current = {
      ipv4: null,
      ipv6: null
    };
  }, [server, open]);

  useEffect(() => {
    checkValidation();
  }, [formData]);

  // Separate useEffect for initial SNAT validation when dialog opens
  useEffect(() => {
    if (open && server) {
      // Trigger SNAT roaming validation for existing configurations when dialog opens
      setTimeout(() => {
        ['ipv4', 'ipv6'].forEach(ipVersion => {
          const ipConfig = formData[ipVersion];
          if (ipConfig.enabled && 
              ipConfig.snat.enabled && 
              ipConfig.snat.roamingMasterInterfaceEnabled &&
              ipConfig.snat.roamingMasterInterface &&
              ipConfig.snat.snatIpNet) {
            validateSNATRoaming(
              ipVersion,
              ipConfig.snat.roamingMasterInterface,
              ipConfig.snat.snatIpNet
            );
          }
        });
      }, 100);
    }
  }, [open, server]);

  const checkValidation = () => {
    const newWarnings = [];

    // At least one IP version must be enabled
    if (!formData.ipv4.enabled && !formData.ipv6.enabled) {
      newWarnings.push('At least one of IPv4 or IPv6 must be enabled');
    }

    setWarnings(newWarnings);
  };

  const validateSNATRoaming = async (ipVersion, masterInterface, snatIpNet) => {
    if (!masterInterface || !snatIpNet) {
      return;
    }

    const addressFamily = ipVersion === 'ipv4' ? '4' : '6';
    
    try {
      const result = await apiService.validateSNATRoamingOffset(masterInterface, snatIpNet, addressFamily);
      
      // Clear any previous errors for this IP version
      setValidationErrors(prev => ({
        ...prev,
        [ipVersion]: {
          snatRoamingInterface: '',
          snatIpNet: ''
        }
      }));
      
      // Store success information with mapped network hint
      setValidationSuccess(prev => ({
        ...prev,
        [ipVersion]: {
          snatIpNet: result?.['mapped network'] 
            ? `Will ${result.type} to ${result['mapped network']} on interface ${masterInterface}`
            : 'Valid'
        }
      }));
    } catch (error) {
      // Use structured error data if available, otherwise fallback to parsing
      let errorData = error.errorData || {};
      
      if (!errorData.error_params) {
        // Fallback error parsing if structured data isn't available
        const errorMessage = error.message;
        errorData.error = errorMessage;
        
        // Determine error_params based on error content
        if (errorMessage.includes('interface') || errorMessage.includes('ifname')) {
          errorData.error_params = 'ifname';
        } else if (errorMessage.includes('offset') || errorMessage.includes('network') || errorMessage.includes('CIDR')) {
          errorData.error_params = 'offset';
        } else {
          errorData.error_params = 'offset'; // Default to offset for unknown errors
        }
      }
      
      setValidationErrors(prev => ({
        ...prev,
        [ipVersion]: {
          ...prev[ipVersion],
          snatRoamingInterface: errorData.error_params === 'ifname' ? (errorData.error || error.message) : '',
          snatIpNet: errorData.error_params === 'offset' ? (errorData.error || error.message) : ''
        }
      }));
      
      // Clear success messages on error
      setValidationSuccess(prev => ({
        ...prev,
        [ipVersion]: {
          snatIpNet: ''
        }
      }));
    }
  };

  const debouncedValidateSNATRoaming = (ipVersion, masterInterface, snatIpNet) => {
    // Clear any existing timeout for this IP version
    const currentTimeout = validationTimeouts.current[ipVersion];
    if (currentTimeout) {
      clearTimeout(currentTimeout);
    }

    // Set a new timeout
    const newTimeout = setTimeout(() => {
      validateSNATRoaming(ipVersion, masterInterface, snatIpNet);
      
      // Clear the timeout from ref
      validationTimeouts.current[ipVersion] = null;
    }, 500);

    // Store the timeout
    validationTimeouts.current[ipVersion] = newTimeout;
  };

  const handleChange = (path, value) => {
    const pathArray = path.split('.');
    setFormData(prev => {
      const newData = { ...prev };
      let current = newData;
      
      for (let i = 0; i < pathArray.length - 1; i++) {
        current[pathArray[i]] = { ...current[pathArray[i]] };
        current = current[pathArray[i]];
      }
      
      current[pathArray[pathArray.length - 1]] = value;
      
      // Trigger SNAT roaming validation when relevant fields change
      if (path.includes('snat.roamingMasterInterface') || path.includes('snat.snatIpNet')) {
        const ipVersion = path.startsWith('ipv4') ? 'ipv4' : 'ipv6';
        const ipConfig = newData[ipVersion];
        
        
        if (ipConfig.snat.enabled && 
            ipConfig.snat.roamingMasterInterfaceEnabled &&
            ipConfig.snat.roamingMasterInterface &&
            ipConfig.snat.snatIpNet) {
          debouncedValidateSNATRoaming(
            ipVersion,
            ipConfig.snat.roamingMasterInterface,
            ipConfig.snat.snatIpNet
          );
        } else {
          // Clear any pending validation timeout
          const currentTimeout = validationTimeouts.current[ipVersion];
          if (currentTimeout) {
            clearTimeout(currentTimeout);
            validationTimeouts.current[ipVersion] = null;
          }
          
          // Clear validation errors and success messages if SNAT roaming is disabled
          setValidationErrors(prev => ({
            ...prev,
            [ipVersion]: {
              snatRoamingInterface: '',
              snatIpNet: ''
            }
          }));
          setValidationSuccess(prev => ({
            ...prev,
            [ipVersion]: {
              snatIpNet: ''
            }
          }));
        }
      }
      
      return newData;
    });
  };

  const handleSave = async () => {
    // Validate required fields
    if (!formData.ipv4.enabled && !formData.ipv6.enabled) {
      setErrorDialog({ 
        open: true, 
        error: 'At least one of IPv4 or IPv6 must be enabled', 
        title: 'Validation Error' 
      });
      return;
    }

    setLoading(true);

    try {
      const data = {
        name: formData.name,
        dns: formData.dns ? formData.dns.split(',').map(s => s.trim()).filter(s => s) : null
      };

      // IPv4 configuration
      if (formData.ipv4.enabled) {
        data.ipv4 = {
          enabled: true,
          network: formData.ipv4.network,
          pseudoBridgeMasterInterface: formData.ipv4.pseudoBridgeMasterInterfaceEnabled ? 
            formData.ipv4.pseudoBridgeMasterInterface : null,
          routedNetworks: formData.ipv4.routedNetworks ? 
            formData.ipv4.routedNetworks.split('\n').map(s => s.trim()).filter(s => s) : null,
          routedNetworksFirewall: formData.ipv4.routedNetworksFirewall,
          snat: {
            enabled: formData.ipv4.snat.enabled,
            snatIpNet: formData.ipv4.snat.snatIpNet || null,
            snatExcludedNetwork: formData.ipv4.snat.snatExcludedNetwork || null,
            roamingMasterInterface: formData.ipv4.snat.roamingMasterInterfaceEnabled ? 
              formData.ipv4.snat.roamingMasterInterface : null
          }
        };
      } else {
        data.ipv4 = { enabled: false };
      }

      // IPv6 configuration
      if (formData.ipv6.enabled) {
        data.ipv6 = {
          enabled: true,
          network: formData.ipv6.network,
          pseudoBridgeMasterInterface: formData.ipv6.pseudoBridgeMasterInterfaceEnabled ? 
            formData.ipv6.pseudoBridgeMasterInterface : null,
          routedNetworks: formData.ipv6.routedNetworks ? 
            formData.ipv6.routedNetworks.split('\n').map(s => s.trim()).filter(s => s) : null,
          routedNetworksFirewall: formData.ipv6.routedNetworksFirewall,
          snat: {
            enabled: formData.ipv6.snat.enabled,
            snatIpNet: formData.ipv6.snat.snatIpNet || null,
            snatExcludedNetwork: formData.ipv6.snat.snatExcludedNetwork || null,
            roamingMasterInterface: formData.ipv6.snat.roamingMasterInterfaceEnabled ? 
              formData.ipv6.snat.roamingMasterInterface : null,
            roamingPseudoBridge: formData.ipv6.snat.roamingPseudoBridge
          }
        };
      } else {
        data.ipv6 = { enabled: false };
      }

      await onSave(data);
      onClose();
    } catch (err) {
      setErrorDialog({ 
        open: true, 
        error: err.message || 'Failed to save server', 
        title: 'Save Failed' 
      });
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!server) return;
    
    setLoading(true);

    try {
      await onDelete(server.id);
      onClose();
    } catch (err) {
      setErrorDialog({ 
        open: true, 
        error: err.message || 'Failed to delete server', 
        title: 'Delete Failed' 
      });
    } finally {
      setLoading(false);
    }
  };

  const renderIPSection = (ipVersion) => {
    const ip = formData[ipVersion];
    const isEnabled = ip.enabled;

    return (
      <Box sx={{ mb: 3 }}>
        <FormControlLabel
          control={
            <Checkbox
              checked={isEnabled}
              onChange={(e) => handleChange(`${ipVersion}.enabled`, e.target.checked)}
            />
          }
          label={<Typography variant="h6">{ipVersion.toUpperCase()}</Typography>}
        />

        <Box sx={{ ml: 4, opacity: isEnabled ? 1 : 0.5 }}>
          <TextField
            label="IP/Network"
            value={ip.network}
            onChange={(e) => handleChange(`${ipVersion}.network`, e.target.value)}
            disabled={!isEnabled}
            fullWidth
            sx={{ mb: 2 }}
            variant="outlined"
          />

          <Box sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
            <FormControlLabel
              control={
                <Checkbox
                  checked={ip.pseudoBridgeMasterInterfaceEnabled}
                  onChange={(e) => handleChange(`${ipVersion}.pseudoBridgeMasterInterfaceEnabled`, e.target.checked)}
                  disabled={!isEnabled}
                />
              }
              label=""
            />
            <TextField
              label="Pseudo-bridge master interface"
              value={ip.pseudoBridgeMasterInterface}
              onChange={(e) => handleChange(`${ipVersion}.pseudoBridgeMasterInterface`, e.target.value)}
              disabled={!isEnabled || !ip.pseudoBridgeMasterInterfaceEnabled}
              fullWidth
              variant="outlined"
            />
          </Box>

          <TextField
            label="Routed Networks"
            value={ip.routedNetworks}
            onChange={(e) => handleChange(`${ipVersion}.routedNetworks`, e.target.value)}
            disabled={!isEnabled}
            fullWidth
            multiline
            rows={3}
            sx={{ mb: 2 }}
            variant="outlined"
          />

          <FormControlLabel
            control={
              <Checkbox
                checked={ip.routedNetworksFirewall}
                onChange={(e) => handleChange(`${ipVersion}.routedNetworksFirewall`, e.target.checked)}
                disabled={!isEnabled}
              />
            }
            label="Block Non-Routed Network Packets"
          />

          <Box sx={{ mb: 1 }}>
            <FormControlLabel
              control={
                <Checkbox
                  checked={ip.snat.enabled}
                  onChange={(e) => handleChange(`${ipVersion}.snat.enabled`, e.target.checked)}
                  disabled={!isEnabled}
                />
              }
              label="SNAT"
            />
          </Box>

          <Box sx={{ ml: 4, opacity: (isEnabled && ip.snat.enabled) ? 1 : 0.5 }}>
            <TextField
              label="SNAT IP/Net"
              value={ip.snat.snatIpNet}
              onChange={(e) => handleChange(`${ipVersion}.snat.snatIpNet`, e.target.value)}
              disabled={!isEnabled || !ip.snat.enabled}
              fullWidth
              sx={{ mb: 2 }}
              variant="outlined"
              error={Boolean(validationErrors[ipVersion]?.snatIpNet)}
              helperText={
                validationErrors[ipVersion]?.snatIpNet || 
                validationSuccess[ipVersion]?.snatIpNet ||
                ''
              }
              FormHelperTextProps={{
                sx: {
                  color: validationSuccess[ipVersion]?.snatIpNet && !validationErrors[ipVersion]?.snatIpNet 
                    ? 'success.main' 
                    : undefined
                }
              }}
            />

            <TextField
              label="SNAT Excluded Network"
              value={ip.snat.snatExcludedNetwork}
              onChange={(e) => handleChange(`${ipVersion}.snat.snatExcludedNetwork`, e.target.value)}
              disabled={!isEnabled || !ip.snat.enabled}
              fullWidth
              sx={{ mb: 2 }}
              variant="outlined"
            />

            <Box sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={ip.snat.roamingMasterInterfaceEnabled}
                    onChange={(e) => handleChange(`${ipVersion}.snat.roamingMasterInterfaceEnabled`, e.target.checked)}
                    disabled={!isEnabled || !ip.snat.enabled}
                  />
                }
                label=""
              />
              <TextField
                label="SNAT Roaming master interface"
                value={ip.snat.roamingMasterInterface}
                onChange={(e) => handleChange(`${ipVersion}.snat.roamingMasterInterface`, e.target.value)}
                disabled={!isEnabled || !ip.snat.enabled || !ip.snat.roamingMasterInterfaceEnabled}
                fullWidth
                variant="outlined"
                error={Boolean(validationErrors[ipVersion]?.snatRoamingInterface)}
                helperText={validationErrors[ipVersion]?.snatRoamingInterface}
              />
            </Box>

            {ipVersion === 'ipv6' && (
              <FormControlLabel
                control={
                  <Checkbox
                    checked={ip.snat.roamingPseudoBridge}
                    onChange={(e) => handleChange(`${ipVersion}.snat.roamingPseudoBridge`, e.target.checked)}
                    disabled={!isEnabled || !ip.snat.enabled}
                  />
                }
                label="SNAT NETMAP pseudo-bridge"
              />
            )}
          </Box>
        </Box>
      </Box>
    );
  };

  return (
    <>
      <Dialog 
        open={open} 
        onClose={onClose}
        maxWidth="md"
        fullWidth
        PaperProps={{
          sx: { borderRadius: 2 }
        }}
      >
      <DialogTitle>{title || (isEdit ? 'Edit Server' : 'New Server')}</DialogTitle>
      
      <DialogContent>
        
        {warnings.length > 0 && (
          <Alert severity="warning" sx={{ mb: 2 }}>
            {warnings.map((warning, index) => (
              <div key={index}>{warning}</div>
            ))}
          </Alert>
        )}
        
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
          <TextField
            label="Name"
            value={formData.name}
            onChange={(e) => handleChange('name', e.target.value)}
            required
            fullWidth
            variant="outlined"
          />
          
          <TextField
            label="DNS"
            value={formData.dns}
            onChange={(e) => handleChange('dns', e.target.value)}
            fullWidth
            variant="outlined"
            placeholder="8.8.8.8, 1.1.1.1"
          />

          <Divider />

          {renderIPSection('ipv4')}
          
          <Divider />
          
          {renderIPSection('ipv6')}
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
          disabled={loading || warnings.length > 0}
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

export default ServerDialog;