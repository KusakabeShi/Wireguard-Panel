import apiService from '../services/apiService';

// Global state management with cookie persistence
class StateManager {
  constructor() {
    this.panelID = null;
    this.initialized = false;
    this.initializationInProgress = false;
    // Local fallback storage when cookies aren't available
    this.localState = {
      sortOrder: ['name-a', 'lastHandshake-d', 'totalTraffic-d', 'enabled-a'],
      trafficDisplayMode: 'total', // Default traffic display mode
      clientsPerPage: 5, // Default clients per page
      uiState: {
        selectedInterfaceId: null,
        collapsedServers: {},
        expandedClients: {},
        serverPages: {}
      }
    };
  }

  // Proactively initialize by fetching service config
  async initializeFromAPI() {
    if (this.initialized || this.initializationInProgress) return;
    
    this.initializationInProgress = true;
    console.log('StateManager: Attempting to initialize from API...');
    
    try {
      const config = await apiService.getServiceConfig();
      console.log('StateManager: Got service config:', config);
      if (config.panelID) {
        console.log('StateManager: Initializing with panelID:', config.panelID);
        this.init(config.panelID);
      } else {
        console.warn('StateManager: No panelID in service config');
      }
    } catch (error) {
      console.warn('StateManager: Failed to fetch service config for initialization:', error);
      // Try to initialize with existing cookies without panelID prefix
      this.initializeFromExistingCookies();
    } finally {
      this.initializationInProgress = false;
    }
  }

  // Fallback: try to read cookies without panelID prefix
  initializeFromExistingCookies() {
    console.log('StateManager: Attempting fallback initialization from existing cookies');
    try {
      // Try to read cookies without prefix first
      const existingSortOrder = this.getCookieInternal('sortOrder', null);
      const existingTrafficDisplayMode = this.getCookieInternal('trafficDisplayMode', null);
      const existingClientsPerPage = this.getCookieInternal('clientsPerPage', null);
      const existingUIState = this.getCookieInternal('uiState', null);
      
      if (existingSortOrder || existingTrafficDisplayMode || existingClientsPerPage || existingUIState) {
        console.log('StateManager: Found existing cookies, using them');
        if (existingSortOrder) {
          this.localState.sortOrder = existingSortOrder;
        }
        if (existingTrafficDisplayMode) {
          this.localState.trafficDisplayMode = existingTrafficDisplayMode;
        }
        if (existingClientsPerPage) {
          this.localState.clientsPerPage = existingClientsPerPage;
        }
        if (existingUIState) {
          this.localState.uiState = existingUIState;
        }
        // Mark as partially initialized so components can use the state
        this.initialized = true;
        this.notifyInitialized();
      }
    } catch (error) {
      console.warn('StateManager: Failed to initialize from existing cookies:', error);
    }
  }

  // Initialize with panel ID from service config
  init(panelID) {
    if (this.initialized && this.panelID === panelID) {
      console.log('StateManager: Already initialized with same panelID, skipping');
      return;
    }
    
    if (this.initialized && this.panelID !== panelID) {
      console.log('StateManager: Reinitializing with new panelID:', panelID);
    }
    
    this.panelID = panelID;
    console.log('StateManager: Initializing with panelID:', panelID);
    
    // Debug: show all cookies
    console.log('StateManager: All document cookies:', document.cookie);
    
    // Sync local state from cookies when initializing, but preserve local changes
    const cookieSortOrder = this.getCookieInternal('sortOrder', null);
    const cookieTrafficDisplayMode = this.getCookieInternal('trafficDisplayMode', null);
    const cookieClientsPerPage = this.getCookieInternal('clientsPerPage', null);
    const cookieUIState = this.getCookieInternal('uiState', null);
    
    console.log('StateManager: Found cookies - sortOrder:', cookieSortOrder, 'trafficDisplayMode:', cookieTrafficDisplayMode, 'clientsPerPage:', cookieClientsPerPage, 'uiState:', cookieUIState);
    
    // Also try reading with the panelID prefix
    const prefixedSortOrder = this.getCookieDirectly(`${panelID}_sortOrder`, null);
    const prefixedTrafficDisplayMode = this.getCookieDirectly(`${panelID}_trafficDisplayMode`, null);
    const prefixedClientsPerPage = this.getCookieDirectly(`${panelID}_clientsPerPage`, null);
    const prefixedUIState = this.getCookieDirectly(`${panelID}_uiState`, null);
    
    console.log('StateManager: Found prefixed cookies - sortOrder:', prefixedSortOrder, 'trafficDisplayMode:', prefixedTrafficDisplayMode, 'clientsPerPage:', prefixedClientsPerPage, 'uiState:', prefixedUIState);
    
    if (prefixedSortOrder || prefixedTrafficDisplayMode || prefixedClientsPerPage || prefixedUIState) {
      console.log('StateManager: Using prefixed cookies');
      if (prefixedSortOrder) {
        this.localState.sortOrder = prefixedSortOrder;
      }
      if (prefixedTrafficDisplayMode) {
        this.localState.trafficDisplayMode = prefixedTrafficDisplayMode;
      }
      if (prefixedClientsPerPage) {
        this.localState.clientsPerPage = prefixedClientsPerPage;
      }
      if (prefixedUIState) {
        this.localState.uiState = prefixedUIState;
      }
    } else if (cookieSortOrder || cookieTrafficDisplayMode || cookieClientsPerPage || cookieUIState) {
      console.log('StateManager: Using non-prefixed cookies');
      if (cookieSortOrder) {
        this.localState.sortOrder = cookieSortOrder;
      }
      if (cookieTrafficDisplayMode) {
        this.localState.trafficDisplayMode = cookieTrafficDisplayMode;
      }
      if (cookieClientsPerPage) {
        this.localState.clientsPerPage = cookieClientsPerPage;
      }
      if (cookieUIState) {
        this.localState.uiState = cookieUIState;
      }
    }
    
    this.initialized = true;
    
    // Now sync any local state changes to cookies
    this.setCookieInternal('sortOrder', this.localState.sortOrder);
    this.setCookieInternal('trafficDisplayMode', this.localState.trafficDisplayMode);
    this.setCookieInternal('clientsPerPage', this.localState.clientsPerPage);
    this.setCookieInternal('uiState', this.localState.uiState);
    
    console.log('StateManager: Initialized with state:', this.localState);
    
    // Notify any listeners that we're now initialized
    this.notifyInitialized();
  }

  // Helper method to read cookies directly by name
  getCookieDirectly(cookieName, defaultValue = null) {
    const nameEQ = cookieName + "=";
    const ca = document.cookie.split(';');
    
    for (let i = 0; i < ca.length; i++) {
      let c = ca[i];
      while (c.charAt(0) === ' ') c = c.substring(1, c.length);
      if (c.indexOf(nameEQ) === 0) {
        try {
          return JSON.parse(decodeURIComponent(c.substring(nameEQ.length, c.length)));
        } catch (e) {
          console.warn('Failed to parse cookie:', cookieName, e);
          return defaultValue;
        }
      }
    }
    return defaultValue;
  }

  // Event system for initialization
  onInitialized(callback) {
    if (this.initialized) {
      callback(); // Call immediately if already initialized
    } else {
      if (!this.initListeners) this.initListeners = [];
      this.initListeners.push(callback);
      // No need to trigger initialization here since it's done at module load
    }
  }

  notifyInitialized() {
    if (this.initListeners) {
      this.initListeners.forEach(callback => callback());
      this.initListeners = [];
    }
  }

  // Cookie utilities with panelID prefix
  getCookieName(key) {
    if (!this.initialized) {
      return key; // Return key without prefix when not initialized
    }
    return `${this.panelID}_${key}`;
  }

  setCookieInternal(key, value, days = 30) {
    const cookieName = this.getCookieName(key);
    const expires = new Date();
    expires.setTime(expires.getTime() + (days * 24 * 60 * 60 * 1000));
    document.cookie = `${cookieName}=${encodeURIComponent(JSON.stringify(value))};expires=${expires.toUTCString()};path=/`;
  }

  getCookieInternal(key, defaultValue = null) {
    const cookieName = this.getCookieName(key);
    const nameEQ = cookieName + "=";
    const ca = document.cookie.split(';');
    
    for (let i = 0; i < ca.length; i++) {
      let c = ca[i];
      while (c.charAt(0) === ' ') c = c.substring(1, c.length);
      if (c.indexOf(nameEQ) === 0) {
        try {
          return JSON.parse(decodeURIComponent(c.substring(nameEQ.length, c.length)));
        } catch (e) {
          console.warn('Failed to parse cookie:', key, e);
          return defaultValue;
        }
      }
    }
    return defaultValue;
  }

  setCookie(key, value, days = 30) {
    if (!this.initialized) {
      // Store in local state as fallback
      if (key === 'sortOrder') {
        this.localState.sortOrder = value;
      } else if (key === 'trafficDisplayMode') {
        this.localState.trafficDisplayMode = value;
      } else if (key === 'clientsPerPage') {
        this.localState.clientsPerPage = value;
      } else if (key === 'uiState') {
        this.localState.uiState = value;
      }
      return;
    }
    this.setCookieInternal(key, value, days);
  }

  getCookie(key, defaultValue = null) {
    if (!this.initialized) {
      // Return from local state as fallback
      if (key === 'sortOrder') {
        return this.localState.sortOrder;
      } else if (key === 'trafficDisplayMode') {
        return this.localState.trafficDisplayMode;
      } else if (key === 'clientsPerPage') {
        return this.localState.clientsPerPage;
      } else if (key === 'uiState') {
        return this.localState.uiState;
      }
      return defaultValue;
    }
    return this.getCookieInternal(key, defaultValue);
  }

  // Sort order management
  getSortOrder() {
    return this.getCookie('sortOrder', ['name-a', 'lastHandshake-d', 'totalTraffic-d', 'enabled-a']);
  }

  setSortOrder(sortOrder) {
    this.setCookie('sortOrder', sortOrder);
  }

  // Traffic display mode management
  getTrafficDisplayMode() {
    return this.getCookie('trafficDisplayMode', 'total');
  }

  setTrafficDisplayMode(mode) {
    this.setCookie('trafficDisplayMode', mode);
  }

  // Clients per page management
  getClientsPerPage() {
    return this.getCookie('clientsPerPage', 5);
  }

  setClientsPerPage(perPage) {
    this.setCookie('clientsPerPage', perPage);
  }

  // Move selected sort method to first position
  updateSortOrder(selectedMethod, isAscending) {
    const currentOrder = this.getSortOrder();
    const newMethod = `${selectedMethod}-${isAscending ? 'a' : 'd'}`;
    
    // Remove the method if it exists
    const filteredOrder = currentOrder.filter(item => !item.startsWith(selectedMethod + '-'));
    
    // Add the new method to the beginning
    const newOrder = [newMethod, ...filteredOrder];
    
    // Keep only first 4 methods
    const trimmedOrder = newOrder.slice(0, 4);
    
    this.setSortOrder(trimmedOrder);
    return trimmedOrder;
  }

  // UI state persistence
  getUIState() {
    return this.getCookie('uiState', {
      selectedInterfaceId: null,
      collapsedServers: {}, // Store collapsed servers (servers expanded by default)
      expandedClients: {}, // Store expanded clients (clients collapsed by default)
      serverPages: {} // serverId -> pageNumber
    });
  }

  setUIState(state) {
    this.setCookie('uiState', state);
  }

  updateUIState(updates) {
    const currentState = this.getUIState();
    const newState = { ...currentState, ...updates };
    this.setUIState(newState);
    return newState;
  }

  // Server-specific page management
  getServerPage(serverId) {
    const uiState = this.getUIState();
    return uiState.serverPages[serverId] || 1;
  }

  setServerPage(serverId, page) {
    const uiState = this.getUIState();
    const newState = {
      ...uiState,
      serverPages: {
        ...uiState.serverPages,
        [serverId]: page
      }
    };
    this.setUIState(newState);
  }

  // Server expansion management (servers expanded by default, store collapsed)
  getCollapsedServers() {
    const uiState = this.getUIState();
    return new Set(Object.keys(uiState.collapsedServers || {}).filter(id => uiState.collapsedServers[id]));
  }

  setServerCollapsed(serverId, collapsed) {
    const uiState = this.getUIState();
    const newState = {
      ...uiState,
      collapsedServers: {
        ...uiState.collapsedServers,
        [serverId]: collapsed
      }
    };
    this.setUIState(newState);
    return this.getCollapsedServers();
  }

  isServerExpanded(serverId) {
    const collapsedServers = this.getCollapsedServers();
    return !collapsedServers.has(serverId); // Expanded by default unless collapsed
  }

  // Client expansion management (clients collapsed by default, store expanded)
  getExpandedClients() {
    const uiState = this.getUIState();
    return new Set(Object.keys(uiState.expandedClients || {}).filter(id => uiState.expandedClients[id]));
  }

  // Check if a specific client is expanded using composite key
  isClientExpanded(interfaceId, serverId, clientId) {
    const compositeKey = `${interfaceId}_${serverId}_${clientId}`;
    const expandedClients = this.getExpandedClients();
    return expandedClients.has(compositeKey);
  }

  setClientExpanded(interfaceId, serverId, clientId, expanded) {
    const compositeKey = `${interfaceId}_${serverId}_${clientId}`;
    const uiState = this.getUIState();
    const newState = {
      ...uiState,
      expandedClients: {
        ...uiState.expandedClients,
        [compositeKey]: expanded
      }
    };
    this.setUIState(newState);
    return this.getExpandedClients();
  }

  // Selected interface management
  getSelectedInterfaceId() {
    const uiState = this.getUIState();
    return uiState.selectedInterfaceId;
  }

  setSelectedInterfaceId(interfaceId) {
    return this.updateUIState({ selectedInterfaceId: interfaceId });
  }
}

// Create singleton instance
const stateManager = new StateManager();

// Initialize immediately when module loads
stateManager.initializeFromAPI();

export default stateManager;