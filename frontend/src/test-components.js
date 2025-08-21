// Simple test to check if components can be imported without errors
import React from 'react';

// Test imports
import Header from './components/layout/Header';
import Sidebar from './components/layout/Sidebar';
import MainContent from './components/layout/MainContent';
import LoginDialog from './components/layout/LoginDialog';
import InterfaceView from './components/interface/InterfaceView';
import InterfaceDialog from './components/dialogs/InterfaceDialog';
import ServerDialog from './components/dialogs/ServerDialog';
import ClientDialog from './components/dialogs/ClientDialog';
import { AuthProvider } from './context/AuthContext';
import apiService from './services/apiService';
import authService from './services/authService';

console.log('All components imported successfully!');

// Test basic component creation
const TestComponents = () => {
  return React.createElement('div', { children: 'Components test passed!' });
};

export default TestComponents;