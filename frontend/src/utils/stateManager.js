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
        collapsedServers: {}, // interfaceId -> { serverId: true }
        expandedClients: {}, // interfaceId -> serverId -> { clientId: true }
        serverPages: {} // interfaceId -> { serverId: pageNumber }
      },
      themeMode: 'auto' // auto, light, dark
    };
  }

  // Initialize immediately using window.WG_PANEL_ID
  initializeFromWindow() {
    if (this.initialized || this.initializationInProgress) return;
    
    this.initializationInProgress = true;
    console.log('StateManager: Attempting to initialize from window.WG_PANEL_ID...');
    
    try {
      // Get panelID from injected window variable (fallback to ReactDBG for development)
      const panelID = window.WG_PANEL_ID || 'ReactDBG';
      console.log('StateManager: Found panelID from window:', panelID);
      this.init(panelID);
    } catch (error) {
      console.warn('StateManager: Failed to get panelID from window, trying existing cookies:', error);
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
      const existingThemeMode = this.getCookieInternal('themeMode', null);
      
      if (existingSortOrder || existingTrafficDisplayMode || existingClientsPerPage || existingUIState || existingThemeMode) {
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
          // Migrate serverPages if needed
          if (existingUIState.serverPages) {
            existingUIState.serverPages = this.migrateServerPagesToNestedStructure(existingUIState.serverPages, null);
          }
          this.localState.uiState = this.mergeUIState(this.localState.uiState, existingUIState);
        }
        if (existingThemeMode) {
          this.localState.themeMode = existingThemeMode;
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
    const cookieThemeMode = this.getCookieInternal('themeMode', null);
    
    console.log('StateManager: Found cookies - sortOrder:', cookieSortOrder, 'trafficDisplayMode:', cookieTrafficDisplayMode, 'clientsPerPage:', cookieClientsPerPage, 'uiState:', cookieUIState, 'themeMode:', cookieThemeMode);
    
    // Also try reading with the panelID prefix
    const prefixedSortOrder = this.getCookieDirectly(`${panelID}_sortOrder`, null);
    const prefixedTrafficDisplayMode = this.getCookieDirectly(`${panelID}_trafficDisplayMode`, null);
    const prefixedClientsPerPage = this.getCookieDirectly(`${panelID}_clientsPerPage`, null);
    const prefixedUIState = this.getCookieDirectly(`${panelID}_uiState`, null);
    const prefixedThemeMode = this.getCookieDirectly(`${panelID}_themeMode`, null);
    
    console.log('StateManager: Found prefixed cookies - sortOrder:', prefixedSortOrder, 'trafficDisplayMode:', prefixedTrafficDisplayMode, 'clientsPerPage:', prefixedClientsPerPage, 'uiState:', prefixedUIState, 'themeMode:', prefixedThemeMode);
    
    if (prefixedSortOrder || prefixedTrafficDisplayMode || prefixedClientsPerPage || prefixedUIState || prefixedThemeMode) {
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
        // Migrate serverPages if needed
        if (prefixedUIState.serverPages) {
          prefixedUIState.serverPages = this.migrateServerPagesToNestedStructure(prefixedUIState.serverPages, null);
        }
        // Protect UI state from being overwritten with empty values before login
        this.localState.uiState = this.mergeUIState(this.localState.uiState, prefixedUIState);
      }
      if (prefixedThemeMode) {
        this.localState.themeMode = prefixedThemeMode;
      }
    } else if (cookieSortOrder || cookieTrafficDisplayMode || cookieClientsPerPage || cookieUIState || cookieThemeMode) {
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
        // Migrate serverPages if needed
        if (cookieUIState.serverPages) {
          cookieUIState.serverPages = this.migrateServerPagesToNestedStructure(cookieUIState.serverPages, null);
        }
        // Protect UI state from being overwritten with empty values before login
        this.localState.uiState = this.mergeUIState(this.localState.uiState, cookieUIState);
      }
      if (cookieThemeMode) {
        this.localState.themeMode = cookieThemeMode;
      }
    }
    
    this.initialized = true;
    
    // Now sync any local state changes to cookies
    this.setCookieInternal('sortOrder', this.localState.sortOrder);
    this.setCookieInternal('trafficDisplayMode', this.localState.trafficDisplayMode);
    this.setCookieInternal('clientsPerPage', this.localState.clientsPerPage);
    this.setCookieInternal('uiState', this.localState.uiState);
    this.setCookieInternal('themeMode', this.localState.themeMode);
    
    console.log('StateManager: Initialized with state:', this.localState);
    
    // Notify any listeners that we're now initialized
    this.notifyInitialized();
  }

  // Migrate old flat serverPages structure to nested structure
  migrateServerPagesToNestedStructure(serverPages, currentInterfaceId) {
    if (!serverPages || typeof serverPages !== 'object') {
      return {};
    }
    
    const keys = Object.keys(serverPages);
    if (keys.length === 0) {
      return {};
    }
    
    // Check if it's already in nested format (interface IDs point to objects)
    const hasInterfaceStructure = keys.every(key => 
      typeof serverPages[key] === 'object' && serverPages[key] !== null && !Array.isArray(serverPages[key])
    );
    
    if (hasInterfaceStructure) {
      // Already nested, but validate interface IDs look reasonable
      // Interface IDs should typically start with 'i' or be UUIDs, not server names
      const suspiciousKeys = keys.filter(key => {
        // If it doesn't look like a proper interface ID, it's likely migrated server data
        return !key.match(/^(i\d+|interface|[0-9a-f-]{8,})/i);
      });
      
      if (suspiciousKeys.length > 0) {
        console.log('StateManager: Found suspicious interface IDs that look like server IDs:', suspiciousKeys);
        console.log('StateManager: Clearing potentially corrupted serverPages data');
        return {}; // Clear corrupted data - will be rebuilt properly
      }
      
      return serverPages; // Already in correct format
    }
    
    // Flat structure detected: { serverId: pageNumber }
    console.log('StateManager: Detected flat serverPages structure, will be cleared by interface cleanup');
    // Don't migrate flat data without interface context - let cleanup handle it
    return {};
  }

  // Helper method to merge UI state while protecting from empty overwrites
  mergeUIState(currentState, newState) {
    if (!newState) return currentState;
    if (!currentState) return newState;
    
    // Merge states, but preserve non-empty collections
    const merged = { ...currentState, ...newState };
    
    // Protect collapsedServers from being overwritten with empty object
    if (currentState.collapsedServers && Object.keys(currentState.collapsedServers).length > 0) {
      if (!newState.collapsedServers || Object.keys(newState.collapsedServers).length === 0) {
        merged.collapsedServers = currentState.collapsedServers;
      }
    }
    
    // Protect expandedClients from being overwritten with empty object
    if (currentState.expandedClients && Object.keys(currentState.expandedClients).length > 0) {
      if (!newState.expandedClients || Object.keys(newState.expandedClients).length === 0) {
        merged.expandedClients = currentState.expandedClients;
      }
    }
    
    // Protect serverPages from being overwritten with empty object
    if (currentState.serverPages && Object.keys(currentState.serverPages).length > 0) {
      if (!newState.serverPages || Object.keys(newState.serverPages).length === 0) {
        merged.serverPages = currentState.serverPages;
      }
    }
    
    return merged;
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
      } else if (key === 'themeMode') {
        this.localState.themeMode = value;
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
      } else if (key === 'themeMode') {
        return this.localState.themeMode;
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
      collapsedServers: {}, // interfaceId -> { serverId: true } (servers expanded by default)
      expandedClients: {}, // interfaceId -> serverId -> { clientId: true } (clients collapsed by default)
      serverPages: {} // interfaceId -> { serverId: pageNumber }
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
  getServerPage(interfaceId, serverId) {
    const uiState = this.getUIState();
    return uiState.serverPages?.[interfaceId]?.[serverId] || 1;
  }

  setServerPage(interfaceId, serverId, page) {
    const uiState = this.getUIState();
    const serverPages = { ...uiState.serverPages };
    
    if (!serverPages[interfaceId]) {
      serverPages[interfaceId] = {};
    }
    
    if (page === 1) {
      // Remove page entry if it's the default page (1)
      const interfacePages = { ...serverPages[interfaceId] };
      delete interfacePages[serverId];
      if (Object.keys(interfacePages).length === 0) {
        delete serverPages[interfaceId];
      } else {
        serverPages[interfaceId] = interfacePages;
      }
    } else {
      serverPages[interfaceId] = { ...serverPages[interfaceId], [serverId]: page };
    }
    
    const newState = {
      ...uiState,
      serverPages
    };
    this.setUIState(newState);
  }

  // Server expansion management (servers expanded by default, store collapsed)
  getCollapsedServers(interfaceId) {
    const uiState = this.getUIState();
    const interfaceCollapsed = uiState.collapsedServers?.[interfaceId] || {};
    return new Set(Object.keys(interfaceCollapsed).filter(serverId => interfaceCollapsed[serverId]));
  }

  setServerCollapsed(interfaceId, serverId, collapsed) {
    const uiState = this.getUIState();
    const collapsedServers = { ...uiState.collapsedServers };
    
    if (!collapsedServers[interfaceId]) {
      collapsedServers[interfaceId] = {};
    }
    
    if (collapsed) {
      collapsedServers[interfaceId] = { ...collapsedServers[interfaceId], [serverId]: true };
    } else {
      const interfaceCollapsed = { ...collapsedServers[interfaceId] };
      delete interfaceCollapsed[serverId];
      if (Object.keys(interfaceCollapsed).length === 0) {
        delete collapsedServers[interfaceId];
      } else {
        collapsedServers[interfaceId] = interfaceCollapsed;
      }
    }
    
    const newState = {
      ...uiState,
      collapsedServers
    };
    this.setUIState(newState);
    return this.getCollapsedServers(interfaceId);
  }

  isServerExpanded(interfaceId, serverId) {
    const collapsedServers = this.getCollapsedServers(interfaceId);
    return !collapsedServers.has(serverId); // Expanded by default unless collapsed
  }

  // Client expansion management (clients collapsed by default, store expanded)
  getExpandedClients() {
    const uiState = this.getUIState();
    return uiState.expandedClients || {};
  }

  // Check if a specific client is expanded
  isClientExpanded(interfaceId, serverId, clientId) {
    const expandedClients = this.getExpandedClients();
    return !!expandedClients[interfaceId]?.[serverId]?.[clientId];
  }

  setClientExpanded(interfaceId, serverId, clientId, expanded) {
    const uiState = this.getUIState();
    const expandedClients = { ...uiState.expandedClients };
    
    if (!expandedClients[interfaceId]) {
      expandedClients[interfaceId] = {};
    }
    
    if (!expandedClients[interfaceId][serverId]) {
      expandedClients[interfaceId][serverId] = {};
    }
    
    if (expanded) {
      expandedClients[interfaceId][serverId] = { ...expandedClients[interfaceId][serverId], [clientId]: true };
    } else {
      const serverClients = { ...expandedClients[interfaceId][serverId] };
      delete serverClients[clientId];
      if (Object.keys(serverClients).length === 0) {
        delete expandedClients[interfaceId][serverId];
        // If interface has no more servers, remove it too
        if (Object.keys(expandedClients[interfaceId]).length === 0) {
          delete expandedClients[interfaceId];
        }
      } else {
        expandedClients[interfaceId][serverId] = serverClients;
      }
    }
    
    const newState = {
      ...uiState,
      expandedClients
    };
    this.setUIState(newState);
    return this.getExpandedClients();
  }

  // Clean up non-existent client entries from expanded clients for a specific interface
  cleanupExpandedClients(interfaceId, existingServers = [], existingClientsByServer = {}) {
    const uiState = this.getUIState();
    const expandedClients = { ...uiState.expandedClients };
    
    // Don't cleanup if we don't have any expanded clients to begin with
    if (!expandedClients || Object.keys(expandedClients).length === 0) {
      return false;
    }
    
    // Don't cleanup if existingServers is empty (likely before login or during loading)
    // Only cleanup when we have actual server data loaded
    if (!existingServers || existingServers.length === 0) {
      return false;
    }
    
    let hasChanges = false;
    
    // Get valid server IDs
    const validServerIds = new Set(existingServers.map(server => server.id));
    
    // Only clean up servers that belong to the current interface
    // Don't touch expanded clients for other interfaces
    if (expandedClients[interfaceId]) {
      const interfaceExpandedClients = { ...expandedClients[interfaceId] };
      
      for (const serverId in interfaceExpandedClients) {
        if (validServerIds.has(serverId)) {
          // Server exists - clean up invalid clients
          // Only cleanup client entries if we have client data for this server
          // Skip if existingClientsByServer[serverId] is undefined (clients not loaded yet)
          if (existingClientsByServer[serverId] !== undefined) {
            const serverClients = existingClientsByServer[serverId] || [];
            const validClientIds = new Set(serverClients.map(client => client.id));
            const expandedServerClients = { ...interfaceExpandedClients[serverId] };
            
            for (const clientId in expandedServerClients) {
              if (!validClientIds.has(clientId)) {
                delete expandedServerClients[clientId];
                hasChanges = true;
              }
            }
            
            // Update server clients or remove empty server entry
            if (Object.keys(expandedServerClients).length === 0) {
              delete interfaceExpandedClients[serverId];
            } else {
              interfaceExpandedClients[serverId] = expandedServerClients;
            }
          }
        } else {
          // Server no longer exists - remove it entirely
          delete interfaceExpandedClients[serverId];
          hasChanges = true;
        }
      }
      
      // Update the interface's expanded clients
      if (Object.keys(interfaceExpandedClients).length === 0) {
        delete expandedClients[interfaceId];
      } else {
        expandedClients[interfaceId] = interfaceExpandedClients;
      }
    }
    
    // Only update state if there were changes
    if (hasChanges) {
      const newState = {
        ...uiState,
        expandedClients
      };
      this.setUIState(newState);
    }
    
    return hasChanges;
  }

  // Clean up non-existent server entries from collapsed servers for a specific interface
  cleanupCollapsedServers(interfaceId, existingServers = []) {
    const uiState = this.getUIState();
    const collapsedServers = { ...uiState.collapsedServers };
    
    // Don't cleanup if we don't have any collapsed servers to begin with
    if (!collapsedServers || Object.keys(collapsedServers).length === 0) {
      return false;
    }
    
    // Don't cleanup if existingServers is empty (likely before login or during loading)
    if (!existingServers || existingServers.length === 0) {
      return false;
    }
    
    let hasChanges = false;
    
    // Get valid server IDs
    const validServerIds = new Set(existingServers.map(server => server.id));
    
    // Only clean up servers that belong to the current interface
    if (collapsedServers[interfaceId]) {
      const interfaceCollapsedServers = { ...collapsedServers[interfaceId] };
      
      for (const serverId in interfaceCollapsedServers) {
        if (!validServerIds.has(serverId)) {
          // Server no longer exists - remove it
          delete interfaceCollapsedServers[serverId];
          hasChanges = true;
        }
      }
      
      // Update the interface's collapsed servers
      if (Object.keys(interfaceCollapsedServers).length === 0) {
        delete collapsedServers[interfaceId];
      } else {
        collapsedServers[interfaceId] = interfaceCollapsedServers;
      }
    }
    
    // Only update state if there were changes
    if (hasChanges) {
      const newState = {
        ...uiState,
        collapsedServers
      };
      this.setUIState(newState);
    }
    
    return hasChanges;
  }

  // Clean up non-existent server entries from server pages for a specific interface
  cleanupServerPages(interfaceId, existingServers = []) {
    const uiState = this.getUIState();
    const serverPages = { ...uiState.serverPages };
    
    // Don't cleanup if we don't have any server pages to begin with
    if (!serverPages || Object.keys(serverPages).length === 0) {
      return false;
    }
    
    // Don't cleanup if existingServers is empty (likely before login or during loading)
    if (!existingServers || existingServers.length === 0) {
      return false;
    }
    
    let hasChanges = false;
    
    // Get valid server IDs
    const validServerIds = new Set(existingServers.map(server => server.id));
    
    // Only clean up servers that belong to the current interface
    if (serverPages[interfaceId]) {
      const interfaceServerPages = { ...serverPages[interfaceId] };
      
      for (const serverId in interfaceServerPages) {
        if (!validServerIds.has(serverId)) {
          // Server no longer exists - remove it
          delete interfaceServerPages[serverId];
          hasChanges = true;
        }
      }
      
      // Update the interface's server pages
      if (Object.keys(interfaceServerPages).length === 0) {
        delete serverPages[interfaceId];
      } else {
        serverPages[interfaceId] = interfaceServerPages;
      }
    }
    
    // Only update state if there were changes
    if (hasChanges) {
      const newState = {
        ...uiState,
        serverPages
      };
      this.setUIState(newState);
    }
    
    return hasChanges;
  }

  // Clean up all UI state for interfaces that no longer exist
  cleanupAllUIStateForDeletedInterfaces(existingInterfaces = []) {
    const uiState = this.getUIState();
    let hasChanges = false;
    
    // Don't cleanup if existingInterfaces is empty (likely before login or during loading)
    if (!existingInterfaces || existingInterfaces.length === 0) {
      return false;
    }
    
    // Get valid interface IDs
    const validInterfaceIds = new Set(existingInterfaces.map(iface => iface.id));
    
    // Clean up expandedClients
    if (uiState.expandedClients && Object.keys(uiState.expandedClients).length > 0) {
      const cleanExpandedClients = { ...uiState.expandedClients };
      for (const interfaceId in cleanExpandedClients) {
        if (!validInterfaceIds.has(interfaceId)) {
          console.log(`StateManager: Removing expandedClients for deleted interface: ${interfaceId}`);
          delete cleanExpandedClients[interfaceId];
          hasChanges = true;
        }
      }
      if (hasChanges) {
        uiState.expandedClients = cleanExpandedClients;
      }
    }
    
    // Clean up collapsedServers
    if (uiState.collapsedServers && Object.keys(uiState.collapsedServers).length > 0) {
      const cleanCollapsedServers = { ...uiState.collapsedServers };
      for (const interfaceId in cleanCollapsedServers) {
        if (!validInterfaceIds.has(interfaceId)) {
          console.log(`StateManager: Removing collapsedServers for deleted interface: ${interfaceId}`);
          delete cleanCollapsedServers[interfaceId];
          hasChanges = true;
        }
      }
      if (hasChanges) {
        uiState.collapsedServers = cleanCollapsedServers;
      }
    }
    
    // Clean up serverPages
    if (uiState.serverPages && Object.keys(uiState.serverPages).length > 0) {
      const cleanServerPages = { ...uiState.serverPages };
      for (const interfaceId in cleanServerPages) {
        if (!validInterfaceIds.has(interfaceId)) {
          console.log(`StateManager: Removing serverPages for deleted interface: ${interfaceId}`);
          delete cleanServerPages[interfaceId];
          hasChanges = true;
        }
      }
      if (hasChanges) {
        uiState.serverPages = cleanServerPages;
      }
    }
    
    // Update state if there were changes
    if (hasChanges) {
      console.log('StateManager: Updating UI state after interface cleanup');
      this.setUIState(uiState);
    }
    
    return hasChanges;
  }

  // Clean up expanded clients for interfaces that no longer exist
  cleanupExpandedClientsForDeletedInterfaces(existingInterfaces = []) {
    const uiState = this.getUIState();
    const expandedClients = { ...uiState.expandedClients };
    
    // Don't cleanup if we don't have any expanded clients to begin with
    if (!expandedClients || Object.keys(expandedClients).length === 0) {
      return false;
    }
    
    // Don't cleanup if existingInterfaces is empty (likely before login or during loading)
    if (!existingInterfaces || existingInterfaces.length === 0) {
      return false;
    }
    
    let hasChanges = false;
    
    // We need to check if any servers in expandedClients no longer belong to existing interfaces
    // Since we don't store interface info in expandedClients, we need to get all servers from all interfaces
    const loadAllServersFromInterfaces = async () => {
      try {
        const allValidServerIds = new Set();
        
        // Get all servers from all existing interfaces
        for (const interface_ of existingInterfaces) {
          try {
            const servers = await apiService.getServers(interface_.id);
            servers.forEach(server => allValidServerIds.add(server.id));
          } catch (error) {
            console.warn(`Failed to load servers for interface ${interface_.id}:`, error);
          }
        }
        
        // Remove servers that are no longer in any existing interface
        for (const interfaceId in expandedClients) {
          const interfaceExpandedClients = { ...expandedClients[interfaceId] };
          
          for (const serverId in interfaceExpandedClients) {
            if (!allValidServerIds.has(serverId)) {
              delete interfaceExpandedClients[serverId];
              hasChanges = true;
            }
          }
          
          // Update or remove the interface
          if (Object.keys(interfaceExpandedClients).length === 0) {
            delete expandedClients[interfaceId];
          } else {
            expandedClients[interfaceId] = interfaceExpandedClients;
          }
        }
        
        // Only update state if there were changes
        if (hasChanges) {
          const newState = {
            ...uiState,
            expandedClients
          };
          this.setUIState(newState);
        }
        
        return hasChanges;
      } catch (error) {
        console.warn('Failed to cleanup expanded clients for deleted interfaces:', error);
        return false;
      }
    };
    
    // Return a promise since this needs to be async
    return loadAllServersFromInterfaces();
  }

  // Selected interface management
  getSelectedInterfaceId() {
    const uiState = this.getUIState();
    return uiState.selectedInterfaceId;
  }

  setSelectedInterfaceId(interfaceId) {
    return this.updateUIState({ selectedInterfaceId: interfaceId });
  }

  // Theme mode management
  getThemeMode() {
    return this.getCookie('themeMode', 'auto');
  }

  setThemeMode(mode) {
    this.setCookie('themeMode', mode);
  }

  // Debug method to inspect cookie contents
  debugCookieContents() {
    const uiState = this.getUIState();
    console.log('=== StateManager Debug ===');
    console.log('Full uiState:', JSON.stringify(uiState, null, 2));
    console.log('serverPages structure:', JSON.stringify(uiState.serverPages, null, 2));
    console.log('collapsedServers structure:', JSON.stringify(uiState.collapsedServers, null, 2));
    console.log('expandedClients structure:', JSON.stringify(uiState.expandedClients, null, 2));
    console.log('Raw cookie value:', this.getCookieDirectly(`${this.panelID}_uiState`, 'NOT FOUND'));
    return uiState;
  }
}

// Create singleton instance
const stateManager = new StateManager();

// Initialize immediately when module loads using window.WG_PANEL_ID
stateManager.initializeFromWindow();

export default stateManager;