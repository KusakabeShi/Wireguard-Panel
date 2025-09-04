class AuthService {
  getApiBaseUrl() {
    // Use injected runtime configuration, fallback to default
    return window.RUNTIME_API_PATH || './api';
  }

  async login(username, password) {
    const response = await fetch(`${this.getApiBaseUrl()}/service/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify({ username, password }),
    });

    if (!response.ok) {
      throw new Error('Login failed');
    }

    return response;
  }

  async logout() {
    const response = await fetch(`${this.getApiBaseUrl()}/service/logout`, {
      method: 'POST',
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Logout failed');
    }

    return response;
  }

  async getServiceConfig() {
    const response = await fetch(`${this.getApiBaseUrl()}/service/config`, {
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Failed to get service config');
    }

    return response.json();
  }

  async setServicePassword(currentPassword, password) {
    const response = await fetch(`${this.getApiBaseUrl()}/service/password`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify({ currentPassword, password }),
    });

    if (!response.ok) {
      throw new Error('Failed to set password');
    }

    return response;
  }
}

export default new AuthService();