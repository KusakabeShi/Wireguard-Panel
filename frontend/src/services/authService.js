const API_BASE_URL = '/api';

class AuthService {
  async login(username, password) {
    const response = await fetch(`${API_BASE_URL}/service/login`, {
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
    const response = await fetch(`${API_BASE_URL}/service/logout`, {
      method: 'POST',
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Logout failed');
    }

    return response;
  }

  async getServiceConfig() {
    const response = await fetch(`${API_BASE_URL}/service/config`, {
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Failed to get service config');
    }

    return response.json();
  }

  async setServicePassword(password) {
    const response = await fetch(`${API_BASE_URL}/service/password`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      body: JSON.stringify({ password }),
    });

    if (!response.ok) {
      throw new Error('Failed to set password');
    }

    return response;
  }
}

export default new AuthService();