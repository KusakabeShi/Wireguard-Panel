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
  Checkbox,
  FormControlLabel,
  Typography,
  Divider
} from '@mui/material';

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
  const [error, setError] = useState('');
  const [warnings, setWarnings] = useState([]);
  const [loading, setLoading] = useState(false);

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
    setError('');
    setWarnings([]);
  }, [server, open]);

  useEffect(() => {
    checkValidation();
  }, [formData]);

  const checkValidation = () => {
    const newWarnings = [];

    // At least one IP version must be enabled
    if (!formData.ipv4.enabled && !formData.ipv6.enabled) {
      newWarnings.push('At least one of IPv4 or IPv6 must be enabled');
    }

    // Check IPv4 SNAT vs Pseudo-bridge conflict
    if (formData.ipv4.enabled && formData.ipv4.snat.enabled && formData.ipv4.pseudoBridgeMasterInterfaceEnabled) {
      newWarnings.push('IPv4 SNAT and Pseudo-bridge are mutually exclusive');
    }

    // Check IPv6 SNAT vs Pseudo-bridge conflict
    if (formData.ipv6.enabled && formData.ipv6.snat.enabled && formData.ipv6.pseudoBridgeMasterInterfaceEnabled) {
      newWarnings.push('IPv6 SNAT and Pseudo-bridge are mutually exclusive');
    }

    setWarnings(newWarnings);
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
      return newData;
    });
  };

  const handleSave = async () => {
    setError('');
    
    // Validate required fields
    if (!formData.ipv4.enabled && !formData.ipv6.enabled) {
      setError('At least one of IPv4 or IPv6 must be enabled');
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
      setError(err.message || 'Failed to save server');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!server) return;
    
    setError('');
    setLoading(true);

    try {
      await onDelete(server.id);
      onClose();
    } catch (err) {
      setError(err.message || 'Failed to delete server');
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
            label="Network"
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
              label="SNAT IP"
              value={ip.snat.snatIpNet}
              onChange={(e) => handleChange(`${ipVersion}.snat.snatIpNet`, e.target.value)}
              disabled={!isEnabled || !ip.snat.enabled}
              fullWidth
              sx={{ mb: 2 }}
              variant="outlined"
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
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}
        
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
  );
};

export default ServerDialog;