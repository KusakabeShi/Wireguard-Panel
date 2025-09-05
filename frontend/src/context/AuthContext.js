import React, { createContext, useContext, useState, useEffect } from 'react';
import authService from '../services/authService';
import stateManager from '../utils/stateManager';

const AuthContext = createContext();

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

export const AuthProvider = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [user, setUser] = useState(null);

  useEffect(() => {
    checkAuthStatus();
  }, []);

  const checkAuthStatus = async () => {
    try {
      const config = await authService.getServiceConfig();
      
      // Initialize state manager with panelID
      if (config.panelID) {
        stateManager.init(config.panelID);
      }
      
      setIsAuthenticated(true);
      setUser({ username: config.user });
    } catch (error) {
      setIsAuthenticated(false);
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  };

  const login = async (username, password) => {
    try {
      await authService.login(username, password);
      
      // After successful login, get service config and reinitialize state manager
      const config = await authService.getServiceConfig();
      if (config.panelID) {
        stateManager.init(config.panelID);
      }
      
      setIsAuthenticated(true);
      setUser({ username: config.user || username });
      return true;
    } catch (error) {
      setIsAuthenticated(false);
      setUser(null);
      throw error;
    }
  };

  const logout = async () => {
    try {
      await authService.logout();
    } catch (error) {
      console.error('Logout error:', error);
    } finally {
      setIsAuthenticated(false);
      setUser(null);
    }
  };

  const value = {
    isAuthenticated,
    isLoading,
    user,
    login,
    logout,
    checkAuthStatus,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};