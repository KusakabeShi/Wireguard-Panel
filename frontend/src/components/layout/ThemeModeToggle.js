import React, { useState, useEffect } from 'react';
import { 
  IconButton, 
  Tooltip
} from '@mui/material';
import { 
  Brightness4 as DarkModeIcon,
  Brightness7 as LightModeIcon,
  BrightnessAuto as AutoModeIcon
} from '@mui/icons-material';
import { useTheme, THEME_MODES } from '../../context/ThemeContext';

const ThemeModeToggle = () => {
  const { themeMode, setThemeMode, isDark } = useTheme();
  const [systemPrefersDark, setSystemPrefersDark] = useState(false);

  // Read system preference on init
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    setSystemPrefersDark(mediaQuery.matches);
    
    const handleChange = (e) => setSystemPrefersDark(e.matches);
    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  const handleClick = () => {
    // Cycle order based on system preference:
    // If system is light: auto -> dark -> light -> auto
    // If system is dark: auto -> light -> dark -> auto
    
    let nextMode;
    if (systemPrefersDark) {
      // System is dark: auto -> light -> dark -> auto
      switch (themeMode) {
        case THEME_MODES.AUTO:
          nextMode = THEME_MODES.LIGHT;
          break;
        case THEME_MODES.LIGHT:
          nextMode = THEME_MODES.DARK;
          break;
        case THEME_MODES.DARK:
          nextMode = THEME_MODES.AUTO;
          break;
        default:
          nextMode = THEME_MODES.AUTO;
      }
    } else {
      // System is light: auto -> dark -> light -> auto
      switch (themeMode) {
        case THEME_MODES.AUTO:
          nextMode = THEME_MODES.DARK;
          break;
        case THEME_MODES.DARK:
          nextMode = THEME_MODES.LIGHT;
          break;
        case THEME_MODES.LIGHT:
          nextMode = THEME_MODES.AUTO;
          break;
        default:
          nextMode = THEME_MODES.AUTO;
      }
    }
    
    setThemeMode(nextMode);
  };

  const getIcon = () => {
    if (themeMode === THEME_MODES.LIGHT) return <LightModeIcon />;
    if (themeMode === THEME_MODES.DARK) return <DarkModeIcon />;
    return <AutoModeIcon />; // Show auto icon for auto mode
  };

  const getTooltip = () => {
    switch (themeMode) {
      case THEME_MODES.LIGHT:
        return 'Theme: Light';
      case THEME_MODES.DARK:
        return `Theme: Dark`;
      case THEME_MODES.AUTO:
        return `Theme: Follow System`;
      default:
        return 'Change theme';
    }
  };

  return (
    <Tooltip title={getTooltip()}>
      <IconButton 
        color="inherit" 
        onClick={handleClick}
        sx={{ color: 'white' }}
      >
        {getIcon()}
      </IconButton>
    </Tooltip>
  );
};

export default ThemeModeToggle;