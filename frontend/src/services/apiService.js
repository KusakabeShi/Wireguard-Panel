class ApiService {
  getApiBaseUrl() {
    // Use injected runtime configuration, fallback to default
    return window.RUNTIME_API_PATH || './api';
  }

  async request(endpoint, options = {}) {
    // debugger; // Uncomment this line to break here in any browser debugger
    const url = `${this.getApiBaseUrl()}${endpoint}`;
    const config = {
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    };

    const response = await fetch(url, config);

    if (response.status === 401) {
      throw new Error('Authentication required');
    }

    if (!response.ok) {
      // Try to parse error response
      let errorMessage = `HTTP error! status: ${response.status}`;
      
      try {
        const contentType = response.headers.get('content-type');
        if (contentType && contentType.includes('application/json')) {
          const errorData = await response.json();
          // Check for msg.error format or direct error field
          if (errorData.error) {
            errorMessage = errorData.error;
          } else if (errorData.message) {
            errorMessage = errorData.message;
          } else {
            errorMessage = JSON.stringify(errorData);
          }
        } else {
          const textError = await response.text();
          if (textError) {
            errorMessage = textError;
          }
        }
      } catch (parseError) {
        // If we can't parse the error, stick with the HTTP status message
        console.error('Failed to parse error response:', parseError);
      }
      
      throw new Error(errorMessage);
    }

    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      return response.json();
    } else if (contentType && contentType.includes('text/')) {
      return response.text();
    }
    
    return response;
  }

  // Interface endpoints
  async getInterfaces() {
    return this.request('/interfaces');
  }

  async getInterface(ifId) {
    return this.request(`/interfaces/${ifId}`);
  }

  async createInterface(interfaceData) {
    return this.request('/interfaces', {
      method: 'POST',
      body: JSON.stringify(interfaceData),
    });
  }

  async updateInterface(ifId, interfaceData) {
    return this.request(`/interfaces/${ifId}`, {
      method: 'PUT',
      body: JSON.stringify(interfaceData),
    });
  }

  async deleteInterface(ifId) {
    return this.request(`/interfaces/${ifId}`, {
      method: 'DELETE',
    });
  }

  async setInterfaceEnabled(ifId, enabled) {
    return this.request(`/interfaces/${ifId}/set-enable`, {
      method: 'POST',
      body: JSON.stringify({ enabled }),
    });
  }

  // Server endpoints
  async getServers(ifId) {
    return this.request(`/interfaces/${ifId}/servers`);
  }

  async getServer(ifId, serverId) {
    return this.request(`/interfaces/${ifId}/servers/${serverId}`);
  }

  async createServer(ifId, serverData) {
    return this.request(`/interfaces/${ifId}/servers`, {
      method: 'POST',
      body: JSON.stringify(serverData),
    });
  }

  async updateServer(ifId, serverId, serverData) {
    return this.request(`/interfaces/${ifId}/servers/${serverId}`, {
      method: 'PUT',
      body: JSON.stringify(serverData),
    });
  }

  async deleteServer(ifId, serverId) {
    return this.request(`/interfaces/${ifId}/servers/${serverId}`, {
      method: 'DELETE',
    });
  }

  async setServerEnabled(ifId, serverId, enabled) {
    return this.request(`/interfaces/${ifId}/servers/${serverId}/set-enable`, {
      method: 'POST',
      body: JSON.stringify({ enabled }),
    });
  }

  async moveServer(ifId, serverId, newInterfaceId) {
    return this.request(`/interfaces/${ifId}/servers/${serverId}/move`, {
      method: 'POST',
      body: JSON.stringify({ newInterfaceId }),
    });
  }

  // Client endpoints
  async getServerClients(ifId, serverId) {
    return this.request(`/interfaces/${ifId}/servers/${serverId}/clients`);
  }

  async getClient(ifId, serverId, clientId) {
    return this.request(`/interfaces/${ifId}/servers/${serverId}/clients/${clientId}`);
  }

  async createClient(ifId, serverId, clientData) {
    return this.request(`/interfaces/${ifId}/servers/${serverId}/clients`, {
      method: 'POST',
      body: JSON.stringify(clientData),
    });
  }

  async updateClient(ifId, serverId, clientId, clientData) {
    return this.request(`/interfaces/${ifId}/servers/${serverId}/clients/${clientId}`, {
      method: 'PUT',
      body: JSON.stringify(clientData),
    });
  }

  async deleteClient(ifId, serverId, clientId) {
    return this.request(`/interfaces/${ifId}/servers/${serverId}/clients/${clientId}`, {
      method: 'DELETE',
    });
  }

  async setClientEnabled(ifId, serverId, clientId, enabled) {
    return this.request(`/interfaces/${ifId}/servers/${serverId}/clients/${clientId}/set-enable`, {
      method: 'POST',
      body: JSON.stringify({ enabled }),
    });
  }

  async getClientState(ifId, serverId, clientId) {
    return this.request(`/interfaces/${ifId}/servers/${serverId}/clients/${clientId}/state`);
  }

  async getClientConfig(ifId, serverId, clientId) {
    return this.request(`/interfaces/${ifId}/servers/${serverId}/clients/${clientId}/config`);
  }

  // Service endpoints
  async getServiceConfig() {
    return this.request('/service/config');
  }

  async updatePassword(password) {
    return this.request('/service/password', {
      method: 'PUT',
      body: JSON.stringify({ password }),
    });
  }

  async logout() {
    return this.request('/service/logout', {
      method: 'POST',
    });
  }
}

export default new ApiService();