import React, { createContext, useContext, useState, useEffect } from 'react';
import { createTheme, ThemeProvider as MuiThemeProvider } from '@mui/material/styles';
import stateManager from '../utils/stateManager';

const ThemeContext = createContext();

export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};

export const THEME_MODES = {
  AUTO: 'auto',
  LIGHT: 'light',
  DARK: 'dark'
};

const createAppTheme = (mode, prefersDarkMode) => {
  const isDark = mode === THEME_MODES.DARK || (mode === THEME_MODES.AUTO && prefersDarkMode);
  
  return createTheme({
    palette: {
      mode: isDark ? 'dark' : 'light',
      primary: {
        main: '#1976d2',
      },
      secondary: {
        main: '#d32f2f',
      },
      background: {
        default: isDark ? '#121212' : '#ffffff',
        paper: isDark ? '#1e1e1e' : '#ffffff',
        sidebar: isDark ? '#1a1a1a' : '#fafafa',
      },
      custom: {
        server: {
          background: isDark ? '#2d5a52' : '#4db6ac',
        },
        client: {
          background: isDark ? '#2d4a2d' : 'rgb(51, 109, 43)',
        },
        expanded: {
          background: isDark ? '#2a2a2a' : '#f5f5f5',
        },
        clientDetails: {
          background: isDark ? '#2a2a2a' : '#f9f9f9',
        }
      }
    },
  });
};

export const ThemeProvider = ({ children }) => {
  const [themeMode, setThemeMode] = useState(THEME_MODES.AUTO);
  const [prefersDarkMode, setPrefersDarkMode] = useState(
    window.matchMedia('(prefers-color-scheme: dark)').matches
  );

  // Listen for system theme changes
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = (e) => setPrefersDarkMode(e.matches);
    
    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  // Load theme preference from stateManager when initialized
  useEffect(() => {
    const loadThemePreference = () => {
      if (stateManager.initialized) {
        const savedTheme = stateManager.getThemeMode();
        if (savedTheme) {
          setThemeMode(savedTheme);
        }
      }
    };

    stateManager.onInitialized(loadThemePreference);
  }, []);

  const changeThemeMode = (newMode) => {
    setThemeMode(newMode);
    if (stateManager.initialized) {
      stateManager.setThemeMode(newMode);
    }
  };

  const theme = createAppTheme(themeMode, prefersDarkMode);

  const contextValue = {
    themeMode,
    setThemeMode: changeThemeMode,
    isDark: theme.palette.mode === 'dark',
    theme
  };

  return (
    <ThemeContext.Provider value={contextValue}>
      <MuiThemeProvider theme={theme}>
        {children}
      </MuiThemeProvider>
    </ThemeContext.Provider>
  );
};